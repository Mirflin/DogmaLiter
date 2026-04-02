<script setup>
import HomeLayout from '@/layouts/HomeLayout.vue'
import { useAuthStore } from '@/stores/auth'
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import api from '@/api'

const auth = useAuthStore()
const router = useRouter()

const title = ref('')
const description = ref('')
const maxPlayers = ref(6)
const showStandardAttrs = ref(true)
const enableChat = ref(true)
const enableItemTrading = ref(true)
const coverFile = ref(null)
const coverPreview = ref(null)

const loading = ref(false)
const errorMsg = ref('')
const successData = ref(null)

const plan = ref(null)
const gameCount = ref(0)

onMounted(async () => {
  try {
    const { data } = await api.get('/me')
    plan.value = data
    const games = await auth.fetchMyGames()
    gameCount.value = games.length
  } catch {}
})

const canCreate = () => {
  if (!plan.value) return false
  return true
}

const limitReached = () => {
  if (!plan.value) return false
  const maxGames = plan.value.max_games_owned
  if (maxGames === -1) return false
  return gameCount.value >= (maxGames || 999)
}

function handleCoverSelect(e) {
  const file = e.target.files?.[0]
  if (!file) return
  const allowed = ['image/jpeg', 'image/png', 'image/webp']
  if (!allowed.includes(file.type)) {
    errorMsg.value = 'Only JPEG, PNG, and WebP images are allowed'
    return
  }
  if (file.size > 5 * 1024 * 1024) {
    errorMsg.value = 'Cover image must be under 5MB'
    return
  }
  coverFile.value = file
  coverPreview.value = URL.createObjectURL(file)
}

function removeCover() {
  coverFile.value = null
  if (coverPreview.value) {
    URL.revokeObjectURL(coverPreview.value)
    coverPreview.value = null
  }
}

async function handleCreate() {
  errorMsg.value = ''
  if (!title.value.trim()) {
    errorMsg.value = 'Game title is required'
    return
  }
  if (limitReached()) {
    errorMsg.value = 'Game limit reached for your plan. Upgrade to create more games.'
    return
  }

  loading.value = true
  try {
    const data = await auth.createGame({
      title: title.value.trim(),
      description: description.value.trim(),
      system: 'custom',
      max_players: maxPlayers.value,
      show_standard_attrs: showStandardAttrs.value,
      enable_chat: enableChat.value,
      enable_item_trading: enableItemTrading.value,
    })
    if (coverFile.value && data.id) {
      try {
        await auth.uploadCoverImage(data.id, coverFile.value)
      } catch {}
    }
    successData.value = data
    startCountdown()
  } catch (err) {
    errorMsg.value = err.response?.data?.error || 'Failed to create game'
  } finally {
    loading.value = false
  }
}

function goToGames() {
  router.push('/games')
}

function copyInviteLink() {
  if (successData.value?.invite_code) {
    const link = `${window.location.origin}/join/${successData.value.invite_code}`
    navigator.clipboard.writeText(link)
  }
}

const countdown = ref(0)
let countdownTimer = null

const countdownDisplay = computed(() => {
  const m = Math.floor(countdown.value / 60)
  const s = countdown.value % 60
  return `${m}:${String(s).padStart(2, '0')}`
})

const codeExpired = computed(() => countdown.value <= 0 && successData.value?.invite_code_expires_at)

function startCountdown() {
  if (!successData.value?.invite_code_expires_at) return
  clearInterval(countdownTimer)
  const update = () => {
    const expiresAt = new Date(successData.value.invite_code_expires_at).getTime()
    const remaining = Math.max(0, Math.floor((expiresAt - Date.now()) / 1000))
    countdown.value = remaining
    if (remaining <= 0) clearInterval(countdownTimer)
  }
  update()
  countdownTimer = setInterval(update, 1000)
}

onUnmounted(() => clearInterval(countdownTimer))
</script>

