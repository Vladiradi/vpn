"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { QRCodeSVG } from "qrcode.react";
import { listMyDevices, listClientApps, listMySubscriptions, getDeviceConfig, getSubscriptionUrl, type ClientApp } from "@/lib/api";
import { useAuthStore } from "@/lib/store";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button, buttonVariants } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { getDeviceQRUrl, getDeviceFileUrl } from "@/lib/api";
import { ClientInstructions } from "@/components/client-instructions";

function groupApps(apps: ClientApp[]) {
  const groups: Record<string, ClientApp[]> = {};
  for (const app of apps) {
    if (!groups[app.platform]) groups[app.platform] = [];
    groups[app.platform].push(app);
  }
  return groups;
}

export default function DashboardPage() {
  const { t } = useTranslation();
  const router = useRouter();
  const { token, user, clearAuth } = useAuthStore();
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!token) {
      router.push("/login");
    }
  }, [token, router]);

  const devicesQuery = useQuery({
    queryKey: ["devices", token],
    queryFn: () => listMyDevices(token!),
    enabled: !!token,
  });

  const subsQuery = useQuery({
    queryKey: ["subscriptions", token],
    queryFn: () => listMySubscriptions(token!),
    enabled: !!token,
  });

  const appsQuery = useQuery({
    queryKey: ["client-apps"],
    queryFn: listClientApps,
  });

  const device = devicesQuery.data?.devices[0];
  const subscription = subsQuery.data?.subscriptions[0];

  const configQuery = useQuery({
    queryKey: ["device-config", device?.id, token],
    queryFn: () => getDeviceConfig(device!.id, token!),
    enabled: !!device && !!token,
  });

  if (!token) {
    return null;
  }

  const isLoading = devicesQuery.isLoading || subsQuery.isLoading || appsQuery.isLoading;

  return (
    <div className="flex flex-1 flex-col items-center p-4 py-8">
      <div className="w-full max-w-5xl space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-semibold">{t("dashboard.title")}</h1>
            <p className="text-muted-foreground">{user?.email}</p>
          </div>
          <Button variant="outline" onClick={() => { clearAuth(); router.push("/login"); }}>
            {t("nav.logout")}
          </Button>
        </div>

        {isLoading ? (
          <Skeleton className="h-96" />
        ) : !device ? (
          <Card>
            <CardContent className="py-12 text-center">
              <p className="text-muted-foreground">{t("dashboard.noDevice")}</p>
              <Button className="mt-4" onClick={() => router.push("/offerings")}>
                {t("offerings.title")}
              </Button>
            </CardContent>
          </Card>
        ) : (
          <>
            <Card>
              <CardHeader>
                <CardTitle>{t("dashboard.subscription")}</CardTitle>
                <CardDescription>
                  {subscription
                    ? `${t(`dashboard.${subscription.status}`)}${
                        subscription.expires_at
                          ? ` · ${t("dashboard.until", { date: new Date(subscription.expires_at).toLocaleDateString() })}`
                          : ""
                      }`
                    : "—"}
                </CardDescription>
              </CardHeader>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>{t("dashboard.qr")}</CardTitle>
                <CardDescription>{device.name}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="flex justify-center">
                  <img
                    src={getDeviceQRUrl(device.id)}
                    alt="QR"
                    className="rounded-lg border"
                    width={256}
                    height={256}
                  />
                </div>

                <div className="flex flex-col gap-2 sm:flex-row sm:justify-center">
                  <a
                    href={getDeviceFileUrl(device.id, "vless")}
                    download
                    className={buttonVariants({ variant: "outline" })}
                  >
                    {t("dashboard.downloadVless")}
                  </a>
                  <a
                    href={getDeviceFileUrl(device.id, "sing-box")}
                    download
                    className={buttonVariants({ variant: "outline" })}
                  >
                    {t("dashboard.downloadSingbox")}
                  </a>
                </div>

                {configQuery.data?.vless_uri && (
                  <div className="rounded-md bg-muted p-3 break-all text-xs font-mono">
                    {configQuery.data.vless_uri}
                  </div>
                )}
              </CardContent>
            </Card>

            {subscription?.subscription_token && (
              <Card>
                <CardHeader>
                  <CardTitle>{t("dashboard.subscriptionLink")}</CardTitle>
                  <CardDescription>
                    Подходит для приложений с поддержкой подписок
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="flex justify-center">
                    <QRCodeSVG
                      value={getSubscriptionUrl(subscription.subscription_token)}
                      size={192}
                      className="rounded-lg border"
                    />
                  </div>
                  <div className="flex items-center gap-2">
                    <input
                      readOnly
                      value={getSubscriptionUrl(subscription.subscription_token)}
                      className="flex-1 rounded-md border bg-muted px-3 py-2 text-xs font-mono"
                    />
                    <Button
                      variant="outline"
                      onClick={() => {
                        navigator.clipboard.writeText(getSubscriptionUrl(subscription.subscription_token!));
                        setCopied(true);
                        setTimeout(() => setCopied(false), 2000);
                      }}
                    >
                      {copied ? t("dashboard.copied") : t("dashboard.copy")}
                    </Button>
                  </div>
                </CardContent>
              </Card>
            )}

            <ClientInstructions />

            <Card>
              <CardHeader>
                <CardTitle>{t("dashboard.apps")}</CardTitle>
              </CardHeader>
              <CardContent>
                <Tabs defaultValue={Object.keys(groupApps(appsQuery.data?.apps || []))[0]}>
                  <TabsList className="flex-wrap h-auto">
                    {Object.keys(groupApps(appsQuery.data?.apps || [])).map((platform) => (
                      <TabsTrigger key={platform} value={platform} className="capitalize">
                        {platform}
                      </TabsTrigger>
                    ))}
                  </TabsList>
                  {Object.entries(groupApps(appsQuery.data?.apps || [])).map(([platform, apps]) => (
                    <TabsContent key={platform} value={platform}>
                      <div className="space-y-2">
                        {apps.map((app) => (
                          <div key={app.name} className="flex items-center justify-between rounded-md border p-3">
                            <div>
                              <p className="font-medium">{app.name}</p>
                              {app.notes && <p className="text-sm text-muted-foreground">{app.notes}</p>}
                            </div>
                            <a
                              href={app.url}
                              target="_blank"
                              rel="noopener noreferrer"
                              className={buttonVariants({ variant: "outline", size: "sm" })}
                            >
                              {t("dashboard.download")}
                            </a>
                          </div>
                        ))}
                      </div>
                    </TabsContent>
                  ))}
                </Tabs>
              </CardContent>
            </Card>
          </>
        )}
      </div>
    </div>
  );
}
