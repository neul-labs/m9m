import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { WorkflowExecution, ExecutionFilters, ExecutionStatus } from '@/types'
import * as executionApi from '@/api/executions'
import { wsConnection } from '@/api/websocket'

export const useExecutionStore = defineStore('execution', () => {
  // State
  const executions = ref<WorkflowExecution[]>([])
  const currentExecution = ref<WorkflowExecution | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)
  const total = ref(0)

  // Real-time execution state
  const runningExecutions = ref<Map<string, WorkflowExecution>>(new Map())

  // Getters
  const recentExecutions = computed(() => {
    return [...executions.value]
      .sort((a, b) => new Date(b.startedAt).getTime() - new Date(a.startedAt).getTime())
      .slice(0, 10)
  })

  const executionsByStatus = computed(() => {
    const grouped: Record<ExecutionStatus, WorkflowExecution[]> = {
      running: [],
      completed: [],
      failed: [],
      cancelled: [],
      waiting: [],
    }
    executions.value.forEach((e) => {
      if (grouped[e.status]) {
        grouped[e.status].push(e)
      }
    })
    return grouped
  })

  const hasRunningExecutions = computed(() => runningExecutions.value.size > 0)

  // Actions
  async function fetchExecutions(filters?: ExecutionFilters) {
    loading.value = true
    error.value = null
    try {
      const response = await executionApi.getExecutions(filters)
      executions.value = response.data
      total.value = response.total
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch executions'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function fetchExecution(id: string) {
    loading.value = true
    error.value = null
    try {
      currentExecution.value = await executionApi.getExecution(id)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch execution'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function deleteExecution(id: string) {
    loading.value = true
    error.value = null
    try {
      await executionApi.deleteExecution(id)
      executions.value = executions.value.filter((e) => e.id !== id)
      if (currentExecution.value?.id === id) {
        currentExecution.value = null
      }
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to delete execution'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function retryExecution(id: string) {
    loading.value = true
    error.value = null
    try {
      const execution = await executionApi.retryExecution(id)
      executions.value.unshift(execution)
      return execution
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to retry execution'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function cancelExecution(id: string) {
    loading.value = true
    error.value = null
    try {
      await executionApi.cancelExecution(id)
      const execution = executions.value.find((e) => e.id === id)
      if (execution) {
        execution.status = 'cancelled'
      }
      runningExecutions.value.delete(id)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to cancel execution'
      throw e
    } finally {
      loading.value = false
    }
  }

  // WebSocket handlers
  function setupWebSocketHandlers() {
    wsConnection.on('executionUpdate', (message) => {
      const data = message.data as {
        executionId: string
        status: ExecutionStatus
        workflowId: string
        startedAt?: string
        finishedAt?: string
        error?: string
      }

      // Update running executions map
      if (data.status === 'running') {
        runningExecutions.value.set(data.executionId, {
          id: data.executionId,
          workflowId: data.workflowId,
          status: data.status,
          mode: 'trigger',
          startedAt: data.startedAt || new Date().toISOString(),
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
        })
      } else {
        runningExecutions.value.delete(data.executionId)
      }

      // Update execution in list if exists
      const execution = executions.value.find((e) => e.id === data.executionId)
      if (execution) {
        execution.status = data.status
        if (data.finishedAt) execution.finishedAt = data.finishedAt
        if (data.error) execution.error = data.error
      }

      // Update current execution if viewing it
      if (currentExecution.value?.id === data.executionId) {
        currentExecution.value.status = data.status
        if (data.finishedAt) currentExecution.value.finishedAt = data.finishedAt
        if (data.error) currentExecution.value.error = data.error
      }
    })
  }

  return {
    // State
    executions,
    currentExecution,
    loading,
    error,
    total,
    runningExecutions,

    // Getters
    recentExecutions,
    executionsByStatus,
    hasRunningExecutions,

    // Actions
    fetchExecutions,
    fetchExecution,
    deleteExecution,
    retryExecution,
    cancelExecution,
    setupWebSocketHandlers,
  }
})
