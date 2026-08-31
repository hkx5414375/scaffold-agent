import { defineStore } from "pinia";
import { ref } from "vue";

import { ApiError, request } from "../api/client";
import type { {{if .Tenancy}}Organization, OrganizationPage, {{end}}Principal, PrincipalResponse } from "../types";

export const useSessionStore = defineStore("session", () => {
  const principal = ref<Principal | null>(null);
  const initialized = ref(false);
{{- if .Tenancy}}
  const organizations = ref<Organization[]>([]);
  const currentOrganizationId = ref("");
{{- end}}

  async function load(): Promise<void> {
    try {
      principal.value = (await request<PrincipalResponse>("/api/v1/auth/me")).principal;
{{- if .Tenancy}}
      await loadOrganizations();
{{- end}}
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
{{- if .Tenancy}}
    await loadOrganizations();
{{- end}}
  }

  async function logout(): Promise<void> {
    await request<void>("/api/v1/auth/logout", { method: "POST" });
    principal.value = null;
{{- if .Tenancy}}
    organizations.value = [];
    currentOrganizationId.value = "";
    localStorage.removeItem("scaffold.organization_id");
{{- end}}
  }

{{- if .Tenancy}}
  async function loadOrganizations(): Promise<void> {
    const page = await request<OrganizationPage>("/api/v1/organizations");
    organizations.value = page.items;
    const stored = localStorage.getItem("scaffold.organization_id") ?? "";
    const selected = page.items.some((item) => item.id === stored)
      ? stored
      : (page.items[0]?.id ?? "");
    selectOrganization(selected);
  }

  function selectOrganization(organizationId: string): void {
    currentOrganizationId.value = organizationId;
    if (organizationId) localStorage.setItem("scaffold.organization_id", organizationId);
    else localStorage.removeItem("scaffold.organization_id");
  }

  async function createOrganization(name: string): Promise<void> {
    const organization = await request<Organization>("/api/v1/organizations", {
      method: "POST",
      body: JSON.stringify({ name }),
    });
    organizations.value.push(organization);
    selectOrganization(organization.id);
  }

  return {
    principal,
    initialized,
    organizations,
    currentOrganizationId,
    load,
    login,
    logout,
    selectOrganization,
    createOrganization,
  };
{{- else}}
  return { principal, initialized, load, login, logout };
{{- end}}
});
