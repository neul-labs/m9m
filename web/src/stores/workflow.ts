import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Workflow, WorkflowFilters } from '@/types'
import * as workflowApi from '@/api/workflows'

export const useWorkflowStore = defineStore('workflow', () => {
  const workflows = ref<Workflow[]>([])
  const currentWorkflow = ref<Workflow | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)
  const total = ref(0)

  const activeWorkflows = computed(() => workflows.value.filter((w) => w.active))
  const inactiveWorkflows = computed(() => workflows.value.filter((w) => !w.active))

  async function fetchWorkflows(filters?: WorkflowFilters) {
    loading.value = true
    error.value = null
    try {
      const response = await workflowApi.getWorkflows(filters)
      workflows.value = response.data
      total.value = response.total
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch workflows'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function fetchWorkflow(id: string) {
    loading.value = true
    error.value = null
    try {
      currentWorkflow.value = await workflowApi.getWorkflow(id)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch workflow'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function saveWorkflow() {
    if (!currentWorkflow.value) return
    loading.value = true
    error.value = null
    try {
      if (currentWorkflow.value.id) {
        currentWorkflow.value = await workflowApi.updateWorkflow(
          currentWorkflow.value.id,
          currentWorkflow.value
        )
      } else {
        currentWorkflow.value = await workflowApi.createWorkflow(currentWorkflow.value)
      }
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to save workflow'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function deleteWorkflow(id: string) {
    loading.value = true
    error.value = null
    try {
      await workflowApi.deleteWorkflow(id)
      workflows.value = workflows.value.filter((w) => w.id !== id)
      if (currentWorkflow.value?.id === id) {
        currentWorkflow.value = null
      }
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to delete workflow'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function toggleWorkflowActive(id: string) {
    const workflow = workflows.value.find((w) => w.id === id)
    if (!workflow) return

    loading.value = true
    error.value = null
    try {
      if (workflow.active) {
        await workflowApi.deactivateWorkflow(id)
        workflow.active = false
      } else {
        await workflowApi.activateWorkflow(id)
        workflow.active = true
      }
      if (currentWorkflow.value?.id === id) {
        currentWorkflow.value.active = workflow.active
      }
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to toggle workflow'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function executeWorkflow(id: string) {
    loading.value = true
    error.value = null
    try {
      const execution = await workflowApi.executeWorkflow(id)
      return execution
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to execute workflow'
      throw e
    } finally {
      loading.value = false
    }
  }

  function setCurrentWorkflow(workflow: Workflow | null) {
    currentWorkflow.value = workflow
  }

  return {
    workflows,
    currentWorkflow,
    loading,
    error,
    total,
    activeWorkflows,
    inactiveWorkflows,
    fetchWorkflows,
    fetchWorkflow,
    saveWorkflow,
    deleteWorkflow,
    toggleWorkflowActive,
    executeWorkflow,
    setCurrentWorkflow,
  }
})
