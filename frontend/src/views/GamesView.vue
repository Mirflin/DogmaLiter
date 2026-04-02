<script setup>
import HomeLayout from '@/layouts/HomeLayout.vue'
import JoinGameModal from '@/components/JoinGameModal.vue'
import GameSettingsModal from '@/components/GameSettingsModal.vue'
import { useAuthStore } from '@/stores/auth'
import { API_URL } from '@/api'
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'

const auth = useAuthStore()
const router = useRouter()
const games = ref([])
const loading = ref(true)
const searchQuery = ref('')

const showJoinModal = ref(false)
const showSettingsModal = ref(false)
const settingsGameId = ref(null)

const contextMenu = ref({ visible: false, x: 0, y: 0, game: null })

onMounted(async () => {
  try {
    games.value = await auth.fetchMyGames()
  } catch {
    games.value = []
  } finally {
    loading.value = false
  }
  document.addEventListener('click', closeContextMenu)
})

onUnmounted(() => {
  document.removeEventListener('click', closeContextMenu)
})

const filteredGames = computed(() => {
  if (!searchQuery.value.trim()) return games.value
  const q = searchQuery.value.toLowerCase()
  return games.value.filter(g => g.title.toLowerCase().includes(q))
})

function isGM(game) {
  return game.owner_id === auth.user?.id
}

function coverUrl(game) {
  if (game.cover_image_id) return `${API_URL}/api/uploads/${game.cover_image_id}`
  return null
}

