# VPN-сервис: план реализации

**Дата:** 2026-09-05
**Архитектура:** управляющий VPS (panel + DB) + VPN-ноды (XRay)
**Бэкенд:** Go 1.25
**Фронтенд:** Next.js (TypeScript)
**База данных:** PostgreSQL 16 + Redis
**Ingress:** Nginx Proxy Manager (контейнер)
**Деплой:** Docker Compose → Kubernetes (minikube)
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

**Ключевое решение:** на каждой VPN-ноде работает `node-agent` в контейнере. Агент сам подключается к панели по WSS, получает конфигурацию и управляет XRay. Не требуется ручная настройка WireGuard или открытие управляющих портов на ноде.

---

## Подтверждённый стек

```text
Control Plane (VPS 1) — всё в Docker:
  - Go 1.25 (Panel API)
  - Next.js (Admin UI + User UI)
  - PostgreSQL 16
  - Redis
  - Nginx Proxy Manager (ingress + Let's Encrypt)

VPN Node (VPS 2+) — всё в Docker:
  - node-agent (Go, контейнер)
  - XRay-core (VLESS + REALITY)
  - decoy-site (nginx, контейнер)

Auth:
  - Email + password (Argon2id + JWT)
  - Google OAuth 2.0
  - Telegram Login Widget

Deployment:
  - Docker Compose (сейчас)
  - Kubernetes / minikube (манифесты готовы)
```

---

## Фактическая структура репозитория

```text
vpn/
├── backend/                    # Go API + node-agent
│   ├── cmd/
│   │   ├── panel/              # Panel API main
│   │   └── agent/              # node-agent main
│   ├── internal/
│   │   ├── config/             # Config под режимы panel и agent
│   │   ├── server/             # HTTP-сервер и роутинг
│   │   ├── auth/               # JWT, Argon2, Google OAuth, Telegram
│   │   ├── users/              # пользователи и роли
│   │   ├── nodes/              # управление VPN-нодами, enroll-токены
│   │   ├── devices/            # устройства пользователей (VLESS UUID)
│   │   ├── offerings/          # страны/тарифы
│   │   ├── billing/            # подписки
│   │   ├── payments/           # Provider-интерфейс, stub, skeleton platega.io
│   │   ├── clientapps/         # ссылки на клиентские приложения
│   │   ├── subscription/       # VLESS URI, sing-box JSON, QR, /sub/{token}
│   │   ├── traffic/            # метрики трафика с нод
│   │   ├── xray/               # генерация XRay config.json, REALITY keys
│   │   ├── agentstream/        # WebSocket hub для связи с агентами
│   │   ├── dockerctl/          # Docker API client для агента
│   │   ├── storage/            # PostgreSQL + Redis + миграции (7 миграций)
│   │   └── middleware/
│   ├── Dockerfile
│   ├── Dockerfile.agent
│   └── go.mod                  # module github.com/vpn-project/backend, go 1.25
├── frontend/
│   ├── admin/                  # Admin UI: login, nodes, users, offerings, subscriptions
│   └── user/                   # User UI: login, offerings, payment, dashboard
├── deploy/
│   ├── docker-compose.control.yml   # VPS 1: panel + DB + Redis + NPM
│   ├── docker-compose.vpn.yml       # VPS 2+: agent + XRay + decoy
│   ├── backup/                      # контейнер pg_dump с ротацией
│   ├── log-server/                  # rsyslog + logrotate для всех сервисов
│   ├── npm/                         # инструкции по настройке NPM
│   └── k8s/minikube/                # namespace, postgres, redis, panel-api,
│                                    # admin-ui, user-ui, ingress, vpn-node, xray-wrapper
├── decoy-site/                 # маскирующий сайт
├── .github/workflows/          # ci.yml (сборка/тесты), deploy.yml (деплой по SSH)
├── .gitlab-ci.yml              # stages: test, build, deploy
└── docs/
    └── REALIZATION.md          # этот файл
```

---

## Реализовано (фазы 0–7)

- **Фундамент:** монорепо, конфигурация под режимы `panel`/`agent`, PostgreSQL + Redis, миграции, email/password auth (Argon2id + JWT), разделённые compose-файлы, Nginx Proxy Manager.
- **Управление нодами:** CRUD нод, enroll-токены, WebSocket endpoint `/api/v1/nodes/stream`, node-agent с heartbeat и реконнектом, статус online/offline в панели.
- **OAuth + Telegram:** Google OAuth 2.0 backend flow, Telegram Login Widget verification, объединение аккаунтов по email.
- **XRay конфиг:** генератор `config.json` (VLESS + REALITY + fallback на decoy), генерация keypair и short IDs, хранение ключей в `nodes`, отправка конфига агенту через WebSocket, агент пишет `/data/xray/config.json` и перезапускает контейнер XRay через Docker API.
- **Пользователи и устройства:** admin CRUD пользователей, устройства пользователя (авто-создание после оплаты, авто-назначение ноды по стране), UUID в XRay inbound clients, доставка конфига (VLESS URI, sing-box JSON, QR PNG, скачивание файла).
- **Тарифы и подписки:** `offerings` (страны, цена, срок, `is_available`), сидеры Германия (доступна) и Финляндия (заглушка), подписки со статусами, заглушка оплаты, `subscription_token` и публичный `GET /sub/{token}` с роутингом по User-Agent (sing-box JSON vs VLESS URI plain text).
- **Фронтенд:** User UI (логин Google/Telegram, выбор страны/тарифа, заглушка оплаты, кабинет с QR/ссылками/приложениями) и Admin UI (пользователи, подписки, ноды со статистикой, тарифы), i18n RU/EN.
- **Обсервбилити и бэкапы:** отдельный контейнер `log-server` (rsyslog + logrotate, ~1 GB лимит), метрики трафика (`traffic_stats`, агент шлёт `xray api statsquery` раз в 60 сек, admin endpoints `GET /api/v1/admin/traffic` и `/api/v1/admin/nodes/{id}/traffic`), контейнер `backup` (pg_dump с ротацией), GitHub Actions CI/CD + GitLab CI, манифесты Minikube.

