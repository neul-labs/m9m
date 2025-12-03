import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { NodeType, NodeCategory } from '@/types'
import { getNodeCategory } from '@/types/node'
import * as nodesApi from '@/api/nodes'

export const useNodesStore = defineStore('nodes', () => {
  // State
  const nodeTypes = ref<NodeType[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const searchQuery = ref('')
  const selectedCategory = ref<NodeCategory | null>(null)

  // Getters
  const nodeTypeMap = computed(() => {
    const map = new Map<string, NodeType>()
    nodeTypes.value.forEach((nt) => map.set(nt.name, nt))
    return map
  })

  const filteredNodeTypes = computed(() => {
    let filtered = nodeTypes.value

    if (selectedCategory.value) {
      filtered = filtered.filter((nt) => getNodeCategory(nt.name) === selectedCategory.value)
    }

    if (searchQuery.value) {
      const query = searchQuery.value.toLowerCase()
      filtered = filtered.filter(
        (nt) =>
          nt.displayName.toLowerCase().includes(query) ||
          nt.description.toLowerCase().includes(query) ||
          nt.name.toLowerCase().includes(query)
      )
    }

    return filtered
  })

  const nodesByCategory = computed(() => {
    const grouped: Record<NodeCategory, NodeType[]> = {
      trigger: [],
      action: [],
      transform: [],
      flow: [],
      core: [],
      data: [],
      communication: [],
      marketing: [],
      productivity: [],
      sales: [],
      development: [],
      utility: [],
    }

    nodeTypes.value.forEach((nt) => {
      const category = getNodeCategory(nt.name)
      if (grouped[category]) {
        grouped[category].push(nt)
      }
    })

    return grouped
  })

  const triggerNodes = computed(() => nodesByCategory.value.trigger)
  const actionNodes = computed(() => nodesByCategory.value.action)
  const transformNodes = computed(() => nodesByCategory.value.transform)

  // Actions
  async function fetchNodeTypes() {
    loading.value = true
    error.value = null
    try {
      nodeTypes.value = await nodesApi.getNodeTypes()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch node types'
      throw e
    } finally {
      loading.value = false
    }
  }

  function getNodeType(name: string): NodeType | undefined {
    return nodeTypeMap.value.get(name)
  }

  function setSearchQuery(query: string) {
    searchQuery.value = query
  }

  function setSelectedCategory(category: NodeCategory | null) {
    selectedCategory.value = category
  }

  function clearFilters() {
    searchQuery.value = ''
    selectedCategory.value = null
  }

  return {
    // State
    nodeTypes,
    loading,
    error,
    searchQuery,
    selectedCategory,

    // Getters
    nodeTypeMap,
    filteredNodeTypes,
    nodesByCategory,
    triggerNodes,
    actionNodes,
    transformNodes,

    // Actions
    fetchNodeTypes,
    getNodeType,
    setSearchQuery,
    setSelectedCategory,
    clearFilters,
  }
})
