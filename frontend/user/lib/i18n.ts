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
        download: "Скачать",
        active: "Активна",
        pending: "Ожидает оплаты",
        expired: "Истекла",
        cancelled: "Отменена",
        until: "Действует до {{date}}",
        qr: "QR-код для подключения",
        subscriptionLink: "Ссылка на подписку",
        copy: "Копировать",
        copied: "Скопировано",
        downloadVless: "Скачать VLESS URI",
        downloadSingbox: "Скачать sing-box JSON",
        apps: "Приложения",
        noDevice: "Устройство ещё не создано. Оформите подписку.",
      },
      instructions: {
        title: "Как подключиться",
        ios: {
          step1: "Установите Streisand или FoXray из App Store.",
          step2: "Откройте приложение и нажмите «Добавить».",
          step3: "Отсканируйте QR-код выше или вставьте ссылку на подписку.",
          step4: "Включите VPN в приложении.",
        },
        android: {
          step1: "Установите v2rayNG или Hiddify.",
          step2: "Откройте приложение и нажмите «+».",
          step3: "Импортируйте из буфера обмена или по ссылке на подписку.",
          step4: "Выберите сервер и нажмите подключить.",
        },
        macos: {
          step1: "Установите V2RayXS, V2RayU или Hiddify.",
          step2: "Откройте приложение и импортируйте VLESS URI.",
          step3: "Или вставьте ссылку на подписку в поддерживаемом приложении.",
          step4: "Нажмите Connect.",
        },
        windows: {
          step1: "Установите v2rayN или Hiddify.",
          step2: "Откройте приложение и нажмите «Импорт из буфера».",
          step3: "Или добавьте подписку по URL.",
          step4: "Выберите сервер и подключитесь.",
        },
        linux: {
          step1: "Установите Hiddify или NekoRay.",
          step2: "Откройте приложение и импортируйте конфигурацию.",
          step3: "Или вставьте ссылку на подписку.",
          step4: "Запустите подключение.",
        },
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
        download: "Download",
        active: "Active",
        pending: "Pending",
        expired: "Expired",
        cancelled: "Cancelled",
        until: "Valid until {{date}}",
        qr: "QR code for connection",
        subscriptionLink: "Subscription link",
        copy: "Copy",
        copied: "Copied",
        downloadVless: "Download VLESS URI",
        downloadSingbox: "Download sing-box JSON",
        apps: "Applications",
        noDevice: "No device yet. Please subscribe first.",
      },
      instructions: {
        title: "How to connect",
        ios: {
          step1: "Install Streisand or FoXray from the App Store.",
          step2: "Open the app and tap Add.",
          step3: "Scan the QR code above or paste the subscription link.",
          step4: "Turn on the VPN in the app.",
        },
        android: {
          step1: "Install v2rayNG or Hiddify.",
          step2: "Open the app and tap +.",
          step3: "Import from clipboard or via subscription link.",
          step4: "Select the server and connect.",
        },
        macos: {
          step1: "Install V2RayXS, V2RayU or Hiddify.",
          step2: "Open the app and import the VLESS URI.",
          step3: "Or paste the subscription link in a supported app.",
          step4: "Press Connect.",
        },
        windows: {
          step1: "Install v2rayN or Hiddify.",
          step2: "Open the app and click Import from clipboard.",
          step3: "Or add subscription by URL.",
          step4: "Select the server and connect.",
        },
        linux: {
          step1: "Install Hiddify or NekoRay.",
          step2: "Open the app and import the configuration.",
          step3: "Or paste the subscription link.",
          step4: "Start the connection.",
        },
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
