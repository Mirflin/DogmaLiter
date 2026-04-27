<script setup>
import HomeLayout from '@/layouts/HomeLayout.vue'
import JoinGameModal from '@/components/JoinGameModal.vue'
import GameSettingsModal from '@/components/GameSettingsModal.vue'
import { useAuthStore } from '@/stores/auth'
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import api, { API_URL } from '@/api'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

const avatarPreview = ref(null)
const games = ref([])
const loadingGames = ref(true)

const contextMenu = ref({ visible: false, x: 0, y: 0, game: null })

const showJoinModal = ref(false)
const joinInitialCode = ref('')
const showSettingsModal = ref(false)
const settingsGameId = ref(null)

const newsPosts = ref([])
const loadingNews = ref(true)

const avatarUrl = computed(() => {
  if (avatarPreview.value) return avatarPreview.value
  if (auth.user?.avatar_id) return `${API_URL}/api/uploads/${auth.user.avatar_id}`
  return null
})

const latestGames = computed(() => games.value.slice(0, 6))

onMounted(async () => {
  try {
    const data = await auth.fetchMyGames()
    games.value = data || []
  } catch {} finally {
    loadingGames.value = false
  }

  try {
    const { data } = await api.get('/news', { params: { limit: 4, offset: 0 } })
    newsPosts.value = data.posts || []
  } catch {} finally {
    loadingNews.value = false
  }

  document.addEventListener('click', closeContextMenu)

  if (route.query.join) {
    joinInitialCode.value = route.query.join
    showJoinModal.value = true
    router.replace({ query: { ...route.query, join: undefined } })
  }
})

onUnmounted(() => {
  document.removeEventListener('click', closeContextMenu)
})

function isGM(game) {
  return game.owner_id === auth.user?.id
}

function coverUrl(game) {
  if (game.cover_image_id) return `${API_URL}/api/uploads/${game.cover_image_id}`
  return null
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
    games.value = (await auth.fetchMyGames()) || []
  } catch {}
}

function onGameDeleted(gameId) {
  games.value = games.value.filter(g => g.id !== gameId)
}

function onGameJoined() {
  refreshGames()
}

function newsImageUrl(post) {
  if (post.image_id) return `${API_URL}/api/uploads/${post.image_id}`
  return null
}

function truncate(text, len = 80) {
  if (!text) return ''
  return text.length > len ? text.substring(0, len) + '…' : text
}

function formatDate(d) {
  if (!d) return ''
  return new Date(d).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}

