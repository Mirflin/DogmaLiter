<script setup>
import { useAuthStore } from '@/stores/auth'
import { API_URL } from '@/api'
import { ref, watch, computed, onUnmounted } from 'vue'

const props = defineProps({
  visible: Boolean,
  gameId: String,
})
const emit = defineEmits(['close', 'updated', 'deleted'])

const auth = useAuthStore()

const loading = ref(true)
const saving = ref(false)
const errorMsg = ref('')
const successMsg = ref('')
const game = ref(null)

const title = ref('')
const description = ref('')
const maxPlayers = ref(6)
const showStandardAttrs = ref(true)
const enableChat = ref(true)
const enableItemTrading = ref(true)

const inviteCode = ref('')
const inviteExpiresAt = ref(null)
const regenerating = ref(false)
const codeCopied = ref(false)
const countdown = ref('')
const codeExpired = ref(false)
let countdownTimer = null

const inviteLinkText = computed(() => `${window.location.origin}/join/${inviteCode.value}`)

const coverFile = ref(null)
const coverPreview = ref(null)
const uploadingCover = ref(false)

const isOwner = computed(() => game.value?.owner_id === auth.user?.id)

watch(() => props.visible, async (val) => {
  if (val && props.gameId) {
    loading.value = true
    errorMsg.value = ''
    successMsg.value = ''
    try {
      const data = await auth.getGame(props.gameId)
      game.value = data
      title.value = data.title
      description.value = data.description || ''
      maxPlayers.value = data.max_players
      showStandardAttrs.value = data.show_standard_attrs
      enableChat.value = data.enable_chat
      enableItemTrading.value = data.enable_item_trading
      inviteCode.value = data.invite_code || ''
      inviteExpiresAt.value = data.invite_code_expires_at || null
      coverPreview.value = data.cover_image_id ? `${API_URL}/api/uploads/${data.cover_image_id}` : null
      coverFile.value = null
      startCountdown()
    } catch {
      errorMsg.value = 'Failed to load game'
    } finally {
      loading.value = false
    }
  } else {
    stopCountdown()
  }
})

onUnmounted(() => stopCountdown())

function startCountdown() {
  stopCountdown()
  updateCountdown()
  countdownTimer = setInterval(updateCountdown, 1000)
}

function stopCountdown() {
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
}

function updateCountdown() {
  if (!inviteExpiresAt.value) {
    countdown.value = ''
    codeExpired.value = true
    return
  }
  const now = Date.now()
  const exp = new Date(inviteExpiresAt.value).getTime()
  const diff = exp - now
  if (diff <= 0) {
    countdown.value = 'Expired'
    codeExpired.value = true
    stopCountdown()
    return
  }
  codeExpired.value = false
  const m = Math.floor(diff / 60000)
  const s = Math.floor((diff % 60000) / 1000)
  countdown.value = `${m}:${String(s).padStart(2, '0')}`
}

async function regenerate() {
  regenerating.value = true
  try {
    const data = await auth.regenerateInviteCode(props.gameId)
    inviteCode.value = data.invite_code
    inviteExpiresAt.value = data.invite_code_expires_at
    codeExpired.value = false
    startCountdown()
  } catch (err) {
    errorMsg.value = err.response?.data?.error || 'Failed to regenerate code'
  } finally {
    regenerating.value = false
  }
}

function copyInviteLink() {
  const text = inviteLinkText.value
  if (navigator.clipboard?.writeText) {
    navigator.clipboard.writeText(text).then(() => {
      codeCopied.value = true
      setTimeout(() => { codeCopied.value = false }, 2000)
    })
  } else {
    const el = document.createElement('textarea')
    el.value = text
    el.style.position = 'fixed'
    el.style.opacity = '0'
    document.body.appendChild(el)
    el.select()
    document.execCommand('copy')
    document.body.removeChild(el)
    codeCopied.value = true
    setTimeout(() => { codeCopied.value = false }, 2000)
  }
}

async function handleSave() {
  saving.value = true
  errorMsg.value = ''
  successMsg.value = ''
  try {
    await auth.updateGame(props.gameId, {
      title: title.value,
      description: description.value,
      max_players: maxPlayers.value,
      show_standard_attrs: showStandardAttrs.value,
      enable_chat: enableChat.value,
      enable_item_trading: enableItemTrading.value,
    })

    if (coverFile.value) {
      await auth.uploadCoverImage(props.gameId, coverFile.value)
    }

    successMsg.value = 'Settings saved'
    setTimeout(() => successMsg.value = '', 2000)
    emit('updated')
  } catch (err) {
    errorMsg.value = err.response?.data?.error || 'Failed to save'
  } finally {
    saving.value = false
  }
}

function handleCoverSelect(e) {
  const file = e.target.files?.[0]
  if (!file) return
  coverFile.value = file
  coverPreview.value = URL.createObjectURL(file)
}

