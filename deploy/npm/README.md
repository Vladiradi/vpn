# Nginx Proxy Manager (NPM)

NPM запускается как контейнер в `docker-compose.control.yml`. Он принимает весь публичный трафик на VPS 1 и раздаёт его на сервисы панели.

## Настройка после первого запуска

1. Откройте `http://<vps1-ip>:81`.
2. Войдите:
   - Email: `admin@example.com`
   - Password: `changeme`
3. Сразу смените пароль.

## Proxy Hosts

### Panel API

- Domain Names: `panel.example.com`
- Scheme: `http`
- Forward Hostname / IP: `panel-api`
- Forward Port: `8080`
- Block Common Exploits: yes
- SSL: запросите Let's Encrypt для `panel.example.com`
- Force SSL: yes

### Admin UI

- Domain Names: `admin.example.com` (или `panel.example.com/admin`)
- Scheme: `http`
- Forward Hostname / IP: `admin-ui`
- Forward Port: `3000`

### User UI

- Domain Names: `panel.example.com` или отдельный `user.example.com`
- Scheme: `http`
- Forward Hostname / IP: `user-ui`
- Forward Port: `3000`

## Важно

- В production оставьте открытыми только порты NPM: `80`, `443`, `81`.
- Порт `8080` панели должен быть доступен только из `net-internal` или `127.0.0.1`.
- Nginx Proxy Manager сам обновляет Let's Encrypt сертификаты.
