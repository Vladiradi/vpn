export const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "";

export interface TokenPair {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface Me {
  user_id: string;
  email: string;
  role: "admin" | "user";
}

export interface User {
  id: string;
  email: string;
  role: "admin" | "user";
  status: "active" | "blocked" | "deleted";
  created_at: string;
  updated_at: string;
}

export interface Offering {
  id: string;
  name: string;
  duration_months: number;
  price_rubles: number;
  speed_mbps: number;
  country_code: string;
  country_name: string;
  region?: string;
  is_active: boolean;
  is_available: boolean;
  created_at: string;
  updated_at: string;
}

export interface Subscription {
  id: string;
  user_id: string;
  offering_id: string;
  status: string;
  started_at?: string;
  expires_at?: string;
  total_traffic_bytes: number;
  created_at: string;
  updated_at: string;
}

export interface Device {
  id: string;
  user_id: string;
  subscription_id?: string;
  node_id?: string;
  name: string;
  uuid: string;
  protocol: string;
  created_at: string;
  revoked_at?: string;
}

export interface Node {
  id: string;
  name: string;
  hostname: string;
  public_ip?: string;
  status: string;
  api_token?: string;
  reality_dest?: string;
  reality_public_key?: string;
  reality_short_ids?: string[];
  decoy_host?: string;
  last_seen_at?: string;
  created_at: string;
  updated_at: string;
}

async function request<T>(
  path: string,
  options: RequestInit = {},
  token?: string
): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options.headers as Record<string, string>),
  };
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }
  const res = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    headers,
  });
  if (!res.ok) {
    let message = res.statusText;
    try {
      const data = await res.json();
      message = data.error || message;
    } catch {
      // ignore
    }
    throw new Error(message);
  }
  if (res.status === 204) {
    return undefined as T;
  }
  return res.json() as Promise<T>;
}

export function login(email: string, password: string) {
  return request<TokenPair>("/api/v1/auth/login", {
    method: "POST",
    body: JSON.stringify({ email, password }),
  });
}

export function getMe(token: string) {
  return request<Me>("/api/v1/me", {}, token);
}

export function listUsers(token: string) {
  return request<{ users: User[] }>("/api/v1/users", {}, token);
}

export function createUser(token: string, payload: Partial<User> & { email: string; password?: string; role: "admin" | "user" }) {
  return request<User>("/api/v1/users", {
    method: "POST",
    body: JSON.stringify(payload),
  }, token);
}

export function updateUser(token: string, id: string, payload: { email: string; role: "admin" | "user" }) {
  return request<User>(`/api/v1/users/${id}`, {
    method: "PUT",
    body: JSON.stringify(payload),
  }, token);
}

export function setUserStatus(token: string, id: string, status: "active" | "blocked") {
  return request<{ status: string }>(`/api/v1/users/${id}/status`, {
    method: "POST",
    body: JSON.stringify({ status }),
  }, token);
}

export function listSubscriptions(token: string) {
  return request<{ subscriptions: Subscription[] }>("/api/v1/admin/subscriptions", {}, token);
}

export function activateSubscription(token: string, id: string) {
  return request<{ status: string; subscription: Subscription }>(
    `/api/v1/admin/subscriptions/${id}/activate`,
    { method: "POST" },
    token
  );
}

export function listNodes(token: string) {
  return request<{ nodes: Node[] }>("/api/v1/nodes", {}, token);
}

export function createNode(token: string, payload: { name: string; hostname: string; public_ip?: string }) {
  return request<{ node: Node; enroll_token: string }>("/api/v1/nodes", {
    method: "POST",
    body: JSON.stringify(payload),
  }, token);
}

export function deleteNode(token: string, id: string) {
  return request<{ status: string }>(`/api/v1/nodes/${id}`, {
    method: "DELETE",
  }, token);
}

export function updateNodeConfig(token: string, id: string, payload: {
  reality_dest?: string;
  reality_public_key?: string;
  reality_short_ids?: string[];
  decoy_host?: string;
}) {
  return request<{ status: string }>(`/api/v1/nodes/${id}/config`, {
    method: "PUT",
    body: JSON.stringify(payload),
  }, token);
}

export function listOfferings(token: string) {
  return request<{ offerings: Offering[] }>("/api/v1/offerings?active=false", {}, token);
}

export function createOffering(token: string, payload: Omit<Offering, "id" | "created_at" | "updated_at">) {
  return request<Offering>("/api/v1/offerings", {
    method: "POST",
    body: JSON.stringify(payload),
  }, token);
}

export function updateOffering(token: string, id: string, payload: Omit<Offering, "id" | "created_at" | "updated_at">) {
  return request<Offering>(`/api/v1/offerings/${id}`, {
    method: "PUT",
    body: JSON.stringify(payload),
  }, token);
}

export function deleteOffering(token: string, id: string) {
  return request<{ status: string }>(`/api/v1/offerings/${id}`, {
    method: "DELETE",
  }, token);
}
