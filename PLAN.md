# VPN-сервис: финальный план

**Дата:** 2026-08-31  
**Архитектура:** управляющий VPS + VPN-ноды, всё в контейнерах  
**Ingress:** Nginx Proxy Manager  
**Бэкенд:** Go 1.27  
**Фронтенд:** Next.js (TypeScript)  
**База данных:** PostgreSQL 16 + Redis 7  

---

## 1. Цель проекта

Построить VPN-сервис с обфускацией трафика, где:

- управляющая панель и сами VPN-ноды находятся на разных VPS;
- на VPS с панелью не работает VPN, только admin/user UI, API и БД;
- VPN-ноды управляются автоматически через `node-agent` в контейнере;
- пользователи могут входить через Google или Telegram (email/password — только для первого админа);
- пользователь выбирает **только страну и тариф**, ноды распределяются автоматически и скрыты от него;
- весь стек работает в контейнерах (Docker/LXC), без установки ПО на хост;
- публичный трафик на панель идёт через Nginx Proxy Manager.

### 1.1. Пользовательский путь (MVP)

1. Авторизация через Google или Telegram.
2. Выбор **страны**: на старте две опции — **Германия** (доступна) и **Финляндия** (заглушка, недоступна).
3. Выбор **тарифа/срока** для выбранной страны: отображается стоимость.
4. **Заглушка оплаты** (реальная платёжная система — позже).
5. После оплаты пользователь попадает в **личный кабинет**, где автоматически создано устройство.
6. В личном кабинете:
   - **QR-код** для быстрого подключения.
   - **Файл конфигурации** (VLESS URI / sing-box JSON) для скачивания.
   - **Ссылки на приложения** по платформам: iOS, Android, macOS, Windows, Linux.
7. Подключение через любое совместимое приложение.

> Описания тарифов, точные цены и финальные регионы будут добавлены позже. Сейчас используются заглушки для демонстрации UI/UX.

### 1.2. Видимость нод

- Пользователь **не видит** отдельные ноды, IP-адреса и технические детали.
- Система автоматически назначает доступную ноду в выбранной стране.
- На MVP доступна только одна нода — Германия, Франкфурт.

---

## 2. Архитектура

```text
┌──────────────────────── VPS 1 (Control Plane) ────────────────────────┐
│  Домен: panel.example.com                                              │
│                                                                        │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐   │
│  │ Admin UI    │  │ User UI     │  │  Panel API  │  │ PostgreSQL  │   │
│  │  Next.js    │  │  Next.js    │  │   Go        │  │   + Redis   │   │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └─────────────┘   │
│         │                │                │                           │
│         └────────────────┴────────────────┘                           │
│                          │                                             │
│           Nginx Proxy Manager (контейнер)                              │
│          :80 :443 :81 — единственные публичные порты                   │
│                          │                                             │
│         ◄────────────────┘──── исходящие HTTPS/WebSocket от нод        │
└──────────────┬─────────────────────────────────────────────────────────┘
               │
               │
               ▼
┌──────────────────────── VPS 2+ (VPN Nodes) ───────────────────────────┐
│                                                                        │
│  ┌───────────┐    ┌─────────────┐    ┌─────────────┐                  │
│  │node-agent │───▶│ XRay-core   │    │ decoy-site  │                  │
│  │ container │    │ VLESS+REALITY│   │  (nginx)    │                  │
│  └───────────┘    └──────┬──────┘    └─────────────┘                  │
│                          │                                            │
│                       :443/tcp                                         │
│                  Публичный IP ноды                                     │
└────────────────────────────────────────────────────────────────────────┘
```

### Принципы

1. **Control plane / data plane разделены.**
   - VPS 1 никогда не обрабатывает пользовательский VPN-трафик.
   - VPS 2+ только принимают соединения пользователей и маскируются под обычный HTTPS.

2. **Автоматическое управление нодами.**
   - `node-agent` в контейнере на каждой VPN-ноде.
   - Агент сам подключается к панели по WebSocket, получает конфиг и пользователей.
   - Новый VPS = `docker compose up -d` + `NODE_TOKEN` из панели.

