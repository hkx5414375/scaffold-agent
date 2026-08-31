<script setup lang="ts">
import { ElMessage } from "element-plus";
import { ref{{if .Tenancy}}, watch{{end}} } from "vue";

import { ApiError, request } from "../api/client";
{{- if .Tenancy}}
import { useSessionStore } from "../stores/session";
{{- end}}
import type { JobItem, JobPage } from "../types";

{{- if .Tenancy}}
const session = useSessionStore();
{{- end}}
const jobs = ref<JobItem[]>([]);
const loading = ref(false);
const status = ref("");
const nextCursor = ref("");

{{- if .Tenancy}}
watch(
  () => session.currentOrganizationId,
  async (organizationId) => {
    jobs.value = [];
    nextCursor.value = "";
    if (organizationId) await load();
  },
  { immediate: true },
);
{{- else}}
void load();
{{- end}}

async function load(append = false): Promise<void> {
  loading.value = true;
  try {
    const parameters = new URLSearchParams({ limit: "50" });
    if (status.value) parameters.set("status", status.value);
    if (append && nextCursor.value) parameters.set("cursor", nextCursor.value);
    const page = await request<JobPage>(`/api/v1/jobs?${parameters.toString()}`);
    jobs.value = append ? [...jobs.value, ...page.items] : page.items;
    nextCursor.value = page.next_cursor ?? "";
  } catch (error) {
    showError(error, "Background jobs could not be loaded");
  } finally {
    loading.value = false;
  }
}

async function retry(item: JobItem): Promise<void> {
  try {
    const updated = await request<JobItem>(`/api/v1/jobs/${item.id}/retry`, {
      method: "POST",
    });
    Object.assign(item, updated);
    ElMessage.success("Background job queued for retry");
  } catch (error) {
    showError(error, "Background job could not be retried");
  }
}

function statusType(value: JobItem["status"]): "success" | "warning" | "danger" | "info" {
  if (value === "succeeded") return "success";
  if (value === "dead") return "danger";
  if (value === "retry" || value === "running") return "warning";
  return "info";
}

function showError(error: unknown, fallback: string): void {
  ElMessage.error(error instanceof ApiError ? error.message : fallback);
}
</script>

<template>
  <section>
    <div class="page-heading">
      <div>
        <p class="eyebrow">Worker operations</p>
        <h1>Background jobs</h1>
      </div>
      <div class="account-actions">
        <el-select
          v-model="status"
          clearable
          placeholder="All statuses"
          style="width: 170px"
          @change="load()"
        >
          <el-option label="Queued" value="queued" />
          <el-option label="Running" value="running" />
          <el-option label="Retry" value="retry" />
          <el-option label="Succeeded" value="succeeded" />
          <el-option label="Dead" value="dead" />
        </el-select>
        <el-button @click="load()">Refresh</el-button>
      </div>
    </div>

    <el-table v-loading="loading" :data="jobs" row-key="id">
      <el-table-column prop="type" label="Type" min-width="190" />
      <el-table-column label="Status" width="120">
        <template #default="scope">
          <el-tag :type="statusType(scope.row.status)" effect="plain">
            {{ "{{" }} scope.row.status {{ "}}" }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="Attempts" width="110">
        <template #default="scope">{{ "{{" }} scope.row.attempts {{ "}}" }}/{{ "{{" }} scope.row.max_attempts {{ "}}" }}</template>
      </el-table-column>
      <el-table-column prop="updated_at" label="Updated" min-width="210" />
      <el-table-column prop="last_error" label="Last error" min-width="260" show-overflow-tooltip />
      <el-table-column label="Actions" width="100" align="right">
        <template #default="scope">
          <el-button
            text
            type="primary"
            :disabled="scope.row.status !== 'dead'"
            @click="retry(scope.row)"
          >
            Retry
          </el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-button v-if="nextCursor" :loading="loading" class="load-more" @click="load(true)">
      Load more
    </el-button>
  </section>
</template>
