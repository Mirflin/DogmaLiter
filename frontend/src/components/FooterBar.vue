<script setup>
import logo from '@/assets/DLlogo.png'
import moment from 'moment';
</script>

<template>
  <footer class="border-t border-[rgba(126,200,227,0.08)]">
    <div class="max-w-[1400px] mx-auto px-6 py-8 flex flex-col sm:flex-row items-center justify-between gap-4">
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
