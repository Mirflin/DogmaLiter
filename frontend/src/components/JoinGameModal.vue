<script setup>
import { useAuthStore } from '@/stores/auth'
import { ref, watch } from 'vue'
import { notify } from '@/notify'

const props = defineProps({
  visible: Boolean,
  initialCode: { type: String, default: '' },
})
const emit = defineEmits(['close', 'joined'])

const auth = useAuthStore()

const code = ref('')
const loading = ref(false)
const checking = ref(false)
const gameInfo = ref(null)

watch(() => props.visible, (val) => {
  if (val) {
    code.value = props.initialCode || ''
    loading.value = false
    checking.value = false
    gameInfo.value = null
    if (code.value) checkInvite()
  }
})

async function checkInvite() {
  if (!code.value.trim()) return
  checking.value = true
  gameInfo.value = null
  try {
    gameInfo.value = await auth.getInviteInfo(code.value.trim())
  } catch (err) {
    notify.error({
      title: 'Invite not found',
      message: err.response?.data?.error || 'Invalid invite code',
    })
  } finally {
    checking.value = false
  }
}

async function handleJoin() {
  if (!code.value.trim()) {
    notify.warning({
      title: 'Invite code required',
      message: 'Enter an invite code to continue.',
    })
    return
  }
  loading.value = true
  try {
    const data = await auth.joinGameByCode(code.value.trim())
    notify.success({
      title: 'Game joined',
      message: `You joined "${data.title}" successfully.`,
    })
    emit('joined', data)
    emit('close')
  } catch (err) {
    notify.error(err, 'Failed to join game')
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
      <div class="relative w-full max-w-[480px] mx-4 bg-n2 border border-line rounded-2xl shadow-[0_16px_64px_rgba(0,0,0,0.6)] p-8">
        <button @click="close" class="absolute top-4 right-4 text-faint hover:text-ink bg-transparent border-none cursor-pointer text-xl leading-none">&times;</button>

        <h2 class="font-heading text-[24px] font-bold text-ink tracking-wide mb-2">Join a Game</h2>
        <p class="text-faint text-[14px] mb-6">Enter an invite code from your GM</p>
        <div class="space-y-5">
          <div>
            <label class="block text-dim text-[13px] font-medium mb-2">Invite Code</label>
            <input
              v-model="code"
              type="text"
              placeholder="e.g. A1B2C3D4E5"
              maxlength="20"
              class="w-full px-4 py-3 bg-n2/60 border border-line rounded-lg text-ink text-[16px] font-mono tracking-[0.15em] text-center placeholder-faint outline-none focus:border-primary/40 transition-colors uppercase"
              @input="code = code.toUpperCase()"
              @blur="checkInvite"
              @keyup.enter="handleJoin"
            />
          </div>

          <div v-if="checking" class="p-4 bg-n2/40 border border-line-soft rounded-xl">
            <p class="text-faint text-[13px]">Checking code...</p>
          </div>

          <div v-else-if="gameInfo" class="p-4 bg-n2/40 border border-line rounded-xl">
            <p class="text-faint text-[11px] uppercase tracking-wider mb-1">Game Found</p>
            <p class="text-ink text-[16px] font-bold">{{ gameInfo.title }}</p>
            <p v-if="gameInfo.member_count" class="text-faint text-[13px] mt-1">{{ gameInfo.member_count }} members</p>
          </div>

          <div class="flex gap-3 pt-1">
            <button
              @click="handleJoin"
              :disabled="loading || !code.trim()"
              class="flex-1 py-3 bg-primary hover:bg-primary-hover text-white text-[14px] font-semibold rounded-lg transition-all duration-300 cursor-pointer border-none disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:translate-y-0"
            >
              {{ loading ? 'Joining...' : 'Join Game' }}
            </button>
            <button
              @click="close"
              class="px-5 py-3 text-ink/60 text-[14px] font-medium border border-line rounded-lg hover:border-line-strong hover:text-ink transition-all duration-300 bg-transparent cursor-pointer"
            >
              Cancel
            </button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>
