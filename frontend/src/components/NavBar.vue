<script setup>
import { ref, computed, onMounted, onBeforeUnmount, nextTick, watch } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useRouter, useRoute } from 'vue-router'
import { API_URL } from '@/api'
import logo from '@/assets/DLlogo.png'

defineProps({
  // Transparent over the page's own ground (landing); stays fixed at the top while scrolling.
  overlay: { type: Boolean, default: false },
})

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()
const showDropdown = ref(false)

const navLinks = ref(null)
const indicatorStyle = ref({ left: '0px', width: '0px', opacity: 0 })

const navItems = [
  { path: '/dashboard', label: 'Home' },
  { path: '/games', label: 'Games' },
  { path: '/news', label: 'News' },
  { path: '/plans', label: 'Subscriptions' },
]

const guestItems = [
  { to: '/#inventory', label: 'How it works' },
  { to: '/plans', label: 'Plans' },
  { to: '/news', label: 'News' },
]

// Over the page's own ground the bar is see-through; once the page scrolls, it gains a ground of
// its own so the content passing underneath never collides with the links.
const scrolled = ref(false)
function onScroll() {
  scrolled.value = window.scrollY > 8
}
onMounted(() => {
  onScroll()
  window.addEventListener('scroll', onScroll, { passive: true })
})
onBeforeUnmount(() => window.removeEventListener('scroll', onScroll))

function isGuestActive(item) {
  return !item.to.includes('#') && route.path.startsWith(item.to)
}

function updateIndicator() {
  if (!navLinks.value) return
  const activeEl = navLinks.value.querySelector('[data-active="true"]')
  if (activeEl) {
    const parentRect = navLinks.value.getBoundingClientRect()
    const rect = activeEl.getBoundingClientRect()
    indicatorStyle.value = {
      left: `${rect.left - parentRect.left}px`,
      width: `${rect.width}px`,
      opacity: 1,
    }
  } else {
    indicatorStyle.value = { ...indicatorStyle.value, opacity: 0 }
  }
}

watch(() => route.path, () => nextTick(updateIndicator))
onMounted(() => nextTick(updateIndicator))

function handleLogout() {
  auth.logout()
  showDropdown.value = false
  router.push('/')
}

const avatarPreview = ref(null)

const avatarUrl = computed(() => {
  if (avatarPreview.value) return avatarPreview.value
  if (auth.user?.avatar_id) return `${API_URL}/api/uploads/${auth.user.avatar_id}`
  return null
})

function isActive(path) {
  if (path === '/dashboard') return route.path === '/dashboard'
  return route.path.startsWith(path)
}
</script>

