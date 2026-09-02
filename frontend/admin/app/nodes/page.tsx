"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuthStore } from "@/lib/store";
import {
  listNodes,
  createNode,
  deleteNode,
  getNodeTraffic,
  type Node,
  type NodeTraffic,
} from "@/lib/api";
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

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
}

export default function NodesPage() {
  const router = useRouter();
  const token = useAuthStore((s) => s.token);
  const qc = useQueryClient();
  const [isOpen, setIsOpen] = useState(false);
  const [form, setForm] = useState({ name: "", hostname: "", public_ip: "" });
  const [enrollToken, setEnrollToken] = useState<string | null>(null);
  const [trafficNode, setTrafficNode] = useState<Node | null>(null);

  useEffect(() => {
    if (!token) router.push("/login");
  }, [token, router]);

  if (!token) return null;

  const { data, isLoading } = useQuery({
    queryKey: ["nodes", token],
    queryFn: () => listNodes(token),
    enabled: !!token,
  });

  const trafficQuery = useQuery<NodeTraffic>({
    queryKey: ["node-traffic", trafficNode?.id, token],
    queryFn: () => getNodeTraffic(token, trafficNode!.id),
    enabled: !!token && !!trafficNode,
  });

  const create = useMutation({
    mutationFn: () =>
      createNode(token, {
        name: form.name,
        hostname: form.hostname,
        public_ip: form.public_ip || undefined,
      }),
    onSuccess: (res) => {
      qc.invalidateQueries({ queryKey: ["nodes", token] });
      setEnrollToken(res.enroll_token);
      setForm({ name: "", hostname: "", public_ip: "" });
      toast.success("Нода создана");
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const remove = useMutation({
    mutationFn: (id: string) => deleteNode(token, id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["nodes", token] }),
    onError: (err: Error) => toast.error(err.message),
  });

  return (
    <div className="flex flex-1 flex-col">
      <Nav />
      <main className="flex-1 p-4">
        <div className="mx-auto max-w-7xl space-y-4">
          <div className="flex items-center justify-between">
            <h1 className="text-2xl font-semibold">Ноды</h1>
            <Dialog open={isOpen} onOpenChange={setIsOpen}>
              <DialogTrigger render={<Button>Добавить ноду</Button>} />
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>Новая нода</DialogTitle>
                </DialogHeader>
                <div className="space-y-4 py-2">
                  <div className="space-y-2">
                    <Label>Название</Label>
                    <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
                  </div>
                  <div className="space-y-2">
                    <Label>Hostname</Label>
                    <Input value={form.hostname} onChange={(e) => setForm({ ...form, hostname: e.target.value })} />
                  </div>
                  <div className="space-y-2">
                    <Label>Публичный IP</Label>
                    <Input value={form.public_ip} onChange={(e) => setForm({ ...form, public_ip: e.target.value })} />
                  </div>
                  <Button className="w-full" onClick={() => create.mutate()} disabled={create.isPending}>
                    Создать
                  </Button>
                  {enrollToken && (
                    <div className="rounded-md bg-muted p-2 text-xs font-mono break-all">
                      Enroll token: {enrollToken}
                    </div>
                  )}
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
                    <TableHead>Hostname</TableHead>
                    <TableHead>IP</TableHead>
                    <TableHead>Статус</TableHead>
                    <TableHead>Public key</TableHead>
                    <TableHead>Трафик</TableHead>
                    <TableHead>Действия</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {isLoading ? (
                    <TableRow>
                      <TableCell colSpan={7}>Загрузка...</TableCell>
                    </TableRow>
                  ) : (
                    data?.nodes.map((node) => (
                      <TableRow key={node.id}>
                        <TableCell>{node.name}</TableCell>
                        <TableCell>{node.hostname}</TableCell>
                        <TableCell>{node.public_ip || "—"}</TableCell>
                        <TableCell>{node.status}</TableCell>
                        <TableCell className="font-mono text-xs max-w-[200px] truncate">
                          {node.reality_public_key || "—"}
                        </TableCell>
                        <TableCell>
                          <Button variant="outline" size="sm" onClick={() => setTrafficNode(node)}>
                            Статистика
                          </Button>
                        </TableCell>
                        <TableCell>
                          <Button variant="destructive" size="sm" onClick={() => remove.mutate(node.id)}>
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

      <Dialog open={!!trafficNode} onOpenChange={(open) => !open && setTrafficNode(null)}>
        <DialogContent className="max-w-3xl">
          <DialogHeader>
            <DialogTitle>Трафик: {trafficNode?.name}</DialogTitle>
          </DialogHeader>
          {trafficQuery.isLoading ? (
            <div>Загрузка...</div>
          ) : trafficQuery.error ? (
            <div className="text-destructive">Ошибка загрузки трафика</div>
          ) : trafficQuery.data ? (
            <div className="space-y-4">
              <div className="grid grid-cols-3 gap-4">
                <Card>
                  <CardHeader className="pb-2">
                    <CardTitle className="text-sm font-medium">Всего</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">
                      {formatBytes(trafficQuery.data.totals.uplink + trafficQuery.data.totals.downlink)}
                    </div>
                  </CardContent>
                </Card>
                <Card>
                  <CardHeader className="pb-2">
                    <CardTitle className="text-sm font-medium">Исходящий</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">{formatBytes(trafficQuery.data.totals.uplink)}</div>
                  </CardContent>
                </Card>
                <Card>
                  <CardHeader className="pb-2">
                    <CardTitle className="text-sm font-medium">Входящий</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">{formatBytes(trafficQuery.data.totals.downlink)}</div>
                  </CardContent>
                </Card>
              </div>

              <Card>
                <CardContent className="p-0 max-h-80 overflow-auto">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Время</TableHead>
                        <TableHead>Устройство</TableHead>
                        <TableHead>Исходящий</TableHead>
                        <TableHead>Входящий</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {trafficQuery.data.traffic.length === 0 ? (
                        <TableRow>
                          <TableCell colSpan={4}>Нет данных</TableCell>
                        </TableRow>
                      ) : (
                        trafficQuery.data.traffic.map((stat) => (
                          <TableRow key={stat.id}>
                            <TableCell>{new Date(stat.ts).toLocaleString()}</TableCell>
                            <TableCell className="font-mono text-xs">{stat.device_id}</TableCell>
                            <TableCell>{formatBytes(stat.uplink)}</TableCell>
                            <TableCell>{formatBytes(stat.downlink)}</TableCell>
                          </TableRow>
                        ))
                      )}
                    </TableBody>
                  </Table>
                </CardContent>
              </Card>
            </div>
          ) : null}
        </DialogContent>
      </Dialog>
    </div>
  );
}
