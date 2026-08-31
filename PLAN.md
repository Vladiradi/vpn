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
- пользователи могут входить через email, Google или Telegram;
- весь стек работает в контейнерах (Docker/LXC), без установки ПО на хост;
- публичный трафик на панель идёт через Nginx Proxy Manager.

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

devices (
  id UUID PK,
  user_id UUID FK,
  node_id UUID FK,
  name VARCHAR,
  uuid UUID UNIQUE,                -- VLESS UUID
  protocol VARCHAR,
  created_at TIMESTAMPTZ,
  revoked_at TIMESTAMPTZ
)

subscriptions (
  id UUID PK,
  user_id UUID FK,
  token_hash VARCHAR UNIQUE,
  expires_at TIMESTAMPTZ,
  traffic_limit_bytes BIGINT,
  created_at TIMESTAMPTZ
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
GET  /api/v1/admin/dashboard

GET  /api/v1/nodes                  # список нод
POST /api/v1/nodes                  # создать ноду
GET  /api/v1/nodes/{id}             # одна нода
DELETE /api/v1/nodes/{id}           # удалить ноду

GET  /api/v1/users                  # пользователи
POST /api/v1/users                  # создать пользователя
POST /api/v1/users/{id}/devices     # создать устройство
```

### User

```text
GET  /api/v1/me                     # профиль
GET  /api/v1/me/devices             # мои устройства
POST /api/v1/me/devices             # добавить устройство
DELETE /api/v1/me/devices/{id}      # удалить устройство
```

### Subscriptions

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

### ⏳ Фаза 2. OAuth + Telegram auth

- [ ] Google OAuth 2.0 backend flow.
- [ ] Telegram Login Widget verification.
- [ ] Объединённый аккаунт по email.
- [ ] Обновление frontend login page.

### ⏳ Фаза 3. Генерация XRay конфига

- [ ] Генератор `config.json`: VLESS + REALITY + fallback на decoy.
- [ ] Хранение REALITY ключей в `nodes`.
- [ ] Отправка конфига агенту через WebSocket.
- [ ] Агент пишет конфиг и перезапускает XRay.

### ⏳ Фаза 4. Пользователи и устройства

- [ ] Админ создаёт пользователя.
- [ ] Пользователь создаёт устройство (выбирает ноду).
- [ ] UUID устройства добавляется в XRay.
- [ ] Генерация VLESS-URI.

### ⏳ Фаза 5. Subscription и QR

- [ ] `GET /sub/{token}` с User-Agent роутингом.
- [ ] QR-код для конфига.
- [ ] Список устройств в пользовательском UI.

### ⏳ Фаза 6. Фронтенд

- [ ] Admin UI: ноды, пользователи, устройства, REALITY.
- [ ] User UI: авторизация, устройства, QR, ссылка.
- [ ] i18n RU/EN.

### ⏳ Фаза 7. Обсервбилити и бэкапы

- [ ] Метрики с нод в панель.
- [ ] Бэкап PostgreSQL в S3.
- [ ] GitHub Actions CI/CD.

### ⏳ Фаза 8. Лимиты и платежи

- [ ] Ограничение количества устройств.
- [ ] Тарифы и оплата.

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
│   │   ├── subscriptions/
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

## 11. Следующий шаг

Реализовать **OAuth + Telegram авторизацию**:

1. Добавить Google OAuth 2.0 flow в `internal/auth`.
2. Добавить верификацию Telegram Login Widget.
3. Обновить таблицу `users` для связи нескольких способов входа с одним email.
4. Подготовить endpoints для фронтенда.

После этого — генерация XRay конфига и управление устройствами.
