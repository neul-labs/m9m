<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  PlayIcon,
  StopIcon,
  ArrowPathIcon,
  CloudArrowUpIcon,
  PencilIcon,
  SparklesIcon,
} from '@heroicons/vue/24/outline'

const props = defineProps<{
  workflowName?: string
  workflowActive?: boolean
  isNewWorkflow: boolean
  isDirty: boolean
  isLoading: boolean
  isExecuting: boolean
  showNodePalette: boolean
  showCopilot: boolean
}>()

const emit = defineEmits<{
  back: []
  rename: [name: string]
  togglePalette: []
  toggleCopilot: []
  toggleActive: []
  execute: []
  save: []
}>()

const editingName = ref(false)
const workflowNameInput = ref('')

const displayName = computed(() => props.workflowName || 'Untitled Workflow')

watch(
  () => props.workflowName,
  (workflowName) => {
    if (!editingName.value) {
      workflowNameInput.value = workflowName || ''
    }
  },
  { immediate: true }
)

function startEditingName() {
  workflowNameInput.value = props.workflowName || ''
  editingName.value = true
}

function saveWorkflowName() {
  const trimmedName = workflowNameInput.value.trim()
  if (trimmedName) {
    emit('rename', trimmedName)
  }
  editingName.value = false
}

function cancelEditingName() {
  workflowNameInput.value = props.workflowName || ''
  editingName.value = false
}
</script>

<template>
  <div class="h-14 bg-white dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700 flex items-center justify-between px-4">
    <div class="flex items-center gap-3">
      <button
        @click="emit('back')"
        class="p-2 rounded-lg text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-700"
      >
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
        </svg>
      </button>

      <div v-if="editingName" class="flex items-center gap-2">
        <input
          v-model="workflowNameInput"
          @keyup.enter="saveWorkflowName"
          @keyup.escape="cancelEditingName"
          class="input py-1 px-2 text-lg font-semibold w-64"
          autofocus
        />
        <button @click="saveWorkflowName" class="btn-primary py-1 px-3">Save</button>
        <button @click="cancelEditingName" class="btn-ghost py-1 px-3">Cancel</button>
      </div>
      <div v-else class="flex items-center gap-2 cursor-pointer group" @click="startEditingName">
        <h2 class="text-lg font-semibold text-slate-900 dark:text-white">
          {{ displayName }}
        </h2>
        <PencilIcon class="w-4 h-4 text-slate-400 opacity-0 group-hover:opacity-100 transition-opacity" />
      </div>

      <span v-if="props.isDirty" class="text-xs text-amber-600 dark:text-amber-400">
        Unsaved changes
      </span>
    </div>

    <div class="flex items-center gap-2">
      <button
        @click="emit('togglePalette')"
        :class="[
          'px-3 py-1.5 text-sm font-medium rounded-lg transition-colors',
          props.showNodePalette
            ? 'bg-primary-100 dark:bg-primary-900/30 text-primary-700 dark:text-primary-400'
            : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700'
        ]"
      >
        Nodes
      </button>
      <button
        @click="emit('toggleCopilot')"
        :class="[
          'px-3 py-1.5 text-sm font-medium rounded-lg transition-colors flex items-center gap-1.5',
          props.showCopilot
            ? 'bg-violet-100 dark:bg-violet-900/30 text-violet-700 dark:text-violet-400'
            : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700'
        ]"
      >
        <SparklesIcon class="w-4 h-4" />
        Copilot
      </button>
    </div>

    <div class="flex items-center gap-2">
      <button
        v-if="!props.isNewWorkflow"
        @click="emit('toggleActive')"
        :class="[
          'flex items-center gap-2 px-3 py-1.5 text-sm font-medium rounded-lg transition-colors',
          props.workflowActive
            ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400'
            : 'bg-slate-100 dark:bg-slate-700 text-slate-600 dark:text-slate-400'
        ]"
      >
        <component :is="props.workflowActive ? StopIcon : PlayIcon" class="w-4 h-4" />
        {{ props.workflowActive ? 'Active' : 'Inactive' }}
      </button>

      <button
        @click="emit('execute')"
        :disabled="props.isExecuting || props.isNewWorkflow"
        class="btn-secondary flex items-center gap-2"
      >
        <ArrowPathIcon v-if="props.isExecuting" class="w-4 h-4 animate-spin" />
        <PlayIcon v-else class="w-4 h-4" />
        <span>{{ props.isExecuting ? 'Running...' : 'Execute' }}</span>
      </button>

      <button
        @click="emit('save')"
        :disabled="props.isLoading"
        class="btn-primary flex items-center gap-2"
      >
        <CloudArrowUpIcon class="w-4 h-4" />
        <span>{{ props.isLoading ? 'Saving...' : 'Save' }}</span>
      </button>
    </div>
  </div>
</template>
