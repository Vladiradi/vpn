import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { Me, TokenPair } from "./api";

interface AuthState {
  token: string | null;
  refreshToken: string | null;
  user: Me | null;
  setAuth: (tokens: TokenPair, user: Me) => void;
  clearAuth: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      refreshToken: null,
      user: null,
      setAuth: (tokens, user) =>
        set({
          token: tokens.access_token,
          refreshToken: tokens.refresh_token,
          user,
        }),
      clearAuth: () =>
        set({ token: null, refreshToken: null, user: null }),
    }),
    { name: "vpn-admin-auth" }
  )
);
