<script setup lang="ts">
import type { CommerceCart } from "../../shared/types/commerce";
import { formatCommerceMoney } from "../utils/commerce";

const currency = ref("USD");
const {
  data: cart,
  error,
  refresh,
  status,
} = await useFetch<CommerceCart>("/api/storefront/cart", {
  query: { currency },
});
const busy = ref(false);

async function update(productID: string, quantity: string): Promise<void> {
  if (!cart.value || busy.value) return;
  busy.value = true;
  try {
    await $fetch(
      `/api/storefront/cart/lines/${encodeURIComponent(productID)}`,
      {
        method: "PUT",
        body: {
          currency: currency.value,
          quantity,
          version: cart.value.version,
        },
      },
    );
    await refresh();
  } finally {
    busy.value = false;
  }
}

async function remove(productID: string): Promise<void> {
  if (!cart.value || busy.value) return;
  busy.value = true;
  try {
    await $fetch(
      `/api/storefront/cart/lines/${encodeURIComponent(productID)}/remove`,
      {
        method: "POST",
        body: { currency: currency.value, version: cart.value.version },
      },
    );
    await refresh();
  } finally {
    busy.value = false;
  }
}
</script>

<template>
  <section class="commerce-page">
    <header>
      <p class="eyebrow">Cart</p>
      <h1>Your current order.</h1>
    </header>
    <p v-if="status === 'pending'">Loading cart…</p>
    <p v-else-if="error" role="alert">Sign in to view your cart.</p>
    <div v-else-if="cart" class="commerce-panel">
      <p v-if="cart.lines.length === 0">Your cart is empty.</p>
      <div
        v-for="line in cart.lines"
        v-else
        :key="line.product_id"
        class="commerce-line"
      >
        <div>
          <strong>{{ line.name }}</strong
          ><br /><small>{{ line.sku }}</small>
        </div>
        <label
          >Quantity
          <input
            :value="line.quantity"
            type="number"
            min="1"
            max="999"
            :disabled="busy"
            @change="
              update(line.product_id, ($event.target as HTMLInputElement).value)
            "
        /></label>
        <button
          class="secondary-action"
          type="button"
          :disabled="busy"
          @click="remove(line.product_id)"
        >
          Remove
        </button>
      </div>
      <div class="commerce-total">
        <span>Subtotal</span
        ><span>{{
          formatCommerceMoney(cart.subtotal_minor, cart.currency)
        }}</span>
      </div>
      <NuxtLink v-if="cart.lines.length" class="primary-action" to="/checkout"
        >Checkout</NuxtLink
      >
    </div>
  </section>
</template>
