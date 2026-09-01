"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuthStore } from "@/lib/store";
import { listOfferings, createOffering, deleteOffering, type Offering } from "@/lib/api";
import { Nav } from "@/components/nav";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { toast } from "sonner";

const emptyForm: Omit<Offering, "id" | "created_at" | "updated_at"> = {
  name: "",
  duration_months: 6,
  price_rubles: 500,
  speed_mbps: 100,
  country_code: "",
  country_name: "",
  region: "",
  is_active: true,
  is_available: true,
};

export default function OfferingsPage() {
  const router = useRouter();
  const token = useAuthStore((s) => s.token);
  const qc = useQueryClient();
  const [isOpen, setIsOpen] = useState(false);
  const [form, setForm] = useState(emptyForm);

  if (!token) {
    router.push("/login");
    return null;
  }

  const { data, isLoading } = useQuery({
    queryKey: ["offerings", token],
    queryFn: () => listOfferings(token),
    enabled: !!token,
  });

  const create = useMutation({
    mutationFn: () => createOffering(token, form),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["offerings", token] });
      setIsOpen(false);
      setForm(emptyForm);
      toast.success("Тариф создан");
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const remove = useMutation({
    mutationFn: (id: string) => deleteOffering(token, id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["offerings", token] }),
    onError: (err: Error) => toast.error(err.message),
  });

  const updateField = (field: keyof typeof form, value: string | number | boolean) => {
    setForm((prev) => ({ ...prev, [field]: value }));
  };

  return (
    <div className="flex flex-1 flex-col">
      <Nav />
      <main className="flex-1 p-4">
        <div className="mx-auto max-w-7xl space-y-4">
          <div className="flex items-center justify-between">
            <h1 className="text-2xl font-semibold">Тарифы</h1>
            <Dialog open={isOpen} onOpenChange={setIsOpen}>
              <DialogTrigger asChild>
                <Button>Добавить тариф</Button>
              </DialogTrigger>
              <DialogContent className="max-w-lg">
                <DialogHeader>
                  <DialogTitle>Новый тариф</DialogTitle>
                </DialogHeader>
                <div className="grid gap-4 py-2 sm:grid-cols-2">
                  <div className="space-y-2">
                    <Label>Название</Label>
                    <Input value={form.name} onChange={(e) => updateField("name", e.target.value)} />
                  </div>
                  <div className="space-y-2">
                    <Label>Страна (код)</Label>
                    <Input value={form.country_code} onChange={(e) => updateField("country_code", e.target.value)} />
                  </div>
                  <div className="space-y-2">
                    <Label>Страна (название)</Label>
                    <Input value={form.country_name} onChange={(e) => updateField("country_name", e.target.value)} />
                  </div>
                  <div className="space-y-2">
                    <Label>Регион</Label>
                    <Input value={form.region} onChange={(e) => updateField("region", e.target.value)} />
                  </div>
                  <div className="space-y-2">
                    <Label>Срок (мес)</Label>
                    <Input type="number" value={form.duration_months} onChange={(e) => updateField("duration_months", parseInt(e.target.value) || 0)} />
                  </div>
                  <div className="space-y-2">
                    <Label>Цена (₽)</Label>
                    <Input type="number" value={form.price_rubles} onChange={(e) => updateField("price_rubles", parseInt(e.target.value) || 0)} />
                  </div>
                  <div className="space-y-2">
                    <Label>Скорость (Мбит/с)</Label>
                    <Input type="number" value={form.speed_mbps} onChange={(e) => updateField("speed_mbps", parseInt(e.target.value) || 0)} />
                  </div>
                  <div className="flex items-center gap-4">
                    <label className="flex items-center gap-2">
                      <input type="checkbox" checked={form.is_active} onChange={(e) => updateField("is_active", e.target.checked)} />
                      Активен
                    </label>
                    <label className="flex items-center gap-2">
                      <input type="checkbox" checked={form.is_available} onChange={(e) => updateField("is_available", e.target.checked)} />
                      Доступен
                    </label>
                  </div>
                  <Button className="sm:col-span-2" onClick={() => create.mutate()} disabled={create.isPending}>
                    Создать
                  </Button>
                </div>
              </DialogContent>
            </Dialog>
          </div>

          <Card>
            <CardContent className="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Название</TableHead>
                    <TableHead>Страна</TableHead>
                    <TableHead>Срок</TableHead>
                    <TableHead>Цена</TableHead>
                    <TableHead>Скорость</TableHead>
                    <TableHead>Статус</TableHead>
                    <TableHead>Действия</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {isLoading ? (
                    <TableRow>
                      <TableCell colSpan={7}>Загрузка...</TableCell>
                    </TableRow>
                  ) : (
                    data?.offerings.map((offering) => (
                      <TableRow key={offering.id}>
                        <TableCell>{offering.name}</TableCell>
                        <TableCell>{offering.country_name}</TableCell>
                        <TableCell>{offering.duration_months} мес</TableCell>
                        <TableCell>{offering.price_rubles} ₽</TableCell>
                        <TableCell>{offering.speed_mbps} Мбит/с</TableCell>
                        <TableCell>
                          {offering.is_active ? "активен" : "неактивен"} /{" "}
                          {offering.is_available ? "доступен" : "недоступен"}
                        </TableCell>
                        <TableCell>
                          <Button variant="destructive" size="sm" onClick={() => remove.mutate(offering.id)}>
                            Удалить
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </div>
      </main>
    </div>
  );
}
