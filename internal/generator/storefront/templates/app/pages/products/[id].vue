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
}
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
    }}</strong>
  </article>
</template>
