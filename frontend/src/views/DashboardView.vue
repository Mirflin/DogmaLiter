<script setup>
import HomeLayout from '@/layouts/HomeLayout.vue'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
</script>

<template>
  <HomeLayout>
  <div>
    <div class="max-w-[1400px] mx-auto px-6 py-8">
      <div class="flex gap-8">
        <div class="flex-1 min-w-0">
          <div class="mb-10">
            <div class="flex items-center justify-between mb-6">
              <h1 class="font-[Cinzel] text-[28px] font-bold text-[#e8e8f0] tracking-wide">Latest Games</h1>
              <div class="flex gap-3">
                <router-link
                  to="/games/create"
                  class="px-5 py-2.5 bg-linear-to-br from-[#e94560] to-[#c23152] text-white text-[13px] font-semibold no-underline rounded-lg border border-[#e94560] hover:-translate-y-0.5 hover:shadow-[0_6px_20px_rgba(233,69,96,0.4)] transition-all duration-300"
                >
                  Create New Game
                </router-link>
                <router-link
                  to="/games/join"
                  class="px-5 py-2.5 bg-transparent text-[#e8e8f0] text-[13px] font-semibold no-underline rounded-lg border border-[rgba(126,200,227,0.25)] hover:border-[#e94560] hover:text-[#e94560] transition-all duration-300"
                >
                  Join a Game
                </router-link>
              </div>
            </div>

            <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
              <div
                v-for="n in 6"
                :key="n"
                class="bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.1)] rounded-xl overflow-hidden hover:border-[rgba(233,69,96,0.3)] hover:-translate-y-1 transition-all duration-300 cursor-pointer group"
              >
                <div class="h-36 bg-[rgba(126,200,227,0.05)] flex items-center justify-center relative overflow-hidden">
                  <span class="text-[#7ec8e3]/20 text-sm">No cover image</span>
                  <div class="absolute inset-0 bg-linear-to-t from-[rgba(15,15,35,0.8)] to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300" />
                </div>
                <div class="p-4">
                  <h3 class="text-[#e8e8f0]/40 text-[15px] font-semibold mb-1">Game placeholder</h3>
                  <p class="text-[#7ec8e3]/30 text-xs mb-2">Next session: Not scheduled</p>
                  <span class="text-[#e94560]/40 text-xs font-medium">Launch game</span>
                </div>
              </div>
            </div>

            <router-link
              to="/games"
              class="inline-block mt-6 px-6 py-2.5 bg-[rgba(126,200,227,0.08)] text-[#7ec8e3]/70 text-[13px] font-semibold no-underline rounded-lg border border-[rgba(126,200,227,0.15)] hover:border-[rgba(126,200,227,0.3)] hover:text-[#7ec8e3] transition-all duration-300"
            >
              View All Games
            </router-link>
          </div>

          <div class="border-t border-[rgba(126,200,227,0.08)] pt-8">
            <div class="flex items-center justify-between mb-6">
              <h2 class="font-[Cinzel] text-[22px] font-bold text-[#e8e8f0] tracking-wide">Latest News</h2>
              <router-link to="/news" class="text-[#e94560] text-[13px] font-medium no-underline hover:text-[#ff6b81] transition-colors">
                View all &rarr;
              </router-link>
            </div>
            <div class="grid grid-cols-1 md:grid-cols-2 gap-5">
              <div v-for="n in 2" :key="n" class="flex gap-4 p-4 bg-[rgba(15,15,35,0.4)] border border-[rgba(126,200,227,0.06)] rounded-xl">
                <div class="w-20 h-20 rounded-lg bg-[rgba(126,200,227,0.05)] flex-shrink-0 flex items-center justify-center">
                  <span class="text-[#7ec8e3]/15 text-[10px]">img</span>
                </div>
                <div>
                  <p class="text-[#7ec8e3]/25 text-[11px] mb-1">Coming soon</p>
                  <h4 class="text-[#e8e8f0]/35 text-[14px] font-semibold">News post placeholder</h4>
                </div>
              </div>
            </div>
          </div>
        </div>

        <aside class="hidden lg:block w-72 flex-shrink-0">
          <div class="bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.1)] rounded-xl p-5 mb-6">
            <div class="flex items-center gap-4 mb-4">
              <div class="w-14 h-14 rounded-full bg-[rgba(233,69,96,0.15)] border border-[rgba(233,69,96,0.3)] flex items-center justify-center">
                <span class="text-[#e94560] text-[20px] font-bold font-[Cinzel]">{{ auth.user?.username?.charAt(0)?.toUpperCase() }}</span>
              </div>
              <div>
                <h3 class="text-[#e8e8f0] text-[16px] font-bold">{{ auth.user?.username }}</h3>
                <p class="text-[#7ec8e3]/40 text-[12px]">{{ auth.user?.email }}</p>
              </div>
            </div>
            <div class="space-y-2 text-[13px]">
              <div class="flex justify-between">
                <span class="text-[#7ec8e3]/40">Plan</span>
                <span class="text-[#e8e8f0]/60">Free</span>
              </div>
              <div class="flex justify-between">
                <span class="text-[#7ec8e3]/40">Role</span>
                <span class="text-[#e8e8f0]/60 capitalize">{{ auth.user?.role }}</span>
              </div>
            </div>
          </div>

          <div class="bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.1)] rounded-xl p-5">
            <h4 class="text-[#e8e8f0]/60 text-[12px] font-semibold tracking-wider uppercase mb-4">Quick Actions</h4>
            <div class="space-y-2">
              <router-link
                to="/games/create"
                class="flex items-center gap-3 px-3 py-2.5 text-[13px] text-[#e8e8f0]/60 no-underline rounded-lg hover:bg-[rgba(233,69,96,0.1)] hover:text-[#e94560] transition-all duration-200"
              >
                <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" /></svg>
                New Game
              </router-link>
              <router-link
                to="/games/join"
                class="flex items-center gap-3 px-3 py-2.5 text-[13px] text-[#e8e8f0]/60 no-underline rounded-lg hover:bg-[rgba(233,69,96,0.1)] hover:text-[#e94560] transition-all duration-200"
              >
                <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M18 7.5v3m0 0v3m0-3h3m-3 0h-3m-2.25-4.125a3.375 3.375 0 11-6.75 0 3.375 3.375 0 016.75 0zM3 19.235v-.11a6.375 6.375 0 0112.75 0v.109A12.318 12.318 0 019.374 21c-2.331 0-4.512-.645-6.374-1.766z" /></svg>
                Join Game
              </router-link>
            </div>
          </div>
        </aside>
      </div>
    </div>
  </div>
  </HomeLayout>
</template>
