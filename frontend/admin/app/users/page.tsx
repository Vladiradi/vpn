"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuthStore } from "@/lib/store";
import { listUsers, createUser, updateUser, setUserStatus, type User } from "@/lib/api";
import { Nav } from "@/components/nav";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { toast } from "sonner";

export default function UsersPage() {
  const router = useRouter();
  const token = useAuthStore((s) => s.token);
  const qc = useQueryClient();
  const [isOpen, setIsOpen] = useState(false);
  const [form, setForm] = useState({ email: "", password: "", role: "user" as "admin" | "user" });

  useEffect(() => {
    if (!token) router.push("/login");
  }, [token, router]);

  if (!token) return null;

  const { data, isLoading } = useQuery({
    queryKey: ["users", token],
    queryFn: () => listUsers(token),
    enabled: !!token,
  });

  const create = useMutation({
    mutationFn: () => createUser(token, form),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["users", token] });
      setIsOpen(false);
      setForm({ email: "", password: "", role: "user" });
      toast.success("Пользователь создан");
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const toggleStatus = useMutation({
    mutationFn: ({ id, status }: { id: string; status: "active" | "blocked" }) =>
      setUserStatus(token, id, status),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["users", token] }),
    onError: (err: Error) => toast.error(err.message),
  });

  return (
    <div className="flex flex-1 flex-col">
      <Nav />
      <main className="flex-1 p-4">
        <div className="mx-auto max-w-7xl space-y-4">
          <div className="flex items-center justify-between">
            <h1 className="text-2xl font-semibold">Пользователи</h1>
            <Dialog open={isOpen} onOpenChange={setIsOpen}>
              <DialogTrigger render={<Button>Добавить</Button>} />
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>Новый пользователь</DialogTitle>
                </DialogHeader>
                <div className="space-y-4 py-2">
                  <div className="space-y-2">
                    <Label>Email</Label>
                    <Input value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} />
                  </div>
                  <div className="space-y-2">
                    <Label>Пароль</Label>
                    <Input value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} />
                  </div>
                  <div className="space-y-2">
                    <Label>Роль</Label>
                    <select
                      className="w-full rounded-md border px-3 py-2"
                      value={form.role}
                      onChange={(e) => setForm({ ...form, role: e.target.value as "admin" | "user" })}
                    >
                      <option value="user">Пользователь</option>
                      <option value="admin">Администратор</option>
                    </select>
                  </div>
                  <Button className="w-full" onClick={() => create.mutate()} disabled={create.isPending}>
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
                    <TableHead>Email</TableHead>
                    <TableHead>Роль</TableHead>
                    <TableHead>Статус</TableHead>
                    <TableHead>Создан</TableHead>
                    <TableHead>Действия</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {isLoading ? (
                    <TableRow>
                      <TableCell colSpan={5}>Загрузка...</TableCell>
                    </TableRow>
                  ) : (
                    data?.users.map((user) => (
                      <TableRow key={user.id}>
                        <TableCell>{user.email}</TableCell>
                        <TableCell>{user.role}</TableCell>
                        <TableCell>{user.status}</TableCell>
                        <TableCell>{new Date(user.created_at).toLocaleDateString()}</TableCell>
                        <TableCell>
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() =>
                              toggleStatus.mutate({
                                id: user.id,
                                status: user.status === "active" ? "blocked" : "active",
                              })
                            }
                          >
                            {user.status === "active" ? "Заблокировать" : "Разблокировать"}
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
