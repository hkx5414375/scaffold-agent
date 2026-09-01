<script setup lang="ts">
import type {
  CustomerEnvelope,
  CustomerPasswordChanged,
} from "../../../shared/types/customer";

const { data, error, refresh, status } = await useFetch<CustomerEnvelope>(
  "/api/storefront/account/me",
  { key: "current-customer" },
);
const profileName = ref(data.value?.customer.display_name ?? "");
const currentPassword = ref("");
const newPassword = ref("");
const profileMessage = ref("");
const passwordMessage = ref("");

useSeoMeta({ title: "Your account · Storefront" });

watch(
  () => data.value?.customer.display_name,
  (value) => {
    if (value) profileName.value = value;
  },
);

async function updateProfile(): Promise<void> {
  if (!data.value) return;
  profileMessage.value = "";
  try {
    data.value = await $fetch<CustomerEnvelope>(
      "/api/storefront/account/profile",
      {
        method: "PUT",
        body: {
          display_name: profileName.value,
          version: data.value.customer.version,
        },
      },
    );
    profileMessage.value = "Profile updated.";
  } catch {
    profileMessage.value = "Profile changed elsewhere or could not be updated.";
  }
}

async function changePassword(): Promise<void> {
  if (!data.value) return;
  passwordMessage.value = "";
  try {
    await $fetch<CustomerPasswordChanged>("/api/storefront/account/password", {
      method: "PUT",
      body: {
        current_password: currentPassword.value,
        new_password: newPassword.value,
        version: data.value.customer.version,
      },
    });
    await navigateTo("/account/login");
  } catch {
    passwordMessage.value =
      "The password was not changed. Verify the current password and reload.";
  }
}

async function logout(): Promise<void> {
  await $fetch("/api/storefront/account/logout", { method: "POST" });
  data.value = undefined;
  await navigateTo("/account/login");
}

async function closeAccount(): Promise<void> {
  if (
    !data.value ||
    !window.confirm("Close this account permanently? This cannot be undone.")
  )
    return;
  await $fetch<CustomerEnvelope>("/api/storefront/account/close", {
    method: "POST",
    body: { version: data.value.customer.version },
  });
  data.value = undefined;
  await navigateTo("/");
}

function reloadAccount(): void {
  void refresh();
}
</script>

<template>
  <section class="account-page">
    <div v-if="status === 'pending'" class="account-card">Loading account…</div>
    <div v-else-if="error || !data" class="account-card account-empty">
      <p class="eyebrow">Customer account</p>
      <h1>Keep your place.</h1>
      <p>Sign in to manage your profile and security settings.</p>
      <div class="hero-actions">
        <NuxtLink class="primary-action" to="/account/login">Sign in</NuxtLink>
        <NuxtLink class="secondary-action" to="/account/register"
          >Create account</NuxtLink
        >
      </div>
    </div>
    <template v-else>
      <div class="account-heading">
        <div>
          <p class="eyebrow">Customer account</p>
          <h1>{{ data.customer.display_name }}</h1>
          <p class="account-summary">
            {{ data.customer.email }} · {{ data.customer.status }}
          </p>
        </div>
        <button class="secondary-action" type="button" @click="logout">
          Sign out
        </button>
      </div>
      <div class="account-grid">
        <form class="account-card account-form" @submit.prevent="updateProfile">
          <h2>Profile</h2>
          <label>
            Display name
            <input v-model.trim="profileName" maxlength="160" required />
          </label>
          <p v-if="profileMessage" class="account-notice" role="status">
            {{ profileMessage }}
          </p>
          <button class="primary-action" type="submit">Save profile</button>
        </form>
        <form
          class="account-card account-form"
          @submit.prevent="changePassword"
        >
          <h2>Password</h2>
          <label>
            Current password
            <input
              v-model="currentPassword"
              type="password"
              autocomplete="current-password"
              required
            />
          </label>
          <label>
            New password
            <input
              v-model="newPassword"
              type="password"
              autocomplete="new-password"
              minlength="12"
              required
            />
          </label>
          <p v-if="passwordMessage" class="account-error" role="alert">
            {{ passwordMessage }}
          </p>
          <button class="primary-action" type="submit">Change password</button>
          <small>Changing the password signs out every device.</small>
        </form>
      </div>
      <div class="account-danger">
        <div>
          <strong>Close account</strong>
          <p>
            Closure is permanent. The principal remains only for stable business
            references.
          </p>
        </div>
        <button type="button" class="danger-action" @click="closeAccount">
          Close permanently
        </button>
      </div>
      <button class="account-refresh" type="button" @click="reloadAccount">
        Reload current version
      </button>
    </template>
  </section>
</template>
