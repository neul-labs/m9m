<script setup lang="ts">
import { computed, markRaw } from 'vue'
import { VueFlow, useVueFlow, Position } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'
import type { Node, Edge, Connection } from '@vue-flow/core'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import '@vue-flow/controls/dist/style.css'
import '@vue-flow/minimap/dist/style.css'

import BaseNode from '@/components/nodes/BaseNode.vue'
import { useWorkflowStore, useNodesStore } from '@/stores'
import { getNodeCategory } from '@/types/node'
import type { WorkflowNode } from '@/types'

const workflowStore = useWorkflowStore()
const nodesStore = useNodesStore()

const { onNodesChange, onEdgesChange, onConnect, project } = useVueFlow()

// Convert workflow nodes to Vue Flow nodes
const flowNodes = computed<Node[]>(() => {
  if (!workflowStore.currentWorkflow) return []

  return workflowStore.currentWorkflow.nodes.map((node) => ({
    id: node.id,
    type: 'custom',
    position: { x: node.position[0], y: node.position[1] },
    data: {
      label: node.name,
      nodeType: node.type,
      parameters: node.parameters,
      category: getNodeCategory(node.type),
    },
    sourcePosition: Position.Right,
    targetPosition: Position.Left,
  }))
})

// Convert workflow connections to Vue Flow edges
const flowEdges = computed<Edge[]>(() => {
  if (!workflowStore.currentWorkflow) return []

  const edges: Edge[] = []
  const connections = workflowStore.currentWorkflow.connections
  const nodesByName = new Map(
    workflowStore.currentWorkflow.nodes.map((n) => [n.name, n])
  )

  Object.entries(connections).forEach(([sourceName, conn]) => {
    const sourceNode = nodesByName.get(sourceName)
    if (!sourceNode || !conn.main) return

    conn.main.forEach((outputs, outputIndex) => {
      outputs.forEach((connection) => {
        const targetNode = nodesByName.get(connection.node)
        if (!targetNode) return

        edges.push({
          id: `${sourceNode.id}-${outputIndex}-${targetNode.id}-${connection.index}`,
          source: sourceNode.id,
          target: targetNode.id,
          sourceHandle: `output-${outputIndex}`,
          targetHandle: `input-${connection.index}`,
          animated: false,
          style: { stroke: 'var(--color-connection)' },
        })
      })
    })
  })

  return edges
})

// Handle node position changes
onNodesChange((changes) => {
  changes.forEach((change) => {
    if (change.type === 'position' && change.position) {
      const node = workflowStore.currentWorkflow?.nodes.find((n) => n.id === change.id)
      if (node) {
        workflowStore.updateNode(change.id, {
          position: [change.position.x, change.position.y],
        })
      }
    }
    if (change.type === 'select') {
      if (change.selected) {
        workflowStore.selectNode(change.id)
      } else {
        workflowStore.clearSelection()
      }
    }
    if (change.type === 'remove') {
      workflowStore.removeNode(change.id)
    }
  })
})

// Handle edge changes
onEdgesChange((changes) => {
  changes.forEach((change) => {
    if (change.type === 'remove') {
      // Parse the edge id to get connection info
      const parts = change.id.split('-')
      if (parts.length >= 4) {
        const sourceNode = workflowStore.currentWorkflow?.nodes.find((n) => n.id === parts[0])
        const targetNode = workflowStore.currentWorkflow?.nodes.find((n) => n.id === parts[2])
        if (sourceNode && targetNode) {
          workflowStore.removeConnection(
            sourceNode.name,
            targetNode.name,
            parseInt(parts[1]),
            parseInt(parts[3])
          )
        }
      }
    }
  })
})

// Handle new connections
onConnect((params: Connection) => {
  const sourceNode = workflowStore.currentWorkflow?.nodes.find((n) => n.id === params.source)
  const targetNode = workflowStore.currentWorkflow?.nodes.find((n) => n.id === params.target)

  if (sourceNode && targetNode) {
    const sourceOutput = parseInt(params.sourceHandle?.replace('output-', '') || '0')
    const targetInput = parseInt(params.targetHandle?.replace('input-', '') || '0')
    workflowStore.addConnection(sourceNode.name, targetNode.name, sourceOutput, targetInput)
  }
})

