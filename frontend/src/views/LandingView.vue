<script setup>
import HomeLayout from '@/layouts/HomeLayout.vue'
import bgImage from '@/assets/background1.jpg'
import bg2Image from '@/assets/background2.jpg'
import logo from '@/assets/DLlogo.png'
</script>

<template>
  <HomeLayout>
  <div>
    <section class="relative overflow-hidden min-h-[600px] md:min-h-[700px] flex items-center">
      <div class="absolute inset-0">
        <img :src="bg2Image" alt="" class="w-full h-full object-cover blur-[3px] scale-105 opacity-30" />
      </div>
      <div class="absolute inset-0 bg-linear-to-b from-[#0a0a1a]/70 via-transparent to-[#0a0a1a]" />
      <div class="absolute inset-0 bg-linear-to-r from-[#0a0a1a]/80 via-[#0a0a1a]/30 to-[#0a0a1a]/60" />
      <div class="absolute inset-0 pointer-events-none">
        <div class="absolute top-32 left-20 w-80 h-80 bg-[#e94560]/10 rounded-full blur-[140px]" />
        <div class="absolute bottom-20 right-32 w-96 h-96 bg-[#7ec8e3]/8 rounded-full blur-[160px]" />
      </div>

      <div class="relative max-w-[1400px] mx-auto px-6 py-24 md:py-32 flex flex-col md:flex-row items-center gap-12">
        <div class="flex-1 max-w-2xl">
          <h1 class="font-[Cinzel] text-[40px] md:text-[54px] font-bold text-[#e8e8f0] leading-tight mb-6 tracking-wide">
            Your Tabletop,<br />
            <span class="text-transparent bg-clip-text bg-linear-to-r from-[#e94560] to-[#ff6b81]">Reimagined.</span>
          </h1>
          <p class="text-[#c8d6e5] text-lg md:text-xl leading-relaxed mb-4 max-w-lg">
            Manage your characters, inventories, and equipment online. Organize your tabletop campaigns and play with friends — all in your browser.
          </p>
          <ul class="list-none p-0 mb-10 space-y-2">
            <li class="flex items-center gap-2 text-[#7ec8e3]/70 text-[15px]">
              <svg class="w-4 h-4 text-[#e94560] flex-shrink-0" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" /></svg>
              Free to play — no credit card required
            </li>
            <li class="flex items-center gap-2 text-[#7ec8e3]/70 text-[15px]">
              <svg class="w-4 h-4 text-[#e94560] flex-shrink-0" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" /></svg>
              No downloads or hosting needed
            </li>
            <li class="flex items-center gap-2 text-[#7ec8e3]/70 text-[15px]">
              <svg class="w-4 h-4 text-[#e94560] flex-shrink-0" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" /></svg>
              Just open your browser and play
            </li>
          </ul>
          <router-link
            to="/auth"
            class="inline-block py-4 px-10 bg-linear-to-br from-[#e94560] to-[#c23152] text-white text-[16px] font-semibold no-underline rounded-xl hover:-translate-y-1 hover:shadow-[0_12px_35px_rgba(233,69,96,0.45)] transition-all duration-300 active:translate-y-0"
          >
            Create Free Account
          </router-link>
        </div>

        <div class="hidden md:flex flex-shrink-0 items-center justify-center">
          <img :src="logo" alt="DogmaLiter" class="w-64 h-64 object-contain drop-shadow-[0_0_40px_rgba(233,69,96,0.3)] animate-float" />
        </div>
      </div>
    </section>

    <section class="relative py-20 overflow-hidden">
      <div class="absolute inset-0 bg-[#0a0a1a]" />
      <div class="relative max-w-[1400px] mx-auto px-6">
        <h2 class="font-[Cinzel] text-[32px] font-bold text-[#e8e8f0] tracking-wide text-center mb-4">How It Works</h2>
        <p class="text-[#7ec8e3]/50 text-center text-[16px] mb-14 max-w-lg mx-auto">Get started in minutes. No complicated setup.</p>

        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
          <div class="relative p-7 bg-[rgba(15,15,35,0.5)] border border-[rgba(126,200,227,0.08)] rounded-2xl text-center hover:border-[rgba(233,69,96,0.25)] hover:-translate-y-1 transition-all duration-300 group">
            <div class="w-16 h-16 mx-auto mb-5 rounded-2xl bg-linear-to-br from-[#e94560] to-[#c23152] flex items-center justify-center shadow-[0_8px_25px_rgba(233,69,96,0.3)] group-hover:shadow-[0_12px_35px_rgba(233,69,96,0.4)] transition-shadow">
              <svg class="w-8 h-8 text-white" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M19 7.5v3m0 0v3m0-3h3m-3 0h-3m-2.25-4.125a3.375 3.375 0 11-6.75 0 3.375 3.375 0 016.75 0zM4 19.235v-.11a6.375 6.375 0 0112.75 0v.109A12.318 12.318 0 0110.374 21c-2.331 0-4.512-.645-6.374-1.766z" /></svg>
            </div>
            <h3 class="font-[Cinzel] text-[18px] font-bold text-[#e8e8f0] mb-2">Sign Up Free</h3>
            <p class="text-[#7ec8e3]/45 text-[14px] leading-relaxed">Create your account in seconds. Free tier available forever, whether you're a GM or player.</p>
          </div>

          <div class="relative p-7 bg-[rgba(15,15,35,0.5)] border border-[rgba(126,200,227,0.08)] rounded-2xl text-center hover:border-[rgba(126,200,227,0.25)] hover:-translate-y-1 transition-all duration-300 group">
            <div class="w-16 h-16 mx-auto mb-5 rounded-2xl bg-linear-to-br from-[#7ec8e3] to-[#4a9bb5] flex items-center justify-center shadow-[0_8px_25px_rgba(126,200,227,0.25)] group-hover:shadow-[0_12px_35px_rgba(126,200,227,0.35)] transition-shadow">
              <svg class="w-8 h-8 text-white" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M12 6.042A8.967 8.967 0 006 3.75c-1.052 0-2.062.18-3 .512v14.25A8.987 8.987 0 016 18c2.305 0 4.408.867 6 2.292m0-14.25a8.966 8.966 0 016-2.292c1.052 0 2.062.18 3 .512v14.25A8.987 8.987 0 0018 18a8.967 8.967 0 00-6 2.292m0-14.25v14.25" /></svg>
            </div>
            <h3 class="font-[Cinzel] text-[18px] font-bold text-[#e8e8f0] mb-2">Create a Game</h3>
            <p class="text-[#7ec8e3]/45 text-[14px] leading-relaxed">Set up your campaign with custom rules, maps, and items. Full creative control in your hands.</p>
          </div>

          <div class="relative p-7 bg-[rgba(15,15,35,0.5)] border border-[rgba(126,200,227,0.08)] rounded-2xl text-center hover:border-[rgba(167,139,250,0.25)] hover:-translate-y-1 transition-all duration-300 group">
            <div class="w-16 h-16 mx-auto mb-5 rounded-2xl bg-linear-to-br from-[#a78bfa] to-[#7c3aed] flex items-center justify-center shadow-[0_8px_25px_rgba(167,139,250,0.25)] group-hover:shadow-[0_12px_35px_rgba(167,139,250,0.35)] transition-shadow">
              <svg class="w-8 h-8 text-white" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M18 18.72a9.094 9.094 0 003.741-.479 3 3 0 00-4.682-2.72m.94 3.198l.001.031c0 .225-.012.447-.037.666A11.944 11.944 0 0112 21c-2.17 0-4.207-.576-5.963-1.584A6.062 6.062 0 016 18.719m12 0a5.971 5.971 0 00-.941-3.197m0 0A5.995 5.995 0 0012 12.75a5.995 5.995 0 00-5.058 2.772m0 0a3 3 0 00-4.681 2.72 8.986 8.986 0 003.74.477m.94-3.197a5.971 5.971 0 00-.94 3.197M15 6.75a3 3 0 11-6 0 3 3 0 016 0zm6 3a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0zm-13.5 0a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0z" /></svg>
            </div>
            <h3 class="font-[Cinzel] text-[18px] font-bold text-[#e8e8f0] mb-2">Find Your Party</h3>
            <p class="text-[#7ec8e3]/45 text-[14px] leading-relaxed">Invite friends with a link or let them join by code. Build your group and get ready to play.</p>
          </div>

          <div class="relative p-7 bg-[rgba(15,15,35,0.5)] border border-[rgba(126,200,227,0.08)] rounded-2xl text-center hover:border-[rgba(251,146,60,0.25)] hover:-translate-y-1 transition-all duration-300 group">
            <div class="w-16 h-16 mx-auto mb-5 rounded-2xl bg-linear-to-br from-[#fb923c] to-[#ea580c] flex items-center justify-center shadow-[0_8px_25px_rgba(251,146,60,0.25)] group-hover:shadow-[0_12px_35px_rgba(251,146,60,0.35)] transition-shadow">
              <svg class="w-8 h-8 text-white" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.347a1.125 1.125 0 010 1.972l-11.54 6.347a1.125 1.125 0 01-1.667-.986V5.653z" /></svg>
            </div>
            <h3 class="font-[Cinzel] text-[18px] font-bold text-[#e8e8f0] mb-2">Play</h3>
            <p class="text-[#7ec8e3]/45 text-[14px] leading-relaxed">That's it. No downloads, no hosting, no setup. Just click and play your adventure.</p>
          </div>
        </div>
      </div>
    </section>

    <section class="max-w-[1400px] mx-auto px-6 py-20">
      <h2 class="font-[Cinzel] text-[32px] font-bold text-[#e8e8f0] tracking-wide text-center mb-4">Everything You Need</h2>
      <p class="text-[#7ec8e3]/50 text-center text-[16px] mb-14 max-w-lg mx-auto">Powerful tools for GMs and players alike.</p>

      <div class="grid grid-cols-1 md:grid-cols-3 gap-8">
        <div class="p-8 bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.1)] rounded-2xl hover:border-[rgba(233,69,96,0.3)] hover:-translate-y-1 transition-all duration-300 group">
          <div class="w-14 h-14 rounded-xl bg-[rgba(233,69,96,0.15)] flex items-center justify-center mb-5 group-hover:bg-[rgba(233,69,96,0.25)] transition-colors">
            <svg class="w-7 h-7 text-[#e94560]" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M20.25 8.511c.884.284 1.5 1.128 1.5 2.097v4.286c0 1.136-.847 2.1-1.98 2.193-.34.027-.68.052-1.02.072v3.091l-3-3c-1.354 0-2.694-.055-4.02-.163a2.115 2.115 0 01-.825-.242m9.345-8.334a2.126 2.126 0 00-.476-.095 48.64 48.64 0 00-8.048 0c-1.131.094-1.976 1.057-1.976 2.192v4.286c0 .837.46 1.58 1.155 1.951m9.345-8.334V6.637c0-1.621-1.152-3.026-2.76-3.235A48.455 48.455 0 0011.25 3c-2.115 0-4.198.137-6.24.402-1.608.209-2.76 1.614-2.76 3.235v6.226c0 1.621 1.152 3.026 2.76 3.235.577.075 1.157.14 1.74.194V21l4.155-4.155" /></svg>
          </div>
          <h3 class="font-[Cinzel] text-[20px] font-bold text-[#e8e8f0] mb-3">Real-Time Chat</h3>
          <p class="text-[#7ec8e3]/50 text-[15px] leading-relaxed">Built-in chat where you can share items for inspection, discuss strategies, and keep the story flowing.</p>
        </div>

        <div class="p-8 bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.1)] rounded-2xl hover:border-[rgba(233,69,96,0.3)] hover:-translate-y-1 transition-all duration-300 group">
          <div class="w-14 h-14 rounded-xl bg-[rgba(233,69,96,0.15)] flex items-center justify-center mb-5 group-hover:bg-[rgba(233,69,96,0.25)] transition-colors">
            <svg class="w-7 h-7 text-[#e94560]" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M15.75 6a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0zM4.501 20.118a7.5 7.5 0 0114.998 0A17.933 17.933 0 0112 21.75c-2.676 0-5.216-.584-7.499-1.632z" /></svg>
          </div>
          <h3 class="font-[Cinzel] text-[20px] font-bold text-[#e8e8f0] mb-3">Character Sheets</h3>
          <p class="text-[#7ec8e3]/50 text-[15px] leading-relaxed">Fully customizable character sheets with attributes, inventory, and equipment management.</p>
        </div>

        <div class="p-8 bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.1)] rounded-2xl hover:border-[rgba(233,69,96,0.3)] hover:-translate-y-1 transition-all duration-300 group">
          <div class="w-14 h-14 rounded-xl bg-[rgba(233,69,96,0.15)] flex items-center justify-center mb-5 group-hover:bg-[rgba(233,69,96,0.25)] transition-colors">
            <svg class="w-7 h-7 text-[#e94560]" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M9 6.75V15m6-6v8.25m.503 3.498l4.875-2.437c.381-.19.622-.58.622-1.006V4.82c0-.836-.88-1.38-1.628-1.006l-3.869 1.934c-.317.159-.69.159-1.006 0L9.503 3.252a1.125 1.125 0 00-1.006 0L3.622 5.689C3.24 5.88 3 6.27 3 6.695V19.18c0 .836.88 1.38 1.628 1.006l3.869-1.934c.317-.159.69-.159 1.006 0l4.994 2.497c.317.158.69.158 1.006 0z" /></svg>
          </div>
          <h3 class="font-[Cinzel] text-[20px] font-bold text-[#e8e8f0] mb-3">Inventory System</h3>
          <p class="text-[#7ec8e3]/50 text-[15px] leading-relaxed">Manage items, equipment, and loot. Trade between characters and track everything your party carries.</p>
        </div>
      </div>
    </section>

    <section class="max-w-[1400px] mx-auto px-6 py-16 border-t border-[rgba(126,200,227,0.08)]">
      <div class="flex items-center justify-between mb-10">
        <h2 class="font-[Cinzel] text-[28px] font-bold text-[#e8e8f0] tracking-wide">Latest News</h2>
        <router-link to="/news" class="text-[#e94560] text-[14px] font-medium no-underline hover:text-[#ff6b81] transition-colors">
          View all &rarr;
        </router-link>
      </div>
      <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div v-for="n in 3" :key="n" class="bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.08)] rounded-xl overflow-hidden hover:border-[rgba(126,200,227,0.18)] hover:-translate-y-1 transition-all duration-300 cursor-pointer">
          <div class="h-40 bg-[rgba(126,200,227,0.05)] flex items-center justify-center">
            <span class="text-[#7ec8e3]/20 text-sm">No image</span>
          </div>
          <div class="p-5">
            <p class="text-[#7ec8e3]/30 text-xs mb-2">Coming soon</p>
            <h3 class="text-[#e8e8f0]/40 text-[16px] font-semibold">News post placeholder</h3>
          </div>
        </div>
      </div>
    </section>

    <section class="relative py-20 overflow-hidden">
      <div class="absolute inset-0">
        <img :src="bgImage" alt="" class="w-full h-full object-cover blur-[4px] scale-105 opacity-20" />
      </div>
      <div class="absolute inset-0 bg-[#0a0a1a]/75" />
      <div class="relative max-w-[700px] mx-auto px-6 text-center">
        <h2 class="font-[Cinzel] text-[32px] md:text-[38px] font-bold text-[#e8e8f0] mb-5 tracking-wide">Ready to Start Your Adventure?</h2>
        <p class="text-[#7ec8e3]/60 text-[16px] mb-10">Join DogmaLiter and bring your tabletop stories to life. It's free.</p>
        <router-link
          to="/auth"
          class="inline-block py-4 px-12 bg-linear-to-br from-[#e94560] to-[#c23152] text-white text-[16px] font-semibold no-underline rounded-xl hover:-translate-y-1 hover:shadow-[0_12px_35px_rgba(233,69,96,0.45)] transition-all duration-300"
        >
          Get Started — It's Free
        </router-link>
      </div>
    </section>
  </div>
  </HomeLayout>
</template>

<style scoped>
@keyframes float {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-12px); }
}
.animate-float {
  animation: float 4s ease-in-out infinite;
}
</style>
