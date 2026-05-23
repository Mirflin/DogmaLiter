<script setup>
import { ref, computed } from 'vue'
import HomeLayout from '@/layouts/HomeLayout.vue'
import { useAuthStore } from '@/stores/auth'
import { API_URL } from '@/api'
import { notify } from '@/notify'

const auth = useAuthStore()

const newUsername = ref(auth.user?.username || '')
const usernameUnchanged = computed(() => newUsername.value === auth.user?.username)

async function handleUpdateUsername() {
  try {
    await auth.updateUsername(newUsername.value)
    notify.success({
      title: 'Username updated',
      message: 'Your profile name was updated successfully.',
    })
  } catch (err) {
    notify.error(err, 'Failed to update username')
  }
}

const currentPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')

async function handleChangePassword() {
  if (newPassword.value !== confirmPassword.value) {
    notify.error('Passwords do not match')
    return
  }
  if (newPassword.value.length < 8) {
    notify.error('Password must be at least 8 characters')
    return
  }
  try {
    await auth.changePassword(currentPassword.value, newPassword.value)
    currentPassword.value = ''
    newPassword.value = ''
    confirmPassword.value = ''
    notify.success({
      title: 'Password changed',
      message: 'Your password has been updated.',
    })
  } catch (err) {
    notify.error(err, 'Failed to change password')
  }
}

const avatarPreview = ref(null)
const avatarFile = ref(null)

const avatarUrl = computed(() => {
  if (avatarPreview.value) return avatarPreview.value
  if (auth.user?.avatar_id) return `${API_URL}/api/uploads/${auth.user.avatar_id}`
  return null
})

function onAvatarSelect(e) {
  const file = e.target.files[0]
  if (!file) return
  if (!['image/jpeg', 'image/png', 'image/webp'].includes(file.type)) {
    notify.error({
      title: 'Invalid avatar format',
      message: 'Only JPEG, PNG, and WebP images are allowed.',
    })
    return
  }
  if (file.size > 2 * 1024 * 1024) {
    notify.error({
      title: 'Avatar too large',
      message: 'The maximum avatar size is 2MB.',
    })
    return
  }
  avatarFile.value = file
  avatarPreview.value = URL.createObjectURL(file)
}

async function handleUploadAvatar() {
  if (!avatarFile.value) return
  try {
    await auth.uploadAvatar(avatarFile.value)
    avatarFile.value = null
    avatarPreview.value = null
    notify.success({
      title: 'Avatar updated',
      message: 'Your new avatar is now visible on your profile.',
    })
  } catch (err) {
    notify.error(err, 'Failed to upload avatar')
  }
}

async function handleDeleteAvatar() {
  try {
    await auth.deleteAvatar()
    avatarFile.value = null
    avatarPreview.value = null
    notify.success({
      title: 'Avatar removed',
      message: 'Your profile avatar has been removed.',
    })
  } catch (err) {
    notify.error(err, 'Failed to remove avatar')
  }
}

function triggerFileInput() {
  document.getElementById('avatar-input').click()
}
</script>

