<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  SunIcon,
  MoonIcon,
  ComputerDesktopIcon,
  UserCircleIcon,
  ShieldCheckIcon,
} from '@heroicons/vue/24/outline'
import { useThemeStore, useAuthStore } from '@/stores'
import type { Theme } from '@/stores/theme'

const themeStore = useThemeStore()
const authStore = useAuthStore()

const themes: { value: Theme; label: string; icon: typeof SunIcon }[] = [
  { value: 'light', label: 'Light', icon: SunIcon },
  { value: 'dark', label: 'Dark', icon: MoonIcon },
  { value: 'system', label: 'System', icon: ComputerDesktopIcon },
]

const profileForm = ref({
  firstName: '',
  lastName: '',
  email: '',
})

const passwordForm = ref({
  currentPassword: '',
  newPassword: '',
  confirmPassword: '',
})

onMounted(() => {
  if (authStore.user) {
    profileForm.value = {
      firstName: authStore.user.firstName || '',
      lastName: authStore.user.lastName || '',
      email: authStore.user.email,
    }
  }
})

const saveProfile = async () => {
  try {
    await authStore.updateProfile({
      firstName: profileForm.value.firstName,
      lastName: profileForm.value.lastName,
    })
    alert('Profile updated successfully')
  } catch (e) {
    console.error('Failed to update profile:', e)
    alert('Failed to update profile. Please try again.')
  }
}

const changePassword = async () => {
  if (passwordForm.value.newPassword !== passwordForm.value.confirmPassword) {
    alert('New passwords do not match')
    return
  }

  try {
    await authStore.changePassword(
      passwordForm.value.currentPassword,
      passwordForm.value.newPassword
    )
    passwordForm.value = { currentPassword: '', newPassword: '', confirmPassword: '' }
    alert('Password changed successfully')
  } catch (e) {
    console.error('Failed to change password:', e)
    alert('Failed to change password. Please check your current password.')
  }
}
</script>

<template>
  <div class="p-6 max-w-4xl mx-auto">
    <h1 class="text-2xl font-bold text-slate-900 dark:text-white mb-6">Settings</h1>

    <div class="space-y-6">
      <!-- Appearance -->
      <div class="card p-6">
        <div class="flex items-center gap-3 mb-4">
          <SunIcon class="w-5 h-5 text-slate-500" />
          <h2 class="text-lg font-semibold text-slate-900 dark:text-white">Appearance</h2>
        </div>

        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
              Theme
            </label>
            <div class="flex gap-3">
              <button
                v-for="theme in themes"
                :key="theme.value"
                @click="themeStore.setTheme(theme.value)"
                :class="[
                  'flex items-center gap-2 px-4 py-2 rounded-lg border-2 transition-colors',
                  themeStore.theme === theme.value
                    ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20 text-primary-700 dark:text-primary-400'
                    : 'border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:border-slate-300 dark:hover:border-slate-600'
                ]"
              >
                <component :is="theme.icon" class="w-5 h-5" />
                <span>{{ theme.label }}</span>
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Profile -->
      <div v-if="authStore.isAuthenticated" class="card p-6">
        <div class="flex items-center gap-3 mb-4">
          <UserCircleIcon class="w-5 h-5 text-slate-500" />
          <h2 class="text-lg font-semibold text-slate-900 dark:text-white">Profile</h2>
        </div>

        <form @submit.prevent="saveProfile" class="space-y-4">
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                First Name
              </label>
              <input
                v-model="profileForm.firstName"
                type="text"
                class="input"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                Last Name
              </label>
              <input
                v-model="profileForm.lastName"
                type="text"
                class="input"
              />
            </div>
          </div>

          <div>
            <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
              Email
            </label>
            <input
              v-model="profileForm.email"
              type="email"
              class="input"
              disabled
            />
          </div>

          <button type="submit" class="btn-primary">
            Save Profile
          </button>
        </form>
      </div>

      <!-- Security -->
      <div v-if="authStore.isAuthenticated" class="card p-6">
        <div class="flex items-center gap-3 mb-4">
          <ShieldCheckIcon class="w-5 h-5 text-slate-500" />
          <h2 class="text-lg font-semibold text-slate-900 dark:text-white">Security</h2>
        </div>

        <form @submit.prevent="changePassword" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
              Current Password
            </label>
            <input
              v-model="passwordForm.currentPassword"
              type="password"
              class="input"
              required
            />
          </div>

          <div>
            <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
              New Password
            </label>
            <input
              v-model="passwordForm.newPassword"
              type="password"
              class="input"
              required
            />
          </div>

          <div>
            <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
              Confirm New Password
            </label>
            <input
              v-model="passwordForm.confirmPassword"
              type="password"
              class="input"
              required
            />
          </div>

          <button type="submit" class="btn-primary">
            Change Password
          </button>
        </form>
      </div>

      <!-- About -->
      <div class="card p-6">
        <h2 class="text-lg font-semibold text-slate-900 dark:text-white mb-4">About</h2>
        <div class="space-y-2 text-sm text-slate-600 dark:text-slate-400">
          <p><span class="font-medium">n8n-go</span> v0.4.0</p>
          <p>High-performance workflow automation platform</p>
          <p class="mt-4">
            <a
              href="https://github.com/yourusername/n8n-go"
              target="_blank"
              class="text-primary-600 dark:text-primary-400 hover:underline"
            >
              GitHub Repository
            </a>
          </p>
        </div>
      </div>
    </div>
  </div>
</template>
