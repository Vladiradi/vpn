# Admin UI

Панель администратора VPN-сервиса.

## Страницы

- `/login` — вход по email/password (только для admin).
- `/users` — список пользователей, блокировка/разблокировка.
- `/subscriptions` — все подписки, ручная активация.
- `/nodes` — управление VPN-нодами.
- `/offerings` — управление тарифами и странами.

## Запуск

```bash
npm install
cp .env.example .env.local
npm run dev
```

## Сборка

```bash
npm run build
```
