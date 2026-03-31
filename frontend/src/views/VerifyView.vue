<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const auth = useAuthStore()

const status = ref('loading')
const message = ref('')
const resendEmail = ref('')
const resendMessage = ref('')

onMounted(async () => {
  const token = router.currentRoute.value.query.token
  if (!token) {
    status.value = 'error'
    message.value = 'Verification token is missing.'
    return
  }

  try {
    const data = await auth.verifyEmail(token)
    status.value = 'success'
    message.value = data?.message || 'Email successfully verified!'
  } catch (err) {
    status.value = 'error'
    message.value = auth.error || 'Invalid or expired verification link.'
  }
})

async function handleResend() {
  if (!resendEmail.value) return
  resendMessage.value = ''
  try {
    const data = await auth.resendVerification(resendEmail.value)
    resendMessage.value = data?.message || 'A new verification email has been sent!'
  } catch {
    resendMessage.value = auth.error || 'Failed to resend verification email.'
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-linear-to-br from-[#0a0a1a] via-[#1a1a3e] to-[#0f2847] font-[Raleway]">
    <div class="w-[480px] max-w-[95vw] bg-[rgba(15,15,35,0.85)] backdrop-blur-[20px] rounded-2xl shadow-[0_25px_60px_rgba(0,0,0,0.5),0_0_40px_rgba(233,69,96,0.08)] border border-[rgba(126,200,227,0.12)] p-10 text-center">

      <div v-if="status === 'loading'" class="flex flex-col items-center gap-4">
        <span class="inline-block w-10 h-10 border-3 border-[#7ec8e3]/30 border-t-[#e94560] rounded-full animate-spin" />
        <p class="text-[#7ec8e3]/60 text-sm">Verifying your email...</p>
      </div>

      <div v-else-if="status === 'success'" class="flex flex-col items-center gap-5">
        <div class="w-16 h-16 rounded-full bg-[rgba(46,204,113,0.15)] border border-[rgba(46,204,113,0.3)] flex items-center justify-center">
          <svg class="w-8 h-8 text-[#6deca9]" fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" /></svg>
        </div>
        <h2 class="font-[Cinzel] text-[24px] font-bold text-[#e8e8f0] tracking-wide">Verified!</h2>
        <p class="text-[#6deca9] text-sm">{{ message }}</p>
        <router-link
          to="/login"
          class="mt-3 inline-block py-3 px-8 bg-linear-to-br from-[#e94560] to-[#c23152] text-white rounded-xl text-[15px] font-semibold no-underline transition-all duration-300 hover:-translate-y-0.5 hover:shadow-[0_8px_25px_rgba(233,69,96,0.4)]"
        >
          Sign In
        </router-link>
      </div>

      <div v-else class="flex flex-col items-center gap-5">
        <div class="w-16 h-16 rounded-full bg-[rgba(233,69,96,0.15)] border border-[rgba(233,69,96,0.3)] flex items-center justify-center">
          <svg class="w-8 h-8 text-[#ff8fa3]" fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
        </div>
        <h2 class="font-[Cinzel] text-[24px] font-bold text-[#e8e8f0] tracking-wide">Verification Failed</h2>
        <p class="text-[#ff8fa3] text-sm">{{ message }}</p>

        <div class="w-full mt-4 pt-5 border-t border-[rgba(126,200,227,0.1)]">
          <p class="text-[#7ec8e3]/50 text-xs mb-3">Enter your email to receive a new verification link:</p>
          <form @submit.prevent="handleResend" class="flex gap-2">
            <input
              v-model="resendEmail"
              type="email"
              placeholder="your@email.com"
              required
              class="flex-1 py-2.5 px-4 bg-[rgba(126,200,227,0.06)] border-[1.5px] border-[rgba(126,200,227,0.15)] rounded-lg text-[#e8e8f0] text-[14px] font-[inherit] outline-none transition-all duration-300 placeholder:text-[#7ec8e3]/25 focus:border-[#e94560] focus:bg-[rgba(233,69,96,0.04)]"
            />
            <button
              type="submit"
              :disabled="auth.loading"
              class="py-2.5 px-5 bg-linear-to-br from-[#e94560] to-[#c23152] text-white border-none rounded-lg text-[13px] font-semibold cursor-pointer transition-all duration-300 hover:shadow-[0_4px_15px_rgba(233,69,96,0.4)] disabled:opacity-60 disabled:cursor-not-allowed whitespace-nowrap"
            >
              <span v-if="auth.loading" class="inline-block w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
              <span v-else>Resend</span>
            </button>
          </form>
          <p v-if="resendMessage" class="mt-3 text-[13px]" :class="resendMessage.includes('sent') ? 'text-[#6deca9]' : 'text-[#ff8fa3]'">
            {{ resendMessage }}
          </p>
        </div>
      </div>

    </div>
  </div>
</template>