<script setup lang="ts">
import { ElMessage } from "element-plus";
import { reactive, ref } from "vue";

import { ApiError } from "../api/client";
import { useSessionStore } from "../stores/session";

const session = useSessionStore();
const submitting = ref(false);
const form = reactive({ email: "", password: "" });

async function submit(): Promise<void> {
  if (submitting.value) return;
  submitting.value = true;
  try {
    await session.login(form.email, form.password);
  } catch (error) {
    ElMessage.error(error instanceof ApiError ? error.message : "Login could not be completed");
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <main class="login-shell">
    <el-card class="login-card" shadow="never">
      <template #header>
        <div>
          <p class="eyebrow">{{.ProjectName}}</p>
          <h1>Administration</h1>
          <p class="muted">Sign in with an authorized account.</p>
        </div>
      </template>
      <el-form label-position="top" @submit.prevent="submit">
        <el-form-item label="Email">
          <el-input v-model="form.email" autocomplete="username" type="email" />
        </el-form-item>
        <el-form-item label="Password">
          <el-input
            v-model="form.password"
            autocomplete="current-password"
            show-password
            type="password"
            @keyup.enter="submit"
          />
        </el-form-item>
        <el-button class="full-width" type="primary" :loading="submitting" @click="submit">
          Sign in
        </el-button>
      </el-form>
    </el-card>
  </main>
</template>
