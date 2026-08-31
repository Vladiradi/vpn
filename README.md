# VPN Service

VPN-сервис с обфускацией трафика.

## Стек

- **Backend:** Go 1.23
- **Frontend:** Next.js (TypeScript)
- **Database:** PostgreSQL 16 + Redis
- **Deployment:** Docker Compose / k3s
- **VPN Core:** XRay-core (VLESS + REALITY)

## Быстрый старт

```bash
# Запуск backend + БД + Redis
cd deploy
docker compose up --build

# Проверка здоровья
curl http://localhost:8080/health
```

## Структура

- `backend/` — Go бэкенд (панель + node-agent)
- `frontend/` — Next.js фронтенд
- `deploy/` — Docker Compose и Helm
- `decoy-site/` — маскирующий сайт
- `docs/` — документация и планы

См. подробный план в `docs/REALIZATION.md`.
