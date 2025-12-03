<script setup lang="ts">
import { onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ArrowLeftIcon,
  CheckCircleIcon,
  XCircleIcon,
  ClockIcon,
  ArrowPathIcon,
} from '@heroicons/vue/24/outline'
import { useExecutionStore, useWorkflowStore } from '@/stores'

const route = useRoute()
const router = useRouter()
const executionStore = useExecutionStore()
const workflowStore = useWorkflowStore()

const executionId = computed(() => route.params.id as string)

onMounted(async () => {
  await executionStore.fetchExecution(executionId.value)
  if (executionStore.currentExecution?.workflowId) {
    await workflowStore.fetchWorkflow(executionStore.currentExecution.workflowId)
  }
})

const execution = computed(() => executionStore.currentExecution)
const workflow = computed(() => workflowStore.currentWorkflow)

const formatDate = (dateStr: string) => {
  return new Date(dateStr).toLocaleString()
}

const formatDuration = (start: string, end?: string) => {
  if (!end) return 'Running...'
  const duration = new Date(end).getTime() - new Date(start).getTime()
  if (duration < 1000) return `${duration}ms`
  if (duration < 60000) return `${(duration / 1000).toFixed(2)}s`
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
    case 'completed': return 'text-green-500 bg-green-100 dark:bg-green-900/30'
    case 'failed': return 'text-red-500 bg-red-100 dark:bg-red-900/30'
    case 'running': return 'text-blue-500 bg-blue-100 dark:bg-blue-900/30'
    default: return 'text-slate-500 bg-slate-100 dark:bg-slate-700'
  }
}

const retryExecution = async () => {
  if (execution.value) {
    try {
      const newExecution = await executionStore.retryExecution(execution.value.id)
      router.push(`/executions/${newExecution.id}`)
    } catch (e) {
      console.error('Failed to retry execution:', e)
    }
  }
}

const goBack = () => {
  router.push('/executions')
}

const openWorkflow = () => {
  if (execution.value?.workflowId) {
    router.push(`/workflows/${execution.value.workflowId}`)
  }
}
</script>

<template>
  <div class="p-6">
    <!-- Header -->
    <div class="flex items-center gap-4 mb-6">
      <button
        @click="goBack"
        class="p-2 rounded-lg text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-700"
      >
        <ArrowLeftIcon class="w-5 h-5" />
      </button>
      <div>
        <h1 class="text-2xl font-bold text-slate-900 dark:text-white">
          Execution {{ execution?.id?.slice(0, 8) }}...
        </h1>
        <p
          v-if="workflow"
          @click="openWorkflow"
          class="text-primary-600 dark:text-primary-400 hover:underline cursor-pointer"
        >
          {{ workflow.name }}
        </p>
      </div>
    </div>

    <div v-if="execution" class="space-y-6">
      <!-- Status Card -->
      <div class="card p-6">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-4">
            <div :class="['p-3 rounded-full', getStatusColor(execution.status)]">
              <component
                :is="getStatusIcon(execution.status)"
                :class="[
                  'w-8 h-8',
                  execution.status === 'running' ? 'animate-spin' : ''
                ]"
              />
            </div>
            <div>
              <h2 class="text-xl font-semibold text-slate-900 dark:text-white capitalize">
                {{ execution.status }}
              </h2>
              <p class="text-slate-500 dark:text-slate-400">
                {{ execution.mode }} execution
              </p>
            </div>
          </div>

          <button
            v-if="execution.status === 'failed'"
            @click="retryExecution"
            class="btn-primary flex items-center gap-2"
          >
            <ArrowPathIcon class="w-4 h-4" />
            Retry
          </button>
        </div>

        <!-- Timing Info -->
        <div class="grid grid-cols-3 gap-6 mt-6 pt-6 border-t border-slate-200 dark:border-slate-700">
          <div>
            <p class="text-sm text-slate-500 dark:text-slate-400">Started</p>
            <p class="font-medium text-slate-900 dark:text-white">
              {{ formatDate(execution.startedAt) }}
            </p>
          </div>
          <div>
            <p class="text-sm text-slate-500 dark:text-slate-400">Finished</p>
            <p class="font-medium text-slate-900 dark:text-white">
              {{ execution.finishedAt ? formatDate(execution.finishedAt) : 'Still running...' }}
            </p>
          </div>
          <div>
            <p class="text-sm text-slate-500 dark:text-slate-400">Duration</p>
            <p class="font-medium text-slate-900 dark:text-white">
              {{ formatDuration(execution.startedAt, execution.finishedAt) }}
            </p>
          </div>
        </div>
      </div>

      <!-- Error Message -->
      <div v-if="execution.error" class="card p-6 border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-900/20">
        <h3 class="font-semibold text-red-700 dark:text-red-400 mb-2">Error</h3>
        <pre class="text-sm text-red-600 dark:text-red-300 whitespace-pre-wrap font-mono">{{ execution.error }}</pre>
      </div>

      <!-- Output Data -->
      <div v-if="execution.data?.length" class="card">
        <div class="p-4 border-b border-slate-200 dark:border-slate-700">
          <h3 class="font-semibold text-slate-900 dark:text-white">Output Data</h3>
        </div>
        <div class="p-4">
          <pre class="text-sm text-slate-600 dark:text-slate-300 whitespace-pre-wrap font-mono bg-slate-50 dark:bg-slate-900 p-4 rounded-lg overflow-auto max-h-96">{{ JSON.stringify(execution.data, null, 2) }}</pre>
        </div>
      </div>

      <!-- Node Data -->
      <div v-if="execution.nodeData && Object.keys(execution.nodeData).length > 0" class="card">
        <div class="p-4 border-b border-slate-200 dark:border-slate-700">
          <h3 class="font-semibold text-slate-900 dark:text-white">Node Outputs</h3>
        </div>
        <div class="divide-y divide-slate-200 dark:divide-slate-700">
          <div
            v-for="(data, nodeName) in execution.nodeData"
            :key="nodeName"
            class="p-4"
          >
            <h4 class="font-medium text-slate-700 dark:text-slate-300 mb-2">{{ nodeName }}</h4>
            <pre class="text-sm text-slate-600 dark:text-slate-400 whitespace-pre-wrap font-mono bg-slate-50 dark:bg-slate-900 p-3 rounded-lg overflow-auto max-h-48">{{ JSON.stringify(data, null, 2) }}</pre>
          </div>
        </div>
      </div>
    </div>

    <!-- Loading State -->
    <div v-else class="text-center py-12">
      <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600 mx-auto" />
      <p class="mt-4 text-slate-500 dark:text-slate-400">Loading execution details...</p>
    </div>
  </div>
</template>
