import { Position, type Edge, type Node } from '@vue-flow/core'
import type { Workflow, WorkflowNode, Connection } from '@/types'
import { getNodeCategory } from '@/types/node'

const EDGE_ID_PREFIX = 'workflow-edge'

function cloneValue<T>(value: T): T {
  return structuredClone(value)
}

function cloneWorkflowNode(node: WorkflowNode): WorkflowNode {
  return {
    ...node,
    position: [...node.position] as [number, number],
    parameters: cloneValue(node.parameters),
    credentials: node.credentials ? cloneValue(node.credentials) : undefined,
  }
}

function cloneConnections(connections: Workflow['connections']): Workflow['connections'] {
  return Object.fromEntries(
    Object.entries(connections).map(([sourceNode, nodeConnections]) => [
      sourceNode,
      {
        main: nodeConnections.main?.map((outputs) =>
          outputs.map((connection) => ({ ...connection }))
        ),
      },
    ])
  )
}

function renameNodeConnections(workflow: Workflow, previousName: string, nextName: string) {
  if (previousName === nextName) {
    return
  }

  const currentConnections = workflow.connections[previousName]
  if (currentConnections) {
    workflow.connections[nextName] = currentConnections
    delete workflow.connections[previousName]
  }

  Object.values(workflow.connections).forEach((nodeConnections) => {
    nodeConnections.main?.forEach((outputs) => {
      outputs.forEach((connection) => {
        if (connection.node === previousName) {
          connection.node = nextName
        }
      })
    })
  })
}

function getNodeNameById(workflow: Workflow, nodeId: string): string | null {
  return workflow.nodes.find((node) => node.id === nodeId)?.name ?? null
}

function buildEdgeId(sourceNodeId: string, sourceOutput: number, targetNodeId: string, targetInput: number) {
  return [
    EDGE_ID_PREFIX,
    encodeURIComponent(sourceNodeId),
    sourceOutput,
    encodeURIComponent(targetNodeId),
    targetInput,
  ].join(':')
}

export function parseEdgeId(edgeId: string) {
  const [prefix, encodedSourceNodeId, sourceOutput, encodedTargetNodeId, targetInput] = edgeId.split(':')
  if (
    prefix !== EDGE_ID_PREFIX ||
    encodedSourceNodeId === undefined ||
    sourceOutput === undefined ||
    encodedTargetNodeId === undefined ||
    targetInput === undefined
  ) {
    return null
  }

  const parsedSourceOutput = Number.parseInt(sourceOutput, 10)
  const parsedTargetInput = Number.parseInt(targetInput, 10)
  if (Number.isNaN(parsedSourceOutput) || Number.isNaN(parsedTargetInput)) {
    return null
  }

  return {
    sourceNodeId: decodeURIComponent(encodedSourceNodeId),
    sourceOutput: parsedSourceOutput,
    targetNodeId: decodeURIComponent(encodedTargetNodeId),
    targetInput: parsedTargetInput,
  }
}

export function createEmptyWorkflow(): Workflow {
  const timestamp = new Date().toISOString()

  return {
    id: '',
    name: 'New Workflow',
    description: '',
    active: false,
    nodes: [],
    connections: {},
    settings: {},
    tags: [],
    createdAt: timestamp,
    updatedAt: timestamp,
  }
}

export function createWorkflowDraft(workflow?: Workflow, overrides: Partial<Workflow> = {}): Workflow {
  if (!workflow) {
    return {
      ...createEmptyWorkflow(),
      ...overrides,
    }
  }

  return {
    ...workflow,
    ...overrides,
    nodes: workflow.nodes.map(cloneWorkflowNode),
    connections: cloneConnections(workflow.connections),
    settings: workflow.settings ? cloneValue(workflow.settings) : {},
    staticData: workflow.staticData ? cloneValue(workflow.staticData) : undefined,
    pinData: workflow.pinData ? cloneValue(workflow.pinData) : undefined,
    tags: workflow.tags ? [...workflow.tags] : [],
  }
}

export function addNodeToWorkflow(workflow: Workflow, node: WorkflowNode) {
  workflow.nodes.push(cloneWorkflowNode(node))
}

export function updateNodeInWorkflow(workflow: Workflow, nodeId: string, updates: Partial<WorkflowNode>) {
  const index = workflow.nodes.findIndex((node) => node.id === nodeId)
  if (index === -1) {
    return false
  }

  const currentNode = workflow.nodes[index]
  const updatedNode: WorkflowNode = {
    ...currentNode,
    ...updates,
    position: updates.position ? [...updates.position] as [number, number] : currentNode.position,
    parameters: updates.parameters ? cloneValue(updates.parameters) : currentNode.parameters,
  }

  workflow.nodes[index] = updatedNode

  if (updates.name && updates.name !== currentNode.name) {
    renameNodeConnections(workflow, currentNode.name, updates.name)
  }

  return true
}

