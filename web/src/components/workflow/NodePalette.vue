<script setup lang="ts">
import { ref, computed } from 'vue'
import { MagnifyingGlassIcon } from '@heroicons/vue/24/outline'
import { useNodesStore } from '@/stores'
import { NODE_CATEGORIES, getNodeCategory } from '@/types/node'
import type { NodeType, NodeCategory } from '@/types'

const nodesStore = useNodesStore()

const searchQuery = ref('')
const expandedCategories = ref<Set<NodeCategory>>(new Set(['trigger', 'action', 'transform']))

const filteredNodes = computed(() => {
  if (!searchQuery.value) {
    return nodesStore.nodeTypes
  }

  const query = searchQuery.value.toLowerCase()
  return nodesStore.nodeTypes.filter(
    (node) =>
      node.displayName.toLowerCase().includes(query) ||
      node.description.toLowerCase().includes(query)
  )
})

const nodesByCategory = computed(() => {
  const grouped: Record<NodeCategory, NodeType[]> = {} as Record<NodeCategory, NodeType[]>

  NODE_CATEGORIES.forEach((cat) => {
    grouped[cat.name] = []
  })

  filteredNodes.value.forEach((node) => {
    const category = getNodeCategory(node.name)
    if (grouped[category]) {
      grouped[category].push(node)
    } else {
      grouped.action.push(node)
    }
  })

  return grouped
})

const toggleCategory = (category: NodeCategory) => {
  if (expandedCategories.value.has(category)) {
    expandedCategories.value.delete(category)
  } else {
    expandedCategories.value.add(category)
  }
}

const onDragStart = (event: DragEvent, nodeType: NodeType) => {
  if (event.dataTransfer) {
    event.dataTransfer.setData('application/vueflow', nodeType.name)
    event.dataTransfer.effectAllowed = 'move'
  }
}

const getCategoryColor = (category: NodeCategory) => {
  switch (category) {
    case 'trigger': return 'bg-green-500'
    case 'action': return 'bg-indigo-500'
    case 'transform': return 'bg-amber-500'
    case 'flow': return 'bg-purple-500'
    default: return 'bg-slate-500'
  }
}

const getCategoryIcon = (category: NodeCategory) => {
  switch (category) {
    case 'trigger': return '⚡'
    case 'action': return '▶️'
    case 'transform': return '🔄'
    case 'flow': return '🔀'
    default: return '📦'
  }
}
</script>

<template>
  <div class="h-full bg-white dark:bg-slate-800 border-r border-slate-200 dark:border-slate-700 flex flex-col">
    <!-- Header -->
    <div class="p-4 border-b border-slate-200 dark:border-slate-700">
      <h3 class="font-semibold text-slate-900 dark:text-white mb-3">Add Nodes</h3>

      <!-- Search -->
      <div class="relative">
        <MagnifyingGlassIcon class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
        <input
          v-model="searchQuery"
          type="text"
          placeholder="Search nodes..."
          class="input pl-9 text-sm"
        />
      </div>
    </div>

    <!-- Node List -->
    <div class="flex-1 overflow-y-auto p-2">
      <div
        v-for="category in NODE_CATEGORIES"
        :key="category.name"
        class="mb-2"
      >
        <!-- Category Header -->
        <button
          v-if="nodesByCategory[category.name]?.length > 0"
          @click="toggleCategory(category.name)"
          class="w-full flex items-center gap-2 px-2 py-2 text-sm font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-lg transition-colors"
        >
          <span class="text-base">{{ getCategoryIcon(category.name) }}</span>
          <span class="flex-1 text-left">{{ category.displayName }}</span>
          <span class="text-xs text-slate-400">
            {{ nodesByCategory[category.name]?.length || 0 }}
          </span>
          <svg
            :class="[
              'w-4 h-4 transition-transform',
              expandedCategories.has(category.name) ? 'rotate-90' : ''
            ]"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
          </svg>
        </button>

        <!-- Category Nodes -->
        <div
          v-if="expandedCategories.has(category.name) && nodesByCategory[category.name]?.length > 0"
          class="mt-1 space-y-1"
        >
          <div
            v-for="node in nodesByCategory[category.name]"
            :key="node.name"
            :draggable="true"
            @dragstart="onDragStart($event, node)"
            class="flex items-center gap-3 px-3 py-2 rounded-lg cursor-grab active:cursor-grabbing hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors group"
          >
            <div :class="[getCategoryColor(category.name), 'w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0']">
              <span class="text-white text-xs font-bold">
                {{ node.displayName.charAt(0) }}
              </span>
            </div>
            <div class="min-w-0 flex-1">
              <p class="text-sm font-medium text-slate-900 dark:text-white truncate">
                {{ node.displayName }}
              </p>
              <p class="text-xs text-slate-500 dark:text-slate-400 truncate">
                {{ node.description }}
              </p>
            </div>
          </div>
        </div>
      </div>

      <!-- Empty State -->
      <div
        v-if="filteredNodes.length === 0"
        class="text-center py-8"
      >
        <p class="text-sm text-slate-500 dark:text-slate-400">
          No nodes found
        </p>
      </div>
    </div>

    <!-- Help Text -->
    <div class="p-4 border-t border-slate-200 dark:border-slate-700">
      <p class="text-xs text-slate-500 dark:text-slate-400">
        Drag nodes onto the canvas to add them to your workflow
      </p>
    </div>
  </div>
</template>
