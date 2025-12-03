<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { BoltIcon } from '@heroicons/vue/24/outline'
import { useAuthStore, useThemeStore } from '@/stores'

const router = useRouter()
const authStore = useAuthStore()
const themeStore = useThemeStore()

const isLogin = ref(true)
const loading = ref(false)
const error = ref('')

const form = ref({
  email: '',
  password: '',
  firstName: '',
  lastName: '',
})

const submit = async () => {
  loading.value = true
  error.value = ''

  try {
    if (isLogin.value) {
      await authStore.login({
        email: form.value.email,
        password: form.value.password,
      })
    } else {
      await authStore.register({
        email: form.value.email,
        password: form.value.password,
        firstName: form.value.firstName,
        lastName: form.value.lastName,
      })
    }
    router.push('/')
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Authentication failed'
  } finally {
    loading.value = false
  }
}

const toggleMode = () => {
  isLogin.value = !isLogin.value
  error.value = ''
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-slate-50 dark:bg-slate-900 px-4">
    <div class="w-full max-w-md">
      <!-- Logo -->
      <div class="text-center mb-8">
        <div class="inline-flex items-center gap-3">
          <div class="w-12 h-12 bg-primary-600 rounded-xl flex items-center justify-center">
            <BoltIcon class="w-7 h-7 text-white" />
          </div>
          <span class="text-3xl font-bold text-slate-900 dark:text-white">n8n-go</span>
        </div>
        <p class="mt-2 text-slate-500 dark:text-slate-400">
          High-performance workflow automation
        </p>
      </div>

      <!-- Card -->
      <div class="card p-6">
        <h2 class="text-xl font-semibold text-slate-900 dark:text-white text-center mb-6">
          {{ isLogin ? 'Sign in to your account' : 'Create an account' }}
        </h2>

        <!-- Error Message -->
        <div
          v-if="error"
          class="mb-4 p-3 rounded-lg bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 text-sm"
        >
          {{ error }}
        </div>

        <form @submit.prevent="submit" class="space-y-4">
          <!-- Name fields (register only) -->
          <div v-if="!isLogin" class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                First Name
              </label>
              <input
                v-model="form.firstName"
                type="text"
                class="input"
                placeholder="John"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                Last Name
              </label>
              <input
                v-model="form.lastName"
                type="text"
                class="input"
                placeholder="Doe"
              />
            </div>
          </div>

          <div>
            <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
              Email
            </label>
            <input
              v-model="form.email"
              type="email"
              class="input"
              placeholder="you@example.com"
              required
            />
          </div>

          <div>
            <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
              Password
            </label>
            <input
              v-model="form.password"
              type="password"
              class="input"
              placeholder="••••••••"
              required
            />
          </div>

          <button
            type="submit"
            :disabled="loading"
            class="w-full btn-primary"
          >
            {{ loading ? 'Please wait...' : (isLogin ? 'Sign in' : 'Create account') }}
          </button>
        </form>

        <div class="mt-6 text-center">
          <button
            @click="toggleMode"
            class="text-sm text-primary-600 dark:text-primary-400 hover:underline"
          >
            {{ isLogin ? "Don't have an account? Sign up" : 'Already have an account? Sign in' }}
          </button>
        </div>
      </div>

      <!-- Theme Toggle -->
      <div class="mt-6 text-center">
        <button
          @click="themeStore.toggleTheme"
          class="text-sm text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-200"
        >
          {{ themeStore.isDark ? 'Switch to light mode' : 'Switch to dark mode' }}
        </button>
      </div>
    </div>
  </div>
</template>
