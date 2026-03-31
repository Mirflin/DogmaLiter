<script setup>
import { ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useRouter } from 'vue-router'

const auth = useAuthStore()
const router = useRouter()

const isRegisterMode = ref(false)

const loginEmail = ref('')
const loginPassword = ref('')

const regUsername = ref('')
const regEmail = ref('')
const regPassword = ref('')
const regPasswordConfirm = ref('')

const successMessage = ref('')
const errorMessage = ref('')

function switchToRegister() {
  isRegisterMode.value = true
  errorMessage.value = ''
  successMessage.value = ''
}

function switchToLogin() {
  isRegisterMode.value = false
  errorMessage.value = ''
  successMessage.value = ''
}

async function handleLogin() {
  errorMessage.value = ''
  successMessage.value = ''
  try {
    await auth.login(loginEmail.value, loginPassword.value)
    router.push('/')
  } catch (err) {
    errorMessage.value = auth.error || 'Login failed'
  }
}

async function handleRegister() {
  errorMessage.value = ''
  successMessage.value = ''

  if (regPassword.value !== regPasswordConfirm.value) {
    errorMessage.value = 'Passwords do not match'
    return
  }
  if (regPassword.value.length < 8) {
    errorMessage.value = 'Password must be at least 8 characters'
    return
  }
  if (regUsername.value.length < 3) {
    errorMessage.value = 'Username must be at least 3 characters'
    return
  }

  try {
    await auth.register(regUsername.value, regEmail.value, regPassword.value)
    successMessage.value = 'Registration successful! Please check your email to verify your account.'
    regUsername.value = ''
    regEmail.value = ''
    regPassword.value = ''
    regPasswordConfirm.value = ''
    setTimeout(() => switchToLogin(), 3000)
  } catch (err) {
    errorMessage.value = auth.error || 'Registration failed'
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-linear-to-br from-[#0a0a1a] via-[#1a1a3e] to-[#0f2847] font-[Raleway] relative overflow-hidden">
    <div class="absolute inset-0 pointer-events-none">
      <span v-for="n in 20" :key="n" class="particle" />
    </div>

    <div
      class="auth-container relative w-[850px] max-w-[95vw] min-h-[620px] bg-[rgba(15,15,35,0.85)] backdrop-blur-[20px] rounded-2xl shadow-[0_25px_60px_rgba(0,0,0,0.5),0_0_40px_rgba(233,69,96,0.08)] flex overflow-hidden border border-[rgba(126,200,227,0.12)] z-1"
      :class="{ 'register-mode': isRegisterMode }"
    >
      <div class="form-side login-side w-1/2 py-12 px-10 flex items-center justify-center transition-all duration-700 z-2">
        <div class="w-full max-w-[320px]">
          <h2 class="font-[Cinzel] text-[28px] font-bold text-[#e8e8f0] mb-2 tracking-wide">Sign In</h2>
          <p class="text-[#7ec8e3]/60 text-sm mb-8">Welcome back, adventurer</p>

          <div v-if="errorMessage && !isRegisterMode" class="alert-error px-4 py-3 rounded-lg text-[13px] mb-5 border border-[rgba(233,69,96,0.3)] bg-[rgba(233,69,96,0.15)] text-[#ff8fa3] animate-shake">
            {{ errorMessage }}
          </div>

          <form @submit.prevent="handleLogin">
            <div class="mb-5 group">
              <label class="block mb-1.5 text-[#7ec8e3]/50 text-xs font-medium tracking-wider uppercase transition-colors duration-300 group-focus-within:text-[#e94560]">Email</label>
              <input
                v-model="loginEmail"
                type="email"
                placeholder="your@email.com"
                required
                autocomplete="email"
                class="w-full py-3 px-4 bg-[rgba(126,200,227,0.06)] border-[1.5px] border-[rgba(126,200,227,0.15)] rounded-lg text-[#e8e8f0] text-[15px] font-[inherit] outline-none transition-all duration-300 placeholder:text-[#7ec8e3]/25 focus:border-[#e94560] focus:bg-[rgba(233,69,96,0.04)] focus:ring-3 focus:ring-[rgba(233,69,96,0.1)]"
              />
            </div>

            <div class="mb-5 group">
              <label class="block mb-1.5 text-[#7ec8e3]/50 text-xs font-medium tracking-wider uppercase transition-colors duration-300 group-focus-within:text-[#e94560]">Password</label>
              <input
                v-model="loginPassword"
                type="password"
                placeholder="Enter password"
                required
                autocomplete="current-password"
                class="w-full py-3 px-4 bg-[rgba(126,200,227,0.06)] border-[1.5px] border-[rgba(126,200,227,0.15)] rounded-lg text-[#e8e8f0] text-[15px] font-[inherit] outline-none transition-all duration-300 placeholder:text-[#7ec8e3]/25 focus:border-[#e94560] focus:bg-[rgba(233,69,96,0.04)] focus:ring-3 focus:ring-[rgba(233,69,96,0.1)]"
              />
            </div>

            <button type="submit" class="btn-primary w-full py-3.5 mt-2 bg-linear-to-br from-[#e94560] to-[#c23152] text-white border-none rounded-xl text-[15px] font-semibold font-[inherit] cursor-pointer transition-all duration-300 relative overflow-hidden hover:-translate-y-0.5 hover:shadow-[0_8px_25px_rgba(233,69,96,0.4)] active:translate-y-0 disabled:opacity-60 disabled:cursor-not-allowed disabled:transform-none" :disabled="auth.loading">
              <span v-if="auth.loading" class="inline-block w-5 h-5 border-2 border-white/30 border-t-white rounded-full animate-spin" />
              <span v-else class="relative z-1">Sign In</span>
            </button>
          </form>
        </div>
      </div>

      <div class="form-side register-side absolute right-0 top-0 bottom-0 w-1/2 py-12 px-10 flex items-center justify-center transition-all duration-700 opacity-0 pointer-events-none z-1">
        <div class="w-full max-w-[320px]">
          <h2 class="font-[Cinzel] text-[28px] font-bold text-[#e8e8f0] mb-2 tracking-wide">Sign Up</h2>
          <p class="text-[#7ec8e3]/60 text-sm mb-8">Start your adventure</p>

          <div v-if="errorMessage && isRegisterMode" class="px-4 py-3 rounded-lg text-[13px] mb-5 border border-[rgba(233,69,96,0.3)] bg-[rgba(233,69,96,0.15)] text-[#ff8fa3] animate-shake">
            {{ errorMessage }}
          </div>
          <div v-if="successMessage" class="px-4 py-3 rounded-lg text-[13px] mb-5 border border-[rgba(46,204,113,0.3)] bg-[rgba(46,204,113,0.15)] text-[#6deca9] animate-shake">
            {{ successMessage }}
          </div>

          <form @submit.prevent="handleRegister">
            <div class="mb-5 group">
              <label class="block mb-1.5 text-[#7ec8e3]/50 text-xs font-medium tracking-wider uppercase transition-colors duration-300 group-focus-within:text-[#e94560]">Username</label>
              <input
                v-model="regUsername"
                type="text"
                placeholder="Choose a username"
                required
                autocomplete="username"
                class="w-full py-3 px-4 bg-[rgba(126,200,227,0.06)] border-[1.5px] border-[rgba(126,200,227,0.15)] rounded-lg text-[#e8e8f0] text-[15px] font-[inherit] outline-none transition-all duration-300 placeholder:text-[#7ec8e3]/25 focus:border-[#e94560] focus:bg-[rgba(233,69,96,0.04)] focus:ring-3 focus:ring-[rgba(233,69,96,0.1)]"
              />
            </div>

            <div class="mb-5 group">
              <label class="block mb-1.5 text-[#7ec8e3]/50 text-xs font-medium tracking-wider uppercase transition-colors duration-300 group-focus-within:text-[#e94560]">Email</label>
              <input
                v-model="regEmail"
                type="email"
                placeholder="your@email.com"
                required
                autocomplete="email"
                class="w-full py-3 px-4 bg-[rgba(126,200,227,0.06)] border-[1.5px] border-[rgba(126,200,227,0.15)] rounded-lg text-[#e8e8f0] text-[15px] font-[inherit] outline-none transition-all duration-300 placeholder:text-[#7ec8e3]/25 focus:border-[#e94560] focus:bg-[rgba(233,69,96,0.04)] focus:ring-3 focus:ring-[rgba(233,69,96,0.1)]"
              />
            </div>

            <div class="mb-5 group">
              <label class="block mb-1.5 text-[#7ec8e3]/50 text-xs font-medium tracking-wider uppercase transition-colors duration-300 group-focus-within:text-[#e94560]">Password</label>
              <input
                v-model="regPassword"
                type="password"
                placeholder="Min 8 characters"
                required
                autocomplete="new-password"
                class="w-full py-3 px-4 bg-[rgba(126,200,227,0.06)] border-[1.5px] border-[rgba(126,200,227,0.15)] rounded-lg text-[#e8e8f0] text-[15px] font-[inherit] outline-none transition-all duration-300 placeholder:text-[#7ec8e3]/25 focus:border-[#e94560] focus:bg-[rgba(233,69,96,0.04)] focus:ring-3 focus:ring-[rgba(233,69,96,0.1)]"
              />
            </div>

            <div class="mb-5 group">
              <label class="block mb-1.5 text-[#7ec8e3]/50 text-xs font-medium tracking-wider uppercase transition-colors duration-300 group-focus-within:text-[#e94560]">Confirm Password</label>
              <input
                v-model="regPasswordConfirm"
                type="password"
                placeholder="Repeat password"
                required
                autocomplete="new-password"
                class="w-full py-3 px-4 bg-[rgba(126,200,227,0.06)] border-[1.5px] border-[rgba(126,200,227,0.15)] rounded-lg text-[#e8e8f0] text-[15px] font-[inherit] outline-none transition-all duration-300 placeholder:text-[#7ec8e3]/25 focus:border-[#e94560] focus:bg-[rgba(233,69,96,0.04)] focus:ring-3 focus:ring-[rgba(233,69,96,0.1)]"
              />
            </div>

            <button type="submit" class="btn-primary w-full py-3.5 mt-2 bg-linear-to-br from-[#e94560] to-[#c23152] text-white border-none rounded-xl text-[15px] font-semibold font-[inherit] cursor-pointer transition-all duration-300 relative overflow-hidden hover:-translate-y-0.5 hover:shadow-[0_8px_25px_rgba(233,69,96,0.4)] active:translate-y-0 disabled:opacity-60 disabled:cursor-not-allowed disabled:transform-none" :disabled="auth.loading">
              <span v-if="auth.loading" class="inline-block w-5 h-5 border-2 border-white/30 border-t-white rounded-full animate-spin" />
              <span v-else class="relative z-1">Create Account</span>
            </button>
          </form>
        </div>
      </div>

      <div class="overlay-container absolute top-0 right-0 w-1/2 h-full overflow-hidden transition-transform duration-700 z-10">
        <div class="overlay relative w-[200%] h-full transition-transform duration-700">
          <div class="overlay-panel absolute top-0 left-0 w-1/2 h-full flex flex-col items-center justify-center text-center p-10 bg-linear-to-br from-[#1a3a5c] via-[#0f2847] to-[#162544]">
            <h2 class="font-[Cinzel] text-[26px] font-bold text-[#e8e8f0] mb-3 tracking-wide relative">Already with us?</h2>
            <p class="text-[#c8d2e6]/70 text-sm leading-relaxed mb-7 max-w-[260px] relative">Sign in to continue your adventures</p>
            <button class="py-3 px-10 bg-transparent text-white border-2 border-white/70 rounded-xl text-[15px] font-semibold font-[inherit] cursor-pointer transition-all duration-300 tracking-wide hover:bg-white/12 hover:border-white hover:-translate-y-0.5 hover:shadow-[0_5px_20px_rgba(255,255,255,0.1)]" @click="switchToLogin">Sign In</button>
          </div>
          <div class="overlay-panel absolute top-0 right-0 w-1/2 h-full flex flex-col items-center justify-center text-center p-10 bg-linear-to-br from-[#2d1b4e] via-[#1a1140] to-[#0f2847]">
            <h2 class="font-[Cinzel] text-[26px] font-bold text-[#e8e8f0] mb-3 tracking-wide relative">New here?</h2>
            <p class="text-[#c8d2e6]/70 text-sm leading-relaxed mb-7 max-w-[260px] relative">Create an account and dive into the world of DogmaLiter</p>
            <button class="py-3 px-10 bg-transparent text-white border-2 border-white/70 rounded-xl text-[15px] font-semibold font-[inherit] cursor-pointer transition-all duration-300 tracking-wide hover:bg-white/12 hover:border-white hover:-translate-y-0.5 hover:shadow-[0_5px_20px_rgba(255,255,255,0.1)]" @click="switchToRegister">Sign Up</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.auth-container .form-side,
.auth-container .overlay-container,
.auth-container .overlay {
  transition-timing-function: cubic-bezier(0.68, -0.15, 0.27, 1.15);
}

.overlay {
  transform: translateX(-50%);
}

.register-mode .login-side {
  transform: translateX(-20%);
  opacity: 0;
  pointer-events: none;
}

.register-mode .register-side {
  opacity: 1;
  pointer-events: all;
  z-index: 2;
}

.register-mode .overlay-container {
  transform: translateX(-100%);
}

.register-mode .overlay {
  transform: translateX(0);
}

.overlay-panel::before {
  content: '';
  position: absolute;
  inset: 0;
  background:
    radial-gradient(circle at 30% 20%, rgba(233, 69, 96, 0.15) 0%, transparent 50%),
    radial-gradient(circle at 70% 80%, rgba(126, 200, 227, 0.1) 0%, transparent 50%);
  pointer-events: none;
}

.btn-primary::before {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(135deg, #ff6b9d, #e94560);
  opacity: 0;
  transition: opacity 0.3s;
}
.btn-primary:hover::before {
  opacity: 1;
}

.particle {
  position: absolute;
  width: 4px;
  height: 4px;
  background: rgba(233, 69, 96, 0.5);
  border-radius: 50%;
  animation: float-particle 15s infinite linear;
}
.particle:nth-child(odd) { background: rgba(126, 200, 227, 0.4); }
.particle:nth-child(1)  { left: 5%;  top: 10%; animation-delay: 0s;    animation-duration: 12s; }
.particle:nth-child(2)  { left: 15%; top: 80%; animation-delay: 1s;    animation-duration: 18s; }
.particle:nth-child(3)  { left: 25%; top: 30%; animation-delay: 2s;    animation-duration: 14s; }
.particle:nth-child(4)  { left: 35%; top: 70%; animation-delay: 3s;    animation-duration: 16s; }
.particle:nth-child(5)  { left: 45%; top: 20%; animation-delay: 4s;    animation-duration: 13s; }
.particle:nth-child(6)  { left: 55%; top: 90%; animation-delay: 5s;    animation-duration: 17s; }
.particle:nth-child(7)  { left: 65%; top: 40%; animation-delay: 6s;    animation-duration: 15s; }
.particle:nth-child(8)  { left: 75%; top: 60%; animation-delay: 7s;    animation-duration: 11s; }
.particle:nth-child(9)  { left: 85%; top: 15%; animation-delay: 8s;    animation-duration: 19s; }
.particle:nth-child(10) { left: 95%; top: 50%; animation-delay: 9s;    animation-duration: 14s; }
.particle:nth-child(11) { left: 10%; top: 45%; animation-delay: 0.5s;  animation-duration: 13s; }
.particle:nth-child(12) { left: 20%; top: 65%; animation-delay: 1.5s;  animation-duration: 16s; }
.particle:nth-child(13) { left: 30%; top: 85%; animation-delay: 2.5s;  animation-duration: 12s; }
.particle:nth-child(14) { left: 40%; top: 35%; animation-delay: 3.5s;  animation-duration: 18s; }
.particle:nth-child(15) { left: 50%; top: 55%; animation-delay: 4.5s;  animation-duration: 15s; }
.particle:nth-child(16) { left: 60%; top: 25%; animation-delay: 5.5s;  animation-duration: 17s; }
.particle:nth-child(17) { left: 70%; top: 75%; animation-delay: 6.5s;  animation-duration: 11s; }
.particle:nth-child(18) { left: 80%; top: 5%;  animation-delay: 7.5s;  animation-duration: 14s; }
.particle:nth-child(19) { left: 90%; top: 95%; animation-delay: 8.5s;  animation-duration: 16s; }
.particle:nth-child(20) { left: 50%; top: 50%; animation-delay: 9.5s;  animation-duration: 13s; }

@keyframes float-particle {
  0%   { transform: translateY(0) translateX(0); opacity: 0; }
  10%  { opacity: 1; }
  90%  { opacity: 1; }
  100% { transform: translateY(-100vh) translateX(30px); opacity: 0; }
}

@keyframes shake {
  0%, 100% { transform: translateX(0); }
  25%      { transform: translateX(-5px); }
  75%      { transform: translateX(5px); }
}

.animate-shake {
  animation: shake 0.4s ease;
}

@media (max-width: 768px) {
  .auth-container {
    flex-direction: column;
    min-height: auto;
    width: 90vw;
    max-width: 420px;
  }
  .form-side { width: 100%; padding: 40px 30px; }
  .register-side { left: 0; right: 0; }
  .overlay-container { position: relative; width: 100%; height: auto; min-height: 200px; order: -1; }
  .register-mode .overlay-container { transform: translateY(100%); position: absolute; bottom: 0; height: auto; min-height: 200px; }
  .overlay { width: 100%; }
  .overlay-panel { width: 100%; padding: 30px; }
  .overlay-left { display: none; }
  .register-mode .overlay { transform: none; }
  .register-mode .overlay-left { display: flex; }
  .register-mode .overlay-right { display: none; }
}
</style>
