<script setup lang="ts">
import { ref, computed } from 'vue'
import type { Schedule, ScheduleCreateRequest, ScheduleUpdateRequest } from '@/types/schedule'
import { TIMEZONES } from '@/types/schedule'
import CronExpressionBuilder from './CronExpressionBuilder.vue'
import { XMarkIcon, ClockIcon } from '@heroicons/vue/24/outline'

interface Workflow {
  id: string
  name: string
}

const props = defineProps<{
  schedule?: Schedule | null
  workflows: Workflow[]
}>()

const emit = defineEmits<{
  submit: [data: ScheduleCreateRequest | ScheduleUpdateRequest]
  cancel: []
}>()

// Form data
const workflowId = ref(props.schedule?.workflowId || '')
const cronExpression = ref(props.schedule?.cronExpression || '0 * * * *')
const timezone = ref(props.schedule?.timezone || 'UTC')
const enabled = ref(props.schedule?.enabled ?? true)
const maxRuns = ref(props.schedule?.maxRuns || 0)
const maxDuration = ref(props.schedule?.maxDuration || 0)

const isEditing = computed(() => !!props.schedule)

const formTitle = computed(() => isEditing.value ? 'Edit Schedule' : 'Create Schedule')

const isValid = computed(() => {
  if (!isEditing.value && !workflowId.value) return false
  if (!cronExpression.value) return false
  return true
})

function handleSubmit() {
  if (!isValid.value) return

  if (isEditing.value) {
    const data: ScheduleUpdateRequest = {
      cronExpression: cronExpression.value,
      timezone: timezone.value,
      enabled: enabled.value,
      maxRuns: maxRuns.value,
      maxDuration: maxDuration.value || undefined,
    }
    emit('submit', data)
  } else {
    const data: ScheduleCreateRequest = {
      workflowId: workflowId.value,
      cronExpression: cronExpression.value,
      timezone: timezone.value,
      enabled: enabled.value,
      maxRuns: maxRuns.value,
      maxDuration: maxDuration.value || undefined,
    }
    emit('submit', data)
  }
}
</script>

<template>
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
    <div class="bg-white dark:bg-slate-800 rounded-xl shadow-2xl w-full max-w-2xl max-h-[90vh] overflow-hidden flex flex-col">
      <!-- Header -->
      <div class="px-6 py-4 border-b border-slate-200 dark:border-slate-700 flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="p-2 bg-violet-100 dark:bg-violet-900/30 rounded-lg">
            <ClockIcon class="w-5 h-5 text-violet-600 dark:text-violet-400" />
          </div>
          <h2 class="text-lg font-semibold text-slate-900 dark:text-white">
            {{ formTitle }}
          </h2>
        </div>
        <button
          @click="emit('cancel')"
          class="p-2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-700"
        >
          <XMarkIcon class="w-5 h-5" />
        </button>
      </div>

      <!-- Form -->
      <form @submit.prevent="handleSubmit" class="flex-1 overflow-y-auto p-6 space-y-6">
        <!-- Workflow Selection (only for create) -->
        <div v-if="!isEditing">
          <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
            Workflow
          </label>
          <select
            v-model="workflowId"
            class="w-full px-4 py-2.5 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"
            required
          >
            <option value="" disabled>Select a workflow</option>
            <option v-for="workflow in workflows" :key="workflow.id" :value="workflow.id">
              {{ workflow.name }}
            </option>
          </select>
        </div>

        <!-- Cron Expression Builder -->
        <div>
          <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
            Schedule
          </label>
          <CronExpressionBuilder v-model="cronExpression" />
        </div>

        <!-- Timezone -->
        <div>
          <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
            Timezone
          </label>
          <select
            v-model="timezone"
            class="w-full px-4 py-2.5 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"
          >
            <option v-for="tz in TIMEZONES" :key="tz" :value="tz">
              {{ tz }}
            </option>
          </select>
        </div>

        <!-- Max Runs -->
        <div>
          <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
            Maximum Runs
          </label>
          <input
            v-model.number="maxRuns"
            type="number"
            min="0"
            class="w-full px-4 py-2.5 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"
            placeholder="0 for unlimited"
          />
          <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">
            Set to 0 for unlimited runs
          </p>
        </div>

        <!-- Max Duration -->
        <div>
          <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
            Maximum Duration (seconds)
          </label>
          <input
            v-model.number="maxDuration"
            type="number"
            min="0"
            class="w-full px-4 py-2.5 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"
            placeholder="0 for no limit"
          />
          <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">
            Maximum execution duration before timeout. Set to 0 for no limit.
          </p>
        </div>

        <!-- Enabled Toggle -->
        <div class="flex items-center justify-between p-4 bg-slate-50 dark:bg-slate-900 rounded-lg">
          <div>
            <h4 class="text-sm font-medium text-slate-900 dark:text-white">
              Enable Schedule
            </h4>
            <p class="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
              When enabled, the workflow will run according to the schedule
            </p>
          </div>
          <button
            type="button"
            @click="enabled = !enabled"
            class="relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-violet-500 focus:ring-offset-2"
            :class="enabled ? 'bg-violet-600' : 'bg-slate-200 dark:bg-slate-700'"
          >
            <span
              class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out"
              :class="enabled ? 'translate-x-5' : 'translate-x-0'"
            />
          </button>
        </div>
      </form>

      <!-- Footer -->
      <div class="px-6 py-4 border-t border-slate-200 dark:border-slate-700 flex justify-end gap-3">
        <button
          type="button"
          @click="emit('cancel')"
          class="px-4 py-2 text-sm font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-lg transition-colors"
        >
          Cancel
        </button>
        <button
          @click="handleSubmit"
          :disabled="!isValid"
          class="px-4 py-2 text-sm font-medium text-white bg-violet-600 hover:bg-violet-700 disabled:opacity-50 disabled:cursor-not-allowed rounded-lg transition-colors"
        >
          {{ isEditing ? 'Save Changes' : 'Create Schedule' }}
        </button>
      </div>
    </div>
  </div>
</template>
