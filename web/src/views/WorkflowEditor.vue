<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter, onBeforeRouteLeave } from 'vue-router'
import WorkflowCanvas from '@/components/workflow/WorkflowCanvas.vue'
import WorkflowEditorToolbar from '@/components/workflow/WorkflowEditorToolbar.vue'
import NodePalette from '@/components/workflow/NodePalette.vue'
import NodePanel from '@/components/workflow/NodePanel.vue'
import AgentCopilot from '@/components/copilot/AgentCopilot.vue'
import { useWorkflowEditorStore, useWorkflowStore } from '@/stores'

const route = useRoute()
const router = useRouter()
const workflowStore = useWorkflowStore()
const workflowEditorStore = useWorkflowEditorStore()

const showNodePalette = ref(true)
const showNodePanel = ref(false)
const showCopilot = ref(false)
const isExecuting = ref(false)

const workflowId = computed(() => route.params.id as string | undefined)
const isNewWorkflow = computed(() => !workflowId.value || workflowId.value === 'new')
const workflow = computed(() => workflowStore.currentWorkflow)
const selectedNode = computed(() => workflowEditorStore.selectedNode)

onMounted(async () => {
  if (!isNewWorkflow.value && workflowId.value) {
    await workflowStore.fetchWorkflow(workflowId.value)
    workflowEditorStore.resetEditorState()
  } else {
    workflowEditorStore.createNewWorkflow()
  }
})

// Watch for selected node changes to show/hide panel
watch(selectedNode, (node) => {
  showNodePanel.value = !!node
})

// Confirm before leaving with unsaved changes
onBeforeRouteLeave((_to, _from, next) => {
  if (workflowEditorStore.isDirty) {
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
    workflowEditorStore.markClean()
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

const renameWorkflow = (name: string) => {
  workflowEditorStore.setWorkflowName(name)
}
</script>

<template>
  <div class="h-full flex flex-col bg-canvas-light dark:bg-canvas-dark">
    <WorkflowEditorToolbar
      :workflow-name="workflow?.name"
      :workflow-active="workflow?.active"
      :is-new-workflow="isNewWorkflow"
      :is-dirty="workflowEditorStore.isDirty"
      :is-loading="workflowStore.loading"
      :is-executing="isExecuting"
      :show-node-palette="showNodePalette"
      :show-copilot="showCopilot"
      @back="router.push('/workflows')"
      @rename="renameWorkflow"
      @toggle-palette="showNodePalette = !showNodePalette"
      @toggle-copilot="showCopilot = !showCopilot"
      @toggle-active="toggleActive"
      @execute="executeWorkflow"
      @save="saveWorkflow"
    />

    <div class="flex-1 flex overflow-hidden">
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

      <div class="flex-1 relative">
        <WorkflowCanvas />
      </div>

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
