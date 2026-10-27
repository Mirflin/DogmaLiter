<script setup>
import { ref, watch } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { notify } from '@/notify'
import { useRouter, useRoute } from 'vue-router'
import LandingSky from '@/components/landing/LandingSky.vue'
import logo from '@/assets/DLlogo.png'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()

const isRegisterMode = ref(route.query.mode === 'register')

const loginEmail = ref('')
const loginPassword = ref('')

const regUsername = ref('')
const regEmail = ref('')
const regPassword = ref('')
const regPasswordConfirm = ref('')
const agreeToTerms = ref(false)
const subscribeNewsletter = ref(false)

// Keep the address in step with the visible form, so a refresh or a shared link opens the same one.
watch(isRegisterMode, (register) => {
  router.replace({ query: { ...route.query, mode: register ? 'register' : undefined } })
})

function switchToRegister() {
  isRegisterMode.value = true
}

function switchToLogin() {
  isRegisterMode.value = false
}

async function handleLogin() {
  try {
    await auth.login(loginEmail.value, loginPassword.value)
    router.push('/')
  } catch (err) {
    notify.error(err, 'Login failed')
  }
}

async function handleRegister() {
  if (regPassword.value !== regPasswordConfirm.value) {
    notify.error('Passwords do not match')
    return
  }
  if (regPassword.value.length < 8) {
    notify.error('Password must be at least 8 characters')
    return
  }
  if (regUsername.value.length < 3) {
    notify.error('Username must be at least 3 characters')
    return
  }
  if (!agreeToTerms.value) {
    notify.error('You must agree to the Terms of Service and Privacy Policy')
    return
  }

  try {
    await auth.register(regUsername.value, regEmail.value, regPassword.value)
    notify.success({
      title: 'Registration successful',
      message: 'Please check your email to verify your account.',
    })
    regUsername.value = ''
    regEmail.value = ''
    regPassword.value = ''
    regPasswordConfirm.value = ''
    agreeToTerms.value = false
    subscribeNewsletter.value = false
    switchToLogin()
  } catch (err) {
    notify.error(err, 'Registration failed')
  }
}
</script>

<template>
  <div class="auth">
    <LandingSky />

    <header class="auth__top">
      <router-link to="/" class="auth__brand">
        <img :src="logo" alt="" class="h-9 w-9 object-contain" />
        <span>DogmaLiter</span>
      </router-link>
    </header>

    <main class="auth__main">
      <section class="auth__panel" :aria-label="isRegisterMode ? 'Create an account' : 'Sign in'">
        <img :src="logo" alt="DogmaLiter" class="auth__die" />
        <h1 class="auth__title">{{ isRegisterMode ? 'Join the table' : 'Welcome back' }}</h1>
        <p class="auth__lede">{{ isRegisterMode ? 'Create a free account for your campaigns and characters.' : 'Sign in to pick up where your party left off.' }}</p>

        <div class="auth__tabs" role="tablist">
          <span class="auth__thumb" :class="{ 'is-right': isRegisterMode }" aria-hidden="true" />
          <button type="button" role="tab" :aria-selected="!isRegisterMode" @click="switchToLogin">Sign in</button>
          <button type="button" role="tab" :aria-selected="isRegisterMode" @click="switchToRegister">Create account</button>
        </div>

        <Transition name="swap" mode="out-in">
          <form v-if="!isRegisterMode" key="login" class="auth__form" @submit.prevent="handleLogin">
            <label class="field">
              <span>Email</span>
              <input v-model="loginEmail" type="email" placeholder="you@example.com" required autocomplete="email" />
            </label>
            <label class="field">
              <span>Password</span>
              <input v-model="loginPassword" type="password" placeholder="Your password" required autocomplete="current-password" />
            </label>
            <button type="submit" class="auth__submit" :disabled="auth.loading">
              <span v-if="auth.loading" class="spinner" aria-hidden="true" />
              <span>{{ auth.loading ? 'Signing in…' : 'Sign in' }}</span>
            </button>
            <p class="auth__switch">New here? <button type="button" @click="switchToRegister">Create a free account</button></p>
          </form>

          <form v-else key="register" class="auth__form" @submit.prevent="handleRegister">
            <label class="field">
              <span>Username</span>
              <input v-model="regUsername" type="text" placeholder="At least 3 characters" required autocomplete="username" />
            </label>
            <label class="field">
              <span>Email</span>
              <input v-model="regEmail" type="email" placeholder="you@example.com" required autocomplete="email" />
            </label>
            <div class="auth__pair">
              <label class="field">
                <span>Password</span>
                <input v-model="regPassword" type="password" placeholder="Min 8 characters" required autocomplete="new-password" />
              </label>
              <label class="field">
                <span>Confirm password</span>
                <input v-model="regPasswordConfirm" type="password" placeholder="Repeat it" required autocomplete="new-password" />
              </label>
            </div>
            <label class="check">
              <input v-model="agreeToTerms" type="checkbox" />
              <span>I agree to the <router-link to="/terms" target="_blank">Terms of Service</router-link> and <router-link to="/privacy" target="_blank">Privacy Policy</router-link></span>
            </label>
            <label class="check">
              <input v-model="subscribeNewsletter" type="checkbox" />
              <span>Send me news and updates by email</span>
            </label>
            <button type="submit" class="auth__submit" :disabled="auth.loading">
              <span v-if="auth.loading" class="spinner" aria-hidden="true" />
              <span>{{ auth.loading ? 'Creating account…' : 'Create account' }}</span>
            </button>
            <p class="auth__switch">Already have an account? <button type="button" @click="switchToLogin">Sign in</button></p>
          </form>
        </Transition>
      </section>
    </main>
  </div>
