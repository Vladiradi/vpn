# VPN-сервис: план реализации

**Дата:** 2026-08-31  
**Архитектура:** управляющий VPS (panel + DB) + VPN-ноды (XRay)  
**Бэкенд:** Go 1.27  
**Фронтенд:** Next.js (TypeScript)  
**База данных:** PostgreSQL 16 + Redis  
**Ingress:** Nginx Proxy Manager (контейнер)  
**Деплой:** Docker Compose / LXC → Kubernetes  
**Принцип:** всё работает в контейнерах, никакой установки ПО на хост VPS.

---

## Архитектура

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
│                          │                                             │
│         ◄────────────────┘──── исходящие HTTPS-соединения от нод       │
└──────────────┬─────────────────────────────────────────────────────────┘
               │
               │
               ▼
┌──────────────────────── VPS 2+ (VPN Nodes) ───────────────────────────┐
│                                                                        │
│  ┌───────────┐    ┌─────────────┐    ┌─────────────┐                  │
│  │node-agent │───▶│ XRay-core   │    │ decoy-site  │                  │
│  │ container │    │ VLESS+REALITY│   │             │                  │
│  └───────────┘    └──────┬──────┘    └─────────────┘                  │
│                          │                                            │
│                       :443/tcp                                         │
│                  Публичный IP ноды                                     │
└────────────────────────────────────────────────────────────────────────┘
```

**Ключевое решение:** на каждой VPN-ноде работает `node-agent` в контейнере. Агент сам подключается к панели по HTTPS/WebSocket, получает конфигурацию и управляет XRay. Не требуется ручная настройка WireGuard или открытие управляющих портов на ноде.

---

## Подтверждённый стек

```text
Control Plane (VPS 1) — всё в Docker:
  - Go 1.27 (Panel API)
  - Next.js 14/15 (Admin UI + User UI)
  - PostgreSQL 16
  - Redis 7
  - Nginx Proxy Manager (ingress + Let's Encrypt)
  - golang-migrate

VPN Node (VPS 2+) — всё в Docker:
  - node-agent (Go, контейнер)
  - XRay-core (VLESS + REALITY)
  - decoy-site (nginx, контейнер)

Auth:
  - Email + password (Argon2id + JWT)
  - Google OAuth 2.0
  - Telegram Login Widget

Deployment:
  - Docker Compose или LXC (сейчас)
  - Helm / Kubernetes (потом)
```

---

## Фазы реализации

### Фаза 0. Переосмысленный фундамент

Цель: разделить деплой на control + vpn, обновить схему БД.

- [x] Подключение к PostgreSQL и Redis.
- [x] Базовая авторизация email/password (Argon2id + JWT).
- [x] Переписать `docs/REALIZATION.md`.
- [x] Разделить `docker-compose` на `control` и `vpn`.
- [x] Обновить миграции: таблица `nodes`, поля OAuth/Telegram в `users`.
- [x] Обновить `Config` под два режима: `panel` и `agent`.
- [x] Nginx Proxy Manager вместо Caddy.

**Критерий готовности:** `docker compose -f deploy/docker-compose.control.yml up` поднимает панель с БД; `docker compose -f deploy/docker-compose.vpn.yml up` поднимает агента и XRay.

---

### Фаза 1. OAuth + Telegram auth

Цель: пользователь входит через email, Google или Telegram.

- [ ] Google OAuth 2.0 backend flow.
- [ ] Telegram Login Widget verification.
- [ ] Объединённый аккаунт: один email, несколько способов входа.
- [ ] Обновить frontend login page.

**Критерий готовности:** можно войти всеми тремя способами и получить JWT.

---

### Фаза 2. Управление нодами

Цель: админ добавляет VPN-ноды, агенты подключаются автоматически.

- [x] API админа: `POST /api/v1/nodes`, `GET /api/v1/nodes`, `DELETE /api/v1/nodes/:id`.
- [x] Генерация `enroll-token` для ноды.
- [x] WebSocket endpoint `/api/v1/nodes/stream` для агентов.
- [x] node-agent подключается к панели и регистрируется.
- [x] Панель видит статус ноды (online/offline).
- [ ] Генерация и отправка XRay конфига через WebSocket.

**Критерий готовности:** админ создаёт ноду в панели, запускает `docker compose` на VPN-VPS, нода появляется онлайн.

---

### Фаза 3. Генерация XRay конфига

Цель: панель управляет XRay на удалённой ноде.

- [ ] Генератор `config.json`: VLESS + REALITY + fallback на decoy.
- [ ] Хранение REALITY ключей в `nodes`.
- [ ] Отправка конфига агенту через WebSocket.
- [ ] Агент пишет конфиг и перезапускает XRay.

**Критерий готовности:** изменение настроек ноды в панели применяется на VPN-VPS в течение нескольких секунд.

---

### Фаза 4. Пользователи и устройства

Цель: админ и пользователь создают устройства и получают доступ.

- [ ] Админ создаёт пользователя.
- [ ] Пользователь создаёт устройство (выбирает ноду).
- [ ] UUID устройства добавляется в XRay на выбранной ноде.
- [ ] Генерация VLESS-URI.

**Критерий готовности:** создание устройства в панели = появление UUID в XRay.

---

### Фаза 5. Subscription и QR

Цель: пользователь скачивает конфиг удобным способом.

- [ ] `GET /sub/:token` с User-Agent роутингом.
- [ ] QR-код для конфига.
- [ ] Список устройств в пользовательском UI.

**Критерий готовности:** пользователь заходит в кабинет, копирует ссылку/QR, подключается.

---

### Фаза 6. Фронтенд

Цель: рабочие UI для админа и пользователя.

- [ ] Admin UI: ноды, пользователи, устройства, REALITY настройки.
- [ ] User UI: авторизация, устройства, QR, ссылка.
- [ ] i18n RU/EN.

---

### Фаза 7. Обсервбилити и бэкапы

- [ ] Метрики с нод в панель.
- [ ] Бэкап PostgreSQL.
- [ ] CI/CD GitHub Actions.

---

### Фаза 8. Лимиты и платежи

- [ ] Ограничение количества устройств.
- [ ] Тарифы и оплата.

---

## Структура репозитория

```text
vpn/
├── backend/                    # Go API + node-agent
│   ├── cmd/
│   │   ├── panel/              # Panel API main
│   │   └── agent/              # node-agent main
│   ├── internal/
│   │   ├── config/
│   │   ├── server/
│   │   ├── auth/               # JWT, Argon2, OAuth, Telegram
│   │   ├── users/
│   │   ├── nodes/              # управление VPN-нодами
│   │   ├── devices/
│   │   ├── subscriptions/
│   │   ├── xray/               # генерация XRay config.json
│   │   ├── agentstream/        # WebSocket для агентов
│   │   ├── storage/
│   │   └── middleware/
│   ├── migrations/
│   ├── Dockerfile
│   ├── Dockerfile.agent
│   └── go.mod
├── frontend/
│   ├── admin/                  # Admin UI
│   └── user/                   # User UI
├── deploy/
│   ├── docker-compose.control.yml   # VPS 1: panel + DB + NPM
│   ├── docker-compose.vpn.yml       # VPS 2+: agent + XRay + decoy
│   ├── helm/                        # future K8s
│   └── npm/                         # примеры конфигурации NPM
├── decoy-site/
└── docs/
    └── REALIZATION.md
```

---

## Безопасность

- Panel API только по HTTPS, домен + Let's Encrypt через Nginx Proxy Manager.
- node-agent подключается к панели исходящим HTTPS/WebSocket.
- XRay gRPC API слушает только `127.0.0.1` внутри VPN-ноды.
- Firewall VPN-VPS: только 22/TCP (SSH с ключом), 443/TCP, 443/UDP.
- OAuth/Telegram проверяются на backend.
- Секреты в `.env` / Docker secrets, не в git.
- Все сервисы запущены в контейнерах, на хосте только Docker.

---

## Быстрый старт

```bash
# VPS 1 — control plane
cd deploy
docker compose -f docker-compose.control.yml up --build

# VPS 2 — VPN node
cd deploy
docker compose -f docker-compose.vpn.yml up -d
```
