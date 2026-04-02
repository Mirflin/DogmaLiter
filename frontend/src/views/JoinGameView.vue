<script setup>
import HomeLayout from '@/layouts/HomeLayout.vue'
import { useAuthStore } from '@/stores/auth'
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()

const code = ref('')
const loading = ref(false)
const checking = ref(false)
const errorMsg = ref('')
const successMsg = ref('')
const gameInfo = ref(null)

onMounted(async () => {
  const inviteCode = route.params.code
  if (inviteCode) {
    code.value = inviteCode
    await checkInvite()
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
    setTimeout(() => router.push('/games'), 1500)
  } catch (err) {
    errorMsg.value = err.response?.data?.error || 'Failed to join game'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <HomeLayout>
    <div class="max-w-[480px] mx-auto px-6 py-8">
      <h1 class="font-[Cinzel] text-[28px] font-bold text-[#e8e8f0] tracking-wide mb-2">Join a Game</h1>
      <p class="text-[#7ec8e3]/40 text-[14px] mb-8">Enter an invite code from your GM to join their game</p>
      <div v-if="successMsg" class="text-center py-12">
        <div class="w-16 h-16 mx-auto mb-4 rounded-full bg-[rgba(76,175,80,0.15)] flex items-center justify-center">
          <svg class="w-8 h-8 text-[#4caf50]" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
          </svg>
        </div>
        <p class="text-[#e8e8f0] text-[18px] font-semibold">{{ successMsg }}</p>
        <p class="text-[#7ec8e3]/40 text-[13px] mt-2">Redirecting...</p>
      </div>
      <div v-else class="space-y-6">
        <div>
          <label class="block text-[#7ec8e3]/60 text-[13px] font-medium mb-2">Invite Code</label>
          <div class="flex gap-3">
            <input
              v-model="code"
              type="text"
              placeholder="e.g. A1B2C3D4E5"
              maxlength="20"
              class="flex-1 px-4 py-3 bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.15)] rounded-lg text-[#e8e8f0] text-[16px] font-mono tracking-[0.15em] text-center placeholder-[#7ec8e3]/30 outline-none focus:border-[rgba(233,69,96,0.4)] transition-colors uppercase"
              @input="code = code.toUpperCase()"
              @blur="checkInvite"
            />
          </div>
        </div>
        <div v-if="checking" class="p-4 bg-[rgba(15,15,35,0.4)] border border-[rgba(126,200,227,0.08)] rounded-xl">
          <p class="text-[#7ec8e3]/40 text-[13px]">Checking code...</p>
        </div>

        <div v-else-if="gameInfo" class="p-5 bg-[rgba(15,15,35,0.4)] border border-[rgba(126,200,227,0.15)] rounded-xl">
          <p class="text-[#7ec8e3]/40 text-[11px] uppercase tracking-wider mb-1">Game Found</p>
          <p class="text-[#e8e8f0] text-[18px] font-bold">{{ gameInfo.title }}</p>
          <p class="text-[#7ec8e3]/40 text-[13px] mt-1">System: {{ gameInfo.system }}</p>
        </div>
        <p v-if="errorMsg" class="text-[#e94560] text-[13px]">{{ errorMsg }}</p>
        <div class="flex gap-4 pt-2">
          <button
            @click="handleJoin"
            :disabled="loading || !code.trim()"
            class="flex-1 py-3 bg-gradient-to-br from-[#e94560] to-[#c23152] text-white text-[14px] font-semibold rounded-lg hover:-translate-y-0.5 hover:shadow-[0_6px_20px_rgba(233,69,96,0.4)] transition-all duration-300 cursor-pointer border-none disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:translate-y-0"
          >
            {{ loading ? 'Joining...' : 'Join Game' }}
          </button>
          <router-link
            to="/dashboard"
            class="px-6 py-3 text-[#e8e8f0]/60 text-[14px] font-medium no-underline border border-[rgba(126,200,227,0.15)] rounded-lg hover:border-[rgba(126,200,227,0.3)] hover:text-[#e8e8f0] transition-all duration-300 flex items-center"
          >
            Cancel
          </router-link>
        </div>
      </div>
    </div>
  </HomeLayout>
</template>
