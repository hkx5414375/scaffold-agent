<script setup lang="ts">
import { ElMessage } from "element-plus";
import { onMounted } from "vue";

import { useSessionStore } from "./stores/session";
{{- if .Business}}
import BusinessView from "./views/BusinessView.vue";
{{- else}}
import DashboardView from "./views/DashboardView.vue";
{{- end}}
import LoginView from "./views/LoginView.vue";
const session = useSessionStore();

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
        <span>{{ "{{" }} session.principal.email {{ "}}" }}</span>
        <el-tag effect="plain">{{ "{{" }} session.principal.role {{ "}}" }}</el-tag>
        <el-button text @click="logout">Sign out</el-button>
      </div>
    </el-header>
    <el-main class="content">
{{- if .Business}}
      <BusinessView />
{{- else}}
      <DashboardView />
{{- end}}
    </el-main>
  </el-container>
</template>