3. **Всё в контейнерах.**
   - На хосте только Docker (или LXC).
   - Нет ручной установки XRay, Nginx, WireGuard и т.д. на хост.

4. **Nginx Proxy Manager.**
   - Единая точка входа на панель.
   - Let's Encrypt, SSL, проксирование на Admin UI / User UI / Panel API.

---

## 3. Стек технологий

### Control Plane (VPS 1)

| Компонент | Технология |
|-----------|------------|
| Panel API | Go 1.27 |
| Admin UI | Next.js 14/15, TypeScript, Tailwind |
| User UI | Next.js 14/15, TypeScript, Tailwind |
| База данных | PostgreSQL 16 |
| Кэш / сессии | Redis 7 |
| Ingress + TLS | Nginx Proxy Manager (контейнер) |
| Миграции БД | golang-migrate |
| Auth | JWT (access + refresh), Argon2id |
| OAuth | Google OAuth 2.0 |
| Telegram | Telegram Login Widget |

### VPN Node (VPS 2+)

| Компонент | Технология |
|-----------|------------|
| node-agent | Go 1.27, контейнер |
| VPN core | XRay-core (VLESS + REALITY) |
| Decoy site | nginx, контейнер |
| Управление | Docker Compose |

### Инфраструктура

| Слой | Решение |
|------|---------|
| Контейнеризация | Docker Compose (сейчас), Kubernetes (потом) |
| Секреты | `.env` / Docker secrets / Kubernetes secrets |
| Бэкапы | pg_dump → S3-совместимое хранилище |
| CI/CD | GitHub Actions |
| Мониторинг | Prometheus + Grafana (опц.) |

---

## 4. Компоненты системы

### 4.1. Panel API (Go)

Модули:

- `auth` — JWT, Argon2id, Google OAuth, Telegram Login.
- `users` — пользователи и роли.
- `nodes` — управление VPN-нодами, генерация enroll-токенов.
- `devices` — устройства пользователей (UUID для XRay).
- `subscriptions` — подписки и ссылки.
- `xray` — генерация `config.json` для XRay.
- `agentstream` — WebSocket hub для связи с агентами.
- `storage` — PostgreSQL + Redis + миграции.

### 4.2. node-agent (Go)

Задачи:

1. При старте подключается к `wss://panel.example.com/api/v1/nodes/stream`.
2. Отправляет `hello` с `NODE_TOKEN`.
3. Получает конфигурацию ноды и список пользователей.
4. Генерирует `/etc/xray/config.json`.
5. Перезапускает контейнер XRay.
6. Раз в 10 секунд шлёт heartbeat.
7. При потере связи переподключается.

### 4.3. Nginx Proxy Manager

Роутинг:

| Домен | Куда |
|-------|------|
| `panel.example.com/api/*` | `panel-api:8080` |
| `panel.example.com/admin/*` | `admin-ui:3000` |
| `panel.example.com/*` | `user-ui:3000` |

Админка NPM: `http://<vps1-ip>:81`.

### 4.4. XRay + REALITY

- Порт `443/TCP`.
- REALITY маскирует трафик под легитимный HTTPS-сайт.
- Fallback для не-VLESS клиентов — `decoy-site`.

---

## 5. База данных

### Основные таблицы

