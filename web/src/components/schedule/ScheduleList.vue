<script setup lang="ts">
import type { Schedule } from '@/types/schedule'
import {
  ClockIcon,
  PencilSquareIcon,
  TrashIcon,
  CalendarDaysIcon,
  CheckCircleIcon,
  XCircleIcon,
} from '@heroicons/vue/24/outline'

defineProps<{
  schedules: Schedule[]
  loading?: boolean
}>()

const emit = defineEmits<{
  edit: [schedule: Schedule]
  delete: [schedule: Schedule]
  toggle: [schedule: Schedule]
  viewHistory: [schedule: Schedule]
}>()

function formatDate(dateString?: string): string {
  if (!dateString) return '-'
  const date = new Date(dateString)
  return date.toLocaleString()
}

function formatNextRun(dateString?: string): string {
  if (!dateString) return 'Not scheduled'
  const date = new Date(dateString)
  const now = new Date()
  const diff = date.getTime() - now.getTime()

  if (diff < 0) return 'Overdue'
  if (diff < 60000) return 'Less than a minute'
  if (diff < 3600000) return `${Math.floor(diff / 60000)} minutes`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)} hours`
  return `${Math.floor(diff / 86400000)} days`
}
</script>

<template>
  <div class="overflow-hidden rounded-lg border border-slate-200 dark:border-slate-700">
    <!-- Loading State -->
    <div v-if="loading" class="p-8 text-center text-slate-500 dark:text-slate-400">
      <div class="animate-spin inline-block w-8 h-8 border-4 border-violet-500 border-t-transparent rounded-full"></div>
      <p class="mt-2">Loading schedules...</p>
    </div>

    <!-- Empty State -->
    <div v-else-if="!schedules.length" class="p-8 text-center text-slate-500 dark:text-slate-400">
      <CalendarDaysIcon class="w-12 h-12 mx-auto mb-3 opacity-50" />
      <p class="font-medium">No schedules found</p>
      <p class="text-sm mt-1">Create a schedule to automate workflow execution</p>
    </div>

    <!-- Schedule List -->
    <table v-else class="min-w-full divide-y divide-slate-200 dark:divide-slate-700">
      <thead class="bg-slate-50 dark:bg-slate-800">
        <tr>
          <th class="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
            Status
          </th>
          <th class="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
            Workflow
          </th>
          <th class="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
            Schedule
          </th>
          <th class="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
            Next Run
          </th>
          <th class="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
            Runs
          </th>
          <th class="px-6 py-3 text-right text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
            Actions
          </th>
        </tr>
      </thead>
      <tbody class="bg-white dark:bg-slate-900 divide-y divide-slate-200 dark:divide-slate-700">
        <tr
          v-for="schedule in schedules"
          :key="schedule.id"
          class="hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors"
        >
          <!-- Status -->
          <td class="px-6 py-4 whitespace-nowrap">
            <button
              @click="emit('toggle', schedule)"
              class="flex items-center gap-2"
              :title="schedule.enabled ? 'Click to disable' : 'Click to enable'"
            >
              <span
                class="flex items-center justify-center w-8 h-8 rounded-full transition-colors"
                :class="schedule.enabled
                  ? 'bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400'
                  : 'bg-slate-100 dark:bg-slate-700 text-slate-400 dark:text-slate-500'"
              >
                <CheckCircleIcon v-if="schedule.enabled" class="w-5 h-5" />
                <XCircleIcon v-else class="w-5 h-5" />
              </span>
              <span
                class="text-sm font-medium"
                :class="schedule.enabled
                  ? 'text-green-600 dark:text-green-400'
                  : 'text-slate-400 dark:text-slate-500'"
              >
                {{ schedule.enabled ? 'Active' : 'Inactive' }}
              </span>
            </button>
          </td>

          <!-- Workflow -->
          <td class="px-6 py-4">
            <div class="text-sm font-medium text-slate-900 dark:text-white">
              {{ schedule.workflowName || 'Unknown Workflow' }}
            </div>
            <div class="text-xs text-slate-500 dark:text-slate-400 font-mono">
              {{ schedule.workflowId }}
            </div>
          </td>

          <!-- Schedule -->
          <td class="px-6 py-4">
            <div class="flex items-center gap-2">
              <ClockIcon class="w-4 h-4 text-slate-400" />
              <code class="text-sm bg-slate-100 dark:bg-slate-800 px-2 py-0.5 rounded text-slate-700 dark:text-slate-300">
                {{ schedule.cronExpression }}
              </code>
            </div>
            <div class="text-xs text-slate-500 dark:text-slate-400 mt-1">
              {{ schedule.timezone }}
            </div>
          </td>

          <!-- Next Run -->
          <td class="px-6 py-4">
            <div v-if="schedule.enabled && schedule.nextRun" class="text-sm text-slate-900 dark:text-white">
              {{ formatNextRun(schedule.nextRun) }}
            </div>
            <div v-else class="text-sm text-slate-400 dark:text-slate-500">
              -
            </div>
            <div v-if="schedule.nextRun" class="text-xs text-slate-500 dark:text-slate-400">
              {{ formatDate(schedule.nextRun) }}
            </div>
          </td>

          <!-- Runs -->
          <td class="px-6 py-4">
            <div class="text-sm text-slate-900 dark:text-white">
              {{ schedule.runCount }} / {{ schedule.maxRuns === 0 ? 'Unlimited' : schedule.maxRuns }}
            </div>
            <div v-if="schedule.lastRun" class="text-xs text-slate-500 dark:text-slate-400">
              Last: {{ formatDate(schedule.lastRun) }}
            </div>
          </td>

          <!-- Actions -->
          <td class="px-6 py-4 whitespace-nowrap text-right">
            <div class="flex items-center justify-end gap-2">
              <button
                @click="emit('viewHistory', schedule)"
                class="p-2 text-slate-400 hover:text-violet-600 dark:hover:text-violet-400 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
                title="View execution history"
              >
                <CalendarDaysIcon class="w-4 h-4" />
              </button>
              <button
                @click="emit('edit', schedule)"
                class="p-2 text-slate-400 hover:text-blue-600 dark:hover:text-blue-400 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
                title="Edit schedule"
              >
                <PencilSquareIcon class="w-4 h-4" />
              </button>
              <button
                @click="emit('delete', schedule)"
                class="p-2 text-slate-400 hover:text-red-600 dark:hover:text-red-400 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
                title="Delete schedule"
              >
                <TrashIcon class="w-4 h-4" />
              </button>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