<template>
  <HomeLayout>
    <div class="max-w-[640px] mx-auto px-6 py-8">
      <h1 class="font-[Cinzel] text-[28px] font-bold text-[#e8e8f0] tracking-wide mb-2">Create Game</h1>
      <p class="text-[#7ec8e3]/40 text-[14px] mb-8">Set up a new game session for your players</p>
      <div v-if="limitReached()" class="mb-6 p-4 bg-[rgba(233,69,96,0.1)] border border-[rgba(233,69,96,0.3)] rounded-lg">
        <p class="text-[#e94560] text-[14px] font-medium">You've reached the game limit for your plan.</p>
        <router-link to="/plans" class="text-[#e94560] text-[13px] underline hover:text-[#ff6b81]">Upgrade your plan</router-link>
      </div>
      <div v-if="successData" class="text-center py-12">
        <div class="w-16 h-16 mx-auto mb-4 rounded-full bg-[rgba(76,175,80,0.15)] flex items-center justify-center">
          <svg class="w-8 h-8 text-[#4caf50]" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
          </svg>
        </div>
        <h2 class="text-[#e8e8f0] text-[22px] font-bold mb-2">Game Created!</h2>
        <p class="text-[#7ec8e3]/50 text-[14px] mb-6">"{{ successData.title }}" is ready</p>

        <div class="mb-8 p-5 bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.15)] rounded-xl inline-block">
          <p class="text-[#7ec8e3]/40 text-[12px] mb-2 uppercase tracking-wider">Invite Code</p>
          <p :class="['text-[28px] font-mono font-bold tracking-[0.3em]', codeExpired ? 'text-[#7ec8e3]/20 line-through' : 'text-[#e8e8f0]']">{{ successData.invite_code }}</p>

          <div v-if="codeExpired" class="mt-3">
            <p class="text-[#e94560] text-[12px] mb-2">Code expired</p>
            <p class="text-[#7ec8e3]/40 text-[11px]">You can regenerate it from game settings</p>
          </div>

          <div v-else class="mt-3">
            <p class="text-[#7ec8e3]/40 text-[11px] mb-2">
              Expires in <span :class="countdown <= 60 ? 'text-[#e94560]' : 'text-[#e8e8f0]'" class="font-mono font-semibold">{{ countdownDisplay }}</span>
            </p>
            <button
              @click="copyInviteLink"
              class="px-4 py-1.5 text-[12px] text-[#7ec8e3]/60 border border-[rgba(126,200,227,0.15)] rounded hover:border-[#e94560] hover:text-[#e94560] transition-all cursor-pointer bg-transparent"
            >
              Copy Invite Link
            </button>
          </div>
        </div>

        <div>
          <button
            @click="goToGames"
            class="px-8 py-3 bg-gradient-to-br from-[#e94560] to-[#c23152] text-white text-[14px] font-semibold rounded-lg hover:-translate-y-0.5 hover:shadow-[0_6px_20px_rgba(233,69,96,0.4)] transition-all duration-300 cursor-pointer border-none"
          >
            Go to My Games
          </button>
        </div>
      </div>
      <div v-else class="space-y-6">
        <div>
          <label class="block text-[#7ec8e3]/60 text-[13px] font-medium mb-2">Game Title *</label>
          <input
            v-model="title"
            type="text"
            maxlength="200"
            placeholder="Enter game title..."
            class="w-full px-4 py-3 bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.15)] rounded-lg text-[#e8e8f0] text-[14px] placeholder-[#7ec8e3]/30 outline-none focus:border-[rgba(233,69,96,0.4)] transition-colors"
          />
        </div>
        <div>
          <label class="block text-[#7ec8e3]/60 text-[13px] font-medium mb-2">Description</label>
          <textarea
            v-model="description"
            rows="3"
            placeholder="Describe your game world, setting, or campaign..."
            class="w-full px-4 py-3 bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.15)] rounded-lg text-[#e8e8f0] text-[14px] placeholder-[#7ec8e3]/30 outline-none focus:border-[rgba(233,69,96,0.4)] transition-colors resize-y min-h-[80px]"
          ></textarea>
        </div>
        <div>
          <label class="block text-[#7ec8e3]/60 text-[13px] font-medium mb-2">Cover Image</label>
          <div v-if="coverPreview" class="relative mb-3">
            <img :src="coverPreview" alt="Cover preview" class="w-full h-48 object-cover rounded-lg border border-[rgba(126,200,227,0.15)]" />
            <button
              @click="removeCover"
              class="absolute top-2 right-2 w-8 h-8 bg-[rgba(0,0,0,0.7)] border border-[rgba(233,69,96,0.4)] rounded-full flex items-center justify-center cursor-pointer hover:bg-[rgba(233,69,96,0.3)] transition-colors"
            >
              <svg class="w-4 h-4 text-[#e94560]" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
            </button>
          </div>
          <label v-else class="flex flex-col items-center justify-center w-full h-36 bg-[rgba(15,15,35,0.6)] border-2 border-dashed border-[rgba(126,200,227,0.15)] rounded-lg cursor-pointer hover:border-[rgba(233,69,96,0.3)] transition-colors">
            <svg class="w-8 h-8 text-[#7ec8e3]/30 mb-2" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M2.25 15.75l5.159-5.159a2.25 2.25 0 013.182 0l5.159 5.159m-1.5-1.5l1.409-1.41a2.25 2.25 0 013.182 0l2.909 2.91m-18 3.75h16.5a1.5 1.5 0 001.5-1.5V6a1.5 1.5 0 00-1.5-1.5H3.75A1.5 1.5 0 002.25 6v12a1.5 1.5 0 001.5 1.5zm10.5-11.25h.008v.008h-.008V8.25zm.375 0a.375.375 0 11-.75 0 .375.375 0 01.75 0z" /></svg>
            <span class="text-[#7ec8e3]/30 text-[13px]">Click to upload cover image</span>
            <span class="text-[#7ec8e3]/20 text-[11px] mt-1">JPEG, PNG or WebP, max 5MB</span>
            <input type="file" accept="image/jpeg,image/png,image/webp" class="hidden" @change="handleCoverSelect" />
          </label>
        </div>
        <div>
          <label class="block text-[#7ec8e3]/60 text-[13px] font-medium mb-2">
            Max Players
            <span class="text-[#7ec8e3]/30 ml-1">(including GM, {{ auth.user?.max_players_per_game === -1 ? 'unlimited on your plan' : `max ${auth.user?.max_players_per_game || '?'} on your plan` }})</span>
          </label>
          <input
            v-model.number="maxPlayers"
            type="number"
            min="2"
            :max="auth.user?.max_players_per_game === -1 ? undefined : (auth.user?.max_players_per_game || 20)"
            class="w-full px-4 py-3 bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.15)] rounded-lg text-[#e8e8f0] text-[14px] outline-none focus:border-[rgba(233,69,96,0.4)] transition-colors"
          />
        </div>
        <div class="space-y-4 p-5 bg-[rgba(15,15,35,0.4)] border border-[rgba(126,200,227,0.08)] rounded-xl">
          <p class="text-[#7ec8e3]/60 text-[13px] font-medium mb-1">Game Options</p>

          <label class="flex items-center justify-between cursor-pointer group">
            <span class="text-[#e8e8f0]/80 text-[14px] group-hover:text-[#e8e8f0] transition-colors">Standard Attributes (STR, DEX, etc.)</span>
            <div class="relative">
              <input type="checkbox" v-model="showStandardAttrs" class="sr-only peer" />
              <div class="w-10 h-5 bg-[rgba(126,200,227,0.15)] rounded-full peer-checked:bg-[#e94560] transition-colors"></div>
              <div class="absolute left-0.5 top-0.5 w-4 h-4 bg-[#e8e8f0] rounded-full transition-transform peer-checked:translate-x-5"></div>
            </div>
          </label>

          <label class="flex items-center justify-between cursor-pointer group">
            <span class="text-[#e8e8f0]/80 text-[14px] group-hover:text-[#e8e8f0] transition-colors">In-Game Chat</span>
            <div class="relative">
              <input type="checkbox" v-model="enableChat" class="sr-only peer" />
              <div class="w-10 h-5 bg-[rgba(126,200,227,0.15)] rounded-full peer-checked:bg-[#e94560] transition-colors"></div>
              <div class="absolute left-0.5 top-0.5 w-4 h-4 bg-[#e8e8f0] rounded-full transition-transform peer-checked:translate-x-5"></div>
            </div>
          </label>

          <label class="flex items-center justify-between cursor-pointer group">
            <span class="text-[#e8e8f0]/80 text-[14px] group-hover:text-[#e8e8f0] transition-colors">Item Trading</span>
            <div class="relative">
              <input type="checkbox" v-model="enableItemTrading" class="sr-only peer" />
              <div class="w-10 h-5 bg-[rgba(126,200,227,0.15)] rounded-full peer-checked:bg-[#e94560] transition-colors"></div>
              <div class="absolute left-0.5 top-0.5 w-4 h-4 bg-[#e8e8f0] rounded-full transition-transform peer-checked:translate-x-5"></div>
            </div>
          </label>
        </div>
        <p v-if="errorMsg" class="text-[#e94560] text-[13px]">{{ errorMsg }}</p>
        <div class="flex gap-4 pt-2">
          <button
            @click="handleCreate"
            :disabled="loading || limitReached()"
            class="flex-1 py-3 bg-gradient-to-br from-[#e94560] to-[#c23152] text-white text-[14px] font-semibold rounded-lg hover:-translate-y-0.5 hover:shadow-[0_6px_20px_rgba(233,69,96,0.4)] transition-all duration-300 cursor-pointer border-none disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:translate-y-0"
          >
            {{ loading ? 'Creating...' : 'Create Game' }}
          </button>
          <router-link
            to="/games"
            class="px-6 py-3 text-[#e8e8f0]/60 text-[14px] font-medium no-underline border border-[rgba(126,200,227,0.15)] rounded-lg hover:border-[rgba(126,200,227,0.3)] hover:text-[#e8e8f0] transition-all duration-300 flex items-center"
          >
            Cancel
          </router-link>
        </div>
      </div>
    </div>
  </HomeLayout>
</template>