</template>

<style scoped>
.auth {
  position: relative;
  min-height: 100svh;
  display: flex;
  flex-direction: column;
  /* Clips the fixed sky to this page. */
  clip-path: inset(0);
}

.auth__top,
.auth__main {
  position: relative;
  z-index: 1;
}

.auth__top {
  padding: 1.25rem 1.5rem 0;
}

.auth__brand {
  display: inline-flex;
  align-items: center;
  gap: 0.6rem;
  font-size: 19px;
  font-weight: 700;
  letter-spacing: -0.01em;
  color: var(--color-ink);
  text-decoration: none;
}

.auth__main {
  flex: 1;
  display: grid;
  place-items: center;
  padding: 2rem 1rem 4rem;
}

.auth__panel {
  width: 100%;
  max-width: 27rem;
  padding: 2.25rem 2rem 2rem;
  border-radius: 2rem;
  background: rgb(8 13 36 / 0.8);
  box-shadow: inset 0 0 0 1px rgb(74 90 150 / 0.35);
  text-align: center;
  animation: panel-in 0.7s cubic-bezier(0.16, 1, 0.3, 1) both;
}

@media (min-width: 640px) {
  .auth__panel {
    backdrop-filter: blur(10px);
    -webkit-backdrop-filter: blur(10px);
  }
}

.auth__die {
  width: 4.5rem;
  margin: -0.25rem auto 1rem;
  filter: drop-shadow(0 10px 18px rgb(2 4 14 / 0.6));
}

.auth__title {
  margin: 0;
  font-family: var(--font-num);
  font-size: 2.6rem;
  font-weight: 700;
  line-height: 1;
  text-transform: uppercase;
  color: var(--color-ink);
}

.auth__lede {
  margin: 0.75rem auto 0;
  max-width: 22rem;
  font-size: 16px;
  line-height: 1.5;
  color: var(--color-dim);
}

/* Segmented control */
.auth__tabs {
  position: relative;
  display: grid;
  grid-template-columns: 1fr 1fr;
  margin: 1.75rem 0 1.5rem;
  padding: 4px;
  border-radius: 0.9rem;
  background: var(--color-n0);
  box-shadow: var(--shadow-well);
}

