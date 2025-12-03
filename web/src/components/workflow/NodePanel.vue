<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { XMarkIcon, TrashIcon } from '@heroicons/vue/24/outline'
import { useWorkflowStore, useNodesStore } from '@/stores'
import { getNodeCategory } from '@/types/node'

const workflowStore = useWorkflowStore()
const nodesStore = useNodesStore()

const node = computed(() => workflowStore.selectedNode)
const nodeType = computed(() => {
  if (!node.value) return null
  return nodesStore.getNodeType(node.value.type)
})
const category = computed(() => {
  if (!node.value) return 'action'
  return getNodeCategory(node.value.type)
})

const localName = ref('')
const localParameters = ref<Record<string, unknown>>({})

// Sync local state with selected node
watch(node, (newNode) => {
  if (newNode) {
    localName.value = newNode.name
    localParameters.value = { ...newNode.parameters }
  }
}, { immediate: true })

const close = () => {
  workflowStore.clearSelection()
}

const deleteNode = () => {
  if (node.value && confirm('Are you sure you want to delete this node?')) {
    workflowStore.removeNode(node.value.id)
  }
}

const updateName = () => {
  if (node.value && localName.value.trim()) {
    workflowStore.updateNode(node.value.id, { name: localName.value.trim() })
  }
}

const updateParameter = (key: string, value: unknown) => {
  if (node.value) {
    const newParams = { ...localParameters.value, [key]: value }
    localParameters.value = newParams
    workflowStore.updateNode(node.value.id, { parameters: newParams })
  }
}

const getCategoryColor = () => {
  switch (category.value) {
    case 'trigger': return 'border-green-500'
    case 'action': return 'border-indigo-500'
    case 'transform': return 'border-amber-500'
    default: return 'border-slate-500'
  }
}
</script>

<template>
  <div class="h-full bg-white dark:bg-slate-800 border-l border-slate-200 dark:border-slate-700 flex flex-col">
    <!-- Header -->
    <div :class="['p-4 border-b border-slate-200 dark:border-slate-700 border-l-4', getCategoryColor()]">
      <div class="flex items-center justify-between">
        <div class="flex-1 min-w-0">
          <input
            v-model="localName"
            @blur="updateName"
            @keyup.enter="updateName"
            class="w-full text-lg font-semibold bg-transparent border-none text-slate-900 dark:text-white focus:outline-none focus:ring-0"
          />
          <p class="text-sm text-slate-500 dark:text-slate-400 truncate">
            {{ nodeType?.displayName || node?.type }}
          </p>
        </div>
        <button
          @click="close"
          class="p-1 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700"
        >
          <XMarkIcon class="w-5 h-5" />
        </button>
      </div>
    </div>

    <!-- Content -->
    <div class="flex-1 overflow-y-auto p-4 space-y-4">
      <!-- Node Description -->
      <div v-if="nodeType?.description" class="text-sm text-slate-600 dark:text-slate-300">
        {{ nodeType.description }}
      </div>

      <!-- Parameters Section -->
      <div class="space-y-4">
        <h4 class="font-medium text-slate-900 dark:text-white">Parameters</h4>

        <!-- Dynamic parameters based on node type -->
        <div v-if="nodeType?.properties?.length" class="space-y-4">
          <div
            v-for="prop in nodeType.properties"
            :key="prop.name"
            class="space-y-1"
          >
            <label class="block text-sm font-medium text-slate-700 dark:text-slate-300">
              {{ prop.displayName }}
              <span v-if="prop.required" class="text-red-500">*</span>
            </label>

            <!-- String input -->
            <input
              v-if="prop.type === 'string'"
              :value="localParameters[prop.name] ?? prop.default ?? ''"
              @input="updateParameter(prop.name, ($event.target as HTMLInputElement).value)"
              :placeholder="prop.placeholder"
              class="input"
            />

            <!-- Number input -->
            <input
              v-else-if="prop.type === 'number'"
              type="number"
              :value="localParameters[prop.name] ?? prop.default ?? 0"
              @input="updateParameter(prop.name, Number(($event.target as HTMLInputElement).value))"
              class="input"
            />

            <!-- Boolean input -->
            <label
              v-else-if="prop.type === 'boolean'"
              class="flex items-center gap-2"
            >
              <input
                type="checkbox"
                :checked="!!localParameters[prop.name]"
                @change="updateParameter(prop.name, ($event.target as HTMLInputElement).checked)"
                class="w-4 h-4 rounded border-slate-300 dark:border-slate-600 text-primary-600 focus:ring-primary-500"
              />
              <span class="text-sm text-slate-600 dark:text-slate-300">
                {{ prop.description }}
              </span>
            </label>

            <!-- Options select -->
            <select
              v-else-if="prop.type === 'options' && prop.options"
              :value="localParameters[prop.name] ?? prop.default"
              @change="updateParameter(prop.name, ($event.target as HTMLSelectElement).value)"
              class="input"
            >
              <option
                v-for="opt in prop.options"
                :key="String(opt.value)"
                :value="opt.value"
              >
                {{ opt.name }}
              </option>
            </select>

            <!-- JSON input -->
            <textarea
              v-else-if="prop.type === 'json'"
              :value="JSON.stringify(localParameters[prop.name] ?? prop.default ?? {}, null, 2)"
              @input="updateParameter(prop.name, JSON.parse(($event.target as HTMLTextAreaElement).value || '{}'))"
              rows="4"
              class="input font-mono text-sm"
            />

            <!-- Default to string -->
            <input
              v-else
              :value="localParameters[prop.name] ?? prop.default ?? ''"
              @input="updateParameter(prop.name, ($event.target as HTMLInputElement).value)"
              class="input"
            />

            <p v-if="prop.description && prop.type !== 'boolean'" class="text-xs text-slate-500 dark:text-slate-400">
              {{ prop.description }}
            </p>
          </div>
        </div>

        <!-- No properties message -->
        <div v-else class="text-sm text-slate-500 dark:text-slate-400">
          No configurable parameters for this node.
        </div>

        <!-- Raw Parameters Editor -->
        <div class="pt-4 border-t border-slate-200 dark:border-slate-700">
          <details class="group">
            <summary class="cursor-pointer text-sm font-medium text-slate-700 dark:text-slate-300 hover:text-slate-900 dark:hover:text-white">
              Advanced: Raw Parameters
            </summary>
            <div class="mt-2">
              <textarea
                :value="JSON.stringify(localParameters, null, 2)"
                @input="localParameters = JSON.parse(($event.target as HTMLTextAreaElement).value || '{}')"
                @blur="node && workflowStore.updateNode(node.id, { parameters: localParameters })"
                rows="6"
                class="input font-mono text-xs"
              />
            </div>
          </details>
        </div>
      </div>
    </div>

    <!-- Footer Actions -->
    <div class="p-4 border-t border-slate-200 dark:border-slate-700 flex items-center gap-2">
      <button
        @click="deleteNode"
        class="btn-ghost text-red-600 dark:text-red-400 flex items-center gap-2"
      >
        <TrashIcon class="w-4 h-4" />
        <span>Delete</span>
      </button>
    </div>
  </div>
</template>
