"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useTranslation } from "react-i18next";
import { useAuthStore } from "@/lib/store";
import { googleLogin, telegramLogin, getMe, type TelegramLoginPayload } from "@/lib/api";
import { GoogleLoginButton } from "@/components/google-login-button";
import { TelegramLoginButton } from "@/components/telegram-login";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "sonner";

export default function LoginPage() {
  const { t } = useTranslation();
  const router = useRouter();
  const { setAuth } = useAuthStore();
  const [email, setEmail] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  const botUsername = process.env.NEXT_PUBLIC_TELEGRAM_BOT_USERNAME || "";
  const googleClientId = process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID || "";

  const handleSetAuth = async (tokens: { access_token: string; refresh_token: string; expires_in: number }) => {
    const me = await getMe(tokens.access_token);
    setAuth(tokens, me);
    if (me.role === "admin") {
      router.push("/dashboard");
    } else {
      router.push("/dashboard");
    }
  };

  const handleGoogleSuccess = async (accessToken: string) => {
    setIsLoading(true);
    try {
      const tokens = await googleLogin(accessToken);
      await handleSetAuth(tokens);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("errors.generic"));
    } finally {
      setIsLoading(false);
    }
  };

  const handleTelegramAuth = async (user: {
    id: number;
    first_name: string;
    username?: string;
    auth_date: number;
    hash: string;
  }) => {
    if (!email) {
      toast.error("Введите email для входа через Telegram");
      return;
    }
    setIsLoading(true);
    try {
      const payload: TelegramLoginPayload = {
        id: user.id,
        first_name: user.first_name,
        username: user.username || "",
        email,
        auth_date: String(user.auth_date),
        hash: user.hash,
      };
      const tokens = await telegramLogin(payload);
      await handleSetAuth(tokens);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("errors.generic"));
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="flex flex-1 items-center justify-center p-4">
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl">{t("login.title")}</CardTitle>
          <CardDescription>{t("login.subtitle")}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="email">{t("login.emailPlaceholder")}</Label>
            <Input
              id="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="user@example.com"
            />
          </div>

          {googleClientId && (
            <GoogleLoginButton
              label={t("login.google")}
              disabled={isLoading}
              onSuccess={handleGoogleSuccess}
              onError={() => toast.error(t("errors.generic"))}
            />
          )}

          {botUsername && (
            <div className="flex justify-center py-2">
              <TelegramLoginButton
                botUsername={botUsername}
                onAuth={handleTelegramAuth}
              />
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
