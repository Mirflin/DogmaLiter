<script setup>
import { ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useRouter } from 'vue-router'
import logo from '@/assets/DLlogo.png'

const auth = useAuthStore()
const router = useRouter()
const showDropdown = ref(false)

function handleLogout() {
  auth.logout()
  showDropdown.value = false
  router.push('/auth')
}
</script>

<template>
  <nav class="sticky top-0 z-50 w-full bg-[rgba(10,10,26,0.95)] backdrop-blur-md border-b border-[rgba(126,200,227,0.1)]">
    <div class="max-w-[1400px] mx-auto px-6 h-16 flex items-center justify-between">
      <div class="flex items-center gap-8">
        <router-link to="/" class="flex items-center gap-3 no-underline hover:opacity-90 transition-opacity">
          <img :src="logo" alt="DogmaLiter" class="h-10 w-10 object-contain" />
          <span class="font-[Cinzel] text-[20px] font-bold text-[#e94560] tracking-wider hidden sm:inline">DogmaLiter</span>
        </router-link>

        <div v-if="auth.isAuthenticated" class="hidden md:flex items-center gap-1">
          <router-link
            to="/games"
            class="px-4 py-2 text-[14px] text-[#e8e8f0]/70 no-underline rounded-lg hover:text-[#e8e8f0] hover:bg-[rgba(126,200,227,0.08)] transition-all duration-200"
          >
            Games
          </router-link>
          <router-link
            to="/news"
            class="px-4 py-2 text-[14px] text-[#e8e8f0]/70 no-underline rounded-lg hover:text-[#e8e8f0] hover:bg-[rgba(126,200,227,0.08)] transition-all duration-200"
          >
            News
          </router-link>
        </div>

        <div v-else class="hidden md:flex items-center gap-1">
          <router-link
            to="/news"
            class="px-4 py-2 text-[14px] text-[#e8e8f0]/70 no-underline rounded-lg hover:text-[#e8e8f0] hover:bg-[rgba(126,200,227,0.08)] transition-all duration-200"
          >
            News
          </router-link>
        </div>
      </div>

      <div class="flex items-center gap-4">
        <div v-if="auth.isAuthenticated" class="relative">
          <button
            @click="showDropdown = !showDropdown"
            class="flex items-center gap-2 px-3 py-2 bg-transparent border-none cursor-pointer rounded-lg hover:bg-[rgba(126,200,227,0.08)] transition-all duration-200"
          >
            <div class="w-8 h-8 rounded-full bg-[rgba(233,69,96,0.2)] border border-[rgba(233,69,96,0.4)] flex items-center justify-center">
              <span class="text-[#e94560] text-[13px] font-bold">{{ auth.user?.username?.charAt(0)?.toUpperCase() }}</span>
            </div>
            <span class="text-[#e8e8f0] text-[14px] font-medium">{{ auth.user?.username }}</span>
            <svg class="w-4 h-4 text-[#7ec8e3]/50 transition-transform duration-200" :class="{ 'rotate-180': showDropdown }" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
            </svg>
          </button>

          <Transition name="dropdown">
            <div
              v-if="showDropdown"
              class="absolute right-0 top-[calc(100%+8px)] w-52 bg-[rgba(15,15,35,0.97)] backdrop-blur-xl border border-[rgba(126,200,227,0.15)] rounded-xl shadow-[0_15px_40px_rgba(0,0,0,0.5)] overflow-hidden"
            >
              <router-link
                to="/profile"
                @click="showDropdown = false"
                class="flex items-center gap-3 px-4 py-3 text-[14px] text-[#e8e8f0]/80 no-underline hover:bg-[rgba(126,200,227,0.08)] hover:text-[#e8e8f0] transition-all duration-200"
              >
                <svg class="w-4 h-4 text-[#7ec8e3]/50" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" /></svg>
                My Profile
              </router-link>
              <router-link
                to="/settings"
                @click="showDropdown = false"
                class="flex items-center gap-3 px-4 py-3 text-[14px] text-[#e8e8f0]/80 no-underline hover:bg-[rgba(126,200,227,0.08)] hover:text-[#e8e8f0] transition-all duration-200"
              >
                <svg class="w-4 h-4 text-[#7ec8e3]/50" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.573-1.066z" /><circle cx="12" cy="12" r="3" /></svg>
                Settings
              </router-link>
              <div class="border-t border-[rgba(126,200,227,0.1)]" />
              <button
                @click="handleLogout"
                class="w-full flex items-center gap-3 px-4 py-3 text-[14px] text-[#ff8fa3] bg-transparent border-none cursor-pointer hover:bg-[rgba(233,69,96,0.1)] transition-all duration-200 font-[inherit]"
              >
                <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" /></svg>
                Sign Out
              </button>
            </div>
          </Transition>
        </div>

        <router-link
          v-else
          to="/auth"
          class="px-5 py-2 text-[14px] font-semibold text-white no-underline bg-linear-to-br from-[#e94560] to-[#c23152] rounded-lg hover:-translate-y-0.5 hover:shadow-[0_6px_20px_rgba(233,69,96,0.4)] transition-all duration-300"
        >
          Sign In
        </router-link>
      </div>
    </div>

    <div v-if="showDropdown" class="fixed inset-0 z-[-1]" @click="showDropdown = false" />
  </nav>
</template>

<style scoped>
.dropdown-enter-active,
.dropdown-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