```sql
users (
  id UUID PK,
  email VARCHAR UNIQUE NOT NULL,
  password_hash VARCHAR,           -- null для OAuth/Telegram
  role VARCHAR,                    -- admin | user
  status VARCHAR,                  -- active | blocked | deleted
  google_id VARCHAR UNIQUE,
  telegram_id BIGINT UNIQUE,
  created_at TIMESTAMPTZ,
  updated_at TIMESTAMPTZ
)

nodes (
  id UUID PK,
  name VARCHAR,
  hostname VARCHAR NOT NULL,
  public_ip VARCHAR,
  status VARCHAR,                  -- online | offline | syncing
  token_hash VARCHAR UNIQUE,       -- enroll token hash
  api_token VARCHAR UNIQUE,        -- agent auth token hash
  reality_dest VARCHAR,            -- маскирующий домен
  reality_private_key VARCHAR,
  reality_public_key VARCHAR,
  reality_short_ids TEXT[],
  decoy_host VARCHAR,
  last_seen_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ,
  updated_at TIMESTAMPTZ
)

offerings (
  id UUID PK,
  name VARCHAR NOT NULL,             -- "Германия / Франкфурт / 100 Мбит/с"
  duration_months INT NOT NULL,      -- 6, 12
  price_rubles INT NOT NULL,         -- 500, 1000
  speed_mbps INT NOT NULL DEFAULT 100,
  country_code VARCHAR NOT NULL,     -- "DE", "FI"
  country_name VARCHAR NOT NULL,     -- "Германия", "Финляндия"
  region VARCHAR,                    -- "Франкфурт", "Хельсинки"
  is_active BOOLEAN DEFAULT true,
  is_available BOOLEAN DEFAULT true, -- false для заглушек
  created_at TIMESTAMPTZ,
  updated_at TIMESTAMPTZ
)

subscriptions (
  id UUID PK,
  user_id UUID FK,
  offering_id UUID FK,
  status VARCHAR,                    -- active | expired | cancelled
  started_at TIMESTAMPTZ,
  expires_at TIMESTAMPTZ,            -- null для бесконечной тестовой подписки
  total_traffic_bytes BIGINT DEFAULT 0,
  created_at TIMESTAMPTZ,
  updated_at TIMESTAMPTZ
)

devices (
  id UUID PK,
  user_id UUID FK,
  subscription_id UUID FK,
  node_id UUID FK,
  name VARCHAR,
  uuid UUID UNIQUE,                  -- VLESS UUID
  protocol VARCHAR,
  created_at TIMESTAMPTZ,
  revoked_at TIMESTAMPTZ
)

traffic_stats (
  id UUID PK,
  device_id UUID FK,
  node_id UUID FK,
  ts TIMESTAMPTZ,
  uplink BIGINT,
  downlink BIGINT
)

audit_log (
  id UUID PK,
  actor_id UUID FK,
  action VARCHAR,
  payload JSONB,
  ts TIMESTAMPTZ
)
```

---

## 6. API

### Auth

```text
POST /api/v1/auth/register          # первый админ (email/password)
POST /api/v1/auth/login             # email/password
POST /api/v1/auth/google            # Google OAuth
POST /api/v1/auth/telegram          # Telegram Login
POST /api/v1/auth/refresh           # обновление access token
```

### Admin

```text
GET  /api/v1/admin/dashboard        # обзор: пользователи, активные подписки, трафик

GET  /api/v1/nodes                  # список нод
POST /api/v1/nodes                  # создать ноду
GET  /api/v1/nodes/{id}             # одна нода
DELETE /api/v1/nodes/{id}           # удалить ноду
PUT  /api/v1/nodes/{id}/config      # настроить REALITY/decoy и отправить конфиг агенту

GET  /api/v1/users                  # пользователи
POST /api/v1/users                  # создать пользователя
GET  /api/v1/users/{id}             # один пользователь
PUT  /api/v1/users/{id}             # обновить пользователя
DELETE /api/v1/users/{id}           # удалить пользователя
POST /api/v1/users/{id}/status      # активировать/заблокировать

GET  /api/v1/offerings              # доступные предложения (фильтр по стране/доступности)
POST /api/v1/offerings              # создать предложение (admin)
PUT  /api/v1/offerings/{id}         # обновить предложение (admin)
DELETE /api/v1/offerings/{id}       # удалить/деактивировать предложение (admin)

GET  /api/v1/subscriptions          # все подписки (admin)
GET  /api/v1/subscriptions/{id}     # одна подписка
POST /api/v1/subscriptions          # оформить подписку (выбор offering, статус pending)
POST /api/v1/subscriptions/{id}/pay # заглушка оплаты
```

