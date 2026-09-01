<script setup lang="ts">
import type { CommerceOrder } from "../../../../shared/types/commerce";
import { formatCommerceMoney } from "../../../utils/commerce";

const route = useRoute();
const id = computed(() => String(route.params.id));
const {
  data: order,
  error,
  refresh,
} = await useFetch<CommerceOrder>(
  () => `/api/storefront/orders/${encodeURIComponent(id.value)}`,
);
const reason = ref("");
const submitting = ref(false);

async function requestReturn(): Promise<void> {
  if (!order.value || submitting.value) return;
  submitting.value = true;
  try {
    await $fetch(
      `/api/storefront/orders/${encodeURIComponent(order.value.id)}/return`,
      {
        method: "POST",
        body: { version: order.value.version, reason: reason.value },
      },
    );
    await refresh();
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <section class="commerce-page">
    <p v-if="error" role="alert">Order could not be loaded.</p>
    <template v-else-if="order">
      <header>
        <p class="eyebrow">Order {{ order.id }}</p>
        <h1>{{ order.status }}</h1>
      </header>
      <div class="commerce-panel">
        <div
          v-for="line in order.lines"
          :key="line.product_id"
          class="commerce-line"
        >
          <span>{{ line.name }} × {{ line.quantity }}</span
          ><strong>{{
            formatCommerceMoney(line.line_minor, order.currency)
          }}</strong>
        </div>
        <div class="commerce-total">
          <span>Total</span
          ><span>{{
            formatCommerceMoney(order.total_minor, order.currency)
          }}</span>
        </div>
        <p>
          Payment: {{ order.payment.status }} · Refunded
          {{
            formatCommerceMoney(order.payment.refunded_minor, order.currency)
          }}
        </p>
        <div v-if="order.status === 'fulfilled'" class="commerce-actions">
          <label class="commerce-field"
            >Return reason
            <input v-model.trim="reason" maxlength="500" /></label
          ><button
            class="secondary-action"
            type="button"
            :disabled="submitting || !reason"
            @click="requestReturn"
          >
            Request return
          </button>
        </div>
        <p v-if="order.return_reason">Return: {{ order.return_reason }}</p>
      </div>
    </template>
  </section>
</template>
