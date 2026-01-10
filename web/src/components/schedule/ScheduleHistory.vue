<script setup lang="ts">
import { computed } from 'vue'
import type { ExecutionHistory, ExecutionRecord } from '@/types/schedule'
import {
  XMarkIcon,
  ClockIcon,
  CheckCircleIcon,
  XCircleIcon,
  ExclamationTriangleIcon,
  ArrowPathIcon,
  ChartBarIcon,
} from '@heroicons/vue/24/outline'

const props = defineProps<{
  history: ExecutionHistory | null
  loading?: boolean
}>()

const emit = defineEmits<{
  close: []
}>()

function formatDate(dateString: string): string {
  return new Date(dateString).toLocaleString()
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
  return `${(ms / 60000).toFixed(1)}m`
}

function getStatusIcon(status: ExecutionRecord['status']) {
  switch (status) {
    case 'success':
      return CheckCircleIcon
    case 'failed':
      return XCircleIcon
    case 'timeout':
      return ExclamationTriangleIcon
    case 'running':
      return ArrowPathIcon
    default:
      return ClockIcon
  }
}

function getStatusClass(status: ExecutionRecord['status']) {
  switch (status) {
    case 'success':
      return 'bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400'
    case 'failed':
      return 'bg-red-100 dark:bg-red-900/30 text-red-600 dark:text-red-400'
    case 'timeout':
      return 'bg-amber-100 dark:bg-amber-900/30 text-amber-600 dark:text-amber-400'
    case 'running':
      return 'bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400'
    default:
      return 'bg-slate-100 dark:bg-slate-700 text-slate-500 dark:text-slate-400'
  }
}

const successRate = computed(() => {
  if (!props.history) return 0
  const total = props.history.successCount + props.history.failureCount
  if (total === 0) return 0
  return Math.round((props.history.successCount / total) * 100)
})
</script>

