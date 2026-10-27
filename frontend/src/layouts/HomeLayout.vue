<script setup>
import NavBar from '@/components/NavBar.vue'
import FooterBar from '@/components/FooterBar.vue'
</script>

<template>
  <div class="min-h-screen flex flex-col bg-n1">
    <NavBar />
    <main class="relative flex-1">
      <slot />
    </main>
    <FooterBar />
  </div>
</template>