.auth__thumb {
  position: absolute;
  top: 4px;
  bottom: 4px;
  left: 4px;
  width: calc(50% - 4px);
  border-radius: 0.65rem;
  background: var(--color-n4);
  transition: transform 0.32s cubic-bezier(0.16, 1, 0.3, 1);
}

.auth__thumb.is-right {
  transform: translateX(100%);
}

.auth__tabs button {
  position: relative;
  z-index: 1;
  min-height: 2.6rem;
  border: 0;
  border-radius: 0.65rem;
  background: none;
  font: inherit;
  font-size: 15px;
  font-weight: 600;
  color: var(--color-dim);
  cursor: pointer;
  transition: color 0.2s ease;
}

.auth__tabs button[aria-selected='true'] {
  color: var(--color-ink);
}

/* Forms */
.auth__form {
  display: grid;
  gap: 1rem;
  text-align: left;
}

.auth__pair {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
}

.field {
  display: grid;
  gap: 0.4rem;
}

.field span {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-ink-soft);
}

.field input {
  width: 100%;
  min-height: 3rem;
  padding: 0 0.95rem;
  border: 1px solid var(--color-n5);
  border-radius: 0.8rem;
  background: var(--color-n0);
  color: var(--color-ink);
  font: inherit;
  font-size: 16px;
  outline: none;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

.field input:hover {
  border-color: var(--color-n6);
}

.field input:focus {
  border-color: var(--color-primary-ink);
  box-shadow: 0 0 0 3px rgb(164 174 255 / 0.18);
}

.check {
  display: flex;
  align-items: flex-start;
  gap: 0.65rem;
  font-size: 14px;
  line-height: 1.45;
  color: var(--color-dim);
  cursor: pointer;
}

.check input {
  margin-top: 0.1rem;
}

.check a {
  color: var(--color-primary-ink);
  text-decoration: underline;
  text-decoration-color: rgb(164 174 255 / 0.35);
  text-underline-offset: 3px;
}

.check a:hover {
  text-decoration-color: currentColor;
}

.auth__submit {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.6rem;
  min-height: 3.25rem;
  margin-top: 0.5rem;
  border: 0;
  border-radius: 0.85rem;
  background: var(--color-primary);
  color: #fff;
  font: inherit;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  transition: background-color 0.2s ease, transform 0.16s ease;
}

.auth__submit:hover {
  background: var(--color-primary-hover);
}

.auth__submit:active {
  transform: scale(0.98);
}

.auth__submit:disabled {
  opacity: 0.65;
  cursor: progress;
}

.spinner {
  width: 1.1rem;
  height: 1.1rem;
  border: 2px solid rgb(255 255 255 / 0.3);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.auth__switch {
  margin: 0.25rem 0 0;
  text-align: center;
  font-size: 14px;
  color: var(--color-faint);
}

.auth__switch button {
  border: 0;
  padding: 0;
  background: none;
  font: inherit;
  font-weight: 600;
  color: var(--color-primary-ink);
  cursor: pointer;
  text-decoration: underline;
  text-decoration-color: rgb(164 174 255 / 0.35);
  text-underline-offset: 3px;
}

.auth__switch button:hover {
  text-decoration-color: currentColor;
}

.swap-enter-active,
.swap-leave-active {
  transition: opacity 0.2s ease, transform 0.24s cubic-bezier(0.16, 1, 0.3, 1);
}

.swap-enter-from {
  opacity: 0;
  transform: translateY(8px);
}

.swap-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}

@keyframes panel-in {
  from { opacity: 0; transform: translateY(16px); }
  to { opacity: 1; transform: none; }
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

@media (max-width: 520px) {
  .auth__panel {
    padding: 1.75rem 1.25rem 1.5rem;
    border-radius: 1.5rem;
  }

  .auth__title {
    font-size: 2.2rem;
  }

  .auth__pair {
    grid-template-columns: 1fr;
  }
}

@media (prefers-reduced-motion: reduce) {
  .auth__panel {
    animation: none;
  }
}
</style>
