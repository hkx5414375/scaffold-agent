<script setup lang="ts">
import { ElMessage } from "element-plus";
import { onMounted{{if .Tenancy}}, ref{{end}} } from "vue";

import { useSessionStore } from "./stores/session";
{{- if .TenancyMembers}}
import MembersView from "./views/MembersView.vue";
{{- end}}
{{- if .TenancyLifecycle}}
import OrganizationSettingsView from "./views/OrganizationSettingsView.vue";
{{- end}}
{{- if .Files}}
import FilesView from "./views/FilesView.vue";
{{- end}}
{{- if .JobAdmin}}
import JobsView from "./views/JobsView.vue";
{{- end}}
{{- if .Approvals}}
import ApprovalsView from "./views/ApprovalsView.vue";
{{- end}}
{{- if .Catalog}}
import CatalogView from "./views/CatalogView.vue";
{{- end}}
{{- if .CustomerAccounts}}
import CustomerAccountsView from "./views/CustomerAccountsView.vue";
{{- end}}
{{- if .CRM}}
import CRMView from "./views/CRMView.vue";
{{- end}}
{{- if .Inventory}}
import InventoryView from "./views/InventoryView.vue";
{{- end}}
{{- if .Business}}
import BusinessView from "./views/BusinessView.vue";
{{- else}}
import DashboardView from "./views/DashboardView.vue";
{{- end}}
import LoginView from "./views/LoginView.vue";
const session = useSessionStore();
{{- if .Tenancy}}
const organizationName = ref("");
{{- if .TenancyMembers}}
const invitationToken = ref("");
{{- end}}
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
{{- if .TenancyMembers}}

