<script setup>
import logo from '@/assets/DLlogo.png'
import moment from 'moment';
</script>

<template>
  <footer class="site-footer border-t border-[rgba(126,200,227,0.08)]">
    <div class="site-footer__inner max-w-[1400px] mx-auto px-6 py-8 flex flex-col sm:flex-row items-center justify-between gap-4">
      <div class="flex items-center gap-3">
        <img :src="logo" alt="DogmaLiter" class="h-7 w-7 object-contain opacity-60" />
        <span class="font-[Cinzel] text-[#e94560]/60 text-sm font-bold">DogmaLiter</span>
      </div>
      <div class="flex items-center gap-4">
        <router-link to="/terms" class="text-[#7ec8e3]/40 text-xs no-underline hover:text-[#7ec8e3]/70 transition-colors">Terms of Service</router-link>
        <span class="text-[#7ec8e3]/20 text-xs">|</span>
        <router-link to="/privacy" class="text-[#7ec8e3]/40 text-xs no-underline hover:text-[#7ec8e3]/70 transition-colors">Privacy Policy</router-link>
      </div>
      <span class="text-[#7ec8e3]/30 text-xs">&copy; {{ moment().format('YYYY') }} DogmaLiter. All rights reserved.</span>
    </div>
  </footer>
</template>

<style scoped>
.site-footer {
  position: relative;
  background: linear-gradient(180deg, rgba(9, 13, 25, 0.76), rgba(7, 10, 19, 0.94));
  border-top-color: rgba(126, 200, 227, 0.1) !important;
}

.site-footer::before {
  content: '';
  position: absolute;
  inset: 0;
  pointer-events: none;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.03), transparent 70%),
    repeating-linear-gradient(90deg, rgba(255, 255, 255, 0.02) 0 1px, transparent 1px 120px);
}

.site-footer__inner {
  position: relative;
  z-index: 1;
}
</style>
