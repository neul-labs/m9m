<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import {
  PlusIcon,
  CalendarDaysIcon,
  FunnelIcon,
  MagnifyingGlassIcon,
} from '@heroicons/vue/24/outline'
import { useScheduleStore, useWorkflowStore } from '@/stores'
import type { Schedule, ScheduleCreateRequest, ScheduleUpdateRequest } from '@/types/schedule'
import ScheduleList from '@/components/schedule/ScheduleList.vue'
import ScheduleForm from '@/components/schedule/ScheduleForm.vue'
import ScheduleHistory from '@/components/schedule/ScheduleHistory.vue'

const scheduleStore = useScheduleStore()
const workflowStore = useWorkflowStore()

// UI State
const showForm = ref(false)
const showHistory = ref(false)
const editingSchedule = ref<Schedule | null>(null)
const historySchedule = ref<Schedule | null>(null)

// Filters
const searchQuery = ref('')
const statusFilter = ref<'all' | 'enabled' | 'disabled'>('all')
const workflowFilter = ref('')

onMounted(async () => {
  await Promise.all([
    scheduleStore.fetchSchedules(),
    workflowStore.fetchWorkflows(),
  ])
})

const filteredSchedules = computed(() => {
  let schedules = scheduleStore.schedules

  if (statusFilter.value === 'enabled') {
    schedules = schedules.filter((s) => s.enabled)
  } else if (statusFilter.value === 'disabled') {
    schedules = schedules.filter((s) => !s.enabled)
  }

  if (workflowFilter.value) {
    schedules = schedules.filter((s) => s.workflowId === workflowFilter.value)
  }

  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    schedules = schedules.filter(
      (s) =>
        s.workflowName?.toLowerCase().includes(query) ||
        s.cronExpression.includes(query)
    )
  }

  return schedules
})

// Workflow list for form
const workflowOptions = computed(() =>
  workflowStore.workflows.map((w) => ({ id: w.id, name: w.name }))
)

// Actions
function openCreateForm() {
  editingSchedule.value = null
  showForm.value = true
}

function openEditForm(schedule: Schedule) {
  editingSchedule.value = schedule
  showForm.value = true
}

async function handleSubmit(data: ScheduleCreateRequest | ScheduleUpdateRequest) {
  try {
    if (editingSchedule.value) {
      await scheduleStore.updateSchedule(editingSchedule.value.id, data as ScheduleUpdateRequest)
    } else {
      await scheduleStore.createSchedule(data as ScheduleCreateRequest)
    }
    showForm.value = false
    editingSchedule.value = null
  } catch (e) {
    console.error('Failed to save schedule:', e)
  }
}

async function handleDelete(schedule: Schedule) {
  if (confirm(`Are you sure you want to delete this schedule for "${schedule.workflowName || 'Unknown'}"?`)) {
    try {
      await scheduleStore.deleteSchedule(schedule.id)
    } catch (e) {
      console.error('Failed to delete schedule:', e)
    }
  }
}

async function handleToggle(schedule: Schedule) {
  try {
    await scheduleStore.toggleScheduleEnabled(schedule.id)
  } catch (e) {
    console.error('Failed to toggle schedule:', e)
  }
}

async function openHistory(schedule: Schedule) {
  historySchedule.value = schedule
  showHistory.value = true
  await scheduleStore.fetchScheduleHistory(schedule.id)
}
</script>

<template>
  <div class="p-6">
    <!-- Header -->
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold text-slate-900 dark:text-white flex items-center gap-3">
          <CalendarDaysIcon class="w-8 h-8 text-violet-500" />
          Schedules
        </h1>
        <p class="text-slate-500 dark:text-slate-400 mt-1">
          Automate workflow execution with cron schedules
        </p>
      </div>
      <button
        @click="openCreateForm"
        class="flex items-center gap-2 px-4 py-2 bg-violet-600 hover:bg-violet-700 text-white rounded-lg transition-colors"
      >
        <PlusIcon class="w-5 h-5" />
        Create Schedule
      </button>
    </div>

    <!-- Stats Cards -->
    <div class="grid grid-cols-4 gap-4 mb-6">
      <div class="card p-4">
        <p class="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">Total</p>
        <p class="text-2xl font-bold text-slate-900 dark:text-white mt-1">
          {{ scheduleStore.total }}
        </p>
      </div>
      <div class="card p-4 bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800">
        <p class="text-xs font-medium text-green-600 dark:text-green-400 uppercase">Active</p>
        <p class="text-2xl font-bold text-green-700 dark:text-green-300 mt-1">
          {{ scheduleStore.enabledSchedules.length }}
        </p>
      </div>
      <div class="card p-4 bg-slate-50 dark:bg-slate-700/50">
        <p class="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">Inactive</p>
        <p class="text-2xl font-bold text-slate-700 dark:text-slate-300 mt-1">
          {{ scheduleStore.disabledSchedules.length }}
        </p>
      </div>
      <div class="card p-4 bg-violet-50 dark:bg-violet-900/20 border-violet-200 dark:border-violet-800">
        <p class="text-xs font-medium text-violet-600 dark:text-violet-400 uppercase">Upcoming</p>
        <p class="text-2xl font-bold text-violet-700 dark:text-violet-300 mt-1">
          {{ scheduleStore.schedulesWithUpcomingRuns.length }}
        </p>
      </div>
    </div>

    <!-- Filters -->
    <div class="flex items-center gap-4 mb-6">
      <!-- Search -->
      <div class="relative flex-1 max-w-md">
        <MagnifyingGlassIcon class="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-400" />
        <input
          v-model="searchQuery"
          type="text"
          placeholder="Search schedules..."
          class="w-full pl-10 pr-4 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"
        />
      </div>

      <!-- Status Filter -->
      <div class="flex items-center gap-2">
        <FunnelIcon class="w-5 h-5 text-slate-400" />
        <select
          v-model="statusFilter"
          class="px-3 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"
        >
          <option value="all">All Status</option>
          <option value="enabled">Active</option>
          <option value="disabled">Inactive</option>
        </select>
      </div>

      <!-- Workflow Filter -->
      <select
        v-model="workflowFilter"
        class="px-3 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"
      >
        <option value="">All Workflows</option>
        <option
          v-for="workflow in workflowStore.workflows"
          :key="workflow.id"
          :value="workflow.id"
        >
          {{ workflow.name }}
        </option>
      </select>
    </div>

    <!-- Schedule List -->
    <ScheduleList
      :schedules="filteredSchedules"
      :loading="scheduleStore.loading"
      @edit="openEditForm"
      @delete="handleDelete"
      @toggle="handleToggle"
      @view-history="openHistory"
    />

    <!-- Create/Edit Form Modal -->
    <ScheduleForm
      v-if="showForm"
      :schedule="editingSchedule"
      :workflows="workflowOptions"
      @submit="handleSubmit"
      @cancel="showForm = false; editingSchedule = null"
    />

    <!-- History Modal -->
    <ScheduleHistory
      v-if="showHistory"
      :history="scheduleStore.currentHistory"
      :loading="scheduleStore.loading"
      @close="showHistory = false; historySchedule = null"
    />
  </div>
</template>
