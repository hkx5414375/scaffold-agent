<script setup lang="ts">
import type { CommerceOrderPage } from "../../../../shared/types/commerce";
import { formatCommerceMoney } from "../../../utils/commerce";

const { data, error, status } = await useFetch<CommerceOrderPage>(
  "/api/storefront/orders",
);
</script>

<template>
  <section class="commerce-page">
    <header>
      <p class="eyebrow">Account</p>
      <h1>Orders</h1>
    </header>
    <p v-if="status === 'pending'">Loading orders…</p>
    <p v-else-if="error" role="alert">Orders could not be loaded.</p>
    <div v-else class="commerce-panel">
      <p v-if="!data?.items.length">No orders yet.</p>
      <NuxtLink
        v-for="order in data?.items"
        :key="order.id"
        class="commerce-line"
        :to="`/account/orders/${encodeURIComponent(order.id)}`"
      >
        <span
          ><strong>{{ order.id }}</strong
          ><br /><small>{{ order.status }}</small></span
        >
        <strong>{{
          formatCommerceMoney(order.total_minor, order.currency)
        }}</strong>
      </NuxtLink>
    </div>
  </section>
</template>
