<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import {
  SunIcon,
  MoonIcon,
  BellIcon,
  UserCircleIcon,
  MagnifyingGlassIcon,
  SignalIcon,
  SignalSlashIcon,
} from '@heroicons/vue/24/outline'
import { Menu, MenuButton, MenuItems, MenuItem } from '@headlessui/vue'
import { useThemeStore, useAuthStore, useExecutionStore } from '@/stores'
import { wsConnection } from '@/api/websocket'

const route = useRoute()
const themeStore = useThemeStore()
const authStore = useAuthStore()
const executionStore = useExecutionStore()

const pageTitle = computed(() => {
  return (route.meta.title as string) || 'm9m'
})

const hasNotifications = computed(() => executionStore.hasRunningExecutions)

// WebSocket connection state
const wsConnected = ref(false)
const wsReconnecting = ref(false)
let connectionCheckInterval: ReturnType<typeof setInterval> | null = null

function updateConnectionState() {
  wsConnected.value = wsConnection.isConnected()
  wsReconnecting.value = !wsConnected.value && wsConnection.isConnected() === false
}

onMounted(() => {
  updateConnectionState()
  // Check connection state periodically
  connectionCheckInterval = setInterval(updateConnectionState, 1000)

  // Listen for connection events
  wsConnection.on('connected', () => {
    wsConnected.value = true
    wsReconnecting.value = false
  })

  wsConnection.on('error', () => {
    wsConnected.value = false
    wsReconnecting.value = true
  })
})

onUnmounted(() => {
  if (connectionCheckInterval) {
    clearInterval(connectionCheckInterval)
  }
})

// Connection status text
const connectionStatus = computed(() => {
  if (wsConnected.value) return 'Connected'
  if (wsReconnecting.value) return 'Reconnecting...'
  return 'Disconnected'
})

const connectionStatusColor = computed(() => {
  if (wsConnected.value) return 'text-green-500'
  if (wsReconnecting.value) return 'text-amber-500'
  return 'text-red-500'
})
</script>

<template>
  <header class="h-16 bg-white dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700 flex items-center justify-between px-6">
    <!-- Left: Page Title -->
    <div class="flex items-center gap-4">
      <h1 class="text-xl font-semibold text-slate-900 dark:text-white">
        {{ pageTitle }}
      </h1>
    </div>

    <!-- Center: Search (optional) -->
    <div class="flex-1 max-w-md mx-8">
      <div class="relative">
        <MagnifyingGlassIcon class="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-400" />
        <input
          type="text"
          placeholder="Search workflows..."
          class="w-full pl-10 pr-4 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-600 bg-slate-50 dark:bg-slate-700 text-slate-900 dark:text-slate-100 placeholder-slate-400 dark:placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
        />
      </div>
    </div>

    <!-- Right: Actions -->
    <div class="flex items-center gap-2">
      <!-- WebSocket Connection Status -->
      <div
        class="flex items-center gap-1.5 px-2 py-1 rounded-lg text-xs font-medium"
        :class="wsConnected ? 'bg-green-50 dark:bg-green-900/20' : 'bg-amber-50 dark:bg-amber-900/20'"
        :title="connectionStatus"
      >
        <span
          class="relative flex h-2 w-2"
        >
          <span
            v-if="wsReconnecting"
            class="animate-ping absolute inline-flex h-full w-full rounded-full opacity-75"
            :class="connectionStatusColor.replace('text-', 'bg-')"
          />
          <span
            class="relative inline-flex rounded-full h-2 w-2"
            :class="connectionStatusColor.replace('text-', 'bg-')"
          />
        </span>
        <component
          :is="wsConnected ? SignalIcon : SignalSlashIcon"
          class="w-4 h-4"
          :class="connectionStatusColor"
        />
      </div>

      <!-- Theme Toggle -->
      <button
        @click="themeStore.toggleTheme"
        class="p-2 rounded-lg text-slate-500 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
        :title="themeStore.isDark ? 'Switch to light mode' : 'Switch to dark mode'"
      >
        <SunIcon v-if="themeStore.isDark" class="w-5 h-5" />
        <MoonIcon v-else class="w-5 h-5" />
      </button>

      <!-- Notifications -->
      <button
        class="relative p-2 rounded-lg text-slate-500 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
        title="Notifications"
      >
        <BellIcon class="w-5 h-5" />
        <span
          v-if="hasNotifications"
          class="absolute top-1.5 right-1.5 w-2 h-2 bg-primary-500 rounded-full"
        />
      </button>

      <!-- User Menu -->
      <Menu as="div" class="relative">
        <MenuButton class="flex items-center gap-2 p-2 rounded-lg text-slate-500 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors">
          <UserCircleIcon class="w-6 h-6" />
        </MenuButton>

        <transition
          enter-active-class="transition duration-100 ease-out"
          enter-from-class="transform scale-95 opacity-0"
          enter-to-class="transform scale-100 opacity-100"
          leave-active-class="transition duration-75 ease-in"
          leave-from-class="transform scale-100 opacity-100"
          leave-to-class="transform scale-95 opacity-0"
        >
          <MenuItems class="absolute right-0 mt-2 w-48 origin-top-right bg-white dark:bg-slate-800 rounded-lg shadow-lg ring-1 ring-black ring-opacity-5 dark:ring-slate-700 focus:outline-none py-1 z-50">
            <MenuItem v-slot="{ active }">
              <RouterLink
                to="/settings"
                class="block px-4 py-2 text-sm text-slate-700 dark:text-slate-200"
                :class="{ 'bg-slate-100 dark:bg-slate-700': active }"
              >
                Settings
              </RouterLink>
            </MenuItem>
            <MenuItem v-if="authStore.isAuthenticated" v-slot="{ active }">
              <button
                @click="authStore.logout"
                class="block w-full text-left px-4 py-2 text-sm text-slate-700 dark:text-slate-200"
                :class="{ 'bg-slate-100 dark:bg-slate-700': active }"
              >
                Sign out
              </button>
            </MenuItem>
            <MenuItem v-else v-slot="{ active }">
              <RouterLink
                to="/login"
                class="block px-4 py-2 text-sm text-slate-700 dark:text-slate-200"
                :class="{ 'bg-slate-100 dark:bg-slate-700': active }"
              >
                Sign in
              </RouterLink>
            </MenuItem>
          </MenuItems>
        </transition>
      </Menu>
    </div>
  </header>
</template>
