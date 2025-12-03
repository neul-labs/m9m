<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'
import {
  HomeIcon,
  BoltIcon,
  ClockIcon,
  KeyIcon,
  Cog6ToothIcon,
  PlusIcon,
} from '@heroicons/vue/24/outline'

const route = useRoute()
const router = useRouter()

interface NavItem {
  name: string
  path: string
  icon: typeof HomeIcon
}

const navItems: NavItem[] = [
  { name: 'Dashboard', path: '/', icon: HomeIcon },
  { name: 'Workflows', path: '/workflows', icon: BoltIcon },
  { name: 'Executions', path: '/executions', icon: ClockIcon },
  { name: 'Credentials', path: '/credentials', icon: KeyIcon },
  { name: 'Settings', path: '/settings', icon: Cog6ToothIcon },
]

const isActive = (path: string) => {
  if (path === '/') {
    return route.path === '/'
  }
  return route.path.startsWith(path)
}

const createNewWorkflow = () => {
  router.push('/workflows/new')
}
</script>

<template>
  <aside class="w-64 bg-white dark:bg-slate-800 border-r border-slate-200 dark:border-slate-700 flex flex-col">
    <!-- Logo -->
    <div class="h-16 flex items-center px-6 border-b border-slate-200 dark:border-slate-700">
      <div class="flex items-center gap-3">
        <div class="w-8 h-8 bg-primary-600 rounded-lg flex items-center justify-center">
          <BoltIcon class="w-5 h-5 text-white" />
        </div>
        <span class="text-xl font-bold text-slate-900 dark:text-white">n8n-go</span>
      </div>
    </div>

    <!-- New Workflow Button -->
    <div class="p-4">
      <button
        @click="createNewWorkflow"
        class="w-full btn-primary flex items-center justify-center gap-2"
      >
        <PlusIcon class="w-5 h-5" />
        <span>New Workflow</span>
      </button>
    </div>

    <!-- Navigation -->
    <nav class="flex-1 px-3 py-2 space-y-1 overflow-y-auto">
      <RouterLink
        v-for="item in navItems"
        :key="item.path"
        :to="item.path"
        class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors"
        :class="[
          isActive(item.path)
            ? 'bg-primary-50 dark:bg-primary-900/20 text-primary-700 dark:text-primary-400'
            : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700/50 hover:text-slate-900 dark:hover:text-slate-200'
        ]"
      >
        <component :is="item.icon" class="w-5 h-5" />
        <span>{{ item.name }}</span>
      </RouterLink>
    </nav>

    <!-- Footer -->
    <div class="p-4 border-t border-slate-200 dark:border-slate-700">
      <div class="text-xs text-slate-500 dark:text-slate-400">
        <div class="font-medium">n8n-go</div>
        <div>v0.4.0</div>
      </div>
    </div>
  </aside>
</template>
