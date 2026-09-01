<script setup lang="ts">
defineProps<{ error: { statusCode?: number } }>();

const handleRecovery = async (): Promise<void> => {
  await clearError({ redirect: "/" });
};
</script>

<template>
  <main class="error-page">
    <p class="eyebrow">Storefront error</p>
    <h1>We could not open this page.</h1>
    <p>The failure was contained. Try the storefront home again.</p>
    <button class="primary-action" type="button" @click="handleRecovery">
      Return home
    </button>
  </main>
</template>
