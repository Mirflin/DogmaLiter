<script setup>
import HomeLayout from '@/layouts/HomeLayout.vue'
import DataTable from '@/components/DataTable.vue'
import { computed, onMounted, ref } from 'vue'
import api from '@/api'
import { notify } from '@/notify'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()

const userColumns = [
  { key: 'username', label: 'User' },
  { key: 'email', label: 'Email' },
  { key: 'role', label: 'Role', filterable: true, filterOptions: [{ value: 'user', label: 'user' }, { value: 'admin', label: 'admin' }] },
  { key: 'plan_name', label: 'Plan', sortable: true },
  { key: 'is_verified', label: 'Status' },
  { key: 'created_at', label: 'Joined' },
  { key: 'actions', label: 'Actions', align: 'right' },
]
const stats = ref(null)
const users = ref([])
const loading = ref(true)
const busyUserId = ref('')
const pendingDeleteUser = ref(null)
const deletingUser = ref(false)

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
    const [statsResponse, usersResponse] = await Promise.all([
      api.get('/admin/stats'),
      api.get('/admin/users'),
    ])
    stats.value = statsResponse.data || {}
    users.value = usersResponse.data?.users || []
  } catch (err) {
    notify.error(err, 'Failed to load admin data')
  } finally {
    loading.value = false
  }
}

async function updateUserRole(user, event) {
  const role = event.target.value
  if (user.role === role) return
  busyUserId.value = user.id
  try {
    await api.patch(`/admin/users/${user.id}`, { role })
    user.role = role
    notify.success({ title: 'Role updated', message: `${user.username} is now ${role}.` })
  } catch (err) {
    event.target.value = user.role
    notify.error(err, 'Failed to update role')
  } finally {
    busyUserId.value = ''
  }
}

async function toggleVerified(user) {
  busyUserId.value = user.id
  const next = !user.is_verified
  try {
    await api.patch(`/admin/users/${user.id}`, { is_verified: next })
    user.is_verified = next
  } catch (err) {
    notify.error(err, 'Failed to update status')
  } finally {
    busyUserId.value = ''
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
</script>

<template>
  <HomeLayout>
    <div class="mx-auto max-w-[1200px] px-6 py-8">
      <div class="mb-8 flex flex-wrap items-end justify-between gap-4">
        <div>
          <p class="text-[11px] uppercase tracking-[0.24em] text-[#7ec8e3]/55">Administration</p>
          <h1 class="mt-2 font-[Cinzel] text-[28px] font-bold tracking-wide text-[#e8e8f0]">Admin Dashboard</h1>
        </div>
        <div class="flex flex-wrap gap-3">
          <router-link
            to="/news/create"
            class="rounded-lg border border-[#e94560] bg-linear-to-br from-[#e94560] to-[#c23152] px-5 py-2.5 text-[13px] font-semibold text-white no-underline transition-all duration-300 hover:-translate-y-0.5 hover:shadow-[0_6px_20px_rgba(233,69,96,0.4)]"
          >
            Create News
          </router-link>
          <router-link
            to="/news"
            class="rounded-lg border border-[rgba(126,200,227,0.2)] bg-[rgba(126,200,227,0.08)] px-5 py-2.5 text-[13px] font-semibold text-[#8fd7ef] no-underline transition-all duration-300 hover:border-[rgba(126,200,227,0.4)]"
          >
            Manage News
          </router-link>
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

        <div class="mt-8">
          <h2 class="mb-3 text-[15px] font-semibold text-[#e8e8f0]">User Management</h2>
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
              <select
                :value="row.role"
                :disabled="busyUserId === row.id"
                @change="updateUserRole(row, $event)"
                class="rounded-lg border border-[rgba(126,200,227,0.2)] bg-[rgba(7,17,31,0.72)] px-2.5 py-1.5 text-[12px] text-[#f6f7fb] outline-none disabled:opacity-50"
              >
                <option value="user">user</option>
                <option value="admin">admin</option>
              </select>
            </template>
            <template #cell-plan_name="{ row }"><span class="text-[#e8e8f0]/60">{{ row.plan_name || '—' }}</span></template>
            <template #cell-is_verified="{ row }">
              <button
                type="button"
                :disabled="busyUserId === row.id"
                @click="toggleVerified(row)"
                class="rounded-full px-2.5 py-1 text-[10px] font-bold uppercase tracking-wider transition-all duration-200 disabled:opacity-50"
                :class="row.is_verified
                  ? 'border border-[rgba(74,222,128,0.3)] bg-[rgba(74,222,128,0.12)] text-[#86efac] hover:border-[rgba(74,222,128,0.5)]'
                  : 'border border-[rgba(248,113,113,0.3)] bg-[rgba(248,113,113,0.12)] text-[#fca5a5] hover:border-[rgba(248,113,113,0.5)]'"
              >
                {{ row.is_verified ? 'Verified' : 'Unverified' }}
              </button>
            </template>
            <template #cell-created_at="{ row }"><span class="text-[#7ec8e3]/45">{{ row.created_at }}</span></template>
            <template #cell-actions="{ row }">
              <button
                type="button"
                :disabled="row.id === auth.user?.id"
                @click="requestDeleteUser(row)"
                class="rounded-lg border border-[rgba(248,113,113,0.24)] bg-[rgba(248,113,113,0.1)] px-3 py-1.5 text-[12px] font-semibold text-[#fecaca] transition-all duration-200 hover:border-[rgba(248,113,113,0.45)] disabled:cursor-not-allowed disabled:opacity-40"
              >
                Delete
              </button>
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
  </HomeLayout>
</template>
