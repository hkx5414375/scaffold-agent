<script setup lang="ts">
import { ElMessage, ElMessageBox } from "element-plus";
import { ref{{if .Tenancy}}, watch{{end}} } from "vue";

import { ApiError, download, request } from "../api/client";
{{- if .Tenancy}}
import { useSessionStore } from "../stores/session";
{{- end}}
import type { FileAsset, FileAssetPage } from "../types";

{{- if .Tenancy}}
const session = useSessionStore();
{{- end}}
const assets = ref<FileAsset[]>([]);
const loading = ref(false);
const uploading = ref(false);
const nextCursor = ref("");

{{- if .Tenancy}}
watch(
  () => session.currentOrganizationId,
  async (organizationId) => {
    assets.value = [];
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
    const cursor =
      append && nextCursor.value ? `&cursor=${encodeURIComponent(nextCursor.value)}` : "";
    const page = await request<FileAssetPage>(`/api/v1/files?limit=50${cursor}`);
    assets.value = append ? [...assets.value, ...page.items] : page.items;
    nextCursor.value = page.next_cursor ?? "";
  } catch (error) {
    showError(error, "Files could not be loaded");
  } finally {
    loading.value = false;
  }
}

async function upload(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) return;
  uploading.value = true;
  try {
    const form = new FormData();
    form.set("file", file);
    await request<FileAsset>("/api/v1/files", { method: "POST", body: form });
    await load();
    ElMessage.success("File uploaded");
  } catch (error) {
    showError(error, "File could not be uploaded");
  } finally {
    uploading.value = false;
    input.value = "";
  }
}

async function downloadAsset(asset: FileAsset): Promise<void> {
  try {
    const blob = await download(`/api/v1/files/${asset.id}/content`);
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = asset.name;
    anchor.click();
    URL.revokeObjectURL(url);
  } catch (error) {
    showError(error, "File could not be downloaded");
  }
}

async function remove(asset: FileAsset): Promise<void> {
  try {
    await ElMessageBox.confirm(`Delete ${asset.name}?`, "Delete file", {
      type: "warning",
      confirmButtonText: "Delete",
    });
  } catch {
    return;
  }
  try {
    await request<void>(`/api/v1/files/${asset.id}`, { method: "DELETE" });
    assets.value = assets.value.filter((item) => item.id !== asset.id);
    ElMessage.success("File deleted");
  } catch (error) {
    showError(error, "File could not be deleted");
  }
}

function formatBytes(value: number): string {
  if (value < 1024) return `${value} B`;
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KiB`;
  return `${(value / (1024 * 1024)).toFixed(1)} MiB`;
}

function showError(error: unknown, fallback: string): void {
  ElMessage.error(error instanceof ApiError ? error.message : fallback);
}
</script>

<template>
  <section>
    <div class="page-heading">
      <div>
        <p class="eyebrow">Durable assets</p>
        <h1>Files</h1>
      </div>
      <label>
        <input type="file" hidden :disabled="uploading" @change="upload" />
        <el-button type="primary" :loading="uploading">Upload file</el-button>
      </label>
    </div>

    <el-table v-loading="loading" :data="assets" row-key="id">
      <el-table-column prop="name" label="Name" min-width="240" />
      <el-table-column prop="media_type" label="Media type" min-width="180" />
      <el-table-column label="Size" width="120">
        <template #default="scope">{{ "{{" }} formatBytes(scope.row.size) {{ "}}" }}</template>
      </el-table-column>
      <el-table-column prop="created_at" label="Created" min-width="210" />
      <el-table-column label="Actions" width="180" align="right">
        <template #default="scope">
          <el-button text @click="downloadAsset(scope.row)">Download</el-button>
          <el-button type="danger" text @click="remove(scope.row)">Delete</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-button v-if="nextCursor" :loading="loading" class="load-more" @click="load(true)">
      Load more
    </el-button>
  </section>
</template>
