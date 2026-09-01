"use client";

import { useTranslation } from "react-i18next";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

const platforms = ["ios", "android", "macos", "windows", "linux"];

export function ClientInstructions() {
  const { t } = useTranslation();

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("instructions.title")}</CardTitle>
      </CardHeader>
      <CardContent>
        <Tabs defaultValue="ios">
          <TabsList className="flex-wrap h-auto">
            {platforms.map((p) => (
              <TabsTrigger key={p} value={p} className="capitalize">
                {p}
              </TabsTrigger>
            ))}
          </TabsList>
          {platforms.map((p) => (
            <TabsContent key={p} value={p}>
              <ol className="list-decimal space-y-2 pl-5 text-sm text-muted-foreground">
                <li>{t(`instructions.${p}.step1`)}</li>
                <li>{t(`instructions.${p}.step2`)}</li>
                <li>{t(`instructions.${p}.step3`)}</li>
                <li>{t(`instructions.${p}.step4`)}</li>
              </ol>
            </TabsContent>
          ))}
        </Tabs>
      </CardContent>
    </Card>
  );
}
