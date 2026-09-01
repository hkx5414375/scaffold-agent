<script setup lang="ts">
import type { CustomerEnvelope } from "../../../shared/types/customer";

const form = reactive({ display_name: "", email: "", password: "" });
const submitting = ref(false);
const message = ref("");

useSeoMeta({ title: "Create account · Storefront" });

async function submit(): Promise<void> {
  if (submitting.value) return;
  submitting.value = true;
  message.value = "";
  try {
    await $fetch<CustomerEnvelope>("/api/storefront/account/register", {
      method: "POST",
      body: form,
    });
    await navigateTo("/account");
  } catch {
    message.value =
      "The account could not be created. Check the details or use another email.";
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <section class="account-page account-narrow">
    <p class="eyebrow">Customer account</p>
    <h1>Start with a clean identity.</h1>
    <p class="account-summary">
      This account is separate from all staff administration access.
    </p>
    <form class="account-card account-form" @submit.prevent="submit">
      <label>
        Display name
        <input
          v-model.trim="form.display_name"
          autocomplete="name"
          maxlength="160"
          required
        />
      </label>
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
          autocomplete="new-password"
          minlength="12"
          required
        />
        <small>Use at least 12 characters.</small>
      </label>
      <p v-if="message" class="account-error" role="alert">{{ message }}</p>
      <button class="primary-action" type="submit" :disabled="submitting">
        {{ submitting ? "Creating…" : "Create account" }}
      </button>
      <p class="account-switch">
        Already registered? <NuxtLink to="/account/login">Sign in</NuxtLink>.
      </p>
    </form>
  </section>
</template>
