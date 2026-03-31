<script setup>
import logo from '@/assets/DLlogo.png'
</script>

<template>
  <footer class="border-t border-[rgba(126,200,227,0.08)]">
    <div class="max-w-[1400px] mx-auto px-6 py-8 flex items-center justify-between">
      <div class="flex items-center gap-3">
        <img :src="logo" alt="DogmaLiter" class="h-7 w-7 object-contain opacity-60" />
        <span class="font-[Cinzel] text-[#e94560]/60 text-sm font-bold">DogmaLiter</span>
      </div>
      <span class="text-[#7ec8e3]/30 text-xs">&copy; 2026 DogmaLiter. All rights reserved.</span>
    </div>
  </footer>
</template>
