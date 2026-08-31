<script setup lang="ts">
import { ElMessage, ElMessageBox } from "element-plus";
import { computed, ref, watch } from "vue";

import { ApiError, request } from "../api/client";
import { useSessionStore } from "../stores/session";
import type { OrganizationInvitation, OrganizationMember, OrganizationMemberPage } from "../types";

const session = useSessionStore();
const members = ref<OrganizationMember[]>([]);
const loading = ref(false);
const inviteEmail = ref("");
const inviteRole = ref<"admin" | "user">("user");
const invitation = ref<OrganizationInvitation | null>(null);
const canManage = computed(
  () =>
    session.organizations.find((item) => item.id === session.currentOrganizationId)?.role ===
    "admin",
);

watch(
  () => session.currentOrganizationId,
  async (organizationId) => {
    members.value = [];
    invitation.value = null;
    if (organizationId) await loadMembers();
  },
  { immediate: true },
);

async function loadMembers(): Promise<void> {
  loading.value = true;
  try {
    const page = await request<OrganizationMemberPage>(
      `/api/v1/organizations/${session.currentOrganizationId}/members`,
    );
    members.value = page.items;
  } catch (error) {
    showError(error, "Members could not be loaded");
  } finally {
    loading.value = false;
  }
}

async function invite(): Promise<void> {
  try {
    invitation.value = await request<OrganizationInvitation>(
      `/api/v1/organizations/${session.currentOrganizationId}/invitations`,
      {
        method: "POST",
        body: JSON.stringify({ email: inviteEmail.value, role: inviteRole.value }),
      },
    );
    inviteEmail.value = "";
    ElMessage.success("Invitation created");
  } catch (error) {
    showError(error, "Invitation could not be created");
  }
}

async function changeRole(member: OrganizationMember, role: "admin" | "user"): Promise<void> {
  if (member.role === role) return;
  try {
    const updated = await request<OrganizationMember>(
      `/api/v1/organizations/${session.currentOrganizationId}/members/${member.user_id}`,
      { method: "PATCH", body: JSON.stringify({ role }) },
    );
    Object.assign(member, updated);
    if (member.user_id === session.principal?.user_id) await session.load();
    ElMessage.success("Member role updated");
  } catch (error) {
    showError(error, "Member role could not be updated");
  }
}

async function remove(member: OrganizationMember): Promise<void> {
  try {
    await ElMessageBox.confirm(`Remove ${member.email} from this organization?`, "Remove member", {
      type: "warning",
      confirmButtonText: "Remove",
    });
  } catch {
    return;
  }
  try {
    await request<void>(
      `/api/v1/organizations/${session.currentOrganizationId}/members/${member.user_id}`,
      { method: "DELETE" },
    );
    if (member.user_id === session.principal?.user_id) {
      await session.load();
      return;
    }
    await loadMembers();
    ElMessage.success("Member removed");
  } catch (error) {
    showError(error, "Member could not be removed");
  }
}

async function copyToken(): Promise<void> {
  if (!invitation.value) return;
  await navigator.clipboard.writeText(invitation.value.acceptance_token);
  ElMessage.success("Invitation token copied");
}

function showError(error: unknown, fallback: string): void {
  ElMessage.error(error instanceof ApiError ? error.message : fallback);
}
</script>

<template>
  <section>
    <div class="page-heading">
      <div>
        <p class="eyebrow">Organization access</p>
        <h1>Members</h1>
      </div>
      <div class="account-actions">
        <el-input
          v-model="inviteEmail"
          :disabled="!canManage"
          placeholder="member@example.com"
          style="width: 240px"
        />
        <el-select v-model="inviteRole" :disabled="!canManage" style="width: 110px">
          <el-option label="User" value="user" />
          <el-option label="Admin" value="admin" />
        </el-select>
        <el-button type="primary" :disabled="!canManage" @click="invite">Invite</el-button>
      </div>
    </div>

    <el-alert
      v-if="invitation"
      class="invitation-token"
      title="Deliver this token securely. It is shown only once."
      type="success"
      :closable="false"
    >
      <template #default>
        <el-input :model-value="invitation.acceptance_token" readonly>
          <template #append><el-button @click="copyToken">Copy</el-button></template>
        </el-input>
      </template>
    </el-alert>

    <el-table v-loading="loading" :data="members" row-key="user_id">
      <el-table-column prop="email" label="Email" min-width="260" />
      <el-table-column label="Role" width="150">
        <template #default="scope">
          <el-select
            :model-value="scope.row.role"
            :disabled="!canManage{{if .TenancyLifecycle}} || scope.row.is_owner{{end}}"
            @change="changeRole(scope.row, $event)"
          >
            <el-option label="User" value="user" />
            <el-option label="Admin" value="admin" />
          </el-select>
        </template>
      </el-table-column>
{{- if .TenancyLifecycle}}
      <el-table-column label="Owner" width="90">
        <template #default="scope">
          <el-tag v-if="scope.row.is_owner" type="warning" effect="plain">Owner</el-tag>
        </template>
      </el-table-column>
{{- end}}
      <el-table-column prop="joined_at" label="Joined" min-width="220" />
      <el-table-column label="Actions" width="120" align="right">
        <template #default="scope">
{{- if .TenancyLifecycle}}
          <el-button
            type="danger"
            text
            :disabled="!canManage || scope.row.is_owner"
            @click="remove(scope.row)"
          >
{{- else}}
          <el-button type="danger" text :disabled="!canManage" @click="remove(scope.row)">
{{- end}}
            Remove
          </el-button>
        </template>
      </el-table-column>
    </el-table>
  </section>
</template>