function getPreviewText(content) {
  try {
    const sections = JSON.parse(content)
    if (Array.isArray(sections) && sections.length > 0) return sections[0].text || ''
  } catch {}
  return content || ''
}
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
            </div>

            <div v-if="loadingGames" class="text-center py-12">
              <div class="w-8 h-8 border-2 border-[#e94560] border-t-transparent rounded-full animate-spin mx-auto"></div>
            </div>

            <div v-else-if="latestGames.length === 0" class="text-center py-12">
              <p class="text-[#7ec8e3]/40 text-[14px] mb-4">You don't have any games yet</p>
              <router-link to="/games/create" class="text-[#e94560] text-[14px] font-medium no-underline hover:text-[#ff6b81]">Create your first game</router-link>
            </div>

            <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
              <div
                v-for="game in latestGames"
                :key="game.id"
                @click="router.push(`/games/${game.id}`)"
                @contextmenu="openContextMenu($event, game)"
                class="bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.1)] rounded-xl overflow-hidden hover:border-[rgba(233,69,96,0.3)] hover:-translate-y-1 transition-all duration-300 cursor-pointer group"
              >
                <div class="h-36 bg-[rgba(126,200,227,0.05)] flex items-center justify-center relative overflow-hidden">
                  <img v-if="coverUrl(game)" :src="coverUrl(game)" :alt="game.title" class="w-full h-full object-cover" />
                  <span v-else class="text-[#7ec8e3]/20 text-sm">No cover image</span>
                  <div class="absolute inset-0 bg-linear-to-t from-[rgba(15,15,35,0.8)] to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300" />
                </div>
                <div class="p-4">
                  <h3 class="text-[#e8e8f0] text-[15px] font-semibold mb-1 truncate">{{ game.title }}</h3>
                  <div class="flex items-center gap-1.5">
                    <span :class="isGM(game) ? 'text-[#e94560]' : 'text-[#7ec8e3]/50'" class="text-[11px]">{{ isGM(game) ? 'GM' : 'Player' }}</span>
                    <span class="text-[#7ec8e3]/20 text-[11px]">&middot;</span>
                    <span class="text-[#7ec8e3]/30 text-[11px]">{{ game.members?.length || 0 }} members</span>
                  </div>
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

            <router-link
              to="/games"
              class="inline-block mt-6 px-6 py-2.5 bg-[rgba(126,200,227,0.08)] text-[#7ec8e3]/70 text-[13px] font-semibold no-underline rounded-lg border border-[rgba(126,200,227,0.15)] hover:border-[rgba(126,200,227,0.3)] hover:text-[#7ec8e3] transition-all duration-300"
            >
              View All Games
            </router-link>
          </div>

          <div>
            <div class="flex items-center justify-between mb-6">
              <h2 class="font-[Cinzel] text-[22px] font-bold text-[#e8e8f0] tracking-wide">Latest News</h2>
              <div class="flex items-center gap-3">
                <router-link
                  v-if="auth.user?.role === 'admin'"
                  to="/news/create"
                  class="px-4 py-2 bg-linear-to-br from-[#e94560] to-[#c23152] text-white text-[13px] font-semibold no-underline rounded-lg border border-[#e94560] hover:-translate-y-0.5 hover:shadow-[0_6px_20px_rgba(233,69,96,0.4)] transition-all duration-300"
                >
                  New post
                </router-link>
                <router-link to="/news" class="text-[#e94560] text-[13px] font-medium no-underline hover:text-[#ff6b81] transition-colors">
                  View all &rarr;
                </router-link>
              </div>
            </div>

            <div v-if="loadingNews" class="text-center py-8">
              <div class="w-6 h-6 border-2 border-[#e94560] border-t-transparent rounded-full animate-spin mx-auto"></div>
            </div>

            <div v-else-if="newsPosts.length === 0" class="text-center py-8">
              <p class="text-[#7ec8e3]/40 text-[14px]">No news yet</p>
            </div>

            <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
              <div
                v-for="post in newsPosts"
                :key="post.id"
                @click="router.push(`/news/${post.id}`)"
                class="bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.1)] rounded-xl overflow-hidden hover:border-[rgba(233,69,96,0.3)] hover:-translate-y-1 transition-all duration-300 cursor-pointer group"
              >
                <div class="h-32 bg-[rgba(126,200,227,0.05)] flex items-center justify-center relative overflow-hidden">
                  <img v-if="newsImageUrl(post)" :src="newsImageUrl(post)" :alt="post.title" class="w-full h-full object-cover" />
                  <span v-else class="text-[#7ec8e3]/20 text-sm">No image</span>
                  <div class="absolute inset-0 bg-linear-to-t from-[rgba(15,15,35,0.8)] to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300" />
                </div>
                <div class="p-3">
                  <p class="text-[#7ec8e3]/40 text-[11px] mb-1">{{ formatDate(post.published_at) }}</p>
                  <h4 class="text-[#e8e8f0] text-[14px] font-semibold mb-1 line-clamp-2">{{ post.title }}</h4>
                  <p class="text-[#e8e8f0]/40 text-[12px] line-clamp-2">{{ truncate(getPreviewText(post.content)) }}</p>
                </div>
              </div>
            </div>
          </div>
        </div>

        <aside class="hidden lg:block w-72 flex-shrink-0">
          <div class="bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.1)] rounded-xl p-5 mb-6">
            <div class="flex items-center gap-4 mb-4">
              <div class="w-14 h-14 rounded-full bg-[rgba(233,69,96,0.15)] border border-[rgba(233,69,96,0.3)] flex items-center justify-center">
                <img v-if="avatarUrl" :src="avatarUrl" alt="Avatar" class="w-full h-full object-cover rounded-full" />
                <span v-else class="text-[#e94560] text-[20px] font-bold font-[Cinzel]">{{ auth.user?.username?.charAt(0)?.toUpperCase() }}</span>
              </div>
              <div>
                <h3 class="text-[#e8e8f0] text-[16px] font-bold">{{ auth.user?.username }}</h3>
                <p class="text-[#7ec8e3]/40 text-[12px]">{{ auth.user?.email }}</p>
              </div>
            </div>
            <div class="space-y-2 text-[13px]">
              <div class="flex justify-between">
                <span class="text-[#7ec8e3]/40">Plan</span>
                <span class="text-[#e94560]">{{ auth.user?.plan_name }}</span>
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
              <button
                @click="showJoinModal = true"
                class="flex items-center gap-3 px-3 py-2.5 text-[13px] text-[#e8e8f0]/60 rounded-lg hover:bg-[rgba(233,69,96,0.1)] hover:text-[#e94560] transition-all duration-200 w-full bg-transparent border-none cursor-pointer text-left"
              >
                <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M18 7.5v3m0 0v3m0-3h3m-3 0h-3m-2.25-4.125a3.375 3.375 0 11-6.75 0 3.375 3.375 0 016.75 0zM3 19.235v-.11a6.375 6.375 0 0112.75 0v.109A12.318 12.318 0 019.374 21c-2.331 0-4.512-.645-6.374-1.766z" /></svg>
                Join Game
              </button>
            </div>
          </div>
        </aside>
      </div>
    </div>
    <JoinGameModal :visible="showJoinModal" :initial-code="joinInitialCode" @close="showJoinModal = false; joinInitialCode = ''" @joined="onGameJoined" />
    <GameSettingsModal :visible="showSettingsModal" :game-id="settingsGameId" @close="showSettingsModal = false" @updated="refreshGames" @deleted="onGameDeleted" />
  </div>
  </HomeLayout>
</template>
