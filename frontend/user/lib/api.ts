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
  status: "pending" | "active" | "expired" | "cancelled";
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

export interface DeviceConfig {
  vless_uri: string;
  sing_box_json: unknown;
}

export interface ClientApp {
  platform: string;
  name: string;
  url: string;
  notes?: string;
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

export function googleLogin(idToken: string) {
  return request<TokenPair>("/api/v1/auth/google", {
    method: "POST",
    body: JSON.stringify({ id_token: idToken }),
  });
}

export interface TelegramLoginPayload {
  id: number;
  first_name: string;
  username?: string;
  email: string;
  auth_date: string;
  hash: string;
}

export function telegramLogin(payload: TelegramLoginPayload) {
  return request<TokenPair>("/api/v1/auth/telegram", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function getMe(token: string) {
  return request<Me>("/api/v1/me", {}, token);
}

export function listOfferings() {
  return request<{ offerings: Offering[] }>("/api/v1/public/offerings");
}

export function createSubscription(offeringId: string, token: string) {
  return request<Subscription>("/api/v1/subscriptions", {
    method: "POST",
    body: JSON.stringify({ offering_id: offeringId }),
  }, token);
}

export function paySubscription(id: string, token: string) {
  return request<{ status: string; subscription: Subscription }>(
    `/api/v1/subscriptions/${id}/pay`,
    { method: "POST" },
    token
  );
}

export function listMySubscriptions(token: string) {
  return request<{ subscriptions: Subscription[] }>("/api/v1/subscriptions", {}, token);
}

export function listMyDevices(token: string) {
  return request<{ devices: Device[] }>("/api/v1/me/devices", {}, token);
}

export function getDeviceConfig(deviceId: string, token: string) {
  return request<DeviceConfig>(`/api/v1/me/devices/${deviceId}/config`, {}, token);
}

export function getDeviceQRUrl(deviceId: string) {
  return `${API_BASE_URL}/api/v1/me/devices/${deviceId}/qr`;
}

export function getDeviceFileUrl(deviceId: string, format: "vless" | "sing-box") {
  return `${API_BASE_URL}/api/v1/me/devices/${deviceId}/file?format=${format}`;
}

export function listClientApps() {
  return request<{ apps: ClientApp[] }>("/api/v1/client-apps");
}
