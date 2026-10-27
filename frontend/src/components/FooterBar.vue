<script setup>
import logo from '@/assets/DLlogo.png'
import moment from 'moment';
</script>

<template>
  <footer class="site-footer">
    <div class="site-footer__inner max-w-[1400px] mx-auto px-6 py-8 flex flex-col sm:flex-row items-center justify-between gap-4">
      <div class="flex items-center gap-3">
        <img :src="logo" alt="DogmaLiter" class="h-7 w-7 object-contain opacity-60" />
        <span class="font-heading text-dim text-sm font-bold">DogmaLiter</span>
      </div>
      <div class="flex items-center gap-4">
        <router-link to="/terms" class="text-faint text-xs no-underline hover:text-dim transition-colors">Terms of Service</router-link>
        <span class="text-faint text-xs">|</span>
        <router-link to="/privacy" class="text-faint text-xs no-underline hover:text-dim transition-colors">Privacy Policy</router-link>
      </div>
      <span class="text-faint text-xs">&copy; {{ moment().format('YYYY') }} DogmaLiter. All rights reserved.</span>
    </div>
  </footer>
</template>

<style scoped>
.site-footer {
  position: relative;
  background: var(--color-n0);
}

.site-footer__inner {
  position: relative;
  z-index: 1;
}
</style>
