<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { notify } from '@/notify'

const router = useRouter()
const auth = useAuthStore()

const status = ref('loading')
const message = ref('')
const resendEmail = ref('')

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
  try {
    const data = await auth.resendVerification(resendEmail.value)
    notify.success({
      title: 'Verification email sent',
      message: data?.message || 'A new verification email has been sent!',
    })
  } catch (err) {
    notify.error(err, 'Failed to resend verification email.')
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-linear-to-br from-n0 via-n3 to-n3 font-sans">
    <div class="w-[480px] max-w-[95vw] bg-n2/85 backdrop-blur-[20px] rounded-2xl shadow-[0_25px_60px_rgba(0,0,0,0.5),0_0_40px_rgba(88,101,221,0.08)] border border-line p-10 text-center">

      <div v-if="status === 'loading'" class="flex flex-col items-center gap-4">
        <span class="inline-block w-10 h-10 border-3 border-sky/30 border-t-primary-ink rounded-full animate-spin" />
        <p class="text-dim text-sm">Verifying your email...</p>
      </div>

      <div v-else-if="status === 'success'" class="flex flex-col items-center gap-5">
        <div class="w-16 h-16 rounded-full bg-success/15 border border-success/30 flex items-center justify-center">
          <svg class="w-8 h-8 text-success" fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" /></svg>
        </div>
        <h2 class="font-heading text-[24px] font-bold text-ink tracking-wide">Verified!</h2>
        <p class="text-success text-sm">{{ message }}</p>
        <router-link
          to="/auth"
          class="mt-3 inline-block py-3 px-8 bg-primary hover:bg-primary-hover text-white rounded-xl text-[15px] font-semibold no-underline transition-all duration-300"
        >
          Sign In
        </router-link>
      </div>

      <div v-else class="flex flex-col items-center gap-5">
        <div class="w-16 h-16 rounded-full bg-primary/15 border border-primary/30 flex items-center justify-center">
          <svg class="w-8 h-8 text-primary-ink" fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
        </div>
        <h2 class="font-heading text-[24px] font-bold text-ink tracking-wide">Verification Failed</h2>
        <p class="text-primary-ink text-sm">{{ message }}</p>

        <div class="w-full mt-4 pt-5 border-t border-line-soft">
          <p class="text-dim text-xs mb-3">Enter your email to receive a new verification link:</p>
          <form @submit.prevent="handleResend" class="flex gap-2">
            <input
              v-model="resendEmail"
              type="email"
              placeholder="your@email.com"
              required
              class="flex-1 py-2.5 px-4 bg-n3 border-[1.5px] border-line rounded-lg text-ink text-[14px] font-[inherit] outline-none transition-all duration-300 placeholder:text-faint focus:border-primary focus:bg-primary/4"
            />
            <button
              type="submit"
              :disabled="auth.loading"
              class="py-2.5 px-5 bg-primary hover:bg-primary-hover text-white border-none rounded-lg text-[13px] font-semibold cursor-pointer transition-all duration-300 disabled:opacity-60 disabled:cursor-not-allowed whitespace-nowrap"
            >
              <span v-if="auth.loading" class="inline-block w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
              <span v-else>Resend</span>
            </button>
          </form>
        </div>
      </div>

    </div>
  </div>
</template>