## В работе (фаза 8)

- [x] Абстракция `payments.Provider` и stub-реализация (`backend/internal/payments/`).
- [x] Skeleton для [platega.io](https://platega.io/en) (`backend/internal/payments/platega.go`).
- [x] Конфигурация провайдера через env: `PAYMENT_PROVIDER`, `PLATEGA_API_KEY`, `PLATEGA_ENDPOINT`.
- [ ] Реальные HTTP-вызовы к platega.io API (нужен API ключ и документация).
- [ ] UI: страница оплаты с редиректом на платёжный шлюз и обработка callback/webhook.
- [ ] Редактирование цен и описаний тарифов в админке.
- [ ] Ограничение количества устройств на подписку.
- [ ] Уведомления об окончании подписки.

---

## API

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
GET  /api/v1/admin/dashboard        # обзор: пользователи, подписки, трафик
GET  /api/v1/admin/traffic          # статистика трафика
GET  /api/v1/admin/nodes/{id}/traffic  # трафик конкретной ноды

GET/POST          /api/v1/nodes
GET/DELETE        /api/v1/nodes/{id}
PUT               /api/v1/nodes/{id}/config

GET/POST          /api/v1/users
GET/PUT/DELETE    /api/v1/users/{id}
POST              /api/v1/users/{id}/status
POST              /api/v1/users/{id}/devices

GET/POST          /api/v1/offerings
PUT/DELETE        /api/v1/offerings/{id}

GET               /api/v1/subscriptions
GET/POST          /api/v1/subscriptions/{id}
POST              /api/v1/subscriptions/{id}/pay   # заглушка оплаты
```

### User

```text
GET  /api/v1/me                     # профиль + подписка + устройства

GET/POST/DELETE  /api/v1/me/devices
GET  /api/v1/me/devices/{id}/config  # VLESS URI + sing-box JSON
GET  /api/v1/me/devices/{id}/qr      # PNG QR-код
GET  /api/v1/me/devices/{id}/file?format=vless|sing-box

GET  /api/v1/client-apps             # ссылки на клиентские приложения
```

### Публичная подписка и агенты

```text
GET /sub/{token}                    # конфиг по токену подписки (без auth),
                                    # формат по User-Agent: sing-box JSON или VLESS URI
WS  /api/v1/nodes/stream            # WebSocket для node-agent
```

---

## База данных

Миграции: `backend/internal/storage/migrations/` (7 миграций: init, nodes+oauth, nullable password, node IP varchar, plans+subscriptions, subscription_token, traffic_stats).

Основные таблицы: `users`, `nodes` (REALITY ключи, токены), `offerings`, `subscriptions` (со `subscription_token`), `devices` (VLESS UUID), `traffic_stats`, `audit_log`.

---

## Безопасность

- Panel API только по HTTPS через Nginx Proxy Manager + Let's Encrypt.
- node-agent подключается к панели исходящим WSS, предъявляет токен.
- XRay gRPC API слушает только `127.0.0.1` внутри VPN-ноды.
- Firewall VPS 1: только `22/TCP` (SSH ключ), `80/TCP`, `443/TCP`, `81/TCP` (NPM admin).
- Firewall VPS 2+: только `22/TCP` (SSH ключ), `443/TCP`, `443/UDP`.
- OAuth/Telegram проверяются на backend.
- Секреты в `.env` / Docker secrets, не в git.
- Все сервисы запущены в контейнерах, на хосте только Docker.

---

## Быстрый старт

```bash
# VPS 1 — control plane
cd deploy
docker compose -f docker-compose.control.yml up -d

# VPS 2 — VPN node
cat > .env <<EOF
PANEL_URL=https://panel.example.com
NODE_TOKEN=<enroll-token-from-admin-panel>
NODE_PUBLIC_IP=<this-vps-public-ip>
EOF
docker compose -f docker-compose.vpn.yml up -d
```

Kubernetes (minikube): `deploy/k8s/minikube/` — `kubectl apply` в порядке: namespace → postgres, redis → panel-api → admin-ui, user-ui → ingress.

---

## Следующий шаг

Фазы 0–7 выполнены, готова база для фазы 8:

1. **Реальная интеграция platega.io** — HTTP-вызовы к API, callback/webhook, UI оплаты.
2. **Лимиты устройств и уведомления** окончания подписки.
3. **Расширение метрик** — графики в админке, online users, агрегация по времени.
4. **Документация по восстановлению из бэкапа** и ручному запуску backup-контейнера.
