"use client";

import { useRouter } from "next/navigation";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuthStore } from "@/lib/store";
import { listSubscriptions, activateSubscription, type Subscription } from "@/lib/api";
import { Nav } from "@/components/nav";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { toast } from "sonner";

export default function SubscriptionsPage() {
  const router = useRouter();
  const token = useAuthStore((s) => s.token);
  const qc = useQueryClient();

  if (!token) {
    router.push("/login");
    return null;
  }

  const { data, isLoading } = useQuery({
    queryKey: ["subscriptions", token],
    queryFn: () => listSubscriptions(token),
    enabled: !!token,
  });

  const activate = useMutation({
    mutationFn: (id: string) => activateSubscription(token, id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["subscriptions", token] }),
    onError: (err: Error) => toast.error(err.message),
  });

  return (
    <div className="flex flex-1 flex-col">
      <Nav />
      <main className="flex-1 p-4">
        <div className="mx-auto max-w-7xl space-y-4">
          <h1 className="text-2xl font-semibold">Подписки</h1>
          <Card>
            <CardContent className="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>ID</TableHead>
                    <TableHead>User ID</TableHead>
                    <TableHead>Offering ID</TableHead>
                    <TableHead>Статус</TableHead>
                    <TableHead>Трафик</TableHead>
                    <TableHead>Действует до</TableHead>
                    <TableHead>Действия</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {isLoading ? (
                    <TableRow>
                      <TableCell colSpan={7}>Загрузка...</TableCell>
                    </TableRow>
                  ) : (
                    data?.subscriptions.map((sub) => (
                      <TableRow key={sub.id}>
                        <TableCell className="font-mono text-xs">{sub.id.slice(0, 8)}</TableCell>
                        <TableCell className="font-mono text-xs">{sub.user_id.slice(0, 8)}</TableCell>
                        <TableCell className="font-mono text-xs">{sub.offering_id.slice(0, 8)}</TableCell>
                        <TableCell>{sub.status}</TableCell>
                        <TableCell>{formatBytes(sub.total_traffic_bytes)}</TableCell>
                        <TableCell>
                          {sub.expires_at ? new Date(sub.expires_at).toLocaleDateString() : "—"}
                        </TableCell>
                        <TableCell>
                          {sub.status === "pending" && (
                            <Button size="sm" onClick={() => activate.mutate(sub.id)} disabled={activate.isPending}>
                              Активировать
                            </Button>
                          )}
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

function formatBytes(bytes: number) {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
}
