<script setup>
import NavBar from '@/components/NavBar.vue'
import FooterBar from '@/components/FooterBar.vue'
import LandingSky from '@/components/landing/LandingSky.vue'

defineProps({
  navOverlay: { type: Boolean, default: false },
  // The calm night sky behind the page; the landing page draws its own, fuller one.
  backdrop: { type: Boolean, default: true },
})
</script>

<template>
  <div class="relative isolate min-h-screen flex flex-col bg-n1">
    <LandingSky v-if="backdrop" calm />
    <NavBar :overlay="navOverlay" />
    <main class="relative z-[1] flex-1">
      <slot />
    </main>
    <FooterBar class="relative z-[1]" />
  </div>
</template>