export function removeNodeFromWorkflow(workflow: Workflow, nodeId: string) {
  const node = workflow.nodes.find((item) => item.id === nodeId)
  if (!node) {
    return false
  }

  workflow.nodes = workflow.nodes.filter((item) => item.id !== nodeId)
  delete workflow.connections[node.name]

  Object.entries(workflow.connections).forEach(([sourceNode, nodeConnections]) => {
    if (!nodeConnections.main) {
      return
    }

    nodeConnections.main = nodeConnections.main.map((outputs) =>
      outputs.filter((connection) => connection.node !== node.name)
    )

    const hasConnections = nodeConnections.main.some((outputs) => outputs.length > 0)
    if (!hasConnections) {
      delete workflow.connections[sourceNode]
    }
  })

  return true
}

export function addConnectionToWorkflow(
  workflow: Workflow,
  sourceNodeName: string,
  targetNodeName: string,
  sourceOutput = 0,
  targetInput = 0
) {
  if (!workflow.connections[sourceNodeName]) {
    workflow.connections[sourceNodeName] = { main: [[]] }
  }

  const nodeConnections = workflow.connections[sourceNodeName]
  while ((nodeConnections.main?.length ?? 0) <= sourceOutput) {
    nodeConnections.main?.push([])
  }

  const newConnection: Connection = {
    node: targetNodeName,
    type: 'main',
    index: targetInput,
  }

  const outputs = nodeConnections.main?.[sourceOutput]
  if (!outputs) {
    return false
  }

  const exists = outputs.some(
    (connection) => connection.node === targetNodeName && connection.index === targetInput
  )

  if (exists) {
    return false
  }

  outputs.push(newConnection)
  return true
}

export function removeConnectionFromWorkflow(
  workflow: Workflow,
  sourceNodeName: string,
  targetNodeName: string,
  sourceOutput = 0,
  targetInput = 0
) {
  const nodeConnections = workflow.connections[sourceNodeName]
  const outputs = nodeConnections?.main?.[sourceOutput]
  if (!outputs) {
    return false
  }

  const nextOutputs = outputs.filter(
    (connection) => !(connection.node === targetNodeName && connection.index === targetInput)
  )

  if (nextOutputs.length === outputs.length) {
    return false
  }

  nodeConnections.main![sourceOutput] = nextOutputs
  const hasConnections = nodeConnections.main?.some((connections) => connections.length > 0)
  if (!hasConnections) {
    delete workflow.connections[sourceNodeName]
  }

  return true
}

export function addConnectionByNodeId(
  workflow: Workflow,
  sourceNodeId: string,
  targetNodeId: string,
  sourceOutput = 0,
  targetInput = 0
) {
  const sourceNodeName = getNodeNameById(workflow, sourceNodeId)
  const targetNodeName = getNodeNameById(workflow, targetNodeId)
  if (!sourceNodeName || !targetNodeName) {
    return false
  }

  return addConnectionToWorkflow(workflow, sourceNodeName, targetNodeName, sourceOutput, targetInput)
}

export function removeConnectionByNodeId(
  workflow: Workflow,
  sourceNodeId: string,
  targetNodeId: string,
  sourceOutput = 0,
  targetInput = 0
) {
  const sourceNodeName = getNodeNameById(workflow, sourceNodeId)
  const targetNodeName = getNodeNameById(workflow, targetNodeId)
  if (!sourceNodeName || !targetNodeName) {
    return false
  }

  return removeConnectionFromWorkflow(workflow, sourceNodeName, targetNodeName, sourceOutput, targetInput)
}

export function buildFlowNodes(workflow: Workflow | null): Node[] {
  if (!workflow) {
    return []
  }

  return workflow.nodes.map((node) => ({
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
}

export function buildFlowEdges(workflow: Workflow | null): Edge[] {
  if (!workflow) {
    return []
  }

  const edges: Edge[] = []
  const nodesByName = new Map(workflow.nodes.map((node) => [node.name, node]))

  Object.entries(workflow.connections).forEach(([sourceName, nodeConnections]) => {
    const sourceNode = nodesByName.get(sourceName)
    if (!sourceNode || !nodeConnections.main) {
      return
    }

    nodeConnections.main.forEach((outputs, outputIndex) => {
      outputs.forEach((connection) => {
        const targetNode = nodesByName.get(connection.node)
        if (!targetNode) {
          return
        }

        edges.push({
          id: buildEdgeId(sourceNode.id, outputIndex, targetNode.id, connection.index),
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
}
