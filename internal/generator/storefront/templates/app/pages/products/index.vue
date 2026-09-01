<script setup lang="ts">
import type {
  CatalogPage,
  CatalogProduct,
} from "../../../shared/types/catalog";
import { formatCatalogPrice } from "../../utils/catalog";

const { data, error, status } = await useFetch<CatalogPage>(
  "/api/storefront/products",
  {
    key: "catalog-products",
    query: { limit: 24 },
  },
);
const products = ref<CatalogProduct[]>(data.value?.items ?? []);
const nextCursor = ref(data.value?.next_cursor ?? "");
const loadingMore = ref(false);
const loadMoreError = ref(false);

async function loadMore(): Promise<void> {
  if (!nextCursor.value || loadingMore.value) return;
  loadingMore.value = true;
  loadMoreError.value = false;
  try {
    const page = await $fetch<CatalogPage>("/api/storefront/products", {
      query: { limit: 24, cursor: nextCursor.value },
    });
    products.value.push(...page.items);
    nextCursor.value = page.next_cursor ?? "";
  } catch {
    loadMoreError.value = true;
  } finally {
    loadingMore.value = false;
  }
}
</script>

<template>
  <section class="catalog-page">
    <header class="catalog-heading">
      <p class="eyebrow">Catalog</p>
      <h1>Products made visible on purpose.</h1>
      <p>Only active products are returned by the public API.</p>
    </header>
    <p v-if="status === 'pending'" class="catalog-state" aria-live="polite">
      Loading products…
    </p>
    <p v-else-if="error" class="catalog-state" role="alert">
      The catalog is temporarily unavailable.
    </p>
    <p v-else-if="products.length === 0" class="catalog-state">
      No published products yet.
    </p>
    <div v-else class="product-grid">
      <NuxtLink
        v-for="product in products"
        :key="product.id"
        class="product-card"
        :to="`/products/${encodeURIComponent(product.id)}`"
      >
        <span class="product-sku">{{ product.sku }}</span>
        <h2>{{ product.name }}</h2>
        <p>{{ product.description || "Product details are coming soon." }}</p>
        <strong>{{
          formatCatalogPrice(product.price_minor, product.currency)
        }}</strong>
      </NuxtLink>
    </div>
    <button
      v-if="nextCursor"
      class="secondary-action load-more"
      type="button"
      :disabled="loadingMore"
      @click="loadMore"
    >
      {{ loadingMore ? "Loading…" : "Load more" }}
    </button>
    <p v-if="loadMoreError" class="catalog-state" role="alert">
      More products could not be loaded. Please try again.
    </p>
  </section>
</template>
