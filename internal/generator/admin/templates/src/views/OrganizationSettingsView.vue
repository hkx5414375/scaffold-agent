<script setup lang="ts">
import { ElMessage, ElMessageBox } from "element-plus";
import { computed, ref, watch } from "vue";

import { request } from "../api/client";
import { useSessionStore } from "../stores/session";
import type { Organization, OrganizationMember, OrganizationMemberPage } from "../types";

const session = useSessionStore();
const name = ref("");
const members = ref<OrganizationMember[]>([]);
const targetUserId = ref("");
const organization = computed(() => session.currentOrganization);

watch(
  organization,
  async (current) => {
    name.value = current?.name ?? "";
    targetUserId.value = "";
    members.value = [];
    if (current?.status === "active" && current.is_owner) {
      try {
        await loadMembers();
      } catch {
        ElMessage.error("Organization members could not be loaded");
      }
    }
  },
  { immediate: true },
);

async function loadMembers(): Promise<void> {
  if (!session.currentOrganizationId) return;
  const page = await request<OrganizationMemberPage>(
    `/api/v1/organizations/${session.currentOrganizationId}/members`,
  );
  members.value = page.items.filter((member) => !member.is_owner);
}

async function rename(): Promise<void> {
  if (!organization.value) return;
  try {
    await request<Organization>(`/api/v1/organizations/${organization.value.id}`, {
      method: "PATCH",
      body: JSON.stringify({ name: name.value }),
    });
    await session.loadOrganizations();
    ElMessage.success("Organization renamed");
  } catch {
    ElMessage.error("Organization could not be renamed");
  }
}

async function transferOwnership(): Promise<void> {
  if (!organization.value || !targetUserId.value) return;
  const target = members.value.find((member) => member.user_id === targetUserId.value);
  try {
    await ElMessageBox.confirm(
      `Transfer ownership to ${target?.email ?? targetUserId.value}?`,
      "Transfer ownership",
      { type: "warning", confirmButtonText: "Transfer" },
    );
    await request<Organization>(
      `/api/v1/organizations/${organization.value.id}/ownership-transfers`,
      { method: "POST", body: JSON.stringify({ target_user_id: targetUserId.value }) },
    );
    await session.loadOrganizations();
    ElMessage.success("Ownership transferred");
  } catch (error) {
    if (error !== "cancel") ElMessage.error("Ownership could not be transferred");
  }
}

async function setActive(active: boolean): Promise<void> {
  if (!organization.value) return;
  const action = active ? "reactivation" : "deactivation";
  try {
    await ElMessageBox.confirm(
      active
        ? "Reactivate this organization?"
        : "Deactivate this organization? New tenant requests will be blocked, but data will be retained.",
      active ? "Reactivate organization" : "Deactivate organization",
      { type: "warning", confirmButtonText: active ? "Reactivate" : "Deactivate" },
    );
    await request<Organization>(`/api/v1/organizations/${organization.value.id}/${action}`, {
      method: "POST",
    });
    await session.loadOrganizations();
    ElMessage.success(active ? "Organization reactivated" : "Organization deactivated");
  } catch (error) {
    if (error !== "cancel") ElMessage.error("Organization state could not be changed");
  }
}
</script>

<template>
  <section v-if="organization" class="panel-stack">
    <el-alert
      v-if="organization.status === 'inactive'"
      title="This organization is inactive. Tenant business and member requests are blocked."
      type="warning"
      :closable="false"
      show-icon
    />
    <el-card shadow="never">
      <template #header><strong>Organization profile</strong></template>
      <el-input v-model="name" maxlength="120" :disabled="organization.status !== 'active'" />
      <el-button
        type="primary"
        :disabled="organization.role !== 'admin' || organization.status !== 'active'"
        @click="rename"
      >
        Save name
      </el-button>
    </el-card>
    <el-card v-if="organization.is_owner" shadow="never">
      <template #header><strong>Ownership and lifecycle</strong></template>
      <template v-if="organization.status === 'active'">
        <el-select v-model="targetUserId" placeholder="Select a member" style="width: 280px">
          <el-option
            v-for="member in members"
            :key="member.user_id"
            :label="member.email"
            :value="member.user_id"
          />
        </el-select>
        <el-button :disabled="!targetUserId" @click="transferOwnership"
          >Transfer ownership</el-button
        >
        <el-divider />
        <el-button type="danger" plain @click="setActive(false)">Deactivate organization</el-button>
      </template>
      <el-button v-else type="primary" @click="setActive(true)">Reactivate organization</el-button>
    </el-card>
  </section>
</template>
