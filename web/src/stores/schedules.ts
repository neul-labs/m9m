import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type {
  Schedule,
  ScheduleFilters,
  ScheduleCreateRequest,
  ScheduleUpdateRequest,
  ExecutionHistory,
} from '@/types/schedule'
import * as scheduleApi from '@/api/schedules'

export const useScheduleStore = defineStore('schedules', () => {
  // State
  const schedules = ref<Schedule[]>([])
  const currentSchedule = ref<Schedule | null>(null)
  const currentHistory = ref<ExecutionHistory | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)
  const total = ref(0)

  // Getters
  const enabledSchedules = computed(() => schedules.value.filter((s) => s.enabled))
  const disabledSchedules = computed(() => schedules.value.filter((s) => !s.enabled))

  const schedulesWithUpcomingRuns = computed(() =>
    schedules.value
      .filter((s) => s.enabled && s.nextRun)
      .sort((a, b) => new Date(a.nextRun!).getTime() - new Date(b.nextRun!).getTime())
  )

  // Actions
  async function fetchSchedules(filters?: ScheduleFilters) {
    loading.value = true
    error.value = null
    try {
      const response = await scheduleApi.getSchedules(filters)
      schedules.value = response.data
      total.value = response.total
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch schedules'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function fetchSchedule(id: string) {
    loading.value = true
    error.value = null
    try {
      currentSchedule.value = await scheduleApi.getSchedule(id)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch schedule'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function createSchedule(data: ScheduleCreateRequest) {
    loading.value = true
    error.value = null
    try {
      const schedule = await scheduleApi.createSchedule(data)
      schedules.value.push(schedule)
      total.value++
      return schedule
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to create schedule'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function updateSchedule(id: string, data: ScheduleUpdateRequest) {
    loading.value = true
    error.value = null
    try {
      const updated = await scheduleApi.updateSchedule(id, data)
      const index = schedules.value.findIndex((s) => s.id === id)
      if (index !== -1) {
        schedules.value[index] = updated
      }
      if (currentSchedule.value?.id === id) {
        currentSchedule.value = updated
      }
      return updated
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to update schedule'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function deleteSchedule(id: string) {
    loading.value = true
    error.value = null
    try {
      await scheduleApi.deleteSchedule(id)
      schedules.value = schedules.value.filter((s) => s.id !== id)
      total.value--
      if (currentSchedule.value?.id === id) {
        currentSchedule.value = null
      }
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to delete schedule'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function toggleScheduleEnabled(id: string) {
    const schedule = schedules.value.find((s) => s.id === id)
    if (!schedule) return

    loading.value = true
    error.value = null
    try {
      let updated: Schedule
      if (schedule.enabled) {
        updated = await scheduleApi.disableSchedule(id)
      } else {
        updated = await scheduleApi.enableSchedule(id)
      }

      const index = schedules.value.findIndex((s) => s.id === id)
      if (index !== -1) {
        schedules.value[index] = updated
      }
      if (currentSchedule.value?.id === id) {
        currentSchedule.value = updated
      }
      return updated
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to toggle schedule'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function fetchScheduleHistory(id: string, limit = 50) {
    loading.value = true
    error.value = null
    try {
      currentHistory.value = await scheduleApi.getScheduleHistory(id, limit)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch schedule history'
      throw e
    } finally {
      loading.value = false
    }
  }

  function clearCurrentSchedule() {
    currentSchedule.value = null
    currentHistory.value = null
  }

  function clearError() {
    error.value = null
  }

  return {
    // State
    schedules,
    currentSchedule,
    currentHistory,
    loading,
    error,
    total,

    // Getters
    enabledSchedules,
    disabledSchedules,
    schedulesWithUpcomingRuns,

    // Actions
    fetchSchedules,
    fetchSchedule,
    createSchedule,
    updateSchedule,
    deleteSchedule,
    toggleScheduleEnabled,
    fetchScheduleHistory,
    clearCurrentSchedule,
    clearError,
  }
})
