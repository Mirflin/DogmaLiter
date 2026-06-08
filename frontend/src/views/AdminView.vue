<script setup>
import HomeLayout from '@/layouts/HomeLayout.vue'
import DataTable from '@/components/DataTable.vue'
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import api from '@/api'
import { notify } from '@/notify'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()

const adminTab = ref('users')
const adminTabs = [
  { id: 'users', label: 'Users' },
  { id: 'news', label: 'News' },
  { id: 'games', label: 'Games' },
]

const userColumns = [
  { key: 'username', label: 'User' },
  { key: 'email', label: 'Email' },
  { key: 'role', label: 'Role', filterable: true, filterOptions: [{ value: 'user', label: 'user' }, { value: 'admin', label: 'admin' }] },
  { key: 'plan_name', label: 'Plan', sortable: true },
  { key: 'is_verified', label: 'Status' },
  { key: 'created_at', label: 'Joined' },
  { key: 'actions', label: 'Actions', align: 'right' },
]
const newsColumns = [
  { key: 'title', label: 'Title' },
  { key: 'author', label: 'Author', filterable: true },
  { key: 'is_published', label: 'Status' },
  { key: 'created_at', label: 'Created' },
  { key: 'actions', label: 'Actions', align: 'right' },
]
const gameColumns = [
  { key: 'title', label: 'Title' },
  { key: 'owner', label: 'Owner' },
  { key: 'players', label: 'Players', align: 'right', sortable: true },
  { key: 'created_at', label: 'Created' },
  { key: 'actions', label: 'Actions', align: 'right' },
]

const stats = ref(null)
const users = ref([])
const news = ref([])
const games = ref([])
const plans = ref([])
const loading = ref(true)
const busyNewsId = ref('')
const pendingDeleteUser = ref(null)
const deletingUser = ref(false)

function formatDate(value) {
  if (!value) return '—'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleDateString()
}

const newsRows = computed(() => news.value.map(post => ({
  id: post.id,
  title: post.title,
  author: post.author?.username || 'Unknown',
  is_published: post.is_published,
  created_at: formatDate(post.created_at),
})))

const statCards = computed(() => {
  const source = stats.value || {}
  return [
    { key: 'users', label: 'Users', value: source.users ?? 0 },
    { key: 'games', label: 'Games', value: source.games ?? 0 },
    { key: 'characters', label: 'Characters', value: source.characters ?? 0 },
    { key: 'items', label: 'Compendium Items', value: source.items ?? 0 },
    { key: 'news', label: 'News Posts', value: source.news ?? 0 },
  ]
})

onMounted(loadAdminData)

async function loadAdminData() {
  loading.value = true
  try {
    const [statsResponse, usersResponse, newsResponse, gamesResponse, plansResponse] = await Promise.all([
      api.get('/admin/stats'),
      api.get('/admin/users'),
      api.get('/news', { params: { limit: 200 } }),
      api.get('/admin/games'),
      api.get('/admin/plans'),
    ])
    stats.value = statsResponse.data || {}
    users.value = usersResponse.data?.users || []
    news.value = newsResponse.data?.posts || []
    games.value = gamesResponse.data?.games || []
    plans.value = plansResponse.data?.plans || []
  } catch (err) {
    notify.error(err, 'Failed to load admin data')
  } finally {
    loading.value = false
  }
}

const editingUser = ref(null)
const editUserForm = ref({ role: 'user', plan_id: 'free', subscription_ends_at: '', is_verified: false })
const savingUser = ref(false)
function openEditUser(user) {
  editingUser.value = user
  editUserForm.value = {
    role: user.role,
    plan_id: user.plan_id || 'free',
    subscription_ends_at: user.subscription_ends_at ? String(user.subscription_ends_at).slice(0, 10) : '',
    is_verified: Boolean(user.is_verified),
  }
}
function cancelEditUser() {
  if (savingUser.value) return
  editingUser.value = null
}
async function saveUser() {
  const user = editingUser.value
  if (!user || savingUser.value) return
  savingUser.value = true
  try {
    const payload = {
      role: editUserForm.value.role,
      plan_id: editUserForm.value.plan_id,
      subscription_ends_at: editUserForm.value.subscription_ends_at || '',
      is_verified: editUserForm.value.is_verified,
    }
    await api.patch(`/admin/users/${user.id}`, payload)
    user.role = payload.role
    user.plan_id = payload.plan_id
    user.plan_name = plans.value.find(plan => plan.id === payload.plan_id)?.name || user.plan_name
    user.subscription_ends_at = payload.subscription_ends_at ? new Date(payload.subscription_ends_at).toISOString() : null
    user.is_verified = payload.is_verified
    notify.success({ title: 'User updated', message: `${user.username} was saved.` })
    editingUser.value = null
  } catch (err) {
    notify.error(err, 'Failed to update user')
  } finally {
    savingUser.value = false
  }
}

