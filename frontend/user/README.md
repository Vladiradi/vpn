# User UI

Личный кабинет пользователя VPN-сервиса.

## Страницы

- `/login` — вход через Google или Telegram.
- `/offerings` — выбор страны и тарифа.
- `/payment/[id]` — заглушка оплаты подписки.
- `/dashboard` — QR-код, файлы конфигурации и ссылки на клиентские приложения.

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