<template>
  <nav class="site-nav top-0 z-50 w-full" :class="overlay ? ['site-nav--overlay', { 'is-scrolled': scrolled }] : 'sticky bg-n0'">
    <div class="site-nav__inner max-w-[1400px] mx-auto px-6 h-16 flex items-center justify-between">
      <div class="flex items-center gap-8">
        <router-link to="/" class="site-nav__brand flex items-center gap-3 no-underline hover:opacity-90 transition-opacity">
          <img :src="logo" alt="DogmaLiter" class="site-nav__brand-mark h-10 w-10 object-contain" />
          <span class="font-heading text-[19px] font-bold text-ink tracking-tight hidden sm:inline">DogmaLiter</span>
        </router-link>

        <div v-if="auth.isAuthenticated" ref="navLinks" class="site-nav__links hidden md:flex items-center gap-1 relative">
          <div
            class="site-nav__indicator absolute bottom-0 h-[2px] bg-primary rounded-full transition-all duration-300 ease-in-out"
            :style="{ left: indicatorStyle.left, width: indicatorStyle.width, opacity: indicatorStyle.opacity }"
          />
          <router-link
            v-for="item in navItems"
            :key="item.path"
            :to="item.path"
            :data-active="isActive(item.path)"
            :aria-current="isActive(item.path) ? 'page' : undefined"
            :class="isActive(item.path) ? 'site-nav__link--active' : 'text-ink/70 hover:text-ink hover:bg-n3/70'"
            class="site-nav__link px-4 py-2 text-[14px] no-underline rounded-lg transition-all duration-200"
          >
            {{ item.label }}
          </router-link>
        </div>

        <div v-else class="hidden md:flex items-center gap-1">
          <router-link
            v-for="item in guestItems"
            :key="item.to"
            :to="item.to"
            :aria-current="isGuestActive(item) ? 'page' : undefined"
            :class="isGuestActive(item) ? 'site-nav__link--active' : 'text-ink/70 hover:text-ink hover:bg-n3/70'"
            class="site-nav__link px-4 py-2 text-[14px] no-underline rounded-lg transition-all duration-200"
          >
            {{ item.label }}
          </router-link>
        </div>
      </div>

      <div class="flex items-center gap-4">
        <div v-if="auth.isAuthenticated" class="relative">
          <button
            @click="showDropdown = !showDropdown"
            class="site-nav__account-button flex items-center gap-2 px-3 py-2 bg-transparent border-none cursor-pointer rounded-lg hover:bg-n3 transition-all duration-200"
          >
            <div class="w-8 h-8 rounded-full bg-primary/20 border border-primary/40 flex items-center justify-center">
              <img v-if="avatarUrl" :src="avatarUrl" alt="Avatar" class="w-full h-full object-cover rounded-full" />
              <span v-else class="text-primary-ink text-[13px] font-bold">{{ auth.user?.username?.charAt(0)?.toUpperCase() }}</span>
            </div>
            <span class="text-ink text-[14px] font-medium">{{ auth.user?.username }}</span>
            <span v-if="auth.user?.role === 'admin'" class="px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wider text-primary-ink bg-primary/15 border border-primary/30 rounded">Admin</span>
            <svg class="w-4 h-4 text-dim transition-transform duration-200" :class="{ 'rotate-180': showDropdown }" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
            </svg>
          </button>

          <Transition name="dropdown">
            <div
              v-if="showDropdown"
              class="site-nav__dropdown absolute right-0 top-[calc(100%+8px)] w-52 bg-n2/97 backdrop-blur-xl border border-line rounded-xl shadow-[0_15px_40px_rgba(0,0,0,0.5)] overflow-hidden"
            >
              <router-link
                to="/profile"
                @click="showDropdown = false"
                class="flex items-center gap-3 px-4 py-3 text-[14px] text-ink/80 no-underline hover:bg-n3 hover:text-ink transition-all duration-200"
              >
                <svg class="w-4 h-4 text-dim" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" /></svg>
                My Profile
              </router-link>
              <router-link
                to="/settings"
                @click="showDropdown = false"
                class="flex items-center gap-3 px-4 py-3 text-[14px] text-ink/80 no-underline hover:bg-n3 hover:text-ink transition-all duration-200"
              >
                <svg class="w-4 h-4 text-dim" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.573-1.066z" /><circle cx="12" cy="12" r="3" /></svg>
                Settings
              </router-link>
              <router-link
                v-if="auth.user?.role === 'admin'"
                to="/admin"
                @click="showDropdown = false"
                class="flex items-center gap-3 px-4 py-3 text-[14px] text-ink/80 no-underline hover:bg-n3 hover:text-ink transition-all duration-200"
              >
                <svg class="w-4 h-4 text-dim" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" /></svg>
                Admin Dashboard
              </router-link>
              <div class="border-t border-line-soft" />
              <button
                @click="handleLogout"
                class="w-full flex items-center gap-3 px-4 py-3 text-[14px] text-primary-ink bg-transparent border-none cursor-pointer hover:bg-primary/10 transition-all duration-200 font-[inherit]"
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
          class="site-nav__signin px-5 py-2 text-[14px] font-semibold text-white no-underline bg-primary hover:bg-primary-hover rounded-lg transition-all duration-300"
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

.site-nav {
  position: sticky;
  background: var(--color-n0) !important;
}

.site-nav--overlay {
  position: fixed;
  left: 0;
  background: transparent !important;
  transition: background-color 0.3s ease, backdrop-filter 0.3s ease;
}

.site-nav--overlay.is-scrolled {
  background: rgb(6 10 28 / 0.72) !important;
  backdrop-filter: blur(14px);
  -webkit-backdrop-filter: blur(14px);
}

.site-nav__link:active {
  transform: scale(0.96);
}

/* The current page: a lifted pill with a lit rim, so it reads at a glance on any ground. */
.site-nav__link--active {
  color: var(--color-ink);
  font-weight: 600;
  background: rgb(88 101 221 / 0.22);
  box-shadow: inset 0 0 0 1px rgb(164 174 255 / 0.35);
}

.site-nav__inner {
  position: relative;
  z-index: 1;
}

.site-nav__brand {
  padding: 0;
  border: none;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
}

.site-nav__brand-mark {
  filter: drop-shadow(0 8px 18px rgba(88, 101, 221, 0.18));
}

.site-nav__links {
  padding: 0;
  border: none;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
}

.site-nav__indicator {
  bottom: 0.35rem;
  height: 2px;
  box-shadow: 0 0 18px rgba(88, 101, 221, 0.32);
}

.site-nav__link {
  position: relative;
  z-index: 1;
}

.site-nav__account-button,
.site-nav__signin {
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.03);
}

.site-nav__account-button {
  border: 1px solid rgba(123, 135, 184, 0.12);
  background: rgba(9, 14, 27, 0.56);
}

.site-nav__dropdown {
  background: linear-gradient(180deg, rgba(13, 18, 33, 0.98), rgba(6, 10, 28, 0.98)) !important;
  box-shadow: 0 22px 55px rgba(0, 0, 0, 0.34);
}

.site-nav__signin {
  border: 1px solid rgba(88, 101, 221, 0.2);
}
</style>