function requestDeleteUser(user) {
  pendingDeleteUser.value = user
}

function cancelDeleteUser() {
  if (deletingUser.value) return
  pendingDeleteUser.value = null
}

async function confirmDeleteUser() {
  const user = pendingDeleteUser.value
  if (!user || deletingUser.value) return
  deletingUser.value = true
  try {
    await api.delete(`/admin/users/${user.id}`)
    users.value = users.value.filter(u => u.id !== user.id)
    if (stats.value && typeof stats.value.users === 'number') stats.value.users -= 1
    notify.success({ title: 'Account deleted', message: `${user.username} was removed.` })
    pendingDeleteUser.value = null
  } catch (err) {
    notify.error(err, 'Failed to delete account')
  } finally {
    deletingUser.value = false
  }
}

// --- News ---
const pendingDeleteNews = ref(null)
const deletingNews = ref(false)
function editNews(post) {
  router.push(`/news/create?edit=${post.id}`)
}
async function toggleNewsPublished(row) {
  const post = news.value.find(item => item.id === row.id)
  if (!post || busyNewsId.value) return
  busyNewsId.value = post.id
  const next = !post.is_published
  try {
    await api.patch(`/news/${post.id}`, { is_published: next })
    post.is_published = next
    notify.success({ title: next ? 'Published' : 'Hidden', message: `"${post.title}" is now ${next ? 'visible' : 'hidden'}.` })
  } catch (err) {
    notify.error(err, 'Failed to change visibility')
  } finally {
    busyNewsId.value = ''
  }
}
function requestDeleteNews(post) {
  pendingDeleteNews.value = post
}
function cancelDeleteNews() {
  if (deletingNews.value) return
  pendingDeleteNews.value = null
}
async function confirmDeleteNews() {
  const post = pendingDeleteNews.value
  if (!post || deletingNews.value) return
  deletingNews.value = true
  try {
    await api.delete(`/news/${post.id}`)
    news.value = news.value.filter(item => item.id !== post.id)
    if (stats.value && typeof stats.value.news === 'number') stats.value.news -= 1
    notify.success({ title: 'News deleted', message: `"${post.title}" was removed.` })
    pendingDeleteNews.value = null
  } catch (err) {
    notify.error(err, 'Failed to delete news post')
  } finally {
    deletingNews.value = false
  }
}

// --- Games ---
const pendingDeleteGame = ref(null)
const deletingGame = ref(false)
function requestDeleteGame(game) {
  pendingDeleteGame.value = game
}
function cancelDeleteGame() {
  if (deletingGame.value) return
  pendingDeleteGame.value = null
}
async function confirmDeleteGame() {
  const game = pendingDeleteGame.value
  if (!game || deletingGame.value) return
  deletingGame.value = true
  try {
    await api.delete(`/admin/games/${game.id}`)
    games.value = games.value.filter(item => item.id !== game.id)
    if (stats.value && typeof stats.value.games === 'number') stats.value.games -= 1
    notify.success({ title: 'Game deleted', message: `"${game.title}" was removed.` })
    pendingDeleteGame.value = null
  } catch (err) {
    notify.error(err, 'Failed to delete game')
  } finally {
    deletingGame.value = false
  }
}
</script>

