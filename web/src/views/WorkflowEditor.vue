<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter, onBeforeRouteLeave } from 'vue-router'
import {
  PlayIcon,
  StopIcon,
  ArrowPathIcon,
  CloudArrowUpIcon,
  PencilIcon,
  SparklesIcon,
} from '@heroicons/vue/24/outline'
import WorkflowCanvas from '@/components/workflow/WorkflowCanvas.vue'
import NodePalette from '@/components/workflow/NodePalette.vue'
import NodePanel from '@/components/workflow/NodePanel.vue'
import AgentCopilot from '@/components/copilot/AgentCopilot.vue'
import { useWorkflowStore } from '@/stores'

const route = useRoute()
const router = useRouter()
const workflowStore = useWorkflowStore()

const showNodePalette = ref(true)
const showNodePanel = ref(false)
const showCopilot = ref(false)
const isExecuting = ref(false)
const editingName = ref(false)
const workflowNameInput = ref('')

const workflowId = computed(() => route.params.id as string | undefined)
const isNewWorkflow = computed(() => !workflowId.value || workflowId.value === 'new')
const workflow = computed(() => workflowStore.currentWorkflow)
const selectedNode = computed(() => workflowStore.selectedNode)

onMounted(async () => {
  if (!isNewWorkflow.value && workflowId.value) {
    await workflowStore.fetchWorkflow(workflowId.value)
  } else {
    workflowStore.createNewWorkflow()
  }
})

// Watch for selected node changes to show/hide panel
watch(selectedNode, (node) => {
  showNodePanel.value = !!node
})

// Confirm before leaving with unsaved changes
onBeforeRouteLeave((_to, _from, next) => {
  if (workflowStore.isDirty) {
    if (confirm('You have unsaved changes. Are you sure you want to leave?')) {
      next()
    } else {
      next(false)
    }
  } else {
    next()
  }
})

const saveWorkflow = async () => {
  try {
    await workflowStore.saveWorkflow()
    if (isNewWorkflow.value && workflow.value?.id) {
      router.replace(`/workflows/${workflow.value.id}`)
    }
  } catch (e) {
    console.error('Failed to save workflow:', e)
    alert('Failed to save workflow. Please try again.')
  }
}

const executeWorkflow = async () => {
  if (!workflow.value?.id) {
    alert('Please save the workflow before executing')
    return
  }

  isExecuting.value = true
  try {
    const execution = await workflowStore.executeWorkflow(workflow.value.id)
    // Could show execution result or navigate to execution detail
    console.log('Execution started:', execution)
  } catch (e) {
    console.error('Failed to execute workflow:', e)
    alert('Failed to execute workflow. Please try again.')
  } finally {
    isExecuting.value = false
  }
}

const toggleActive = async () => {
  if (!workflow.value?.id) return
  await workflowStore.toggleWorkflowActive(workflow.value.id)
}

const startEditingName = () => {
  if (workflow.value) {
    workflowNameInput.value = workflow.value.name
    editingName.value = true
  }
}

const saveWorkflowName = () => {
  if (workflowNameInput.value.trim()) {
    workflowStore.setWorkflowName(workflowNameInput.value.trim())
  }
  editingName.value = false
}

const cancelEditingName = () => {
  editingName.value = false
}
</script>