<template>
  <HomeLayout>
    <div class="max-w-[700px] mx-auto px-6 py-10">
      <h1 class="font-[Cinzel] text-[28px] font-bold text-[#e8e8f0] tracking-wide mb-8">Settings</h1>
      <div class="bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.1)] rounded-xl p-6 mb-6">
        <h2 class="font-[Cinzel] text-[18px] font-bold text-[#e8e8f0] tracking-wide mb-5">Avatar</h2>

        <div class="flex items-center gap-6">
          <div
            class="w-24 h-24 rounded-full flex-shrink-0 flex items-center justify-center overflow-hidden cursor-pointer border-2 border-[rgba(126,200,227,0.15)] hover:border-[#e94560] transition-all duration-300"
            @click="triggerFileInput"
          >
            <img v-if="avatarUrl" :src="avatarUrl" alt="Avatar" class="w-full h-full object-cover" />
            <span v-else class="text-[#e94560] text-[32px] font-bold font-[Cinzel] bg-[rgba(233,69,96,0.15)] w-full h-full flex items-center justify-center">
              {{ auth.user?.username?.charAt(0)?.toUpperCase() }}
            </span>
          </div>
          <div class="flex flex-col gap-3">
            <input id="avatar-input" type="file" accept="image/jpeg,image/png,image/webp" class="hidden" @change="onAvatarSelect" />
            <div class="flex gap-3">
              <button
                v-if="avatarFile"
                @click="handleUploadAvatar"
                :disabled="auth.loading"
                class="px-5 py-2.5 bg-linear-to-br from-[#e94560] to-[#c23152] text-white text-[13px] font-semibold rounded-lg border-none cursor-pointer hover:-translate-y-0.5 hover:shadow-[0_6px_20px_rgba(233,69,96,0.4)] transition-all duration-300 disabled:opacity-60 disabled:cursor-not-allowed disabled:transform-none"
              >
                Save Avatar
              </button>
              <button
                v-else
                @click="triggerFileInput"
                class="px-5 py-2.5 bg-transparent text-[#e8e8f0]/70 text-[13px] font-semibold rounded-lg border border-[rgba(126,200,227,0.25)] cursor-pointer hover:border-[#e94560] hover:text-[#e94560] transition-all duration-300"
              >
                Upload Photo
              </button>
              <button
                v-if="auth.user?.avatar_id"
                @click="handleDeleteAvatar"
                :disabled="auth.loading"
                class="px-5 py-2.5 bg-transparent text-[#ff8fa3] text-[13px] font-semibold rounded-lg border border-[rgba(233,69,96,0.25)] cursor-pointer hover:bg-[rgba(233,69,96,0.1)] transition-all duration-300 disabled:opacity-60 disabled:cursor-not-allowed"
              >
                Remove
              </button>
            </div>
            <p class="text-[#7ec8e3]/40 text-[12px]">JPEG, PNG or WebP. Max 2MB.</p>
          </div>
        </div>
      </div>
      <div class="bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.1)] rounded-xl p-6 mb-6">
        <h2 class="font-[Cinzel] text-[18px] font-bold text-[#e8e8f0] tracking-wide mb-5">Username</h2>

        <form @submit.prevent="handleUpdateUsername" class="flex gap-3 items-end">
          <div class="flex-1 group">
            <label class="block mb-1.5 text-[#7ec8e3]/50 text-xs font-medium tracking-wider uppercase transition-colors duration-300 group-focus-within:text-[#e94560]">Username</label>
            <input
              v-model="newUsername"
              type="text"
              required
              minlength="3"
              maxlength="50"
              class="w-full py-3 px-4 bg-[rgba(126,200,227,0.06)] border-[1.5px] border-[rgba(126,200,227,0.15)] rounded-lg text-[#e8e8f0] text-[15px] font-[inherit] outline-none transition-all duration-300 placeholder:text-[#7ec8e3]/25 focus:border-[#e94560] focus:bg-[rgba(233,69,96,0.04)] focus:ring-3 focus:ring-[rgba(233,69,96,0.1)]"
            />
          </div>
          <button
            type="submit"
            :disabled="auth.loading || usernameUnchanged"
            class="px-6 py-3 bg-linear-to-br from-[#e94560] to-[#c23152] text-white text-[14px] font-semibold rounded-lg border-none cursor-pointer hover:-translate-y-0.5 hover:shadow-[0_6px_20px_rgba(233,69,96,0.4)] transition-all duration-300 disabled:opacity-60 disabled:cursor-not-allowed disabled:transform-none whitespace-nowrap"
          >
            Save
          </button>
        </form>
      </div>
      <div class="bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.1)] rounded-xl p-6">
        <h2 class="font-[Cinzel] text-[18px] font-bold text-[#e8e8f0] tracking-wide mb-5">Change Password</h2>

        <form @submit.prevent="handleChangePassword" class="space-y-4">
          <div class="group">
            <label class="block mb-1.5 text-[#7ec8e3]/50 text-xs font-medium tracking-wider uppercase transition-colors duration-300 group-focus-within:text-[#e94560]">Current Password</label>
            <input
              v-model="currentPassword"
              type="password"
              required
              autocomplete="current-password"
              placeholder="Enter current password"
              class="w-full py-3 px-4 bg-[rgba(126,200,227,0.06)] border-[1.5px] border-[rgba(126,200,227,0.15)] rounded-lg text-[#e8e8f0] text-[15px] font-[inherit] outline-none transition-all duration-300 placeholder:text-[#7ec8e3]/25 focus:border-[#e94560] focus:bg-[rgba(233,69,96,0.04)] focus:ring-3 focus:ring-[rgba(233,69,96,0.1)]"
            />
          </div>
          <div class="group">
            <label class="block mb-1.5 text-[#7ec8e3]/50 text-xs font-medium tracking-wider uppercase transition-colors duration-300 group-focus-within:text-[#e94560]">New Password</label>
            <input
              v-model="newPassword"
              type="password"
              required
              autocomplete="new-password"
              placeholder="Min 8 characters"
              class="w-full py-3 px-4 bg-[rgba(126,200,227,0.06)] border-[1.5px] border-[rgba(126,200,227,0.15)] rounded-lg text-[#e8e8f0] text-[15px] font-[inherit] outline-none transition-all duration-300 placeholder:text-[#7ec8e3]/25 focus:border-[#e94560] focus:bg-[rgba(233,69,96,0.04)] focus:ring-3 focus:ring-[rgba(233,69,96,0.1)]"
            />
          </div>
          <div class="group">
            <label class="block mb-1.5 text-[#7ec8e3]/50 text-xs font-medium tracking-wider uppercase transition-colors duration-300 group-focus-within:text-[#e94560]">Confirm New Password</label>
            <input
              v-model="confirmPassword"
              type="password"
              required
              autocomplete="new-password"
              placeholder="Repeat new password"
              class="w-full py-3 px-4 bg-[rgba(126,200,227,0.06)] border-[1.5px] border-[rgba(126,200,227,0.15)] rounded-lg text-[#e8e8f0] text-[15px] font-[inherit] outline-none transition-all duration-300 placeholder:text-[#7ec8e3]/25 focus:border-[#e94560] focus:bg-[rgba(233,69,96,0.04)] focus:ring-3 focus:ring-[rgba(233,69,96,0.1)]"
            />
          </div>
          <button
            type="submit"
            :disabled="auth.loading"
            class="px-6 py-3 mt-2 bg-linear-to-br from-[#e94560] to-[#c23152] text-white text-[14px] font-semibold rounded-lg border-none cursor-pointer hover:-translate-y-0.5 hover:shadow-[0_6px_20px_rgba(233,69,96,0.4)] transition-all duration-300 disabled:opacity-60 disabled:cursor-not-allowed disabled:transform-none"
          >
            Change Password
          </button>
        </form>
      </div>
    </div>
  </HomeLayout>
</template>
