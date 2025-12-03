<script setup lang="ts">
import { computed } from 'vue'
import { Handle, Position } from '@vue-flow/core'
import type { NodeCategory } from '@/types'

interface Props {
  id: string
  data: {
    label: string
    nodeType: string
    category: NodeCategory
    parameters?: Record<string, unknown>
  }
  selected?: boolean
}

const props = defineProps<Props>()

const categoryStyles = computed(() => {
  switch (props.data.category) {
    case 'trigger':
      return {
        border: 'border-l-green-500',
        icon: 'bg-green-500',
        iconText: '⚡',
      }
    case 'action':
      return {
        border: 'border-l-indigo-500',
        icon: 'bg-indigo-500',
        iconText: '▶',
      }
    case 'transform':
      return {
        border: 'border-l-amber-500',
        icon: 'bg-amber-500',
        iconText: '🔄',
      }
    case 'flow':
      return {
        border: 'border-l-purple-500',
        icon: 'bg-purple-500',
        iconText: '🔀',
      }
    default:
      return {
        border: 'border-l-slate-500',
        icon: 'bg-slate-500',
        iconText: '📦',
      }
  }
})

const nodeTypeName = computed(() => {
  const type = props.data.nodeType
  // Extract display name from type like "n8n-nodes-base.httpRequest"
  const parts = type.split('.')
  return parts[parts.length - 1]
    .replace(/([A-Z])/g, ' $1')
    .replace(/^./, (str) => str.toUpperCase())
    .trim()
})
</script>

<template>
  <div
    :class="[
      'workflow-node min-w-[180px] max-w-[220px]',
      categoryStyles.border,
      'border-l-4',
      selected ? 'selected' : ''
    ]"
  >
    <!-- Input Handle -->
    <Handle
      v-if="data.category !== 'trigger'"
      id="input-0"
      type="target"
      :position="Position.Left"
      class="connection-handle !-left-1.5"
    />

    <!-- Node Content -->
    <div class="p-3">
      <div class="flex items-center gap-2">
        <div :class="[categoryStyles.icon, 'w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0']">
          <span class="text-white text-sm">{{ categoryStyles.iconText }}</span>
        </div>
        <div class="min-w-0 flex-1">
          <h4 class="text-sm font-semibold text-slate-900 dark:text-white truncate">
            {{ data.label }}
          </h4>
          <p class="text-xs text-slate-500 dark:text-slate-400 truncate">
            {{ nodeTypeName }}
          </p>
        </div>
      </div>

      <!-- Parameters preview (optional) -->
      <div
        v-if="data.parameters && Object.keys(data.parameters).length > 0"
        class="mt-2 pt-2 border-t border-slate-200 dark:border-slate-600"
      >
        <div
          v-for="(value, key) in data.parameters"
          :key="key"
          class="text-xs text-slate-500 dark:text-slate-400 truncate"
        >
          <span class="font-medium">{{ key }}:</span>
          {{ typeof value === 'object' ? '...' : value }}
        </div>
      </div>
    </div>

    <!-- Output Handle -->
    <Handle
      id="output-0"
      type="source"
      :position="Position.Right"
      class="connection-handle !-right-1.5"
    />
  </div>
</template>

<style scoped>
.workflow-node {
  @apply bg-white dark:bg-slate-800 rounded-lg;
  @apply border-2 border-slate-200 dark:border-slate-600;
  @apply shadow-md;
  @apply transition-all duration-150;
}

.workflow-node:hover {
  @apply border-primary-400 dark:border-primary-500;
  @apply shadow-lg;
}

.workflow-node.selected {
  @apply border-primary-500 dark:border-primary-400;
}

.connection-handle {
  @apply w-3 h-3 rounded-full;
  @apply !bg-slate-400 dark:!bg-slate-500;
  @apply !border-2 !border-white dark:!border-slate-800;
  @apply transition-all duration-150;
}

.connection-handle:hover {
  @apply !bg-primary-500 scale-125;
}
</style>
