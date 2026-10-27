<script setup>
import HomeLayout from '@/layouts/HomeLayout.vue'
import { useAuthStore } from '@/stores/auth'
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { notify } from '@/notify'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()

const code = ref('')
const loading = ref(false)
const checking = ref(false)
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
    router.push('/games')
  } catch (err) {
    notify.error(err, 'Failed to join game')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <HomeLayout>
    <div class="max-w-[480px] mx-auto px-6 py-8">
      <h1 class="font-heading text-[28px] font-bold text-ink tracking-wide mb-2">Join a Game</h1>
      <p class="text-faint text-[14px] mb-8">Enter an invite code from your GM to join their game</p>
      <div class="space-y-6">
        <div>
          <label class="block text-dim text-[13px] font-medium mb-2">Invite Code</label>
          <div class="flex gap-3">
            <input
              v-model="code"
              type="text"
              placeholder="e.g. A1B2C3D4E5"
              maxlength="20"
              class="flex-1 px-4 py-3 bg-n2/60 border border-line rounded-lg text-ink text-[16px] font-mono tracking-[0.15em] text-center placeholder-faint outline-none focus:border-primary/40 transition-colors uppercase"
              @input="code = code.toUpperCase()"
              @blur="checkInvite"
            />
          </div>
        </div>
        <div v-if="checking" class="p-4 bg-n2/40 border border-line-soft rounded-xl">
          <p class="text-faint text-[13px]">Checking code...</p>
        </div>

        <div v-else-if="gameInfo" class="p-5 bg-n2/40 border border-line rounded-xl">
          <p class="text-faint text-[11px] uppercase tracking-wider mb-1">Game Found</p>
          <p class="text-ink text-[18px] font-bold">{{ gameInfo.title }}</p>
          <p class="text-faint text-[13px] mt-1">System: {{ gameInfo.system }}</p>
        </div>
        <div class="flex gap-4 pt-2">
          <button
            @click="handleJoin"
            :disabled="loading || !code.trim()"
            class="flex-1 py-3 bg-primary hover:bg-primary-hover text-white text-[14px] font-semibold rounded-lg transition-all duration-300 cursor-pointer border-none disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:translate-y-0"
          >
            {{ loading ? 'Joining...' : 'Join Game' }}
          </button>
          <router-link
            to="/dashboard"
            class="px-6 py-3 text-ink/60 text-[14px] font-medium no-underline border border-line rounded-lg hover:border-line-strong hover:text-ink transition-all duration-300 flex items-center"
          >
            Cancel
          </router-link>
        </div>
      </div>
    </div>
  </HomeLayout>
</template>
