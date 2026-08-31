<script setup lang="ts">
import { ElMessage } from "element-plus";
import { onMounted{{if .Tenancy}}, ref{{end}} } from "vue";

import { useSessionStore } from "./stores/session";
{{- if .Business}}
import BusinessView from "./views/BusinessView.vue";
{{- else}}
import DashboardView from "./views/DashboardView.vue";
{{- end}}
import LoginView from "./views/LoginView.vue";
const session = useSessionStore();
{{- if .Tenancy}}
const organizationName = ref("");
{{- end}}

onMounted(async () => {
  try {
    await session.load();
  } catch {
    ElMessage.error("The administration API is unavailable");
  }
});

async function logout(): Promise<void> {
  try {
    await session.logout();
  } catch {
    ElMessage.error("Logout could not be completed");
  }
}
{{- if .Tenancy}}

async function createOrganization(): Promise<void> {
  try {
    await session.createOrganization(organizationName.value);
    organizationName.value = "";
    ElMessage.success("Organization created");
  } catch {
    ElMessage.error("Organization could not be created");
  }
}
{{- end}}
</script>

<template>
  <div v-if="!session.initialized" class="center-screen">
    <span class="muted">Loading administration…</span>
  </div>
  <LoginView v-else-if="!session.principal" />
  <el-container v-else class="application-shell">
    <el-header class="topbar">
      <div>
        <p class="eyebrow">{{.ProjectName}}</p>
        <strong>Administration</strong>
      </div>
      <div class="account-actions">
{{- if .Tenancy}}
        <el-select
          v-if="session.organizations.length"
          :model-value="session.currentOrganizationId"
          aria-label="Current organization"
          style="width: 190px"
          @change="session.selectOrganization"
        >
          <el-option
            v-for="organization in session.organizations"
            :key="organization.id"
            :label="organization.name"
            :value="organization.id"
          />
        </el-select>
{{- end}}
        <span>{{ "{{" }} session.principal.email {{ "}}" }}</span>
        <el-tag effect="plain">{{ "{{" }} session.principal.role {{ "}}" }}</el-tag>
        <el-button text @click="logout">Sign out</el-button>
      </div>
    </el-header>
    <el-main class="content">
{{- if .Tenancy}}
      <el-card v-if="!session.currentOrganizationId" class="setup-card" shadow="never">
        <p class="eyebrow">First workspace</p>
        <h1>Create an organization</h1>
        <p class="muted">Business data is isolated by organization.</p>
        <el-input v-model="organizationName" maxlength="120" placeholder="Organization name" />
        <el-button type="primary" @click="createOrganization">Create organization</el-button>
      </el-card>
{{- end}}
{{- if .Business}}
{{- if .Tenancy}}
      <BusinessView v-else :key="session.currentOrganizationId" />
{{- else}}
      <BusinessView />
{{- end}}
{{- else}}
{{- if .Tenancy}}
      <DashboardView v-else />
{{- else}}
      <DashboardView />
{{- end}}
{{- end}}
    </el-main>
  </el-container>
</template>
