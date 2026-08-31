import { defineStore } from "pinia";
{{if .Tenancy}}import { computed, ref } from "vue";{{else}}import { ref } from "vue";{{end}}

import { ApiError, request } from "../api/client";
{{if .TenancyMembers}}import type {
  Organization,
  OrganizationPage,
  OrganizationMember,
  Principal,
  PrincipalResponse,
} from "../types";
{{else}}import type { {{if .Tenancy}}Organization, OrganizationPage, {{end}}Principal, PrincipalResponse } from "../types";
{{end}}
export const useSessionStore = defineStore("session", () => {
  const principal = ref<Principal | null>(null);
  const initialized = ref(false);
{{- if .Tenancy}}
  const organizations = ref<Organization[]>([]);
  const currentOrganizationId = ref("");
  const currentOrganization = computed(
    () => organizations.value.find((item) => item.id === currentOrganizationId.value) ?? null,
  );
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
{{if .Tenancy}}
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
{{if .TenancyMembers}}
  async function acceptInvitation(token: string): Promise<void> {
    const member = await request<OrganizationMember>("/api/v1/organization-invitations/accept", {
      method: "POST",
      body: JSON.stringify({ token }),
    });
    await loadOrganizations();
    selectOrganization(member.organization_id);
  }
{{end}}
  return {
    principal,
    initialized,
    organizations,
    currentOrganizationId,
    currentOrganization,
    load,
    login,
    logout,
    selectOrganization,
    createOrganization,
    loadOrganizations,
{{- if .TenancyMembers}}
    acceptInvitation,
{{- end}}
  };
{{- else}}
  return { principal, initialized, load, login, logout };
{{- end}}
});
