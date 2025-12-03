<script setup lang="ts">
import { onMounted } from 'vue'
import Sidebar from './Sidebar.vue'
import Header from './Header.vue'
import { useNodesStore, useAuthStore } from '@/stores'
import { wsConnection } from '@/api/websocket'

const nodesStore = useNodesStore()
const authStore = useAuthStore()

onMounted(async () => {
  // Initialize auth
  authStore.initAuth()

  // Fetch node types
  try {
    await nodesStore.fetchNodeTypes()
  } catch (e) {
    console.error('Failed to fetch node types:', e)
  }

  // Connect WebSocket
  wsConnection.connect()
})
</script>

<template>
  <div class="flex h-screen overflow-hidden bg-slate-50 dark:bg-slate-900">
    <!-- Sidebar -->
    <Sidebar />

    <!-- Main content area -->
    <div class="flex-1 flex flex-col overflow-hidden">
      <!-- Header -->
      <Header />

      <!-- Page content -->
      <main class="flex-1 overflow-auto">
        <RouterView />
      </main>
    </div>
  </div>
</template>
