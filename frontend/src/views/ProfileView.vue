<script setup>
import HomeLayout from '@/layouts/HomeLayout.vue'
import { useAuthStore } from '@/stores/auth'
import { ref, computed, onMounted } from 'vue'
import api, { API_URL } from '@/api'

const auth = useAuthStore()

const storage = ref(null)
const loadingProfile = ref(true)
const showCancelConfirm = ref(false)
const canceling = ref(false)
const cancelMessage = ref('')

const hasPaidPlan = computed(() => {
  const pid = auth.user?.plan_id
  return pid && pid !== 'free'
})

const avatarUrl = computed(() => {
  if (auth.user?.avatar_id) return `${API_URL}/api/uploads/${auth.user.avatar_id}`
  return null
})

const storagePercent = computed(() => {
  if (!storage.value || !storage.value.storage_limit_mb) return 0
  const limitBytes = storage.value.storage_limit_mb * 1024 * 1024
  return Math.min(100, Math.round((storage.value.used_bytes / limitBytes) * 100))
})

const storageUsedFormatted = computed(() => {
  if (!storage.value) return '0 B'
  return formatBytes(storage.value.used_bytes)
})

const storageLimitFormatted = computed(() => {
  if (!storage.value) return '0 MB'
  const mb = storage.value.storage_limit_mb
  if (mb >= 1024) return (mb / 1024).toFixed(1) + ' GB'
  return mb + ' MB'
})

