<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import {
  CheckCircleIcon,
  XCircleIcon,
  ClockIcon,
  ArrowPathIcon,
  TrashIcon,
  FunnelIcon,
} from '@heroicons/vue/24/outline'
import { useExecutionStore, useWorkflowStore } from '@/stores'
import type { ExecutionStatus } from '@/types'

const router = useRouter()
const executionStore = useExecutionStore()
const workflowStore = useWorkflowStore()

const statusFilter = ref<ExecutionStatus | 'all'>('all')
const workflowFilter = ref<string>('')

onMounted(async () => {
  await Promise.all([
    executionStore.fetchExecutions(),
    workflowStore.fetchWorkflows(),
  ])
  executionStore.setupWebSocketHandlers()
})

const filteredExecutions = computed(() => {
  let executions = executionStore.executions

  if (statusFilter.value !== 'all') {
    executions = executions.filter((e) => e.status === statusFilter.value)
  }

  if (workflowFilter.value) {
    executions = executions.filter((e) => e.workflowId === workflowFilter.value)
  }

  return executions.sort(
    (a, b) => new Date(b.startedAt).getTime() - new Date(a.startedAt).getTime()
  )
})

const formatDate = (dateStr: string) => {
  return new Date(dateStr).toLocaleString()
}

const formatDuration = (start: string, end?: string) => {
  if (!end) return 'Running...'
  const duration = new Date(end).getTime() - new Date(start).getTime()
  if (duration < 1000) return `${duration}ms`
  if (duration < 60000) return `${(duration / 1000).toFixed(1)}s`
  return `${Math.floor(duration / 60000)}m ${Math.floor((duration % 60000) / 1000)}s`
}

const getStatusIcon = (status: string) => {
  switch (status) {
    case 'completed': return CheckCircleIcon
    case 'failed': return XCircleIcon
    case 'running': return ArrowPathIcon
    default: return ClockIcon
  }
}

const getStatusColor = (status: string) => {
  switch (status) {
    case 'completed': return 'text-green-500'
    case 'failed': return 'text-red-500'
    case 'running': return 'text-blue-500 animate-spin'
    case 'cancelled': return 'text-amber-500'
    default: return 'text-slate-500'
  }
}

const getWorkflowName = (workflowId: string) => {
  const workflow = workflowStore.workflows.find((w) => w.id === workflowId)
  return workflow?.name || 'Unknown Workflow'
}

const viewExecution = (id: string) => {
  router.push(`/executions/${id}`)
}

const retryExecution = async (id: string) => {
  try {
    await executionStore.retryExecution(id)
  } catch (e) {
    console.error('Failed to retry execution:', e)
  }
}

const deleteExecution = async (id: string) => {
  if (confirm('Are you sure you want to delete this execution?')) {
    try {
      await executionStore.deleteExecution(id)
    } catch (e) {
      console.error('Failed to delete execution:', e)
    }
  }
}
</script>

<template>
  <div class="p-6">
    <!-- Header -->
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold text-slate-900 dark:text-white">Execution History</h1>
        <p class="text-slate-500 dark:text-slate-400">
          View and manage workflow executions
        </p>
      </div>
    </div>

    <!-- Filters -->
    <div class="flex items-center gap-4 mb-6">
      <!-- Status Filter -->
      <div class="flex items-center gap-2">
        <FunnelIcon class="w-5 h-5 text-slate-400" />
        <select
          v-model="statusFilter"
          class="input py-1.5"
        >
          <option value="all">All Statuses</option>
          <option value="completed">Completed</option>
          <option value="failed">Failed</option>
          <option value="running">Running</option>
          <option value="cancelled">Cancelled</option>
        </select>
      </div>

      <!-- Workflow Filter -->
      <select
        v-model="workflowFilter"
        class="input py-1.5"
      >
        <option value="">All Workflows</option>
        <option
          v-for="workflow in workflowStore.workflows"
          :key="workflow.id"
          :value="workflow.id"
        >
          {{ workflow.name }}
        </option>
      </select>
    </div>

    <!-- Executions Table -->
    <div class="card overflow-hidden">
      <table class="w-full">
        <thead class="bg-slate-50 dark:bg-slate-700/50">
          <tr>
            <th class="px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
              Status
            </th>
            <th class="px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
              Workflow
            </th>
            <th class="px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
              Started
            </th>
            <th class="px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
              Duration
            </th>
            <th class="px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
              Mode
            </th>
            <th class="px-4 py-3 text-right text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
              Actions
            </th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-200 dark:divide-slate-700">
          <tr
            v-for="execution in filteredExecutions"
            :key="execution.id"
            @click="viewExecution(execution.id)"
            class="hover:bg-slate-50 dark:hover:bg-slate-700/30 cursor-pointer transition-colors"
          >
            <td class="px-4 py-4">
              <div class="flex items-center gap-2">
                <component
                  :is="getStatusIcon(execution.status)"
                  :class="['w-5 h-5', getStatusColor(execution.status)]"
                />
                <span class="text-sm font-medium text-slate-900 dark:text-white capitalize">
                  {{ execution.status }}
                </span>
              </div>
            </td>
            <td class="px-4 py-4">
              <span class="text-sm text-slate-900 dark:text-white">
                {{ getWorkflowName(execution.workflowId) }}
              </span>
            </td>
            <td class="px-4 py-4">
              <span class="text-sm text-slate-600 dark:text-slate-300">
                {{ formatDate(execution.startedAt) }}
              </span>
            </td>
            <td class="px-4 py-4">
              <span class="text-sm text-slate-600 dark:text-slate-300">
                {{ formatDuration(execution.startedAt, execution.finishedAt) }}
              </span>
            </td>
            <td class="px-4 py-4">
              <span class="badge-info capitalize">
                {{ execution.mode }}
              </span>
            </td>
            <td class="px-4 py-4 text-right">
              <div class="flex items-center justify-end gap-2" @click.stop>
                <button
                  v-if="execution.status === 'failed'"
                  @click="retryExecution(execution.id)"
                  class="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700"
                  title="Retry"
                >
                  <ArrowPathIcon class="w-4 h-4" />
                </button>
                <button
                  @click="deleteExecution(execution.id)"
                  class="p-1.5 rounded-lg text-slate-400 hover:text-red-600 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20"
                  title="Delete"
                >
                  <TrashIcon class="w-4 h-4" />
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>

      <!-- Empty State -->
      <div
        v-if="filteredExecutions.length === 0 && !executionStore.loading"
        class="text-center py-12"
      >
        <ClockIcon class="w-12 h-12 mx-auto text-slate-300 dark:text-slate-600" />
        <h3 class="mt-4 text-lg font-medium text-slate-900 dark:text-white">
          No executions found
        </h3>
        <p class="mt-2 text-slate-500 dark:text-slate-400">
          Run a workflow to see executions here
        </p>
      </div>

      <!-- Loading State -->
      <div v-if="executionStore.loading" class="text-center py-12">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600 mx-auto" />
        <p class="mt-4 text-slate-500 dark:text-slate-400">Loading executions...</p>
      </div>
    </div>
  </div>
</template>
