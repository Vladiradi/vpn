"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { listOfferings, createSubscription, type Offering } from "@/lib/api";
import { useAuthStore } from "@/lib/store";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { toast } from "sonner";

interface CountryGroup {
  countryCode: string;
  countryName: string;
  offerings: Offering[];
}

export default function OfferingsPage() {
  const { t } = useTranslation();
  const router = useRouter();
  const token = useAuthStore((s) => s.token);
  const [selectedCountry, setSelectedCountry] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState<string | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ["offerings"],
    queryFn: listOfferings,
  });

  const countries = useMemo<CountryGroup[]>(() => {
    if (!data?.offerings) return [];
    const map = new Map<string, CountryGroup>();
    for (const offering of data.offerings) {
      if (!map.has(offering.country_code)) {
        map.set(offering.country_code, {
          countryCode: offering.country_code,
          countryName: offering.country_name,
          offerings: [],
        });
      }
      map.get(offering.country_code)!.offerings.push(offering);
    }
    return Array.from(map.values());
  }, [data]);

  const selectedGroup = useMemo(
    () => countries.find((c) => c.countryCode === selectedCountry),
    [countries, selectedCountry]
  );

  const handleChoose = async (offering: Offering) => {
    if (!offering.is_available || !token) return;
    setIsSubmitting(offering.id);
    try {
      const sub = await createSubscription(offering.id, token);
      router.push(`/payment/${sub.id}`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("errors.generic"));
    } finally {
      setIsSubmitting(null);
    }
  };

  if (isLoading) {
    return (
      <div className="flex flex-1 flex-col items-center justify-center p-4">
        <Skeleton className="h-8 w-64 mb-8" />
        <div className="grid gap-4 w-full max-w-4xl md:grid-cols-2">
          <Skeleton className="h-40" />
          <Skeleton className="h-40" />
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-1 flex-col items-center p-4 py-8">
      <h1 className="text-2xl font-semibold mb-6">{t("offerings.title")}</h1>

      <div className="grid gap-4 w-full max-w-4xl md:grid-cols-2 lg:grid-cols-3">
        {countries.map((country) => {
          const available = country.offerings.some((o) => o.is_available);
          return (
            <Card
              key={country.countryCode}
              className={`cursor-pointer transition-shadow hover:shadow-md ${
                selectedCountry === country.countryCode ? "ring-2 ring-primary" : ""
              } ${!available ? "opacity-70" : ""}`}
              onClick={() => setSelectedCountry(country.countryCode)}
            >
              <CardHeader>
                <CardTitle>{country.countryName}</CardTitle>
                <CardDescription>
                  {available
                    ? country.offerings
                        .map((o) => t("offerings.speed", { speed: o.speed_mbps }))
                        .join(", ")
                    : t("offerings.unavailable")}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground">
                  {country.offerings.length} {t("offerings.duration").toLowerCase()}
                </p>
              </CardContent>
            </Card>
          );
        })}
      </div>

      {selectedGroup && (
        <div className="w-full max-w-4xl mt-8 space-y-4">
          <h2 className="text-xl font-medium">{selectedGroup.countryName}</h2>
          <div className="grid gap-4 md:grid-cols-2">
            {selectedGroup.offerings.map((offering) => (
              <Card key={offering.id}>
                <CardHeader>
                  <CardTitle>{offering.name}</CardTitle>
                  <CardDescription>
                    {t("offerings.speed", { speed: offering.speed_mbps })}
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="text-2xl font-semibold">
                    {t("offerings.price", { price: offering.price_rubles })}
                  </div>
                  <div className="text-sm text-muted-foreground">
                    {t("offerings.months", { count: offering.duration_months })}
                  </div>
                  <Button
                    className="w-full"
                    disabled={!offering.is_available || isSubmitting === offering.id}
                    onClick={() => handleChoose(offering)}
                  >
                    {isSubmitting === offering.id
                      ? "..."
                      : offering.is_available
                      ? t("offerings.choose")
                      : t("offerings.unavailable")}
                  </Button>
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