function removeCover() {
  coverFile.value = null
  coverPreview.value = game.value?.cover_image_id ? `${API_URL}/api/uploads/${game.value.cover_image_id}` : null
}

async function handleDelete() {
  if (!confirm(`Delete "${title.value}"? This cannot be undone.`)) return
  try {
    await auth.deleteGame(props.gameId)
    emit('deleted', props.gameId)
    emit('close')
  } catch (err) {
    errorMsg.value = err.response?.data?.error || 'Failed to delete'
  }
}

function close() {
  if (!saving.value) emit('close')
}
</script>

<template>
  <Teleport to="body">
    <div v-if="visible" class="fixed inset-0 z-[10000] flex items-center justify-center" @click.self="close">
      <div class="absolute inset-0 bg-black/60 backdrop-blur-sm" @click="close"></div>
      <div class="relative w-full max-w-[600px] max-h-[90vh] overflow-y-auto mx-4 bg-[#0f0f23] border border-[rgba(126,200,227,0.15)] rounded-2xl shadow-[0_16px_64px_rgba(0,0,0,0.6)] p-8">
        <button @click="close" class="absolute top-4 right-4 text-[#7ec8e3]/40 hover:text-[#e8e8f0] bg-transparent border-none cursor-pointer text-xl leading-none">&times;</button>

        <h2 class="font-[Cinzel] text-[24px] font-bold text-[#e8e8f0] tracking-wide mb-6">Game Settings</h2>

        <div v-if="loading" class="text-center py-12">
          <div class="w-8 h-8 border-2 border-[#e94560] border-t-transparent rounded-full animate-spin mx-auto"></div>
        </div>

        <div v-else-if="!isOwner" class="text-center py-8">
          <p class="text-[#7ec8e3]/40 text-[14px]">Only the game owner can manage settings</p>
        </div>

        <div v-else class="space-y-6">
          <div>
            <label class="block text-[#7ec8e3]/60 text-[13px] font-medium mb-2">Cover Image</label>
            <div class="flex items-center gap-4">
              <div class="w-24 h-16 rounded-lg overflow-hidden bg-[rgba(126,200,227,0.05)] border border-[rgba(126,200,227,0.1)] flex items-center justify-center">
                <img v-if="coverPreview" :src="coverPreview" class="w-full h-full object-cover" />
                <span v-else class="text-[#7ec8e3]/20 text-[10px]">No cover</span>
              </div>
              <label class="px-4 py-2 text-[13px] text-[#e8e8f0]/60 border border-[rgba(126,200,227,0.15)] rounded-lg hover:border-[rgba(126,200,227,0.3)] cursor-pointer transition-colors">
                Change
                <input type="file" accept="image/jpeg,image/png,image/webp" class="hidden" @change="handleCoverSelect" />
              </label>
              <button v-if="coverFile" @click="removeCover" class="text-[#7ec8e3]/40 text-[12px] bg-transparent border-none cursor-pointer hover:text-[#e94560]">Reset</button>
            </div>
          </div>
          <div>
            <label class="block text-[#7ec8e3]/60 text-[13px] font-medium mb-2">Title</label>
            <input v-model="title" type="text" maxlength="200" class="w-full px-4 py-3 bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.15)] rounded-lg text-[#e8e8f0] text-[14px] outline-none focus:border-[rgba(233,69,96,0.4)] transition-colors" />
          </div>
          <div>
            <label class="block text-[#7ec8e3]/60 text-[13px] font-medium mb-2">Description</label>
            <textarea v-model="description" rows="3" class="w-full px-4 py-3 bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.15)] rounded-lg text-[#e8e8f0] text-[14px] outline-none focus:border-[rgba(233,69,96,0.4)] transition-colors resize-y"></textarea>
          </div>
          <div>
            <label class="block text-[#7ec8e3]/60 text-[13px] font-medium mb-2">Max Players</label>
            <div class="flex items-center gap-3">
              <input v-model.number="maxPlayers" type="number" min="1" :max="auth.user?.max_players_per_game === -1 ? undefined : (auth.user?.max_players_per_game || 50)" class="w-24 px-4 py-3 bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.15)] rounded-lg text-[#e8e8f0] text-[14px] outline-none focus:border-[rgba(233,69,96,0.4)] transition-colors" />
              <span class="text-[#7ec8e3]/30 text-[12px]">{{ auth.user?.max_players_per_game === -1 ? 'unlimited on your plan' : `max ${auth.user?.max_players_per_game || '?'} on your plan` }}</span>
            </div>
          </div>
          <div class="space-y-3">
            <label class="flex items-center gap-3 cursor-pointer group">
              <span class="relative flex items-center justify-center w-5 h-5 rounded-md border transition-all duration-150"
                :class="showStandardAttrs ? 'bg-[#e94560] border-[#e94560]' : 'bg-transparent border-[rgba(126,200,227,0.25)] group-hover:border-[rgba(233,69,96,0.5)]'">
                <svg v-if="showStandardAttrs" class="w-3 h-3 text-white" fill="none" stroke="currentColor" stroke-width="3" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                </svg>
                <input type="checkbox" v-model="showStandardAttrs" class="absolute inset-0 opacity-0 w-full h-full cursor-pointer m-0" />
              </span>
              <span class="text-[#e8e8f0]/70 text-[13px] select-none">Standard Attributes</span>
            </label>
            <label class="flex items-center gap-3 cursor-pointer group">
              <span class="relative flex items-center justify-center w-5 h-5 rounded-md border transition-all duration-150"
                :class="enableChat ? 'bg-[#e94560] border-[#e94560]' : 'bg-transparent border-[rgba(126,200,227,0.25)] group-hover:border-[rgba(233,69,96,0.5)]'">
                <svg v-if="enableChat" class="w-3 h-3 text-white" fill="none" stroke="currentColor" stroke-width="3" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                </svg>
                <input type="checkbox" v-model="enableChat" class="absolute inset-0 opacity-0 w-full h-full cursor-pointer m-0" />
              </span>
              <span class="text-[#e8e8f0]/70 text-[13px] select-none">Chat</span>
            </label>
            <label class="flex items-center gap-3 cursor-pointer group">
              <span class="relative flex items-center justify-center w-5 h-5 rounded-md border transition-all duration-150"
                :class="enableItemTrading ? 'bg-[#e94560] border-[#e94560]' : 'bg-transparent border-[rgba(126,200,227,0.25)] group-hover:border-[rgba(233,69,96,0.5)]'">
                <svg v-if="enableItemTrading" class="w-3 h-3 text-white" fill="none" stroke="currentColor" stroke-width="3" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                </svg>
                <input type="checkbox" v-model="enableItemTrading" class="absolute inset-0 opacity-0 w-full h-full cursor-pointer m-0" />
              </span>
              <span class="text-[#e8e8f0]/70 text-[13px] select-none">Item Trading</span>
            </label>
          </div>
          <div class="border-t border-[rgba(126,200,227,0.08)] pt-5">
            <label class="block text-[#7ec8e3]/60 text-[13px] font-medium mb-3">Invite Code</label>
            <div class="flex items-center gap-3 mb-2">
              <span :class="[codeExpired ? 'line-through text-[#7ec8e3]/25' : 'text-[#e8e8f0]']" class="font-mono text-[18px] tracking-[0.2em] font-bold">{{ inviteCode || '—' }}</span>
              <span v-if="countdown" :class="codeExpired ? 'text-[#e94560]' : 'text-[#7ec8e3]/40'" class="text-[12px]">{{ countdown }}</span>
            </div>
            <div class="flex gap-2">
              <button
                @click="regenerate"
                :disabled="regenerating"
                class="px-4 py-2 text-[12px] font-medium text-[#e8e8f0]/60 border border-[rgba(126,200,227,0.15)] rounded-lg hover:border-[#e94560] hover:text-[#e94560] transition-all bg-transparent cursor-pointer disabled:opacity-50"
              >
                {{ regenerating ? 'Regenerating...' : 'New Code' }}
              </button>
              <button
                v-if="!codeExpired && inviteCode"
                @click="copyInviteLink"
                class="px-4 py-2 text-[12px] font-medium text-[#e8e8f0]/60 border border-[rgba(126,200,227,0.15)] rounded-lg hover:border-[#e94560] hover:text-[#e94560] transition-all bg-transparent cursor-pointer"
              >
                {{ codeCopied ? 'Copied!' : 'Copy Link' }}
              </button>
            </div>
          </div>
          <p v-if="errorMsg" class="text-[#e94560] text-[13px]">{{ errorMsg }}</p>
          <p v-if="successMsg" class="text-[#4caf50] text-[13px]">{{ successMsg }}</p>
          <div class="flex items-center justify-between pt-2 border-t border-[rgba(126,200,227,0.08)]">
            <button
              @click="handleDelete"
              class="px-4 py-2.5 text-[13px] text-[#e94560] bg-transparent border border-[rgba(233,69,96,0.2)] rounded-lg hover:bg-[rgba(233,69,96,0.1)] transition-colors cursor-pointer"
            >
              Delete Game
            </button>
            <div class="flex gap-3">
              <button @click="close" class="px-5 py-2.5 text-[13px] text-[#e8e8f0]/60 bg-transparent border border-[rgba(126,200,227,0.15)] rounded-lg hover:border-[rgba(126,200,227,0.3)] transition-colors cursor-pointer">Cancel</button>
              <button
                @click="handleSave"
                :disabled="saving"
                class="px-6 py-2.5 text-[13px] font-semibold text-white bg-gradient-to-br from-[#e94560] to-[#c23152] rounded-lg hover:-translate-y-0.5 hover:shadow-[0_6px_20px_rgba(233,69,96,0.4)] transition-all cursor-pointer border-none disabled:opacity-50"
              >
                {{ saving ? 'Saving...' : 'Save' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>
