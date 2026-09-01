<template>
  <div class="site-frame">
    <NuxtRouteAnnouncer />
    <header class="site-header">
      <NuxtLink class="brand" to="/" aria-label="Storefront home">
        <span class="brand-mark" aria-hidden="true">S</span>
        <span>Storefront</span>
      </NuxtLink>
      <nav aria-label="Primary navigation">
        <NuxtLink to="/">Home</NuxtLink>[[if .Catalog]]
        <NuxtLink to="/products">Products</NuxtLink>[[end]][[if .CustomerAccounts]]
        <NuxtLink to="/account">Account</NuxtLink>[[end]][[if .Commerce]]
        <NuxtLink to="/cart">Cart</NuxtLink>
        <NuxtLink to="/account/orders">Orders</NuxtLink>[[end]]
      </nav>
    </header>
    <main>
      <NuxtPage />
    </main>
    <footer class="site-footer">
      <span>Deterministic by design.</span>
      <span>Built for safe extension.</span>
    </footer>
  </div>
</template>
