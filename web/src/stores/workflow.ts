import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Workflow, WorkflowNode, WorkflowFilters, Connection } from '@/types'
import * as workflowApi from '@/api/workflows'

export const useWorkflowStore = defineStore('workflow', () => {
  // State
  const workflows = ref<Workflow[]>([])
  const currentWorkflow = ref<Workflow | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)
  const total = ref(0)

  // Editor state
  const selectedNodeIds = ref<string[]>([])
  const isDirty = ref(false)

  // Getters
  const activeWorkflows = computed(() => workflows.value.filter((w) => w.active))
  const inactiveWorkflows = computed(() => workflows.value.filter((w) => !w.active))

  const selectedNodes = computed(() => {
    if (!currentWorkflow.value) return []
    return currentWorkflow.value.nodes.filter((n) => selectedNodeIds.value.includes(n.id))
  })

  const selectedNode = computed(() => {
    if (selectedNodeIds.value.length !== 1 || !currentWorkflow.value) return null
    return currentWorkflow.value.nodes.find((n) => n.id === selectedNodeIds.value[0]) || null
  })

  // Actions
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
      isDirty.value = false
      selectedNodeIds.value = []
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
      isDirty.value = false
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

  // Editor actions
  function createNewWorkflow() {
    currentWorkflow.value = {
      id: '',
      name: 'New Workflow',
      description: '',
      active: false,
      nodes: [],
      connections: {},
      settings: {},
      tags: [],
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    }
    isDirty.value = true
    selectedNodeIds.value = []
  }

  function addNode(node: WorkflowNode) {
    if (!currentWorkflow.value) return
    currentWorkflow.value.nodes.push(node)
    isDirty.value = true
    selectedNodeIds.value = [node.id]
  }

  function updateNode(nodeId: string, updates: Partial<WorkflowNode>) {
    if (!currentWorkflow.value) return
    const index = currentWorkflow.value.nodes.findIndex((n) => n.id === nodeId)
    if (index !== -1) {
      currentWorkflow.value.nodes[index] = {
        ...currentWorkflow.value.nodes[index],
        ...updates,
      }
      isDirty.value = true
    }
  }

  function removeNode(nodeId: string) {
    if (!currentWorkflow.value) return
    currentWorkflow.value.nodes = currentWorkflow.value.nodes.filter((n) => n.id !== nodeId)

    // Remove connections to/from this node
    const nodeName = currentWorkflow.value.nodes.find((n) => n.id === nodeId)?.name
    if (nodeName) {
      delete currentWorkflow.value.connections[nodeName]
      Object.values(currentWorkflow.value.connections).forEach((conn) => {
        if (conn.main) {
          conn.main = conn.main.map((outputs) =>
            outputs.filter((c) => c.node !== nodeName)
          )
        }
      })
    }

    selectedNodeIds.value = selectedNodeIds.value.filter((id) => id !== nodeId)
    isDirty.value = true
  }

  function addConnection(sourceNode: string, targetNode: string, sourceOutput = 0, targetInput = 0) {
    if (!currentWorkflow.value) return

    if (!currentWorkflow.value.connections[sourceNode]) {
      currentWorkflow.value.connections[sourceNode] = { main: [[]] }
    }

    const connections = currentWorkflow.value.connections[sourceNode]
    while (connections.main!.length <= sourceOutput) {
      connections.main!.push([])
    }

    const newConnection: Connection = {
      node: targetNode,
      type: 'main',
      index: targetInput,
    }

    // Check if connection already exists
    const exists = connections.main![sourceOutput].some(
      (c) => c.node === targetNode && c.index === targetInput
    )

    if (!exists) {
      connections.main![sourceOutput].push(newConnection)
      isDirty.value = true
    }
  }

  function removeConnection(sourceNode: string, targetNode: string, sourceOutput = 0, targetInput = 0) {
    if (!currentWorkflow.value) return

    const connections = currentWorkflow.value.connections[sourceNode]
    if (!connections?.main?.[sourceOutput]) return

    connections.main[sourceOutput] = connections.main[sourceOutput].filter(
      (c) => !(c.node === targetNode && c.index === targetInput)
    )
    isDirty.value = true
  }

  function selectNode(nodeId: string, addToSelection = false) {
    if (addToSelection) {
      if (selectedNodeIds.value.includes(nodeId)) {
        selectedNodeIds.value = selectedNodeIds.value.filter((id) => id !== nodeId)
      } else {
        selectedNodeIds.value.push(nodeId)
      }
    } else {
      selectedNodeIds.value = [nodeId]
    }
  }

  function selectNodes(nodeIds: string[]) {
    selectedNodeIds.value = nodeIds
  }

  function clearSelection() {
    selectedNodeIds.value = []
  }

  function setWorkflowName(name: string) {
    if (!currentWorkflow.value) return
    currentWorkflow.value.name = name
    isDirty.value = true
  }

  return {
    // State
    workflows,
    currentWorkflow,
    loading,
    error,
    total,
    selectedNodeIds,
    isDirty,

    // Getters
    activeWorkflows,
    inactiveWorkflows,
    selectedNodes,
    selectedNode,

    // Actions
    fetchWorkflows,
    fetchWorkflow,
    saveWorkflow,
    deleteWorkflow,
    toggleWorkflowActive,
    executeWorkflow,

    // Editor actions
    createNewWorkflow,
    addNode,
    updateNode,
    removeNode,
    addConnection,
    removeConnection,
    selectNode,
    selectNodes,
    clearSelection,
    setWorkflowName,
  }
})
