<script setup lang="ts">
import type { CommerceCart, CommerceOrder } from "../../shared/types/commerce";
import { formatCommerceMoney } from "../utils/commerce";

const config = useRuntimeConfig();
const currency = ref("USD");
const coupon = ref("");
const submitting = ref(false);
const failure = ref("");
const { data: cart } = await useFetch<CommerceCart>("/api/storefront/cart", {
  query: { currency },
});

async function checkout(): Promise<void> {
  if (!cart.value || submitting.value) return;
  submitting.value = true;
  failure.value = "";
  try {
    let order = await $fetch<CommerceOrder>("/api/storefront/checkout", {
      method: "POST",
      body: {
        currency: currency.value,
        coupon_code: coupon.value || undefined,
        idempotency_key: globalThis.crypto.randomUUID(),
        version: cart.value.version,
      },
    });
    if (
      config.public.commerceSandboxEnabled &&
      order.payment.status === "requires_action"
    ) {
      order = await $fetch<CommerceOrder>(
        `/api/storefront/sandbox/payments/${encodeURIComponent(order.payment.provider_ref)}`,
        {
          method: "POST",
          body: {
            event_id: globalThis.crypto.randomUUID(),
            status: "succeeded",
          },
        },
      );
    }
    await navigateTo(`/account/orders/${encodeURIComponent(order.id)}`);
  } catch {
    failure.value =
      "Checkout could not be completed. Reload your cart and try again.";
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <section class="commerce-page">
    <header>
      <p class="eyebrow">Checkout</p>
      <h1>Confirm the immutable price snapshot.</h1>
    </header>
    <div v-if="cart" class="commerce-panel">
      <div
        v-for="line in cart.lines"
        :key="line.product_id"
        class="commerce-line"
      >
        <span>{{ line.name }} × {{ line.quantity }}</span
        ><strong>{{
          formatCommerceMoney(line.line_minor, cart.currency)
        }}</strong>
      </div>
      <div class="commerce-total">
        <span>Subtotal before promotion</span
        ><span>{{
          formatCommerceMoney(cart.subtotal_minor, cart.currency)
        }}</span>
      </div>
      <label class="commerce-field"
        >Coupon code
        <input v-model.trim="coupon" maxlength="64" autocomplete="off"
      /></label>
      <p v-if="failure" role="alert">{{ failure }}</p>
      <button
        class="primary-action"
        type="button"
        :disabled="submitting || cart.lines.length === 0"
        @click="checkout"
      >
        {{ submitting ? "Submitting…" : "Place order" }}
      </button>
      <p v-if="config.public.commerceSandboxEnabled">
        <small
          >Local sandbox payment completion is enabled. No real charge is
          made.</small
        >
      </p>
    </div>
  </section>
</template>
