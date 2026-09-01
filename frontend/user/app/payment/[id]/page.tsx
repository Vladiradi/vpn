"use client";

import { useParams, useRouter } from "next/navigation";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { paySubscription, listMySubscriptions, listOfferings } from "@/lib/api";
import { useAuthStore } from "@/lib/store";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { toast } from "sonner";

export default function PaymentPage() {
  const { t } = useTranslation();
  const router = useRouter();
  const params = useParams<{ id: string }>();
  const token = useAuthStore((s) => s.token);
  const id = params.id;

  const { data: subsData, isLoading: isLoadingSubs } = useQuery({
    queryKey: ["subscriptions", token],
    queryFn: () => listMySubscriptions(token!),
    enabled: !!token,
  });

  const { data: offeringsData, isLoading: isLoadingOfferings } = useQuery({
    queryKey: ["offerings"],
    queryFn: listOfferings,
  });

  const subscription = subsData?.subscriptions.find((s) => s.id === id);
  const offering = offeringsData?.offerings.find((o) => o.id === subscription?.offering_id);
  const isLoading = isLoadingSubs || isLoadingOfferings;

  const pay = useMutation({
    mutationFn: () => paySubscription(id, token!),
    onSuccess: () => {
      toast.success(t("payment.success"));
      router.push("/dashboard");
    },
    onError: (err: Error) => toast.error(err.message || t("errors.generic")),
  });

  if (isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center p-4">
        <Skeleton className="h-64 w-full max-w-md" />
      </div>
    );
  }

  if (!subscription) {
    return (
      <div className="flex flex-1 items-center justify-center p-4">
        <Card className="w-full max-w-md">
          <CardHeader>
            <CardTitle>{t("payment.title")}</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-muted-foreground">Подписка не найдена</p>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="flex flex-1 items-center justify-center p-4">
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <CardTitle>{t("payment.title")}</CardTitle>
          <CardDescription>{t("payment.stub")}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="text-3xl font-bold text-center">
            {t("payment.amount", { amount: offering ? offering.price_rubles : "—" })}
          </div>
          <Button
            className="w-full"
            size="lg"
            onClick={() => pay.mutate()}
            disabled={pay.isPending}
          >
            {pay.isPending ? "..." : t("payment.pay")}
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
