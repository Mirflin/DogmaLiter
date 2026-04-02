<script setup>
import HomeLayout from '@/layouts/HomeLayout.vue'
import GameSettingsModal from '@/components/GameSettingsModal.vue'
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
const playerSearch = ref('')

const showSettingsModal = ref(false)

const gameId = computed(() => route.params.id)

const isGM = computed(() => game.value?.owner_id === auth.user?.id)

const filteredMembers = computed(() => {
  if (!game.value?.members) return []
  if (!playerSearch.value.trim()) return game.value.members
  const q = playerSearch.value.toLowerCase()
  return game.value.members.filter(m => m.username?.toLowerCase().includes(q))
})

function avatarUrl(avatarId) {
  if (avatarId) return `${API_URL}/api/uploads/${avatarId}`
  return null
}

function coverUrl() {
  if (game.value?.cover_image_id) return `${API_URL}/api/uploads/${game.value.cover_image_id}`
  return null
}

function formatDate(dateStr) {
  if (!dateStr) return '—'
  const d = new Date(dateStr)
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  const yy = String(d.getFullYear()).slice(2)
  return `${mm}/${dd}/${yy}`
}

async function loadGame() {
  loading.value = true
  error.value = null
  try {
    game.value = await auth.getGame(gameId.value)
  } catch (err) {
    error.value = err.response?.data?.error || 'Failed to load game'
  } finally {
    loading.value = false
  }
}

async function handleLeave() {
  if (!confirm(`Leave "${game.value.title}"?`)) return
  try {
    await auth.leaveGame(gameId.value)
    router.push('/games')
  } catch {}
}

async function handleDelete() {
  if (!confirm(`Delete "${game.value.title}"? This cannot be undone.`)) return
  try {
    await auth.deleteGame(gameId.value)
    router.push('/games')
  } catch {}
}

function onGameUpdated() {
  loadGame()
}

function onGameDeleted() {
  router.push('/games')
}

onMounted(loadGame)
</script>

