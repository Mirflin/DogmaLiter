<script setup>
import NavBar from '@/components/NavBar.vue'
import FooterBar from '@/components/FooterBar.vue'

defineProps({
  navOverlay: { type: Boolean, default: false },
})
</script>

<template>
  <div class="min-h-screen flex flex-col bg-n1">
    <NavBar :overlay="navOverlay" />
    <main class="relative flex-1">
      <slot />
    </main>
    <FooterBar />
  </div>
</template>
