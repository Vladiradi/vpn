# VPN Service

VPN-сервис с обфускацией трафика.

## Стек

- **Backend:** Go 1.23
- **Frontend:** Next.js (TypeScript)
- **Database:** PostgreSQL 16 + Redis
- **Deployment:** Docker Compose / Kubernetes (minikube)
- **VPN Core:** XRay-core (VLESS + REALITY)

## Быстрый старт

### Docker Compose

```bash
# Запуск backend + БД + Redis
cd deploy
docker compose -f docker-compose.control.yml up -d

# Проверка здоровья
curl http://localhost:8080/health
```

### Minikube

```bash
cd deploy/k8s/minikube
kubectl apply -f namespace.yaml
kubectl apply -f postgres.yaml -f redis.yaml -f panel-api.yaml
kubectl apply -f admin-ui.yaml -f user-ui.yaml -f ingress.yaml
```

Подробнее в `deploy/k8s/minikube/README.md`.

## Структура

- `backend/` — Go бэкенд (панель + node-agent)
- `frontend/` — Next.js фронтенд
- `deploy/` — Docker Compose и Helm
- `decoy-site/` — маскирующий сайт
- `docs/` — документация и планы

См. подробный план в `docs/REALIZATION.md`.
