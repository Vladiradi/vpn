import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import LanguageDetector from "i18next-browser-languagedetector";

const resources = {
  ru: {
    translation: {
      appName: "VPN",
      login: {
        title: "Войдите в VPN",
        subtitle: "Используйте Telegram или Google для входа",
        telegram: "Войти через Telegram",
        google: "Войти через Google",
        emailPlaceholder: "Email",
      },
      offerings: {
        title: "Выберите страну и тариф",
        country: "Страна",
        duration: "Срок",
        price: "{{price}} ₽",
        months: "{{count}} мес",
        speed: "{{speed}} Мбит/с",
        unavailable: "Недоступно",
        choose: "Выбрать",
      },
      payment: {
        title: "Оплата",
        amount: "Сумма к оплате: {{amount}} ₽",
        stub: "Это заглушка оплаты. Нажмите кнопку ниже, чтобы активировать подписку.",
        pay: "Оплатить",
        success: "Оплата прошла успешно!",
      },
      dashboard: {
        title: "Ваш VPN",
        subscription: "Подписка",
        active: "Активна",
        pending: "Ожидает оплаты",
        expired: "Истекла",
        cancelled: "Отменена",
        until: "Действует до {{date}}",
        qr: "QR-код для подключения",
        downloadVless: "Скачать VLESS URI",
        downloadSingbox: "Скачать sing-box JSON",
        apps: "Приложения",
        noDevice: "Устройство ещё не создано. Оформите подписку.",
      },
      errors: {
        generic: "Произошла ошибка. Попробуйте позже.",
      },
      nav: {
        logout: "Выйти",
      },
    },
  },
  en: {
    translation: {
      appName: "VPN",
      login: {
        title: "Sign in to VPN",
        subtitle: "Use Telegram or Google to sign in",
        telegram: "Sign in with Telegram",
        google: "Sign in with Google",
        emailPlaceholder: "Email",
      },
      offerings: {
        title: "Choose country and plan",
        country: "Country",
        duration: "Duration",
        price: "{{price}} ₽",
        months: "{{count}} mo",
        speed: "{{speed}} Mbps",
        unavailable: "Unavailable",
        choose: "Choose",
      },
      payment: {
        title: "Payment",
        amount: "Amount due: {{amount}} ₽",
        stub: "This is a payment stub. Press the button below to activate your subscription.",
        pay: "Pay",
        success: "Payment successful!",
      },
      dashboard: {
        title: "Your VPN",
        subscription: "Subscription",
        active: "Active",
        pending: "Pending",
        expired: "Expired",
        cancelled: "Cancelled",
        until: "Valid until {{date}}",
        qr: "QR code for connection",
        downloadVless: "Download VLESS URI",
        downloadSingbox: "Download sing-box JSON",
        apps: "Applications",
        noDevice: "No device yet. Please subscribe first.",
      },
      errors: {
        generic: "Something went wrong. Please try again later.",
      },
      nav: {
        logout: "Logout",
      },
    },
  },
};

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources,
    fallbackLng: "ru",
    interpolation: { escapeValue: false },
  });

export default i18n;
