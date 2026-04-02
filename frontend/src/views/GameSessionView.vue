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
          <h2 class="font-[Cinzel] text-[24px] font-bold text-[#e8e8f0] mb-2">Game Session Active</h2>
          <p class="text-[#7ec8e3]/40 text-[14px] mb-10">This is your game workspace. Game features will appear here.</p>
          <div class="flex flex-col gap-2">
            <div class="flex items-center gap-3 px-5 py-3.5 bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.06)] rounded-xl text-[#e8e8f0]/50 opacity-50 cursor-default">
              <svg class="w-6 h-6" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 6a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0zM4.501 20.118a7.5 7.5 0 0114.998 0A17.933 17.933 0 0112 21.75c-2.676 0-5.216-.584-7.499-1.632z" />
              </svg>
              <span class="text-[14px] font-medium">Characters</span>
              <span class="ml-auto text-[11px] text-[#7ec8e3]/25 italic">Coming soon</span>
            </div>
            <div class="flex items-center gap-3 px-5 py-3.5 bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.06)] rounded-xl text-[#e8e8f0]/50 opacity-50 cursor-default">
              <svg class="w-6 h-6" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5M10 11.25h4M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z" />
              </svg>
              <span class="text-[14px] font-medium">Inventory</span>
              <span class="ml-auto text-[11px] text-[#7ec8e3]/25 italic">Coming soon</span>
            </div>
            <div class="flex items-center gap-3 px-5 py-3.5 bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.06)] rounded-xl text-[#e8e8f0]/50 opacity-50 cursor-default">
              <svg class="w-6 h-6" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9 6.75V15m6-6v8.25m.503 3.498l4.875-2.437c.381-.19.622-.58.622-1.006V4.82c0-.836-.88-1.38-1.628-1.006l-3.869 1.934c-.317.159-.69.159-1.006 0L9.503 3.252a1.125 1.125 0 00-1.006 0L3.622 5.689C3.24 5.88 3 6.27 3 6.695V19.18c0 .836.88 1.38 1.628 1.006l3.869-1.934c.317-.159.69-.159 1.006 0l4.994 2.497c.317.158.69.158 1.006 0z" />
              </svg>
              <span class="text-[14px] font-medium">Maps</span>
              <span class="ml-auto text-[11px] text-[#7ec8e3]/25 italic">Coming soon</span>
            </div>
            <div v-if="game.enable_chat" class="flex items-center gap-3 px-5 py-3.5 bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.06)] rounded-xl text-[#e8e8f0]/50 opacity-50 cursor-default">
              <svg class="w-6 h-6" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" d="M8.625 12a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H8.25m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H12m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0h-.375M21 12c0 4.556-4.03 8.25-9 8.25a9.764 9.764 0 01-2.555-.337A5.972 5.972 0 015.41 20.97a5.969 5.969 0 01-.474-.065 4.48 4.48 0 00.978-2.025c.09-.457-.133-.901-.467-1.226C3.93 16.178 3 14.189 3 12c0-4.556 4.03-8.25 9-8.25s9 3.694 9 8.25z" />
              </svg>
              <span class="text-[14px] font-medium">Chat</span>
              <span class="ml-auto text-[11px] text-[#7ec8e3]/25 italic">Coming soon</span>
            </div>
            <div v-if="isGM" class="flex items-center gap-3 px-5 py-3.5 bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.06)] rounded-xl text-[#e8e8f0]/50 opacity-50 cursor-default">
              <svg class="w-6 h-6" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.325.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 011.37.49l1.296 2.247a1.125 1.125 0 01-.26 1.431l-1.003.827c-.293.241-.438.613-.43.992a7.723 7.723 0 010 .255c-.008.378.137.75.43.991l1.004.827c.424.35.534.955.26 1.43l-1.298 2.247a1.125 1.125 0 01-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.47 6.47 0 01-.22.128c-.331.183-.581.495-.644.869l-.213 1.281c-.09.543-.56.94-1.11.94h-2.594c-.55 0-1.019-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 01-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 01-1.369-.49l-1.297-2.247a1.125 1.125 0 01.26-1.431l1.004-.827c.292-.24.437-.613.43-.991a6.932 6.932 0 010-.255c.007-.38-.138-.751-.43-.992l-1.004-.827a1.125 1.125 0 01-.26-1.43l1.297-2.247a1.125 1.125 0 011.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.086.22-.128.332-.183.582-.495.644-.869l.214-1.28z" /><circle cx="12" cy="12" r="3" />
              </svg>
              <span class="text-[14px] font-medium">GM Tools</span>
              <span class="ml-auto text-[11px] text-[#7ec8e3]/25 italic">Coming soon</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