### User

```text
GET  /api/v1/me                     # профиль + активная подписка + устройство

GET  /api/v1/offerings              # доступные предложения (страны + тарифы)

GET  /api/v1/me/devices             # мои устройства (создаётся после оплаты)
DELETE /api/v1/me/devices/{id}      # удалить/пересоздать устройство

GET /api/v1/me/devices/{id}/config  # VLESS URI + sing-box JSON
GET /api/v1/me/devices/{id}/qr      # PNG QR-код
GET /api/v1/me/devices/{id}/file?format=vless|sing-box  # скачать конфиг

GET /api/v1/client-apps             # ссылки на клиентские приложения по платформам
```

### Subscription public endpoint

```text
GET /sub/{token}                    # конфиг по подписке
```

### Agents

```text
WS /api/v1/nodes/stream             # WebSocket для node-agent
```

---

## 7. Безопасность

- **Panel API** только по HTTPS через Nginx Proxy Manager + Let's Encrypt.
- **node-agent** подключается исходящим WSS, предъявляет токен.
- **XRay gRPC API** слушает только `127.0.0.1` внутри VPN-ноды.
- **Firewall VPS 1**: только `22/TCP` (SSH ключ), `80/TCP`, `443/TCP`, `81/TCP` (NPM admin).
- **Firewall VPS 2+**: только `22/TCP` (SSH ключ), `443/TCP`, `443/UDP`.
- **SSH** — ключи, нестандартный порт, fail2ban.
- **OAuth/Telegram** проверяются на backend.
- **Секреты** в `.env` / Docker secrets, никогда в git.
- **Все сервисы в контейнерах**, на хосте только Docker.

---

## 8. Деплой

### VPS 1 — Control Plane

```bash
cd deploy
docker compose -f docker-compose.control.yml up -d
```

После запуска:

1. Открыть `http://<vps1-ip>:81`.
2. Войти в NPM (`admin@example.com` / `changeme`), сменить пароль.
3. Настроить Proxy Hosts для домена `panel.example.com` → `panel-api:8080`.
4. Запросить Let's Encrypt.

### VPS 2+ — VPN Node

```bash
# 1. Установить Docker
curl -fsSL https://get.docker.com | sh

# 2. Создать .env
 cat > .env <<EOF
PANEL_URL=https://panel.example.com
NODE_TOKEN=<enroll-token-from-admin-panel>
NODE_PUBLIC_IP=<this-vps-public-ip>
EOF

# 3. Запустить
docker compose -f docker-compose.vpn.yml up -d
```

---

## 9. Фазы реализации

### ✅ Фаза 0. Фундамент

- [x] Монорепо-структура.
- [x] Go backend с конфигурацией.
- [x] PostgreSQL + Redis + миграции.
- [x] Email/password auth (Argon2id + JWT).
- [x] Разделение `docker-compose.control.yml` и `docker-compose.vpn.yml`.
- [x] Nginx Proxy Manager в control plane.
- [x] `Config` под режимы `panel` и `agent`.

### ✅ Фаза 1. Управление нодами

- [x] API создания/списка/удаления нод.
- [x] Генерация `enroll-token`.
- [x] WebSocket endpoint `/api/v1/nodes/stream`.
- [x] node-agent подключается и держит соединение.
- [x] Статус ноды online/offline в панели.

### ✅ Фаза 2. OAuth + Telegram auth

- [x] Google OAuth 2.0 backend flow (`POST /api/v1/auth/google`).
- [x] Telegram Login Widget verification (`POST /api/v1/auth/telegram`).
- [x] Объединённый аккаунт по email (link Google/Telegram ID к существующему пользователю).
- [ ] Обновление frontend login page (отложено до фронтенд-фазы).

### ✅ Фаза 3. Генерация XRay конфига