<template>
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
    <div class="bg-white dark:bg-slate-800 rounded-xl shadow-2xl w-full max-w-4xl max-h-[90vh] overflow-hidden flex flex-col">
      <!-- Header -->
      <div class="px-6 py-4 border-b border-slate-200 dark:border-slate-700 flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="p-2 bg-violet-100 dark:bg-violet-900/30 rounded-lg">
            <ChartBarIcon class="w-5 h-5 text-violet-600 dark:text-violet-400" />
          </div>
          <h2 class="text-lg font-semibold text-slate-900 dark:text-white">
            Execution History
          </h2>
        </div>
        <button
          @click="emit('close')"
          class="p-2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-700"
        >
          <XMarkIcon class="w-5 h-5" />
        </button>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex-1 flex items-center justify-center p-8">
        <div class="animate-spin inline-block w-8 h-8 border-4 border-violet-500 border-t-transparent rounded-full"></div>
      </div>

      <!-- Content -->
      <div v-else-if="history" class="flex-1 overflow-y-auto">
        <!-- Stats Cards -->
        <div class="grid grid-cols-4 gap-4 p-6 border-b border-slate-200 dark:border-slate-700">
          <div class="p-4 bg-slate-50 dark:bg-slate-900 rounded-lg">
            <p class="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">Total Executions</p>
            <p class="text-2xl font-bold text-slate-900 dark:text-white mt-1">
              {{ history.successCount + history.failureCount }}
            </p>
          </div>
          <div class="p-4 bg-green-50 dark:bg-green-900/20 rounded-lg">
            <p class="text-xs font-medium text-green-600 dark:text-green-400 uppercase">Successful</p>
            <p class="text-2xl font-bold text-green-700 dark:text-green-300 mt-1">
              {{ history.successCount }}
            </p>
          </div>
          <div class="p-4 bg-red-50 dark:bg-red-900/20 rounded-lg">
            <p class="text-xs font-medium text-red-600 dark:text-red-400 uppercase">Failed</p>
            <p class="text-2xl font-bold text-red-700 dark:text-red-300 mt-1">
              {{ history.failureCount }}
            </p>
          </div>
          <div class="p-4 bg-violet-50 dark:bg-violet-900/20 rounded-lg">
            <p class="text-xs font-medium text-violet-600 dark:text-violet-400 uppercase">Success Rate</p>
            <p class="text-2xl font-bold text-violet-700 dark:text-violet-300 mt-1">
              {{ successRate }}%
            </p>
          </div>
        </div>

        <!-- Additional Stats -->
        <div class="grid grid-cols-2 gap-4 px-6 py-4 border-b border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/50">
          <div class="flex items-center gap-3">
            <ClockIcon class="w-5 h-5 text-slate-400" />
            <div>
              <p class="text-xs text-slate-500 dark:text-slate-400">Average Duration</p>
              <p class="text-sm font-medium text-slate-900 dark:text-white">
                {{ formatDuration(history.averageTime) }}
              </p>
            </div>
          </div>
          <div class="flex items-center gap-3">
            <CheckCircleIcon class="w-5 h-5 text-green-500" />
            <div>
              <p class="text-xs text-slate-500 dark:text-slate-400">Last Success</p>
              <p class="text-sm font-medium text-slate-900 dark:text-white">
                {{ history.lastSuccess ? formatDate(history.lastSuccess) : 'Never' }}
              </p>
            </div>
          </div>
        </div>

        <!-- Execution List -->
        <div class="p-6">
          <h3 class="text-sm font-medium text-slate-700 dark:text-slate-300 mb-4">
            Recent Executions
          </h3>

          <div v-if="history.executions.length === 0" class="text-center py-8 text-slate-500 dark:text-slate-400">
            <ClockIcon class="w-12 h-12 mx-auto mb-3 opacity-50" />
            <p>No executions yet</p>
          </div>

          <div v-else class="space-y-3">
            <div
              v-for="execution in history.executions"
              :key="execution.id"
              class="p-4 bg-slate-50 dark:bg-slate-900 rounded-lg border border-slate-200 dark:border-slate-700"
            >
              <div class="flex items-start justify-between">
                <div class="flex items-center gap-3">
                  <span
                    class="flex items-center justify-center w-8 h-8 rounded-full"
                    :class="getStatusClass(execution.status)"
                  >
                    <component :is="getStatusIcon(execution.status)" class="w-4 h-4" />
                  </span>
                  <div>
                    <p class="text-sm font-medium text-slate-900 dark:text-white capitalize">
                      {{ execution.status }}
                    </p>
                    <p class="text-xs text-slate-500 dark:text-slate-400">
                      {{ formatDate(execution.startTime) }}
                    </p>
                  </div>
                </div>
                <div class="text-right">
                  <p class="text-sm text-slate-900 dark:text-white">
                    {{ formatDuration(execution.duration) }}
                  </p>
                  <p v-if="execution.metrics" class="text-xs text-slate-500 dark:text-slate-400">
                    {{ execution.metrics.nodesExecuted }} nodes
                  </p>
                </div>
              </div>

              <!-- Error Message -->
              <div
                v-if="execution.error"
                class="mt-3 p-3 bg-red-50 dark:bg-red-900/20 rounded border border-red-200 dark:border-red-800"
              >
                <p class="text-sm text-red-700 dark:text-red-300 font-mono">
                  {{ execution.error }}
                </p>
              </div>

              <!-- Metrics -->
              <div
                v-if="execution.metrics"
                class="mt-3 flex items-center gap-4 text-xs text-slate-500 dark:text-slate-400"
              >
                <span>Data: {{ (execution.metrics.dataProcessed / 1024).toFixed(1) }}KB</span>
                <span>Memory: {{ (execution.metrics.memoryUsed / 1024 / 1024).toFixed(1) }}MB</span>
                <span>CPU: {{ execution.metrics.cpuTime }}ms</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Empty State -->
      <div v-else class="flex-1 flex items-center justify-center p-8 text-slate-500 dark:text-slate-400">
        <div class="text-center">
          <ChartBarIcon class="w-12 h-12 mx-auto mb-3 opacity-50" />
          <p>No history available</p>
        </div>
      </div>
    </div>
  </div>
</template>
