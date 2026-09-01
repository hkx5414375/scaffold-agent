<script setup lang="ts">
import type { CustomerEnvelope } from "../../../shared/types/customer";

const form = reactive({ email: "", password: "" });
const submitting = ref(false);
const message = ref("");

useSeoMeta({ title: "Sign in · Storefront" });

async function submit(): Promise<void> {
  if (submitting.value) return;
  submitting.value = true;
  message.value = "";
  try {
    await $fetch<CustomerEnvelope>("/api/storefront/account/login", {
      method: "POST",
      body: form,
    });
    await navigateTo("/account");
  } catch {
    message.value = "Email or password is incorrect.";
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <section class="account-page account-narrow">
    <p class="eyebrow">Customer account</p>
    <h1>Welcome back.</h1>
    <p class="account-summary">
      Sign in with the storefront identity you created here.
    </p>
    <form class="account-card account-form" @submit.prevent="submit">
      <label>
        Email
        <input
          v-model.trim="form.email"
          type="email"
          autocomplete="email"
          maxlength="254"
          required
        />
      </label>
      <label>
        Password
        <input
          v-model="form.password"
          type="password"
          autocomplete="current-password"
          required
        />
      </label>
      <p v-if="message" class="account-error" role="alert">{{ message }}</p>
      <button class="primary-action" type="submit" :disabled="submitting">
        {{ submitting ? "Signing in…" : "Sign in" }}
      </button>
      <p class="account-switch">
        New here? <NuxtLink to="/account/register">Create an account</NuxtLink>.
      </p>
    </form>
  </section>
</template>