<template>
  <HomeLayout>
    <div class="max-w-[1200px] mx-auto px-6 py-8">
      <div v-if="loading" class="text-center py-20">
        <div class="w-10 h-10 border-2 border-[#e94560] border-t-transparent rounded-full animate-spin mx-auto mb-4"></div>
        <p class="text-[#7ec8e3]/40 text-sm">Loading game...</p>
      </div>
      <div v-else-if="error" class="text-center py-20">
        <p class="text-[#e94560] text-sm mb-4">{{ error }}</p>
        <router-link to="/games" class="text-[#7ec8e3]/60 text-sm no-underline hover:text-[#7ec8e3]">&larr; Back to games</router-link>
      </div>
      <div v-else-if="game" class="flex gap-8">
        <div class="flex-1 min-w-0">
          <div class="w-full h-[300px] rounded-xl overflow-hidden bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.1)] mb-6">
            <img v-if="coverUrl()" :src="coverUrl()" :alt="game.title" class="w-full h-full object-cover" />
            <div v-else class="w-full h-full flex items-center justify-center">
              <svg class="w-16 h-16 text-[#7ec8e3]/10" fill="none" stroke="currentColor" stroke-width="1" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 15.75l5.159-5.159a2.25 2.25 0 013.182 0l5.159 5.159m-1.5-1.5l1.409-1.409a2.25 2.25 0 013.182 0l2.909 2.909M3.75 21h16.5A2.25 2.25 0 0022.5 18.75V5.25A2.25 2.25 0 0020.25 3H3.75A2.25 2.25 0 001.5 5.25v13.5A2.25 2.25 0 003.75 21z" />
              </svg>
            </div>
          </div>
          <h1 class="font-[Cinzel] text-[36px] font-bold text-[#e8e8f0] tracking-wide mb-6 leading-tight">{{ game.title }}</h1>
          <div class="flex items-center gap-4 mb-8">
            <button
              @click="router.push(`/games/${gameId}/play`)"
              class="flex items-center gap-2 px-6 py-3 bg-linear-to-br from-[#e94560] to-[#c23152] text-white text-[14px] font-semibold rounded-lg border border-[#e94560] hover:-translate-y-0.5 hover:shadow-[0_6px_20px_rgba(233,69,96,0.4)] transition-all duration-300 cursor-pointer"
            >
              <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24"><path d="M8 5v14l11-7z"/></svg>
              Launch Game
            </button>

            <button
              v-if="isGM"
              @click="showSettingsModal = true"
              class="flex items-center gap-2 px-5 py-3 bg-[rgba(126,200,227,0.08)] text-[#e8e8f0]/70 text-[14px] font-semibold rounded-lg border border-[rgba(126,200,227,0.15)] hover:border-[rgba(126,200,227,0.3)] hover:text-[#e8e8f0] transition-all duration-300 cursor-pointer"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.325.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 011.37.49l1.296 2.247a1.125 1.125 0 01-.26 1.431l-1.003.827c-.293.241-.438.613-.43.992a7.723 7.723 0 010 .255c-.008.378.137.75.43.991l1.004.827c.424.35.534.955.26 1.43l-1.298 2.247a1.125 1.125 0 01-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.47 6.47 0 01-.22.128c-.331.183-.581.495-.644.869l-.213 1.281c-.09.543-.56.94-1.11.94h-2.594c-.55 0-1.019-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 01-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 01-1.369-.49l-1.297-2.247a1.125 1.125 0 01.26-1.431l1.004-.827c.292-.24.437-.613.43-.991a6.932 6.932 0 010-.255c.007-.38-.138-.751-.43-.992l-1.004-.827a1.125 1.125 0 01-.26-1.43l1.297-2.247a1.125 1.125 0 011.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.086.22-.128.332-.183.582-.495.644-.869l.214-1.28z" /><circle cx="12" cy="12" r="3" /></svg>
              Settings
            </button>

            <button
              v-if="!isGM"
              @click="handleLeave"
              class="text-[#7ec8e3]/50 text-[14px] hover:text-[#e94560] transition-colors bg-transparent border-none cursor-pointer"
            >
              Leave Game
            </button>

            <button
              v-if="isGM"
              @click="handleDelete"
              class="text-[#7ec8e3]/50 text-[14px] hover:text-[#e94560] transition-colors bg-transparent border-none cursor-pointer"
            >
              Delete Game
            </button>
          </div>
          <div v-if="game.description" class="mb-8">
            <h3 class="text-[#e8e8f0]/60 text-[12px] font-semibold tracking-wider uppercase mb-3">Description</h3>
            <p class="text-[#e8e8f0]/70 text-[14px] leading-relaxed whitespace-pre-wrap">{{ game.description }}</p>
          </div>
        </div>
        <aside class="hidden lg:block w-[300px] flex-shrink-0">
          <div class="bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.1)] rounded-xl p-5 mb-6">
            <h4 class="text-[#e8e8f0]/60 text-[11px] font-semibold tracking-wider uppercase mb-4">Creator</h4>
            <div class="flex items-center gap-3">
              <div class="w-14 h-14 rounded-full bg-[rgba(233,69,96,0.15)] border border-[rgba(233,69,96,0.3)] flex items-center justify-center overflow-hidden flex-shrink-0">
                <img v-if="avatarUrl(game.owner?.avatar_id)" :src="avatarUrl(game.owner?.avatar_id)" class="w-full h-full object-cover" />
                <span v-else class="text-[#e94560] text-[18px] font-bold font-[Cinzel]">{{ game.owner?.username?.charAt(0)?.toUpperCase() }}</span>
              </div>
              <div class="min-w-0">
                <p class="text-[#e8e8f0] text-[15px] font-bold truncate">{{ game.owner?.username }}</p>
                <p v-if="game.owner?.plan_name" class="text-[#e94560] text-[12px]">{{ game.owner.plan_name }}</p>
                <p class="text-[#7ec8e3]/40 text-[11px]">Joined: {{ formatDate(game.owner?.created_at) }}</p>
              </div>
            </div>
          </div>
        </aside>
      </div>
    </div>

    <GameSettingsModal
      :visible="showSettingsModal"
      :game-id="gameId"
      @close="showSettingsModal = false"
      @updated="onGameUpdated"
      @deleted="onGameDeleted"
    />
  </HomeLayout>
</template>
