<script setup lang="ts">
import { onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import {
  BoltIcon,
  PlayIcon,
  CheckCircleIcon,
  XCircleIcon,
  ClockIcon,
} from '@heroicons/vue/24/outline'
import { useWorkflowStore, useExecutionStore } from '@/stores'

const router = useRouter()
const workflowStore = useWorkflowStore()
const executionStore = useExecutionStore()

onMounted(async () => {
  await Promise.all([
    workflowStore.fetchWorkflows({ limit: 5 }),
    executionStore.fetchExecutions({ limit: 10 }),
  ])
})

const stats = computed(() => [
  {
    name: 'Total Workflows',
    value: workflowStore.total,
    icon: BoltIcon,
    color: 'bg-blue-500',
  },
  {
    name: 'Active Workflows',
    value: workflowStore.activeWorkflows.length,
    icon: PlayIcon,
    color: 'bg-green-500',
  },
  {
    name: 'Successful',
    value: executionStore.executionsByStatus.completed.length,
    icon: CheckCircleIcon,
    color: 'bg-emerald-500',
  },
  {
    name: 'Failed',
    value: executionStore.executionsByStatus.failed.length,
    icon: XCircleIcon,
    color: 'bg-red-500',
  },
])

const formatDate = (dateStr: string) => {
  const date = new Date(dateStr)
  return date.toLocaleString()
}

const formatDuration = (start: string, end?: string) => {
  if (!end) return 'Running...'
  const duration = new Date(end).getTime() - new Date(start).getTime()
  if (duration < 1000) return `${duration}ms`
  if (duration < 60000) return `${(duration / 1000).toFixed(1)}s`
  return `${Math.floor(duration / 60000)}m ${Math.floor((duration % 60000) / 1000)}s`
}

const getStatusColor = (status: string) => {
  switch (status) {
    case 'completed': return 'badge-success'
    case 'failed': return 'badge-error'
    case 'running': return 'badge-info'
    case 'cancelled': return 'badge-warning'
    default: return 'badge-info'
  }
}
</script>

<template>
  <div class="p-6 space-y-6">
    <!-- Stats Grid -->
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
      <div
        v-for="stat in stats"
        :key="stat.name"
        class="card p-6"
      >
        <div class="flex items-center gap-4">
          <div :class="[stat.color, 'p-3 rounded-lg']">
            <component :is="stat.icon" class="w-6 h-6 text-white" />
          </div>
          <div>
            <p class="text-2xl font-bold text-slate-900 dark:text-white">
              {{ stat.value }}
            </p>
            <p class="text-sm text-slate-500 dark:text-slate-400">
              {{ stat.name }}
            </p>
          </div>
        </div>
      </div>
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- Recent Workflows -->
      <div class="card">
        <div class="p-4 border-b border-slate-200 dark:border-slate-700 flex items-center justify-between">
          <h2 class="text-lg font-semibold text-slate-900 dark:text-white">Recent Workflows</h2>
          <RouterLink to="/workflows" class="text-sm text-primary-600 dark:text-primary-400 hover:underline">
            View all
          </RouterLink>
        </div>
        <div class="divide-y divide-slate-200 dark:divide-slate-700">
          <div
            v-for="workflow in workflowStore.workflows.slice(0, 5)"
            :key="workflow.id"
            @click="router.push(`/workflows/${workflow.id}`)"
            class="p-4 hover:bg-slate-50 dark:hover:bg-slate-700/50 cursor-pointer transition-colors"
          >
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-3">
                <div :class="[
                  workflow.active ? 'bg-green-100 dark:bg-green-900/30' : 'bg-slate-100 dark:bg-slate-700',
                  'p-2 rounded-lg'
                ]">
                  <BoltIcon :class="[
                    workflow.active ? 'text-green-600 dark:text-green-400' : 'text-slate-500 dark:text-slate-400',
                    'w-5 h-5'
                  ]" />
                </div>
                <div>
                  <p class="font-medium text-slate-900 dark:text-white">{{ workflow.name }}</p>
                  <p class="text-sm text-slate-500 dark:text-slate-400">
                    {{ workflow.nodes.length }} nodes
                  </p>
                </div>
              </div>
              <span :class="workflow.active ? 'badge-success' : 'badge-info'">
                {{ workflow.active ? 'Active' : 'Inactive' }}
              </span>
            </div>
          </div>
          <div v-if="workflowStore.workflows.length === 0" class="p-8 text-center">
            <BoltIcon class="w-12 h-12 mx-auto text-slate-300 dark:text-slate-600" />
            <p class="mt-2 text-slate-500 dark:text-slate-400">No workflows yet</p>
            <button
              @click="router.push('/workflows/new')"
              class="mt-4 btn-primary"
            >
              Create your first workflow
            </button>
          </div>
        </div>
      </div>

      <!-- Recent Executions -->
      <div class="card">
        <div class="p-4 border-b border-slate-200 dark:border-slate-700 flex items-center justify-between">
          <h2 class="text-lg font-semibold text-slate-900 dark:text-white">Recent Executions</h2>
          <RouterLink to="/executions" class="text-sm text-primary-600 dark:text-primary-400 hover:underline">
            View all
          </RouterLink>
        </div>
        <div class="divide-y divide-slate-200 dark:divide-slate-700">
          <div
            v-for="execution in executionStore.recentExecutions"
            :key="execution.id"
            @click="router.push(`/executions/${execution.id}`)"
            class="p-4 hover:bg-slate-50 dark:hover:bg-slate-700/50 cursor-pointer transition-colors"
          >
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-3">
                <div :class="[
                  execution.status === 'completed' ? 'bg-green-100 dark:bg-green-900/30' :
                  execution.status === 'failed' ? 'bg-red-100 dark:bg-red-900/30' :
                  'bg-blue-100 dark:bg-blue-900/30',
                  'p-2 rounded-lg'
                ]">
                  <ClockIcon :class="[
                    execution.status === 'completed' ? 'text-green-600 dark:text-green-400' :
                    execution.status === 'failed' ? 'text-red-600 dark:text-red-400' :
                    'text-blue-600 dark:text-blue-400',
                    'w-5 h-5'
                  ]" />
                </div>
                <div>
                  <p class="font-medium text-slate-900 dark:text-white text-sm">
                    {{ execution.id.slice(0, 8) }}...
                  </p>
                  <p class="text-xs text-slate-500 dark:text-slate-400">
                    {{ formatDate(execution.startedAt) }}
                  </p>
                </div>
              </div>
              <div class="text-right">
                <span :class="getStatusColor(execution.status)">
                  {{ execution.status }}
                </span>
                <p class="text-xs text-slate-500 dark:text-slate-400 mt-1">
                  {{ formatDuration(execution.startedAt, execution.finishedAt) }}
                </p>
              </div>
            </div>
          </div>
          <div v-if="executionStore.executions.length === 0" class="p-8 text-center">
            <ClockIcon class="w-12 h-12 mx-auto text-slate-300 dark:text-slate-600" />
            <p class="mt-2 text-slate-500 dark:text-slate-400">No executions yet</p>
            <p class="text-sm text-slate-400 dark:text-slate-500">
              Run a workflow to see executions here
            </p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