// Handle drop from node palette
const onDrop = (event: DragEvent) => {
  const nodeType = event.dataTransfer?.getData('application/vueflow')
  if (!nodeType) return

  const nodeTypeInfo = nodesStore.getNodeType(nodeType)
  if (!nodeTypeInfo) return

  const { left, top } = (event.target as HTMLElement).getBoundingClientRect()
  const position = project({
    x: event.clientX - left,
    y: event.clientY - top,
  })

  const newNode: WorkflowNode = {
    id: `node_${Date.now()}`,
    name: nodeTypeInfo.displayName,
    type: nodeType,
    typeVersion: nodeTypeInfo.version,
    position: [position.x, position.y],
    parameters: {},
  }

  workflowStore.addNode(newNode)
}

const onDragOver = (event: DragEvent) => {
  event.preventDefault()
  if (event.dataTransfer) {
    event.dataTransfer.dropEffect = 'move'
  }
}

// Node types for Vue Flow - use markRaw to prevent reactivity issues
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const nodeTypes: Record<string, any> = {
  custom: markRaw(BaseNode),
}

// MiniMap node color
const getMinimapNodeColor = (node: Node) => {
  const category = node.data?.category
  switch (category) {
    case 'trigger': return '#10b981'
    case 'action': return '#6366f1'
    case 'transform': return '#f59e0b'
    default: return '#94a3b8'
  }
}
</script>

<template>
  <div
    class="w-full h-full"
    @drop="onDrop"
    @dragover="onDragOver"
  >
    <VueFlow
      :nodes="flowNodes"
      :edges="flowEdges"
      :node-types="nodeTypes"
      :default-viewport="{ x: 100, y: 100, zoom: 1 }"
      :min-zoom="0.1"
      :max-zoom="2"
      :snap-to-grid="true"
      :snap-grid="[20, 20]"
      fit-view-on-init
      class="bg-slate-50 dark:bg-slate-900"
    >
      <Background
        :pattern-color="'var(--color-canvas-grid)'"
        :gap="20"
      />
      <Controls
        class="!bg-white dark:!bg-slate-800 !border-slate-200 dark:!border-slate-700 !shadow-lg"
      />
      <MiniMap
        :node-color="getMinimapNodeColor"
        class="!bg-white dark:!bg-slate-800 !border-slate-200 dark:!border-slate-700 !shadow-lg"
      />
    </VueFlow>

    <!-- Empty state overlay -->
    <div
      v-if="!workflowStore.currentWorkflow?.nodes.length"
      class="absolute inset-0 flex items-center justify-center pointer-events-none"
    >
      <div class="text-center">
        <div class="w-16 h-16 mx-auto mb-4 rounded-full bg-slate-100 dark:bg-slate-800 flex items-center justify-center">
          <svg class="w-8 h-8 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
          </svg>
        </div>
        <h3 class="text-lg font-medium text-slate-700 dark:text-slate-300">
          Drag nodes here to build your workflow
        </h3>
        <p class="mt-2 text-sm text-slate-500 dark:text-slate-400">
          Select nodes from the palette on the left
        </p>
      </div>
    </div>
  </div>
</template>

<style>
/* Vue Flow custom styles */
.vue-flow__node {
  @apply transition-shadow duration-150;
}

.vue-flow__node.selected {
  @apply ring-2 ring-primary-500 ring-offset-2 dark:ring-offset-slate-900;
}

.vue-flow__edge-path {
  @apply stroke-slate-400 dark:stroke-slate-500;
  stroke-width: 2;
}

.vue-flow__edge.selected .vue-flow__edge-path {
  @apply stroke-primary-500;
  stroke-width: 3;
}

.vue-flow__handle {
  @apply w-3 h-3 !bg-slate-400 dark:!bg-slate-500 !border-2 !border-white dark:!border-slate-800;
  @apply transition-all duration-150;
}

.vue-flow__handle:hover {
  @apply !bg-primary-500 scale-125;
}

.vue-flow__handle-connecting {
  @apply !bg-primary-500;
}

.vue-flow__handle-valid {
  @apply !bg-green-500;
}
</style>
