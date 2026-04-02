<script setup>
import { useAuthStore } from '@/stores/auth'
import { ref, watch } from 'vue'

const props = defineProps({
  visible: Boolean,
  initialCode: { type: String, default: '' },
})
const emit = defineEmits(['close', 'joined'])

const auth = useAuthStore()

const code = ref('')
const loading = ref(false)
const checking = ref(false)
const errorMsg = ref('')
const successMsg = ref('')
const gameInfo = ref(null)

watch(() => props.visible, (val) => {
  if (val) {
    code.value = props.initialCode || ''
    loading.value = false
    checking.value = false
    errorMsg.value = ''
    successMsg.value = ''
    gameInfo.value = null
    if (code.value) checkInvite()
  }
})

async function checkInvite() {
  if (!code.value.trim()) return
  checking.value = true
  errorMsg.value = ''
  gameInfo.value = null
  try {
    gameInfo.value = await auth.getInviteInfo(code.value.trim())
  } catch (err) {
    errorMsg.value = err.response?.data?.error || 'Invalid invite code'
  } finally {
    checking.value = false
  }
}

async function handleJoin() {
  errorMsg.value = ''
  successMsg.value = ''
  if (!code.value.trim()) {
    errorMsg.value = 'Enter an invite code'
    return
  }
  loading.value = true
  try {
    const data = await auth.joinGameByCode(code.value.trim())
    successMsg.value = `Joined "${data.title}" successfully!`
    setTimeout(() => {
      emit('joined', data)
      emit('close')
    }, 1200)
  } catch (err) {
    errorMsg.value = err.response?.data?.error || 'Failed to join game'
  } finally {
    loading.value = false
  }
}

function close() {
  if (!loading.value) emit('close')
}
</script>

<template>
  <Teleport to="body">
    <div v-if="visible" class="fixed inset-0 z-[10000] flex items-center justify-center" @click.self="close">
      <div class="absolute inset-0 bg-black/60 backdrop-blur-sm" @click="close"></div>
      <div class="relative w-full max-w-[480px] mx-4 bg-[#0f0f23] border border-[rgba(126,200,227,0.15)] rounded-2xl shadow-[0_16px_64px_rgba(0,0,0,0.6)] p-8">
        <button @click="close" class="absolute top-4 right-4 text-[#7ec8e3]/40 hover:text-[#e8e8f0] bg-transparent border-none cursor-pointer text-xl leading-none">&times;</button>

        <h2 class="font-[Cinzel] text-[24px] font-bold text-[#e8e8f0] tracking-wide mb-2">Join a Game</h2>
        <p class="text-[#7ec8e3]/40 text-[14px] mb-6">Enter an invite code from your GM</p>

        <div v-if="successMsg" class="text-center py-8">
          <div class="w-14 h-14 mx-auto mb-3 rounded-full bg-[rgba(76,175,80,0.15)] flex items-center justify-center">
            <svg class="w-7 h-7 text-[#4caf50]" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <p class="text-[#e8e8f0] text-[16px] font-semibold">{{ successMsg }}</p>
        </div>

        <div v-else class="space-y-5">
          <div>
            <label class="block text-[#7ec8e3]/60 text-[13px] font-medium mb-2">Invite Code</label>
            <input
              v-model="code"
              type="text"
              placeholder="e.g. A1B2C3D4E5"
              maxlength="20"
              class="w-full px-4 py-3 bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.15)] rounded-lg text-[#e8e8f0] text-[16px] font-mono tracking-[0.15em] text-center placeholder-[#7ec8e3]/30 outline-none focus:border-[rgba(233,69,96,0.4)] transition-colors uppercase"
              @input="code = code.toUpperCase()"
              @blur="checkInvite"
              @keyup.enter="handleJoin"
            />
          </div>

          <div v-if="checking" class="p-4 bg-[rgba(15,15,35,0.4)] border border-[rgba(126,200,227,0.08)] rounded-xl">
            <p class="text-[#7ec8e3]/40 text-[13px]">Checking code...</p>
          </div>

          <div v-else-if="gameInfo" class="p-4 bg-[rgba(15,15,35,0.4)] border border-[rgba(126,200,227,0.15)] rounded-xl">
            <p class="text-[#7ec8e3]/40 text-[11px] uppercase tracking-wider mb-1">Game Found</p>
            <p class="text-[#e8e8f0] text-[16px] font-bold">{{ gameInfo.title }}</p>
            <p v-if="gameInfo.member_count" class="text-[#7ec8e3]/40 text-[13px] mt-1">{{ gameInfo.member_count }} members</p>
          </div>

          <p v-if="errorMsg" class="text-[#e94560] text-[13px]">{{ errorMsg }}</p>

          <div class="flex gap-3 pt-1">
            <button
              @click="handleJoin"
              :disabled="loading || !code.trim()"
              class="flex-1 py-3 bg-gradient-to-br from-[#e94560] to-[#c23152] text-white text-[14px] font-semibold rounded-lg hover:-translate-y-0.5 hover:shadow-[0_6px_20px_rgba(233,69,96,0.4)] transition-all duration-300 cursor-pointer border-none disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:translate-y-0"
            >
              {{ loading ? 'Joining...' : 'Join Game' }}
            </button>
            <button
              @click="close"
              class="px-5 py-3 text-[#e8e8f0]/60 text-[14px] font-medium border border-[rgba(126,200,227,0.15)] rounded-lg hover:border-[rgba(126,200,227,0.3)] hover:text-[#e8e8f0] transition-all duration-300 bg-transparent cursor-pointer"
            >
              Cancel
            </button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>
