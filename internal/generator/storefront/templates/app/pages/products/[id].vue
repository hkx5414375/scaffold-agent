<script setup lang="ts">
import type { CatalogProduct } from "../../../shared/types/catalog";
import { formatCatalogPrice } from "../../utils/catalog";

const route = useRoute();
const productID = Array.isArray(route.params.id)
  ? route.params.id[0]
  : route.params.id;
const { data: product, error } = await useFetch<CatalogProduct>(
  `/api/storefront/products/${encodeURIComponent(productID ?? "")}`,
  { key: `catalog-product-${productID ?? "missing"}` },
);
if (error.value) {
  throw createError({
    statusCode: error.value.statusCode === 404 ? 404 : 502,
    statusMessage:
      error.value.statusCode === 404
        ? "Product was not found"
        : "Catalog is temporarily unavailable",
  });
}[[if .Commerce]]
const adding = ref(false);
const addError = ref(false);

async function addToCart(): Promise<void> {
  if (!product.value || adding.value) return;
  adding.value = true;
  addError.value = false;
  try {
    const cart = await $fetch<{ version: string }>("/api/storefront/cart", {
      query: { currency: product.value.currency },
    });
    await $fetch(
      `/api/storefront/cart/lines/${encodeURIComponent(product.value.id)}`,
      {
        method: "PUT",
        body: {
          currency: product.value.currency,
          quantity: "1",
          version: cart.version,
        },
      },
    );
    await navigateTo("/cart");
  } catch {
    addError.value = true;
  } finally {
    adding.value = false;
  }
}[[end]]
</script>

<template>
  <article v-if="product" class="product-detail">
    <NuxtLink class="back-link" to="/products">← All products</NuxtLink>
    <p class="product-sku">{{ product.sku }}</p>
    <h1>{{ product.name }}</h1>
    <p class="product-description">
      {{ product.description || "No description is available." }}
    </p>
    <strong class="product-price">{{
      formatCatalogPrice(product.price_minor, product.currency)
    }}</strong>[[if .Commerce]]
    <button
      class="primary-action"
      type="button"
      :disabled="adding"
      @click="addToCart"
    >
      {{ adding ? "Adding…" : "Add to cart" }}
    </button>
    <p v-if="addError" role="alert">
      Sign in before adding this product, then try again.
    </p>[[end]]
  </article>
</template>
