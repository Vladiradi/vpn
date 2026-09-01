# Frontend

Два независимых Next.js-приложения:

- `user/` — личный кабинет пользователя.
- `admin/` — панель администратора.

## Стек

- Next.js 16 (App Router)
- TypeScript
- Tailwind CSS v4
- shadcn/ui (base-ui)
- TanStack Query
- Zustand
- react-hook-form + zod
- i18next (RU/EN)

## Запуск

```bash
cd frontend/user
npm install
cp .env.example .env.local
npm run dev
```

```bash
cd frontend/admin
npm install
cp .env.example .env.local
npm run dev
```

## Переменные окружения

```env
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_GOOGLE_CLIENT_ID=
NEXT_PUBLIC_TELEGRAM_BOT_USERNAME=
```

При локальной разработке Next.js проксирует запросы с `/api/*` на `NEXT_PUBLIC_API_URL`.
