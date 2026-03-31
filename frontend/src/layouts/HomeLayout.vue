<script setup>
import NavBar from '@/components/NavBar.vue'
import FooterBar from '@/components/FooterBar.vue'
</script>

<template>
  <div class="min-h-screen flex flex-col bg-[#0a0a1a]">
    <NavBar />
    <main class="flex-1">
      <slot />
    </main>
    <FooterBar />
  </div>
</template>
