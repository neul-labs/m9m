<script setup lang="ts">
import { useRouter } from 'vue-router'
import { HomeIcon } from '@heroicons/vue/24/outline'

const router = useRouter()

const goHome = () => {
  router.push('/')
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-slate-50 dark:bg-slate-900 px-4">
    <div class="text-center">
      <h1 class="text-9xl font-bold text-slate-200 dark:text-slate-700">404</h1>
      <h2 class="mt-4 text-2xl font-semibold text-slate-900 dark:text-white">
        Page not found
      </h2>
      <p class="mt-2 text-slate-500 dark:text-slate-400">
        The page you're looking for doesn't exist or has been moved.
      </p>
      <button @click="goHome" class="mt-6 btn-primary inline-flex items-center gap-2">
        <HomeIcon class="w-5 h-5" />
        <span>Go to Dashboard</span>
      </button>
    </div>
  </div>
</template>