- [x] Генератор `config.json`: VLESS + REALITY + fallback на decoy (`internal/xray`).
- [x] Генерация REALITY keypair и short IDs (`internal/xray`).
- [x] Хранение REALITY ключей в `nodes`.
- [x] Admin endpoint `PUT /api/v1/nodes/{id}/config` с автогенерацией недостающих ключей.
- [x] Отправка полного конфига агенту через WebSocket (`agentstream.Hub.PushConfig`).
- [x] Агент пишет `/data/xray/config.json` и перезапускает контейнер `xray` через Docker API (`internal/dockerctl`).

### ✅ Фаза 4. Пользователи, устройства и доставка конфига

- [x] Admin CRUD пользователей (`GET/POST /api/v1/users`, `GET/PUT/DELETE /api/v1/users/{id}`, `POST /api/v1/users/{id}/status`).
- [x] Пользователь создаёт/удаляет своё устройство (`GET/POST/DELETE /api/v1/me/devices`), выбирая ноду.
- [x] Admin может создать устройство для пользователя (`POST /api/v1/users/{id}/devices`).
- [x] UUID устройства добавляется в XRay inbound clients и пушится агенту при создании/отзыве.
- [x] Генерация VLESS-URI и sing-box JSON (`internal/subscription`).
- [x] Доставка конфига:
  - `GET /api/v1/me/devices/{id}/config` — VLESS URI + sing-box JSON.
  - `GET /api/v1/me/devices/{id}/qr` — PNG QR-код для VLESS URI.
  - `GET /api/v1/me/devices/{id}/file?format=vless|sing-box` — скачать `.txt` или `.json`.

### ✅ Фаза 5. Страны, тарифы-заглушки, подписки и пользовательский UI

- [x] Вернуть в `offerings` цену и срок; добавить флаг `is_available`.
- [x] Сидеры: Германия (доступна) и Финляндия (недоступна заглушка).
- [x] API предложений с фильтром по доступности.
- [x] Таблица `subscriptions` (user_id, offering_id, status, expires_at).
- [x] API подписок: оформление, заглушка оплаты, активация.
- [x] Автоматическое создание устройства после успешной оплаты.
- [x] Автоматическое назначение ноды по стране (MVP — только Германия).
- [x] API `/api/v1/client-apps` — ссылки на приложения по платформам.
- [x] User UI (`frontend/user/`): логин через Google/Telegram, выбор страны/тарифа, заглушка оплаты, личный кабинет с QR/файлом и ссылками на приложения.
- [x] Admin UI (`frontend/admin/`): пользователи, подписки, ноды, тарифы (offerings).

### ⏳ Фаза 6. Публичная подписка и i18n

- [ ] `GET /sub/{token}` с роутингом по User-Agent.
- [ ] i18n RU/EN.
- [ ] Инструкции по установке клиентов в UI.

### ⏳ Фаза 7. Обсервбилити и бэкапы

- [ ] Метрики с нод в панель (онлайн, трафик).
- [ ] Бэкап PostgreSQL в S3.
- [ ] GitHub Actions CI/CD.

### ⏳ Фаза 8. Реальные платежи и лимиты

- [ ] Интеграция платёжной системы (Stripe, ЮKassa, CryptoBot и т.п.).
- [ ] Редактирование цен и описаний тарифов в админке.
- [ ] Ограничение количества устройств на подписку.
- [ ] Уведомления об окончании подписки.

---

## 10. Структура репозитория

