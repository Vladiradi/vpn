# Деплой

Все сервисы запускаются через Docker Compose. На хосте не устанавливается ПО, кроме Docker.

## Control Plane (VPS 1)

```bash
cd deploy
docker compose -f docker-compose.control.yml up -d
```

Сервисы:

- `log-server` — центральный syslog + logrotate для всех логов
- `postgres` — PostgreSQL 16
- `redis` — Redis 7
- `panel-api` — Go backend
- `npm` — Nginx Proxy Manager (ingress + TLS)

### Логи

Логи **не хранятся в PostgreSQL** и **не в Docker internal json-file**. Все контейнеры отправляют логи по syslog в отдельный контейнер `log-server`, который пишет их в named volume `logs` (`/var/log/vpn/<service>.log`).

Ротация настроена через `logrotate` внутри `log-server`:

- `size 100M`
- `rotate 10`
- `compress`

Общий лимит ~1 GB на все логи (можно увеличить количество файлов в `deploy/log-server/logrotate.conf`).

Просмотр логов:

```bash
docker exec deploy-log-server-1 tail -f /var/log/vpn/deploy-panel-api-1.log
docker exec deploy-log-server-1 tail -f /var/log/vpn/deploy-postgres-1.log
```

## VPN Node (VPS 2+)

```bash
cd deploy
docker compose -f docker-compose.vpn.yml up -d
```

Перед запуском создайте `.env`:

```env
PANEL_URL=https://panel.example.com
NODE_TOKEN=<enroll-token-from-admin-panel>
NODE_PUBLIC_IP=<this-vps-public-ip>
```

VPN-нода также отправляет логи в локальный `log-server`.

## CI/CD

Поддерживаются GitHub Actions и GitLab CI.

### GitHub Actions

- `.github/workflows/ci.yml` — запускается на push/PR: сборка и тесты backend, сборка user и admin UI.
- `.github/workflows/deploy.yml` — ручной деплой на VPS 1 по SSH.

Секреты:

- `VPS_HOST`
- `VPS_USER`
- `VPS_SSH_KEY`

### GitLab CI

- `.gitlab-ci.yml` — аналогичные stages: `test`, `build`, `deploy`.
- Деплой вручную (`when: manual`) на push в `main`.

Variables:

- `VPS_HOST`
- `VPS_USER`
- `VPS_SSH_KEY`

Деплой не использует S3. Код забирается с GitHub/GitLab прямо на сервер через `git pull`, затем пересобираются и перезапускаются контейнеры.

## Платежи

По умолчанию используется `PAYMENT_PROVIDER=stub` — оплата всегда успешна (для MVP).

Для подключения [platega.io](https://platega.io/en):

1. Зарегистрироваться и получить API ключ.
2. Установить переменные в `docker-compose.control.yml`:
   ```yaml
   PAYMENT_PROVIDER: "platega"
   PLATEGA_API_KEY: "your-api-key"
   PLATEGA_ENDPOINT: "https://api.platega.io/v1"
   ```
3. Реализовать реальные HTTP-вызовы в `backend/internal/payments/platega.go` (сейчас там skeleton).

Код плательщика изолирован за интерфейсом `payments.Provider`, поэтому добавить любой другой провайдер можно через новую реализацию интерфейса.
