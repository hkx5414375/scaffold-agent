import { defineStore } from "pinia";
import { ref } from "vue";

import { ApiError, request } from "../api/client";
import type { Principal, PrincipalResponse } from "../types";

export const useSessionStore = defineStore("session", () => {
  const principal = ref<Principal | null>(null);
  const initialized = ref(false);

  async function load(): Promise<void> {
    try {
      principal.value = (await request<PrincipalResponse>("/api/v1/auth/me")).principal;
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        principal.value = null;
        return;
      }
      throw error;
    } finally {
      initialized.value = true;
    }
  }

  async function login(email: string, password: string): Promise<void> {
    principal.value = (
      await request<PrincipalResponse>("/api/v1/auth/login", {
        method: "POST",
        body: JSON.stringify({ email, password }),
      })
    ).principal;
  }

  async function logout(): Promise<void> {
    await request<void>("/api/v1/auth/logout", { method: "POST" });
    principal.value = null;
  }

  return { principal, initialized, load, login, logout };
});
