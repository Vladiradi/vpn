# Minikube (локальный Kubernetes)

Минимальный набор манифестов для запуска control plane и VPN-ноды в minikube.

## Требования

- [minikube](https://minikube.sigs.k8s.io/docs/start/)
- kubectl
- Docker (для сборки образов)

## 1. Запуск minikube

```bash
minikube start --driver=docker --cpus=4 --memory=8192
```

Включить ingress-nginx:

```bash
minikube addons enable ingress
```

## 2. Сборка образов внутри minikube

```bash
eval $(minikube docker-env)

# backend
cd ../../../backend
docker build -t vpn/panel:latest -f Dockerfile .
docker build -t vpn/agent:latest -f Dockerfile.agent .

# frontend
cd ../frontend/admin
docker build -t vpn/admin-ui:latest .
cd ../user
docker build -t vpn/user-ui:latest .
```

## 3. Применить манифесты

```bash
cd deploy/k8s/minikube
kubectl apply -f namespace.yaml
kubectl apply -f postgres.yaml
kubectl apply -f redis.yaml
kubectl apply -f panel-api.yaml
kubectl apply -f admin-ui.yaml
kubectl apply -f user-ui.yaml
kubectl apply -f ingress.yaml
```

Дождаться, пока `panel-api` поднимется:

```bash
kubectl wait --for=condition=ready pod -l app=panel-api -n vpn --timeout=120s
```

## 4. Создать первого админа

```bash
kubectl port-forward svc/panel-api 8080:8080 -n vpn
```

В другом терминале:

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"admin123"}'
```

## 5. Доступ к UI

Добавить в `/etc/hosts`:

```
$(minikube ip) vpn.local admin.vpn.local api.vpn.local
```

Открыть:

- User UI: http://vpn.local
- Admin UI: http://admin.vpn.local

## 6. VPN-нода (опционально)

1. В Admin UI создать ноду и скопировать enroll token.
2. Подставить токен в `vpn-node.yaml`:
   ```yaml
   stringData:
     NODE_TOKEN: "your-token-here"
   ```
3. Применить:
   ```bash
   kubectl apply -f xray-wrapper.yaml
   kubectl apply -f vpn-node.yaml
   ```

VPN будет доступен на порту `30443` minikube-ноды:

```bash
minikube service vpn-node -n vpn --url
```

> Примечание: в Kubernetes `node-agent` работает в `KUBERNETES_MODE=1` — конфиг пишется в shared volume, xray-sidecar перезагружает его автоматически. Сбор статистики через Docker API в Kubernetes не работает.