<template>
  <HomeLayout>
    <div class="mx-auto max-w-[1200px] px-6 py-8">
      <div class="mb-8 flex flex-wrap items-end justify-between gap-4">
        <div>
          <p class="text-[11px] uppercase tracking-[0.24em] text-[#7ec8e3]/55">Administration</p>
          <h1 class="mt-2 font-[Cinzel] text-[28px] font-bold tracking-wide text-[#e8e8f0]">Admin Dashboard</h1>
        </div>
      </div>

      <div v-if="loading" class="py-16 text-center">
        <div class="mx-auto h-8 w-8 animate-spin rounded-full border-2 border-[#e94560] border-t-transparent"></div>
      </div>

      <template v-else>
        <div class="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-5">
          <div
            v-for="card in statCards"
            :key="card.key"
            class="rounded-xl border border-[rgba(126,200,227,0.12)] bg-[rgba(15,15,35,0.6)] px-5 py-5"
          >
            <p class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">{{ card.label }}</p>
            <p class="mt-2 text-[28px] font-bold text-[#f6f7fb]">{{ card.value }}</p>
          </div>
        </div>

        <div class="mt-8 inline-flex flex-wrap gap-1.5 rounded-2xl border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.6)] p-1.5">
          <button
            v-for="tab in adminTabs"
            :key="tab.id"
            type="button"
            @click="adminTab = tab.id"
            class="cursor-pointer rounded-xl px-4 py-2 text-[13px] font-semibold transition-all duration-200"
            :class="adminTab === tab.id
              ? 'bg-[linear-gradient(135deg,rgba(233,69,96,0.95),rgba(194,49,82,0.95))] text-white shadow-[0_8px_20px_rgba(233,69,96,0.3)]'
              : 'text-[#d8dce7]/70 hover:bg-[rgba(126,200,227,0.06)] hover:text-[#f6f7fb]'"
          >
            {{ tab.label }}
          </button>
        </div>

        <div v-show="adminTab === 'users'" class="mt-5">
          <DataTable
            :columns="userColumns"
            :rows="users"
            :search-keys="['username', 'email']"
            search-placeholder="Search users by name or email"
            :page-size="10"
            min-width="760px"
            empty-text="No users found"
          >
            <template #cell-username="{ row }">
              <span class="font-semibold text-[#f6f7fb]">{{ row.username }}</span>
              <span v-if="row.id === auth.user?.id" class="ml-1 text-[10px] font-normal text-[#7ec8e3]/45">(you)</span>
            </template>
            <template #cell-email="{ row }"><span class="text-[#e8e8f0]/60">{{ row.email }}</span></template>
            <template #cell-role="{ row }">
              <span class="rounded-full px-2.5 py-1 text-[10px] font-bold uppercase tracking-wider" :class="row.role === 'admin' ? 'border border-[rgba(233,69,96,0.32)] bg-[rgba(233,69,96,0.12)] text-[#ffadbd]' : 'border border-[rgba(126,200,227,0.2)] bg-[rgba(126,200,227,0.08)] text-[#8fd7ef]'">{{ row.role }}</span>
            </template>
            <template #cell-plan_name="{ row }">
              <span class="text-[#e8e8f0]/70">{{ row.plan_name || '—' }}</span>
              <span v-if="row.subscription_ends_at" class="ml-1 text-[11px] text-[#7ec8e3]/45">until {{ formatDate(row.subscription_ends_at) }}</span>
            </template>
            <template #cell-is_verified="{ row }">
              <span class="rounded-full px-2.5 py-1 text-[10px] font-bold uppercase tracking-wider" :class="row.is_verified ? 'border border-[rgba(74,222,128,0.3)] bg-[rgba(74,222,128,0.12)] text-[#86efac]' : 'border border-[rgba(248,113,113,0.3)] bg-[rgba(248,113,113,0.12)] text-[#fca5a5]'">{{ row.is_verified ? 'Verified' : 'Unverified' }}</span>
            </template>
            <template #cell-created_at="{ row }"><span class="text-[#7ec8e3]/45">{{ row.created_at }}</span></template>
            <template #cell-actions="{ row }">
              <div class="flex justify-end gap-2">
                <button type="button" @click="openEditUser(row)" class="rounded-lg border border-[rgba(126,200,227,0.2)] bg-[rgba(126,200,227,0.08)] px-3 py-1.5 text-[12px] font-semibold text-[#8fd7ef] transition-all duration-200 hover:border-[rgba(126,200,227,0.4)]">Edit</button>
                <button
                  type="button"
                  :disabled="row.id === auth.user?.id"
                  @click="requestDeleteUser(row)"
                  class="rounded-lg border border-[rgba(248,113,113,0.24)] bg-[rgba(248,113,113,0.1)] px-3 py-1.5 text-[12px] font-semibold text-[#fecaca] transition-all duration-200 hover:border-[rgba(248,113,113,0.45)] disabled:cursor-not-allowed disabled:opacity-40"
                >
                  Delete
                </button>
              </div>
            </template>
          </DataTable>
        </div>

        <div v-show="adminTab === 'news'" class="mt-5">
          <div class="mb-3 flex items-center justify-between gap-3">
            <h2 class="text-[15px] font-semibold text-[#e8e8f0]">News</h2>
            <router-link to="/news/create" class="rounded-lg border border-[#e94560] bg-linear-to-br from-[#e94560] to-[#c23152] px-4 py-2 text-[12px] font-semibold text-white no-underline transition-all duration-300 hover:-translate-y-0.5">Create News</router-link>
          </div>
          <DataTable
            :columns="newsColumns"
            :rows="newsRows"
            :search-keys="['title', 'author']"
            search-placeholder="Search news by title or author"
            :page-size="10"
            min-width="680px"
            empty-text="No news posts"
          >
            <template #cell-title="{ row }"><span class="font-semibold text-[#f6f7fb]">{{ row.title }}</span></template>
            <template #cell-author="{ row }"><span class="text-[#e8e8f0]/60">{{ row.author }}</span></template>
            <template #cell-is_published="{ row }">
              <button
                type="button"
                :disabled="busyNewsId === row.id"
                @click="toggleNewsPublished(row)"
                :title="row.is_published ? 'Click to hide' : 'Click to publish'"
                class="rounded-full px-2.5 py-1 text-[10px] font-bold uppercase tracking-wider transition-all duration-200 disabled:opacity-50"
                :class="row.is_published
                  ? 'border border-[rgba(74,222,128,0.3)] bg-[rgba(74,222,128,0.12)] text-[#86efac] hover:border-[rgba(74,222,128,0.5)]'
                  : 'border border-[rgba(126,200,227,0.2)] bg-[rgba(126,200,227,0.08)] text-[#8fd7ef] hover:border-[rgba(126,200,227,0.45)]'"
              >
                {{ row.is_published ? 'Published' : 'Hidden' }}
              </button>
            </template>
            <template #cell-created_at="{ row }"><span class="text-[#7ec8e3]/45">{{ row.created_at }}</span></template>
            <template #cell-actions="{ row }">
              <div class="flex justify-end gap-2">
                <button type="button" @click="editNews(row)" class="rounded-lg border border-[rgba(126,200,227,0.2)] bg-[rgba(126,200,227,0.08)] px-3 py-1.5 text-[12px] font-semibold text-[#8fd7ef] transition-all duration-200 hover:border-[rgba(126,200,227,0.4)]">Edit</button>
                <button type="button" @click="requestDeleteNews(row)" class="rounded-lg border border-[rgba(248,113,113,0.24)] bg-[rgba(248,113,113,0.1)] px-3 py-1.5 text-[12px] font-semibold text-[#fecaca] transition-all duration-200 hover:border-[rgba(248,113,113,0.45)]">Delete</button>
              </div>
            </template>
          </DataTable>
        </div>

        <div v-show="adminTab === 'games'" class="mt-5">
          <h2 class="mb-3 text-[15px] font-semibold text-[#e8e8f0]">Games</h2>
          <DataTable
            :columns="gameColumns"
            :rows="games"
            :search-keys="['title', 'owner']"
            search-placeholder="Search games by title or owner"
            :page-size="10"
            min-width="640px"
            empty-text="No games"
          >
            <template #cell-title="{ row }"><span class="font-semibold text-[#f6f7fb]">{{ row.title }}</span></template>
            <template #cell-owner="{ row }"><span class="text-[#e8e8f0]/60">{{ row.owner }}</span></template>
            <template #cell-players="{ row }"><span class="text-[#f6f7fb]">{{ row.players }}/{{ row.max_players }}</span></template>
            <template #cell-created_at="{ row }"><span class="text-[#7ec8e3]/45">{{ row.created_at }}</span></template>
            <template #cell-actions="{ row }">
              <div class="flex justify-end gap-2">
                <button type="button" @click="requestDeleteGame(row)" class="rounded-lg border border-[rgba(248,113,113,0.24)] bg-[rgba(248,113,113,0.1)] px-3 py-1.5 text-[12px] font-semibold text-[#fecaca] transition-all duration-200 hover:border-[rgba(248,113,113,0.45)]">Delete</button>
              </div>
            </template>
          </DataTable>
        </div>
      </template>
    </div>

    <Teleport to="body">
      <div v-if="pendingDeleteUser" class="fixed inset-0 z-[12550] flex items-center justify-center p-4">
        <div class="absolute inset-0 bg-[rgba(5,8,12,0.82)] backdrop-blur-md" @click="cancelDeleteUser"></div>
        <div class="relative w-full max-w-[28rem] overflow-hidden rounded-[1.6rem] border border-[rgba(248,113,113,0.24)] bg-[linear-gradient(180deg,rgba(9,18,34,0.98),rgba(5,10,22,0.98))] p-6 shadow-[0_40px_120px_rgba(0,0,0,0.55)]">
          <p class="text-[11px] uppercase tracking-[0.24em] text-[#fca5a5]/70">Delete Account</p>
          <h3 class="mt-2 break-words font-[Cinzel] text-[22px] font-bold leading-tight text-[#f6f7fb] [overflow-wrap:anywhere]">{{ pendingDeleteUser.username }}</h3>
          <p class="mt-3 text-[14px] leading-relaxed text-[#d8dce7]/68">
            This permanently deletes the account and all games they own (with characters, items, and chat). Characters and items they created in other people's games are reassigned, and authored news is transferred to you. This cannot be undone.
          </p>
          <div class="mt-6 flex flex-wrap justify-end gap-3">
            <button type="button" @click="cancelDeleteUser" :disabled="deletingUser" class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.3)] disabled:cursor-not-allowed disabled:opacity-60">
              Cancel
            </button>
            <button type="button" @click="confirmDeleteUser" :disabled="deletingUser" class="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-[rgba(248,113,113,0.28)] bg-[linear-gradient(135deg,rgba(248,113,113,0.9),rgba(220,38,38,0.9))] px-4 py-2.5 text-[13px] font-semibold text-white transition-all duration-200 hover:-translate-y-0.5 disabled:cursor-not-allowed disabled:opacity-60">
              {{ deletingUser ? 'Deleting...' : 'Delete Account' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="pendingDeleteNews" class="fixed inset-0 z-[12550] flex items-center justify-center p-4">
        <div class="absolute inset-0 bg-[rgba(5,8,12,0.82)] backdrop-blur-md" @click="cancelDeleteNews"></div>
        <div class="relative w-full max-w-[28rem] overflow-hidden rounded-[1.6rem] border border-[rgba(248,113,113,0.24)] bg-[linear-gradient(180deg,rgba(9,18,34,0.98),rgba(5,10,22,0.98))] p-6 shadow-[0_40px_120px_rgba(0,0,0,0.55)]">
          <p class="text-[11px] uppercase tracking-[0.24em] text-[#fca5a5]/70">Delete News</p>
          <h3 class="mt-2 break-words font-[Cinzel] text-[22px] font-bold leading-tight text-[#f6f7fb] [overflow-wrap:anywhere]">{{ pendingDeleteNews.title }}</h3>
          <p class="mt-3 text-[14px] leading-relaxed text-[#d8dce7]/68">This permanently removes the news post. This action cannot be undone.</p>
          <div class="mt-6 flex flex-wrap justify-end gap-3">
            <button type="button" @click="cancelDeleteNews" :disabled="deletingNews" class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.3)] disabled:cursor-not-allowed disabled:opacity-60">Cancel</button>
            <button type="button" @click="confirmDeleteNews" :disabled="deletingNews" class="cursor-pointer rounded-xl border border-[rgba(248,113,113,0.28)] bg-[linear-gradient(135deg,rgba(248,113,113,0.9),rgba(220,38,38,0.9))] px-4 py-2.5 text-[13px] font-semibold text-white transition-all duration-200 hover:-translate-y-0.5 disabled:cursor-not-allowed disabled:opacity-60">Delete</button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="pendingDeleteGame" class="fixed inset-0 z-[12550] flex items-center justify-center p-4">
        <div class="absolute inset-0 bg-[rgba(5,8,12,0.82)] backdrop-blur-md" @click="cancelDeleteGame"></div>
        <div class="relative w-full max-w-[28rem] overflow-hidden rounded-[1.6rem] border border-[rgba(248,113,113,0.24)] bg-[linear-gradient(180deg,rgba(9,18,34,0.98),rgba(5,10,22,0.98))] p-6 shadow-[0_40px_120px_rgba(0,0,0,0.55)]">
          <p class="text-[11px] uppercase tracking-[0.24em] text-[#fca5a5]/70">Delete Game</p>
          <h3 class="mt-2 break-words font-[Cinzel] text-[22px] font-bold leading-tight text-[#f6f7fb] [overflow-wrap:anywhere]">{{ pendingDeleteGame.title }}</h3>
          <p class="mt-3 text-[14px] leading-relaxed text-[#d8dce7]/68">This permanently removes the game with all its characters, inventories, items and chat. This action cannot be undone.</p>
          <div class="mt-6 flex flex-wrap justify-end gap-3">
            <button type="button" @click="cancelDeleteGame" :disabled="deletingGame" class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.3)] disabled:cursor-not-allowed disabled:opacity-60">Cancel</button>
            <button type="button" @click="confirmDeleteGame" :disabled="deletingGame" class="cursor-pointer rounded-xl border border-[rgba(248,113,113,0.28)] bg-[linear-gradient(135deg,rgba(248,113,113,0.9),rgba(220,38,38,0.9))] px-4 py-2.5 text-[13px] font-semibold text-white transition-all duration-200 hover:-translate-y-0.5 disabled:cursor-not-allowed disabled:opacity-60">Delete</button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="editingUser" class="fixed inset-0 z-[12550] flex items-center justify-center p-4">
        <div class="absolute inset-0 bg-[rgba(5,8,12,0.82)] backdrop-blur-md" @click="cancelEditUser"></div>
        <div class="relative w-full max-w-[30rem] overflow-hidden rounded-[1.6rem] border border-[rgba(126,200,227,0.18)] bg-[linear-gradient(180deg,rgba(9,18,34,0.98),rgba(5,10,22,0.98))] p-6 shadow-[0_40px_120px_rgba(0,0,0,0.55)]">
          <p class="text-[11px] uppercase tracking-[0.24em] text-[#7ec8e3]/58">Edit User</p>
          <h3 class="mt-2 break-words font-[Cinzel] text-[22px] font-bold leading-tight text-[#f6f7fb] [overflow-wrap:anywhere]">{{ editingUser.username }}</h3>

          <label class="mt-4 block text-[11px] uppercase tracking-[0.16em] text-[#7ec8e3]/55">Role</label>
          <select v-model="editUserForm.role" class="mt-1.5 w-full rounded-lg border border-[rgba(126,200,227,0.2)] bg-[rgba(7,17,31,0.72)] px-3 py-2.5 text-[14px] text-[#f6f7fb] outline-none focus:border-[rgba(126,200,227,0.45)]">
            <option value="user">user</option>
            <option value="admin">admin</option>
          </select>

          <label class="mt-4 block text-[11px] uppercase tracking-[0.16em] text-[#7ec8e3]/55">Subscription plan</label>
          <select v-model="editUserForm.plan_id" class="mt-1.5 w-full rounded-lg border border-[rgba(126,200,227,0.2)] bg-[rgba(7,17,31,0.72)] px-3 py-2.5 text-[14px] text-[#f6f7fb] outline-none focus:border-[rgba(126,200,227,0.45)]">
            <option v-for="plan in plans" :key="plan.id" :value="plan.id">{{ plan.name }}</option>
          </select>

          <label class="mt-4 block text-[11px] uppercase tracking-[0.16em] text-[#7ec8e3]/55">Subscription ends</label>
          <input v-model="editUserForm.subscription_ends_at" type="date" class="mt-1.5 w-full rounded-lg border border-[rgba(126,200,227,0.2)] bg-[rgba(7,17,31,0.72)] px-3 py-2.5 text-[14px] text-[#f6f7fb] outline-none focus:border-[rgba(126,200,227,0.45)]" />
          <p class="mt-1 text-[11px] text-[#7ec8e3]/40">Leave empty for no expiry. Set manually — no payment required.</p>

          <label class="mt-4 flex cursor-pointer items-center gap-2.5 text-[13px] text-[#e8e8f0]/80">
            <input v-model="editUserForm.is_verified" type="checkbox" class="h-4 w-4 accent-[#86efac]" />
            Email verified
          </label>

          <div class="mt-6 flex flex-wrap justify-end gap-3">
            <button type="button" @click="cancelEditUser" :disabled="savingUser" class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.3)] disabled:cursor-not-allowed disabled:opacity-60">Cancel</button>
            <button type="button" @click="saveUser" :disabled="savingUser" class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.28)] bg-[linear-gradient(135deg,rgba(126,200,227,0.22),rgba(126,200,227,0.12))] px-4 py-2.5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:-translate-y-0.5 disabled:cursor-not-allowed disabled:opacity-60">{{ savingUser ? 'Saving...' : 'Save' }}</button>
          </div>
        </div>
      </div>
    </Teleport>
  </HomeLayout>
</template>
