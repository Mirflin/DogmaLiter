import { ref } from 'vue'
import { defineStore } from 'pinia'
import api from '@/api'

export const useAuthStore = defineStore('auth', () => {
  const user = ref(null)
  const isAuthenticated = ref(false)
  const loading = ref(false)
  const error = ref(null)

  function setTokens(accessToken, refreshToken) {
    localStorage.setItem('access_token', accessToken)
    localStorage.setItem('refresh_token', refreshToken)
  }

  function clearTokens() {
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
  }

  async function register(username, email, password) {
    loading.value = true
    error.value = null
    try {
      const { data } = await api.post('/auth/register', { username, email, password })
      return data
    } catch (err) {
      error.value = err.response?.data?.error || 'Registration failed'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function login(email, password) {
    loading.value = true
    error.value = null
    try {
      const { data } = await api.post('/auth/login', { email, password })
      setTokens(data.access_token, data.refresh_token)
      user.value = data.user
      isAuthenticated.value = true
      return data
    } catch (err) {
      error.value = err.response?.data?.error || 'Login failed'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchUser() {
    try {
      const { data } = await api.get('/me')
      user.value = data
      isAuthenticated.value = true
    } catch {
      logout()
    }
  }

  function logout() {
    user.value = null
    isAuthenticated.value = false
    clearTokens()
  }

  async function forgotPassword(email) {
    loading.value = true
    error.value = null
    try {
      const { data } = await api.post('/auth/forgot-password', { email })
      return data
    } catch (err) {
      error.value = err.response?.data?.error || 'Something went wrong'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function resetPassword(token, password) {
    loading.value = true
    error.value = null
    try {
      const { data } = await api.post('/auth/reset-password', { token, password })
      return data
    } catch (err) {
      error.value = err.response?.data?.error || 'Password reset failed'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function verifyEmail(token) {
    loading.value = true
    error.value = null
    try {
      const { data } = await api.get('/auth/verify', { params: { token } })
      return data
    } catch (err) {
      error.value = err.response?.data?.error || 'Email verification failed'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function resendVerification(email) {
    loading.value = true
    error.value = null
    try {
      const { data } = await api.post('/auth/resend-verification', { email })
      return data
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to resend verification email'
      throw err
    } finally {
      loading.value = false
    }
  }


  const initPromise = localStorage.getItem('access_token') ? fetchUser() : Promise.resolve()

  async function waitForInit() {
    await initPromise
  }

  return {
    user,
    isAuthenticated,
    loading,
    error,
    register,
    login,
    logout,
    fetchUser,
    waitForInit,
    forgotPassword,
    resetPassword,
    verifyEmail,
    resendVerification,
  }
})