```text
vpn/
├── backend/                    # Go: panel API + node-agent
│   ├── cmd/
│   │   ├── panel/              # Panel API main
│   │   └── agent/              # node-agent main
│   ├── internal/
│   │   ├── config/
│   │   ├── server/
│   │   ├── auth/               # JWT, Argon2, OAuth, Telegram
│   │   ├── users/
│   │   ├── nodes/              # управление нодами
│   │   ├── devices/
│   │   ├── offerings/          # страны/тарифы-заглушки
│   │   ├── billing/            # подписки и оплата-заглушка
│   │   ├── clientapps/         # ссылки на клиентские приложения
│   │   ├── subscription/       # VLESS URI, sing-box JSON, QR
│   │   ├── dockerctl/          # Docker API client для агента
│   │   ├── xray/               # генерация XRay config.json
│   │   ├── agentstream/        # WebSocket для агентов
│   │   ├── storage/            # PostgreSQL + Redis + миграции
│   │   └── middleware/
│   ├── migrations/
│   ├── Dockerfile
│   ├── Dockerfile.agent
│   └── go.mod
├── frontend/
│   ├── admin/                  # Admin UI (Next.js)
│   └── user/                   # User UI (Next.js)
├── deploy/
│   ├── docker-compose.control.yml   # VPS 1: panel + DB + NPM
│   ├── docker-compose.vpn.yml       # VPS 2+: agent + XRay + decoy
│   ├── npm/                         # инструкции по настройке NPM
│   └── helm/                        # future K8s
├── decoy-site/                 # маскирующий сайт
├── PLAN.md                     # этот файл
├── README.md
└── docs/
    └── REALIZATION.md
```

---

## 11. Рекомендуемые бесплатные клиентские приложения

Форматы, которые отдаёт панель:

- **VLESS URI** — универсальная ссылка. Поддерживается почти всеми VLESS-клиентами.
- **sing-box JSON** — JSON-фрагмент outbound для sing-box-совместимых клиентов.
- **QR-код** — тот же VLESS URI в виде PNG.

| Платформа | Приложение | Лицензия | Поддержка VLESS+REALITY | Удобный формат импорта |
|-----------|------------|----------|-------------------------|------------------------|
| iOS | **Streisand** | бесплатно | ✅ | QR, sing-box JSON, ссылка |
| iOS | **FoXray** | бесплатно | ✅ | QR, VLESS URI, sing-box JSON |
| Android | **v2rayNG** | бесплатно (open-source) | ✅ | QR, VLESS URI, буфер обмена |
| Android | **NekoBox** | бесплатно (open-source) | ✅ | sing-box JSON, ссылка |
| Android | **Hiddify** | бесплатно | ✅ | sing-box JSON, ссылка, QR |
| Windows | **v2rayN** | бесплатно (open-source) | ✅ | QR, VLESS URI, буфер обмена |
| Windows | **Hiddify** | бесплатно | ✅ | sing-box JSON, ссылка |
| Windows | **NekoRay** | бесплатно (open-source, archived) | ✅ | VLESS URI |
| macOS | **V2RayXS** | бесплатно (open-source) | ✅ | VLESS URI |
| macOS | **V2RayU** | бесплатно (open-source) | ✅ | VLESS URI |
| macOS | **Hiddify** | бесплатно | ✅ | sing-box JSON, ссылка |
| Linux | **Hiddify** | бесплатно | ✅ | sing-box JSON, ссылка |
| Linux | **NekoRay** | бесплатно (open-source, archived) | ✅ | VLESS URI |

> **Примечание.** На мобильных устройствах проще всего отсканировать QR-код. На десктопе — скопировать VLESS URI или импортировать sing-box JSON. Для sing-box-совместимых приложений JSON-фрагмент из `/file?format=sing-box` вставляется в секцию `outbounds` конфига sing-box.

---

## 12. Следующий шаг

Реализовать **фронтенд** для фазы 5:

1. User UI: логин, выбор страны/срока, заглушка оплаты, личный кабинет с QR, файлом и ссылками на приложения.
2. Admin UI: список пользователей, подписок, устройств и нод.

Бэкенд уже готов:
- `GET /api/v1/public/offerings` — страны и сроки.
- `POST /api/v1/subscriptions` + `POST /api/v1/subscriptions/{id}/pay` — оформление и оплата-заглушка.
- `GET /api/v1/me/devices/{id}/config|qr|file` — конфиг/QR/файл.
- `GET /api/v1/client-apps` — ссылки на приложения.
- `GET /api/v1/admin/subscriptions` — подписки для админа.

После фронтенда — публичная подписка `/sub/{token}`, i18n, обсервбилити, CI/CD, и в конце — реальные платежи.
