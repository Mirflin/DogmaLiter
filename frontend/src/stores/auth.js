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
      if (data.access_token && data.refresh_token) {
        setTokens(data.access_token, data.refresh_token)
      }
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

  async function updateUsername(username) {
    loading.value = true
    error.value = null
    try {
      const { data } = await api.put('/me/username', { username })
      user.value = { ...user.value, username }
      return data
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to update username'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function changePassword(currentPassword, newPassword) {
    loading.value = true
    error.value = null
    try {
      const { data } = await api.put('/me/password', {
        current_password: currentPassword,
        new_password: newPassword,
      })
      return data
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to change password'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function uploadAvatar(file) {
    loading.value = true
    error.value = null
    try {
      const formData = new FormData()
      formData.append('avatar', file)
      const { data } = await api.post('/me/avatar', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })
      user.value = { ...user.value, avatar_id: data.avatar_id }
      return data
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to upload avatar'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function deleteAvatar() {
    loading.value = true
    error.value = null
    try {
      const { data } = await api.delete('/me/avatar')
      user.value = { ...user.value, avatar_id: null }
      return data
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to remove avatar'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchProfile() {
    const { data } = await api.get('/me')
    user.value = data
    return data
  }

  async function fetchStorageUsage() {
    const { data } = await api.get('/me/storage')
    return data
  }

  async function fetchMyGames() {
    const { data } = await api.get('/games')
    return data
  }

  async function createGame(gameData) {
    const { data } = await api.post('/games', gameData)
    return data
  }

  async function joinGameByCode(code) {
    const { data } = await api.post('/games/join', { code })
    return data
  }

  async function getInviteInfo(code) {
    const { data } = await api.get(`/games/invite/${code}`)
    return data
  }

  async function leaveGame(gameID) {
    const { data } = await api.post(`/games/${gameID}/leave`)
    return data
  }

  async function deleteGame(gameID) {
    const { data } = await api.delete(`/games/${gameID}`)
    return data
  }

  async function uploadCoverImage(gameID, file) {
    const formData = new FormData()
    formData.append('cover', file)
    const { data } = await api.post(`/games/${gameID}/cover`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    return data
  }

  async function getGame(gameID) {
    const { data } = await api.get(`/games/${gameID}`)
    return data
  }

  async function updateGame(gameID, updates) {
    const { data } = await api.put(`/games/${gameID}`, updates)
    return data
  }

  async function regenerateInviteCode(gameID) {
    const { data } = await api.post(`/games/${gameID}/regenerate-code`)
    return data
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
    updateUsername,
    changePassword,
    uploadAvatar,
    deleteAvatar,
    fetchProfile,
    fetchStorageUsage,
    fetchMyGames,
    createGame,
    joinGameByCode,
    getInviteInfo,
    leaveGame,
    deleteGame,
    uploadCoverImage,
    getGame,
    updateGame,
    regenerateInviteCode,
  }
})