function memberAvatarUrl(member) {
  if (member.avatar_id) return `${API_URL}/api/uploads/${member.avatar_id}`
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

function openContextMenu(e, game) {
  e.preventDefault()
  contextMenu.value = { visible: true, x: e.clientX, y: e.clientY, game }
}

function closeContextMenu() {
  contextMenu.value.visible = false
}

function openSettings(game) {
  closeContextMenu()
  settingsGameId.value = game.id
  showSettingsModal.value = true
}

async function handleLeave(game) {
  closeContextMenu()
  if (!confirm(`Leave "${game.title}"?`)) return
  try {
    await auth.leaveGame(game.id)
    games.value = games.value.filter(g => g.id !== game.id)
  } catch {}
}

async function handleDelete(game) {
  closeContextMenu()
  if (!confirm(`Delete "${game.title}"? This cannot be undone.`)) return
  try {
    await auth.deleteGame(game.id)
    games.value = games.value.filter(g => g.id !== game.id)
  } catch {}
}

async function refreshGames() {
  try {
    games.value = await auth.fetchMyGames()
  } catch {}
}

function onGameDeleted(gameId) {
  games.value = games.value.filter(g => g.id !== gameId)
}

function onGameJoined() {
  refreshGames()
}
</script>

<template>
  <HomeLayout>
    <div class="max-w-[900px] mx-auto px-6 py-8">
      <div class="flex items-center justify-between mb-6">
        <h1 class="font-[Cinzel] text-[28px] font-bold text-[#e8e8f0] tracking-wide">My Games</h1>
        <div class="flex gap-3">
          <router-link
            to="/games/create"
            class="px-5 py-2.5 bg-linear-to-br from-[#e94560] to-[#c23152] text-white text-[13px] font-semibold no-underline rounded-lg border border-[#e94560] hover:-translate-y-0.5 hover:shadow-[0_6px_20px_rgba(233,69,96,0.4)] transition-all duration-300"
          >
            Create New Game
          </router-link>
          <button
            @click="showJoinModal = true"
            class="px-5 py-2.5 bg-transparent text-[#e8e8f0] text-[13px] font-semibold rounded-lg border border-[rgba(126,200,227,0.25)] hover:border-[#e94560] hover:text-[#e94560] transition-all duration-300 cursor-pointer"
          >
            Join a Game
          </button>
        </div>
      </div>
      <div class="relative mb-8">
        <svg class="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-[#7ec8e3]/40" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z" />
        </svg>
        <input
          v-model="searchQuery"
          type="text"
          placeholder="Search: enter a game name..."
          class="w-full pl-12 pr-4 py-3 bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.15)] rounded-lg text-[#e8e8f0] text-[14px] placeholder-[#7ec8e3]/30 outline-none focus:border-[rgba(233,69,96,0.4)] transition-colors"
        />
      </div>
      <div v-if="loading" class="text-center py-16">
        <div class="w-8 h-8 border-2 border-[#e94560] border-t-transparent rounded-full animate-spin mx-auto mb-3"></div>
        <p class="text-[#7ec8e3]/40 text-sm">Loading games...</p>
      </div>
      <div v-else-if="filteredGames.length === 0 && !searchQuery" class="text-center py-16">
        <p class="text-[#7ec8e3]/40 text-sm mb-4">You don't have any games yet</p>
      </div>
      <div v-else-if="filteredGames.length === 0 && searchQuery" class="text-center py-16">
        <p class="text-[#7ec8e3]/40 text-sm">No results found for "{{ searchQuery }}"</p>
      </div>
      <div v-else class="space-y-4">
        <div
          v-for="game in filteredGames"
          :key="game.id"
          @click="router.push(`/games/${game.id}`)"
          @contextmenu="openContextMenu($event, game)"
          class="flex gap-5 p-5 bg-[rgba(15,15,35,0.5)] border border-[rgba(126,200,227,0.08)] rounded-xl hover:border-[rgba(233,69,96,0.25)] transition-all duration-300 cursor-pointer"
        >
          <div class="w-[120px] h-[120px] flex-shrink-0 rounded-lg overflow-hidden bg-[rgba(126,200,227,0.05)]">
            <img
              v-if="coverUrl(game)"
              :src="coverUrl(game)"
              :alt="game.title"
              class="w-full h-full object-cover"
            />
            <div v-else class="w-full h-full flex items-center justify-center">
              <svg class="w-10 h-10 text-[#7ec8e3]/15" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 15.75l5.159-5.159a2.25 2.25 0 013.182 0l5.159 5.159m-1.5-1.5l1.409-1.409a2.25 2.25 0 013.182 0l2.909 2.909M3.75 21h16.5A2.25 2.25 0 0022.5 18.75V5.25A2.25 2.25 0 0020.25 3H3.75A2.25 2.25 0 001.5 5.25v13.5A2.25 2.25 0 003.75 21z" />
              </svg>
            </div>
          </div>
          <div class="flex-1 min-w-0">
            <div class="flex items-start gap-3 mb-1">
              <h2 class="text-[#e8e8f0] text-[18px] font-bold leading-tight">{{ game.title }}</h2>
            </div>
            <div class="w-8 h-px bg-[rgba(126,200,227,0.15)] mb-3"></div>
            <div class="flex items-center gap-1 mb-3 flex-wrap">
              <div
                v-for="member in game.members"
                :key="member.user_id"
                class="w-9 h-9 rounded-full border-2 border-[rgba(15,15,35,0.8)] -ml-1 first:ml-0 overflow-hidden bg-[rgba(126,200,227,0.1)] flex items-center justify-center"
                :title="member.username"
              >
                <img
                  v-if="memberAvatarUrl(member)"
                  :src="memberAvatarUrl(member)"
                  :alt="member.username"
                  class="w-full h-full object-cover"
                />
                <span v-else class="text-[#7ec8e3]/50 text-[11px] font-bold">
                  {{ member.username?.charAt(0)?.toUpperCase() }}
                </span>
              </div>
            </div>
            <p class="text-[#7ec8e3]/35 text-[12px]">
              Last played: {{ formatDate(game.updated_at) }}
            </p>
          </div>
        </div>
      </div>
      <Teleport to="body">
        <div v-if="contextMenu.visible" :style="{ position: 'fixed', left: contextMenu.x + 'px', top: contextMenu.y + 'px', zIndex: 9999 }" class="bg-[#0f0f23] border border-[rgba(126,200,227,0.2)] rounded-lg shadow-[0_8px_32px_rgba(0,0,0,0.6)] py-1 min-w-[180px]">
          <button v-if="isGM(contextMenu.game)" @click="openSettings(contextMenu.game)" class="w-full px-4 py-2.5 text-left text-[13px] text-[#e8e8f0]/80 hover:bg-[rgba(233,69,96,0.1)] hover:text-[#e8e8f0] transition-colors bg-transparent border-none cursor-pointer flex items-center gap-2.5">
            <svg class="w-4 h-4 text-[#7ec8e3]/40" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.325.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 011.37.49l1.296 2.247a1.125 1.125 0 01-.26 1.431l-1.003.827c-.293.241-.438.613-.43.992a7.723 7.723 0 010 .255c-.008.378.137.75.43.991l1.004.827c.424.35.534.955.26 1.43l-1.298 2.247a1.125 1.125 0 01-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.47 6.47 0 01-.22.128c-.331.183-.581.495-.644.869l-.213 1.281c-.09.543-.56.94-1.11.94h-2.594c-.55 0-1.019-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 01-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 01-1.369-.49l-1.297-2.247a1.125 1.125 0 01.26-1.431l1.004-.827c.292-.24.437-.613.43-.991a6.932 6.932 0 010-.255c.007-.38-.138-.751-.43-.992l-1.004-.827a1.125 1.125 0 01-.26-1.43l1.297-2.247a1.125 1.125 0 011.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.086.22-.128.332-.183.582-.495.644-.869l.214-1.28z" /><circle cx="12" cy="12" r="3" /></svg>
            Game Settings
          </button>
          <button v-if="!isGM(contextMenu.game)" @click="handleLeave(contextMenu.game)" class="w-full px-4 py-2.5 text-left text-[13px] text-[#e8e8f0]/80 hover:bg-[rgba(233,69,96,0.1)] hover:text-[#e94560] transition-colors bg-transparent border-none cursor-pointer flex items-center gap-2.5">
            <svg class="w-4 h-4 text-[#7ec8e3]/40" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M15.75 9V5.25A2.25 2.25 0 0013.5 3h-6a2.25 2.25 0 00-2.25 2.25v13.5A2.25 2.25 0 007.5 21h6a2.25 2.25 0 002.25-2.25V15m3 0l3-3m0 0l-3-3m3 3H9" /></svg>
            Leave Game
          </button>
          <button v-if="isGM(contextMenu.game)" @click="handleDelete(contextMenu.game)" class="w-full px-4 py-2.5 text-left text-[13px] text-[#e94560] hover:bg-[rgba(233,69,96,0.1)] transition-colors bg-transparent border-none cursor-pointer flex items-center gap-2.5">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" /></svg>
            Delete Game
          </button>
        </div>
      </Teleport>
      <JoinGameModal :visible="showJoinModal" @close="showJoinModal = false" @joined="onGameJoined" />
      <GameSettingsModal :visible="showSettingsModal" :game-id="settingsGameId" @close="showSettingsModal = false" @updated="refreshGames" @deleted="onGameDeleted" />
    </div>
  </HomeLayout>
</template>