<template>
  <div class="h-full flex flex-col bg-canvas-light dark:bg-canvas-dark">
    <!-- Editor Toolbar -->
    <div class="h-14 bg-white dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700 flex items-center justify-between px-4">
      <!-- Left: Workflow Name -->
      <div class="flex items-center gap-3">
        <button
          @click="router.push('/workflows')"
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
            {{ workflow?.name || 'Untitled Workflow' }}
          </h2>
          <PencilIcon class="w-4 h-4 text-slate-400 opacity-0 group-hover:opacity-100 transition-opacity" />
        </div>

        <span v-if="workflowStore.isDirty" class="text-xs text-amber-600 dark:text-amber-400">
          Unsaved changes
        </span>
      </div>

      <!-- Center: Quick Actions -->
      <div class="flex items-center gap-2">
        <button
          @click="showNodePalette = !showNodePalette"
          :class="[
            'px-3 py-1.5 text-sm font-medium rounded-lg transition-colors',
            showNodePalette
              ? 'bg-primary-100 dark:bg-primary-900/30 text-primary-700 dark:text-primary-400'
              : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700'
          ]"
        >
          Nodes
        </button>
        <button
          @click="showCopilot = !showCopilot"
          :class="[
            'px-3 py-1.5 text-sm font-medium rounded-lg transition-colors flex items-center gap-1.5',
            showCopilot
              ? 'bg-violet-100 dark:bg-violet-900/30 text-violet-700 dark:text-violet-400'
              : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700'
          ]"
        >
          <SparklesIcon class="w-4 h-4" />
          Copilot
        </button>
      </div>

      <!-- Right: Actions -->
      <div class="flex items-center gap-2">
        <!-- Active Toggle -->
        <button
          v-if="!isNewWorkflow"
          @click="toggleActive"
          :class="[
            'flex items-center gap-2 px-3 py-1.5 text-sm font-medium rounded-lg transition-colors',
            workflow?.active
              ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400'
              : 'bg-slate-100 dark:bg-slate-700 text-slate-600 dark:text-slate-400'
          ]"
        >
          <component :is="workflow?.active ? StopIcon : PlayIcon" class="w-4 h-4" />
          {{ workflow?.active ? 'Active' : 'Inactive' }}
        </button>

        <!-- Execute Button -->
        <button
          @click="executeWorkflow"
          :disabled="isExecuting || !workflow?.id"
          class="btn-secondary flex items-center gap-2"
        >
          <ArrowPathIcon v-if="isExecuting" class="w-4 h-4 animate-spin" />
          <PlayIcon v-else class="w-4 h-4" />
          <span>{{ isExecuting ? 'Running...' : 'Execute' }}</span>
        </button>

        <!-- Save Button -->
        <button
          @click="saveWorkflow"
          :disabled="workflowStore.loading"
          class="btn-primary flex items-center gap-2"
        >
          <CloudArrowUpIcon class="w-4 h-4" />
          <span>{{ workflowStore.loading ? 'Saving...' : 'Save' }}</span>
        </button>
      </div>
    </div>

    <!-- Main Editor Area -->
    <div class="flex-1 flex overflow-hidden">
      <!-- Node Palette (Left) -->
      <transition
        enter-active-class="transition-all duration-200 ease-out"
        enter-from-class="-translate-x-full opacity-0"
        enter-to-class="translate-x-0 opacity-100"
        leave-active-class="transition-all duration-150 ease-in"
        leave-from-class="translate-x-0 opacity-100"
        leave-to-class="-translate-x-full opacity-0"
      >
        <NodePalette v-if="showNodePalette" class="w-72" />
      </transition>

      <!-- Canvas (Center) -->
      <div class="flex-1 relative">
        <WorkflowCanvas />
      </div>

      <!-- Node Panel (Right) -->
      <transition
        enter-active-class="transition-all duration-200 ease-out"
        enter-from-class="translate-x-full opacity-0"
        enter-to-class="translate-x-0 opacity-100"
        leave-active-class="transition-all duration-150 ease-in"
        leave-from-class="translate-x-0 opacity-100"
        leave-to-class="translate-x-full opacity-0"
      >
        <NodePanel v-if="showNodePanel && selectedNode" class="w-80" />
      </transition>

      <!-- Agent Copilot Panel (Right) -->
      <transition
        enter-active-class="transition-all duration-200 ease-out"
        enter-from-class="translate-x-full opacity-0"
        enter-to-class="translate-x-0 opacity-100"
        leave-active-class="transition-all duration-150 ease-in"
        leave-from-class="translate-x-0 opacity-100"
        leave-to-class="translate-x-full opacity-0"
      >
        <AgentCopilot
          v-if="showCopilot"
          class="w-96"
          @close="showCopilot = false"
        />
      </transition>
    </div>
  </div>
</template>