async function acceptInvitation(): Promise<void> {
  try {
    await session.acceptInvitation(invitationToken.value);
    invitationToken.value = "";
    ElMessage.success("Invitation accepted");
  } catch {
    ElMessage.error("Invitation is invalid or expired");
  }
}
{{- end}}
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
            :label="organization.name{{if .TenancyLifecycle}} + (organization.status === 'inactive' ? ' (inactive)' : ''){{end}}"
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
{{- if .TenancyMembers}}
        <el-divider>or join an organization</el-divider>
        <el-input v-model="invitationToken" placeholder="Invitation token" />
        <el-button @click="acceptInvitation">Accept invitation</el-button>
{{- end}}
      </el-card>
{{- end}}
{{- if .TenancyMembers}}
      <el-tabs v-else class="workspace-tabs">
{{- if .Business}}
{{- if .TenancyLifecycle}}
        <el-tab-pane
          label="Business"
          name="business"
          :disabled="session.currentOrganization?.status !== 'active'"
        >
{{- else}}
        <el-tab-pane label="Business" name="business">
{{- end}}
          <BusinessView :key="session.currentOrganizationId" />
        </el-tab-pane>
{{- else}}
        <el-tab-pane label="Overview" name="overview">
          <DashboardView />
        </el-tab-pane>
{{- end}}
{{- if .TenancyLifecycle}}
        <el-tab-pane
          label="Members"
          name="members"
          :disabled="session.currentOrganization?.status !== 'active'"
        >
{{- else}}
        <el-tab-pane label="Members" name="members">
{{- end}}
          <MembersView :key="session.currentOrganizationId" />
        </el-tab-pane>
{{- if .TenancyLifecycle}}
        <el-tab-pane label="Organization" name="organization">
          <OrganizationSettingsView :key="session.currentOrganizationId" />
        </el-tab-pane>
{{- end}}
{{- if .Files}}
{{- if .TenancyLifecycle}}
        <el-tab-pane
          label="Files"
          name="files"
          :disabled="session.currentOrganization?.status !== 'active'"
        >
{{- else}}
        <el-tab-pane label="Files" name="files">
{{- end}}
          <FilesView :key="session.currentOrganizationId" />
        </el-tab-pane>
{{- end}}
{{- if .JobAdmin}}
{{- if .TenancyLifecycle}}
        <el-tab-pane
          label="Jobs"
          name="jobs"
          :disabled="session.currentOrganization?.status !== 'active'"
        >
{{- else}}
        <el-tab-pane label="Jobs" name="jobs">
{{- end}}
          <JobsView :key="session.currentOrganizationId" />
        </el-tab-pane>
{{- end}}
{{- if .Approvals}}
{{- if .TenancyLifecycle}}
        <el-tab-pane
          label="Approvals"
          name="approvals"
          :disabled="session.currentOrganization?.status !== 'active'"
        >
{{- else}}
        <el-tab-pane label="Approvals" name="approvals">
{{- end}}
          <ApprovalsView :key="session.currentOrganizationId" />
        </el-tab-pane>
{{- end}}
{{- if .Catalog}}
{{- if .TenancyLifecycle}}
        <el-tab-pane
          label="Catalog"
          name="catalog"
          :disabled="session.currentOrganization?.status !== 'active'"
        >
{{- else}}
        <el-tab-pane label="Catalog" name="catalog">
{{- end}}
          <CatalogView :key="session.currentOrganizationId" />
        </el-tab-pane>
{{- end}}
{{- if .CustomerAccounts}}
{{- if .TenancyLifecycle}}
        <el-tab-pane
          label="Customers"
          name="customers"
          :disabled="session.currentOrganization?.status !== 'active'"
        >
{{- else}}
        <el-tab-pane label="Customers" name="customers">
{{- end}}
          <CustomerAccountsView :key="session.currentOrganizationId" />
        </el-tab-pane>
{{- end}}
{{- if .CRM}}
{{- if .TenancyLifecycle}}
        <el-tab-pane
          label="CRM"
          name="crm"
          :disabled="session.currentOrganization?.status !== 'active'"
        >
{{- else}}
        <el-tab-pane label="CRM" name="crm">
{{- end}}
          <CRMView :key="session.currentOrganizationId" />
        </el-tab-pane>
{{- end}}
{{- if .Inventory}}
{{- if .TenancyLifecycle}}
        <el-tab-pane
          label="Inventory"
          name="inventory"
          :disabled="session.currentOrganization?.status !== 'active'"
        >
{{- else}}
        <el-tab-pane label="Inventory" name="inventory">
{{- end}}
          <InventoryView :key="session.currentOrganizationId" />
        </el-tab-pane>
{{- end}}
      </el-tabs>
{{- else}}
{{- if or .Files .JobAdmin .Approvals .Catalog .CustomerAccounts .CRM .Inventory}}
{{- if .Tenancy}}
      <el-tabs v-else class="workspace-tabs">
{{- else}}
      <el-tabs class="workspace-tabs">
{{- end}}
{{- if .Business}}
        <el-tab-pane label="Business" name="business">
          <BusinessView{{if .Tenancy}} :key="session.currentOrganizationId"{{end}} />
        </el-tab-pane>
{{- else}}
        <el-tab-pane label="Overview" name="overview">
          <DashboardView />
        </el-tab-pane>
{{- end}}
{{- if .Files}}
        <el-tab-pane label="Files" name="files">
          <FilesView{{if .Tenancy}} :key="session.currentOrganizationId"{{end}} />
        </el-tab-pane>
{{- end}}
{{- if .JobAdmin}}
        <el-tab-pane label="Jobs" name="jobs">
          <JobsView{{if .Tenancy}} :key="session.currentOrganizationId"{{end}} />
        </el-tab-pane>
{{- end}}
{{- if .Approvals}}
        <el-tab-pane label="Approvals" name="approvals">
          <ApprovalsView{{if .Tenancy}} :key="session.currentOrganizationId"{{end}} />
        </el-tab-pane>
{{- end}}
{{- if .Catalog}}
        <el-tab-pane label="Catalog" name="catalog">
          <CatalogView{{if .Tenancy}} :key="session.currentOrganizationId"{{end}} />
        </el-tab-pane>
{{- end}}
{{- if .CustomerAccounts}}
        <el-tab-pane label="Customers" name="customers">
          <CustomerAccountsView{{if .Tenancy}} :key="session.currentOrganizationId"{{end}} />
        </el-tab-pane>
{{- end}}
{{- if .CRM}}
        <el-tab-pane label="CRM" name="crm">
          <CRMView{{if .Tenancy}} :key="session.currentOrganizationId"{{end}} />
        </el-tab-pane>
{{- end}}
{{- if .Inventory}}
        <el-tab-pane label="Inventory" name="inventory">
          <InventoryView{{if .Tenancy}} :key="session.currentOrganizationId"{{end}} />
        </el-tab-pane>
{{- end}}
      </el-tabs>
{{- else}}
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
{{- end}}
{{- end}}
    </el-main>
  </el-container>
</template>