function formatBytes(bytes) {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

const barColor = computed(() => {
  if (storagePercent.value >= 90) return '#e94560'
  if (storagePercent.value >= 70) return '#f0a500'
  return '#7ec8e3'
})

async function cancelSubscription() {
  canceling.value = true
  cancelMessage.value = ''
  try {
    await api.post('/payment/cancel')
    await auth.fetchProfile()
    showCancelConfirm.value = false
    cancelMessage.value = 'Subscription canceled. You have been downgraded to the Free plan.'
  } catch (err) {
    cancelMessage.value = err.response?.data?.error || 'Failed to cancel subscription'
  } finally {
    canceling.value = false
  }
}

onMounted(async () => {
  try {
    const [p, s] = await Promise.all([
      auth.fetchProfile(),
      auth.fetchStorageUsage(),
    ])
    storage.value = s
  } catch {

  } finally {
    loadingProfile.value = false
  }
})
</script>

<template>
  <HomeLayout>
    <div class="max-w-[1000px] mx-auto px-6 py-10">
      <h1 class="font-[Cinzel] text-[28px] font-bold text-[#e8e8f0] tracking-wide mb-8">My Profile</h1>

      <div v-if="loadingProfile" class="text-center py-20">
        <div class="inline-block w-8 h-8 border-2 border-[#e94560] border-t-transparent rounded-full animate-spin" />
      </div>

      <template v-else>
        <div class="flex flex-col md:flex-row gap-8 bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.1)] rounded-xl p-8">
          <div class="flex-shrink-0 flex flex-col items-center gap-3">
            <div class="w-48 h-48 rounded-xl overflow-hidden bg-[rgba(233,69,96,0.1)] border border-[rgba(233,69,96,0.25)] flex items-center justify-center">
              <img
                v-if="avatarUrl"
                :src="avatarUrl"
                alt="Avatar"
                class="w-full h-full object-cover"
              />
              <span v-else class="text-[#e94560] text-[72px] font-bold font-[Cinzel] select-none">
                {{ auth.user?.username?.charAt(0)?.toUpperCase() }}
              </span>
            </div>
            <router-link
              to="/settings"
              class="text-[#7ec8e3]/60 text-[13px] no-underline hover:text-[#e94560] transition-colors duration-200"
            >
              Edit profile →
            </router-link>
          </div>

          <div class="flex-1 min-w-0">
            <h2 class="font-[Cinzel] text-[32px] font-bold text-[#e8e8f0] mb-1 tracking-wide">
              {{ auth.user?.username }}
            </h2>
            <p class="text-[#7ec8e3]/40 text-[14px] mb-6">{{ auth.user?.email }}</p>

            <div class="space-y-4">
              <div class="flex items-center gap-3">
                <svg class="w-4 h-4 text-[#7ec8e3]/40 flex-shrink-0" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
                <span class="text-[#7ec8e3]/40 text-[13px] w-36">Registration date</span>
                <span class="text-[#e8e8f0]/80 text-[14px]">{{ auth.user?.created_at }}</span>
              </div>

              <div class="flex items-center gap-3">
                <svg class="w-4 h-4 text-[#7ec8e3]/40 flex-shrink-0" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
                </svg>
                <span class="text-[#7ec8e3]/40 text-[13px] w-36">Plan</span>
                <span class="text-[#e94560] text-[14px] font-semibold">{{ auth.user?.plan_name }}</span>
                <button
                  v-if="hasPaidPlan"
                  @click="showCancelConfirm = true"
                  class="ml-3 text-[12px] text-[#7ec8e3]/40 hover:text-[#e94560] transition-colors duration-200 cursor-pointer underline underline-offset-2"
                >
                  Cancel subscription
                </button>
                <router-link
                  v-else
                  to="/plans"
                  class="ml-3 text-[12px] text-[#7ec8e3]/40 hover:text-[#e94560] transition-colors duration-200 no-underline"
                >
                  Upgrade →
                </router-link>
              </div>

              <div class="flex items-center gap-3">
                <svg class="w-4 h-4 text-[#7ec8e3]/40 flex-shrink-0" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                </svg>
                <span class="text-[#7ec8e3]/40 text-[13px] w-36">Email status</span>
                <span
                  class="text-[13px] font-medium px-2 py-0.5 rounded-full"
                  :class="auth.user?.is_verified
                    ? 'text-green-400 bg-green-400/10 border border-green-400/20'
                    : 'text-amber-400 bg-amber-400/10 border border-amber-400/20'"
                >
                  {{ auth.user?.is_verified ? 'Verified' : 'Not verified' }}
                </span>
              </div>
            </div>
          </div>
        </div>

        <div class="mt-8 bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.1)] rounded-xl p-8">
          <div class="flex items-center justify-between mb-5">
            <h3 class="font-[Cinzel] text-[18px] font-bold text-[#e8e8f0] tracking-wide">Upload Storage</h3>
            <span class="text-[#7ec8e3]/40 text-[13px]">
              {{ storage?.files_count ?? 0 }} {{ (storage?.files_count ?? 0) === 1 ? 'file' : 'files' }}
            </span>
          </div>

          <div class="relative h-6 bg-[rgba(126,200,227,0.06)] rounded-full overflow-hidden border border-[rgba(126,200,227,0.08)]">
            <div
              class="h-full rounded-full transition-all duration-700 ease-out"
              :style="{ width: storagePercent + '%', backgroundColor: barColor }"
            />
            <span
              v-if="storagePercent > 8"
              class="absolute inset-0 flex items-center justify-center text-[12px] font-semibold text-white mix-blend-difference"
            >
              {{ storagePercent }}%
            </span>
          </div>

          <div class="flex items-center justify-between mt-3">
            <span class="text-[#7ec8e3]/40 text-[13px]">
              {{ storageUsedFormatted }} used
            </span>
            <span class="text-[#7ec8e3]/40 text-[13px]">
              {{ storageLimitFormatted }} total
            </span>
          </div>

          <div class="flex items-center gap-6 mt-5 pt-5 border-t border-[rgba(126,200,227,0.06)]">
            <div class="flex items-center gap-2">
              <div class="w-3 h-3 rounded-sm" :style="{ backgroundColor: barColor }" />
              <span class="text-[#7ec8e3]/50 text-[12px]">Used ({{ storageUsedFormatted }})</span>
            </div>
            <div class="flex items-center gap-2">
              <div class="w-3 h-3 rounded-sm bg-[rgba(126,200,227,0.06)] border border-[rgba(126,200,227,0.1)]" />
              <span class="text-[#7ec8e3]/50 text-[12px]">Free ({{ storageLimitFormatted }})</span>
            </div>
          </div>
        </div>

        <div v-if="cancelMessage" class="mt-6 p-4 rounded-xl text-[14px] text-center"
          :class="cancelMessage.includes('canceled')
            ? 'bg-green-500/10 border border-green-500/20 text-green-400'
            : 'bg-[rgba(233,69,96,0.1)] border border-[rgba(233,69,96,0.2)] text-[#e94560]'"
        >
          {{ cancelMessage }}
        </div>
      </template>
    </div>

    <Teleport to="body">
      <div v-if="showCancelConfirm" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm" @click.self="showCancelConfirm = false">
        <div class="bg-[#0f0f23] border border-[rgba(233,69,96,0.3)] rounded-2xl p-8 max-w-md w-full mx-4 shadow-[0_20px_60px_rgba(0,0,0,0.6)]">
          <div class="flex items-center gap-3 mb-4">
            <svg class="w-6 h-6 text-[#e94560]" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z" />
            </svg>
            <h3 class="font-[Cinzel] text-[20px] font-bold text-[#e8e8f0]">Cancel Subscription</h3>
          </div>
          <p class="text-[#7ec8e3]/60 text-[14px] mb-2">
            Are you sure you want to cancel your <span class="text-[#e94560] font-semibold">{{ auth.user?.plan_name }}</span> subscription?
          </p>
          <p class="text-[#7ec8e3]/40 text-[13px] mb-6">
            You will be immediately downgraded to the Free plan. Your uploaded files will be kept, but you may exceed the Free plan storage limit.
          </p>
          <div class="flex gap-3">
            <button
              @click="showCancelConfirm = false"
              class="flex-1 py-2.5 rounded-xl text-[14px] font-medium bg-transparent border border-[rgba(126,200,227,0.2)] text-[#e8e8f0] hover:bg-[rgba(126,200,227,0.05)] cursor-pointer transition-all duration-200"
            >
              Keep subscription
            </button>
            <button
              @click="cancelSubscription"
              :disabled="canceling"
              class="flex-1 py-2.5 rounded-xl text-[14px] font-semibold bg-[#e94560] border border-[#e94560] text-white hover:bg-[#d63b55] cursor-pointer transition-all duration-200 disabled:opacity-50"
            >
              {{ canceling ? 'Canceling...' : 'Yes, cancel' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </HomeLayout>
</template>
