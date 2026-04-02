<script setup>
import { useAuthStore } from '@/stores/auth'
import { API_URL } from '@/api'
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

const game = ref(null)
const loading = ref(true)
const error = ref(null)

const gameId = computed(() => route.params.id)
const isGM = computed(() => game.value?.owner_id === auth.user?.id)

function avatarUrl(avatarId) {
  if (avatarId) return `${API_URL}/api/uploads/${avatarId}`
  return null
}

function coverUrl() {
  if (game.value?.cover_image_id) return `${API_URL}/api/uploads/${game.value.cover_image_id}`
  return null
}

async function loadGame() {
  loading.value = true
  error.value = null
  try {
    game.value = await auth.getGame(gameId.value)
  } catch (err) {
    if (err.response?.status === 403) {
      error.value = 'You do not have access to this game'
    } else {
      error.value = err.response?.data?.error || 'Failed to load game'
    }
  } finally {
    loading.value = false
  }
}

function goBack() {
  router.push(`/games/${gameId.value}`)
}

onMounted(loadGame)
</script>

<template>
  <div class="w-screen h-screen bg-[#0a0a1a] overflow-hidden flex flex-col">
    <div v-if="loading" class="flex-1 flex flex-col items-center justify-center">
      <div class="w-10 h-10 border-2 border-[#e94560] border-t-transparent rounded-full animate-spin"></div>
      <p class="text-[#7ec8e3]/40 text-[14px] mt-4">Loading game session...</p>
    </div>
    <div v-else-if="error" class="flex-1 flex flex-col items-center justify-center">
      <svg class="w-16 h-16 text-[#e94560] mb-4" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
      </svg>
      <p class="text-[#e94560] text-[16px] font-semibold mb-2">{{ error }}</p>
      <button
        @click="router.push('/games')"
        class="px-6 py-2.5 bg-transparent text-[#7ec8e3] border border-[rgba(126,200,227,0.2)] rounded-lg cursor-pointer text-[14px] transition-all duration-200 hover:border-[#e94560] hover:text-[#e94560]"
      >
        Back to Games
      </button>
    </div>
    <div v-else-if="game" class="flex flex-col h-full">
      <header class="flex items-center justify-between px-6 py-3 bg-[rgba(15,15,35,0.8)] border-b border-[rgba(126,200,227,0.08)] shrink-0">
        <div class="flex items-center gap-4">
          <button
            @click="goBack"
            title="Exit session"
            class="w-9 h-9 flex items-center justify-center bg-[rgba(126,200,227,0.06)] border border-[rgba(126,200,227,0.1)] rounded-lg text-[#e8e8f0]/60 cursor-pointer transition-all duration-200 hover:border-[#e94560] hover:text-[#e94560]"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" d="M10.5 19.5L3 12m0 0l7.5-7.5M3 12h18" />
            </svg>
          </button>
          <div class="flex items-center gap-2.5">
            <h1 class="font-[Cinzel] text-[18px] font-bold text-[#e8e8f0] m-0 tracking-wide">{{ game.title }}</h1>
            <span
              class="px-2 py-0.5 rounded text-[11px] font-semibold uppercase tracking-wider"
              :class="isGM
                ? 'bg-[rgba(233,69,96,0.15)] text-[#e94560] border border-[rgba(233,69,96,0.3)]'
                : 'bg-[rgba(126,200,227,0.1)] text-[#7ec8e3]/60 border border-[rgba(126,200,227,0.2)]'"
            >
              {{ isGM ? 'GM' : 'Player' }}
            </span>
          </div>
        </div>
      </header>
      <div class="flex-1 flex items-center justify-center overflow-auto">
        <div class="text-center max-w-[500px] p-10">
          <div class="text-[#7ec8e3]/15 mb-6 flex justify-center">
            <svg class="w-20 h-20" fill="none" stroke="currentColor" stroke-width="1" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6zM3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25zM13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6zM13.5 15.75a2.25 2.25 0 012.25-2.25H18a2.25 2.25 0 012.25 2.25V18A2.25 2.25 0 0118 20.25h-2.25A2.25 2.25 0 0113.5 18v-2.25z" />
            </svg>
          </div>
          <h2 class="font-[Cinzel] text-[24px] font-bold text-[#e8e8f0] mb-2">Game Session placehodler</h2>
          <div class="flex flex-col gap-2">

          </div>
        </div>
      </div>
    </div>
  </div>
</template>
