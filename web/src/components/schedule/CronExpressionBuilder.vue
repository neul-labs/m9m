<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { CRON_PRESETS } from '@/types/schedule'
import { ClockIcon, CalendarIcon, InformationCircleIcon } from '@heroicons/vue/24/outline'

const props = defineProps<{
  modelValue: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const mode = ref<'preset' | 'custom'>('preset')
const selectedPreset = ref('')

// Custom cron parts
const minute = ref('*')
const hour = ref('*')
const dayOfMonth = ref('*')
const month = ref('*')
const dayOfWeek = ref('*')

// Parse initial value
function parseExpression(expr: string) {
  const parts = expr.split(' ')
  if (parts.length === 5) {
    minute.value = parts[0]
    hour.value = parts[1]
    dayOfMonth.value = parts[2]
    month.value = parts[3]
    dayOfWeek.value = parts[4]
  }
}

// Initialize
if (props.modelValue) {
  parseExpression(props.modelValue)
  const preset = CRON_PRESETS.find(p => p.value === props.modelValue)
  if (preset) {
    selectedPreset.value = preset.value
    mode.value = 'preset'
  } else {
    mode.value = 'custom'
  }
}

// Build expression from parts
const customExpression = computed(() => {
  return `${minute.value} ${hour.value} ${dayOfMonth.value} ${month.value} ${dayOfWeek.value}`
})

// Watch for changes
watch([minute, hour, dayOfMonth, month, dayOfWeek], () => {
  if (mode.value === 'custom') {
    emit('update:modelValue', customExpression.value)
  }
})

watch(selectedPreset, (val) => {
  if (val && mode.value === 'preset') {
    emit('update:modelValue', val)
    parseExpression(val)
  }
})

watch(mode, (val) => {
  if (val === 'preset' && selectedPreset.value) {
    emit('update:modelValue', selectedPreset.value)
  } else if (val === 'custom') {
    emit('update:modelValue', customExpression.value)
  }
})

// Get human-readable description
function getCronDescription(expr: string): string {
  const parts = expr.split(' ')
  if (parts.length !== 5) return 'Invalid expression'

  const [min, hr, dom, mon, dow] = parts

  // Simple pattern matching
  if (expr === '* * * * *') return 'Every minute'
  if (min.startsWith('*/') && hr === '*' && dom === '*' && mon === '*' && dow === '*') {
    return `Every ${min.slice(2)} minutes`
  }
  if (min === '0' && hr.startsWith('*/') && dom === '*' && mon === '*' && dow === '*') {
    return `Every ${hr.slice(2)} hours`
  }
  if (min === '0' && hr === '0' && dom === '*' && mon === '*' && dow === '*') {
    return 'Daily at midnight'
  }
  if (min === '0' && hr !== '*' && dom === '*' && mon === '*' && dow === '*') {
    return `Daily at ${hr}:00`
  }
  if (min === '0' && hr === '0' && dom === '*' && mon === '*' && dow !== '*') {
    const days = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']
    const dayNum = parseInt(dow)
    if (!isNaN(dayNum) && dayNum >= 0 && dayNum <= 6) {
      return `Weekly on ${days[dayNum]}`
    }
  }
  if (min === '0' && hr === '0' && dom !== '*' && mon === '*' && dow === '*') {
    return `Monthly on day ${dom}`
  }

  return `At ${min} ${hr} ${dom} ${mon} ${dow}`
}

const cronDescription = computed(() => {
  const expr = mode.value === 'preset' ? selectedPreset.value : customExpression.value
  return getCronDescription(expr)
})

// Field options
const minuteOptions = [
  { label: 'Every minute', value: '*' },
  { label: 'Every 5 minutes', value: '*/5' },
  { label: 'Every 10 minutes', value: '*/10' },
  { label: 'Every 15 minutes', value: '*/15' },
  { label: 'Every 30 minutes', value: '*/30' },
  { label: 'At minute 0', value: '0' },
]

const hourOptions = [
  { label: 'Every hour', value: '*' },
  { label: 'Every 2 hours', value: '*/2' },
  { label: 'Every 6 hours', value: '*/6' },
  { label: 'Every 12 hours', value: '*/12' },
  ...Array.from({ length: 24 }, (_, i) => ({ label: `At ${i}:00`, value: String(i) })),
]

const dayOfMonthOptions = [
  { label: 'Every day', value: '*' },
  ...Array.from({ length: 31 }, (_, i) => ({ label: `Day ${i + 1}`, value: String(i + 1) })),
]

const monthOptions = [
  { label: 'Every month', value: '*' },
  { label: 'January', value: '1' },
  { label: 'February', value: '2' },
  { label: 'March', value: '3' },
  { label: 'April', value: '4' },
  { label: 'May', value: '5' },
  { label: 'June', value: '6' },
  { label: 'July', value: '7' },
  { label: 'August', value: '8' },
  { label: 'September', value: '9' },
  { label: 'October', value: '10' },
  { label: 'November', value: '11' },
  { label: 'December', value: '12' },
]

const dayOfWeekOptions = [
  { label: 'Every day', value: '*' },
  { label: 'Sunday', value: '0' },
  { label: 'Monday', value: '1' },
  { label: 'Tuesday', value: '2' },
  { label: 'Wednesday', value: '3' },
  { label: 'Thursday', value: '4' },
  { label: 'Friday', value: '5' },
  { label: 'Saturday', value: '6' },
  { label: 'Weekdays', value: '1-5' },
  { label: 'Weekends', value: '0,6' },
]
</script>

<template>
  <div class="space-y-4">
    <!-- Mode Toggle -->
    <div class="flex rounded-lg bg-slate-100 dark:bg-slate-800 p-1">
      <button
        @click="mode = 'preset'"
        class="flex-1 px-4 py-2 text-sm font-medium rounded-md transition-colors"
        :class="mode === 'preset'
          ? 'bg-white dark:bg-slate-700 text-slate-900 dark:text-white shadow'
          : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'"
      >
        <ClockIcon class="w-4 h-4 inline mr-2" />
        Presets
      </button>
      <button
        @click="mode = 'custom'"
        class="flex-1 px-4 py-2 text-sm font-medium rounded-md transition-colors"
        :class="mode === 'custom'
          ? 'bg-white dark:bg-slate-700 text-slate-900 dark:text-white shadow'
          : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'"
      >
        <CalendarIcon class="w-4 h-4 inline mr-2" />
        Custom
      </button>
    </div>

    <!-- Preset Selection -->
    <div v-if="mode === 'preset'" class="space-y-2">
      <div class="grid grid-cols-2 gap-2">
        <button
          v-for="preset in CRON_PRESETS"
          :key="preset.value"
          @click="selectedPreset = preset.value"
          class="px-4 py-3 text-left rounded-lg border transition-all"
          :class="selectedPreset === preset.value
            ? 'border-violet-500 bg-violet-50 dark:bg-violet-900/20 text-violet-700 dark:text-violet-300'
            : 'border-slate-200 dark:border-slate-700 hover:border-slate-300 dark:hover:border-slate-600 text-slate-700 dark:text-slate-300'"
        >
          <div class="text-sm font-medium">{{ preset.label }}</div>
          <code class="text-xs text-slate-500 dark:text-slate-400">{{ preset.value }}</code>
        </button>
      </div>
    </div>

    <!-- Custom Builder -->
    <div v-else class="space-y-4">
      <div class="grid grid-cols-5 gap-3">
        <!-- Minute -->
        <div>
          <label class="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
            Minute
          </label>
          <select
            v-model="minute"
            class="w-full px-3 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"
          >
            <option v-for="opt in minuteOptions" :key="opt.value" :value="opt.value">
              {{ opt.label }}
            </option>
          </select>
        </div>

        <!-- Hour -->
        <div>
          <label class="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
            Hour
          </label>
          <select
            v-model="hour"
            class="w-full px-3 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"
          >
            <option v-for="opt in hourOptions" :key="opt.value" :value="opt.value">
              {{ opt.label }}
            </option>
          </select>
        </div>

        <!-- Day of Month -->
        <div>
          <label class="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
            Day
          </label>
          <select
            v-model="dayOfMonth"
            class="w-full px-3 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"
          >
            <option v-for="opt in dayOfMonthOptions" :key="opt.value" :value="opt.value">
              {{ opt.label }}
            </option>
          </select>
        </div>

        <!-- Month -->
        <div>
          <label class="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
            Month
          </label>
          <select
            v-model="month"
            class="w-full px-3 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"
          >
            <option v-for="opt in monthOptions" :key="opt.value" :value="opt.value">
              {{ opt.label }}
            </option>
          </select>
        </div>

        <!-- Day of Week -->
        <div>
          <label class="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
            Weekday
          </label>
          <select
            v-model="dayOfWeek"
            class="w-full px-3 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"
          >
            <option v-for="opt in dayOfWeekOptions" :key="opt.value" :value="opt.value">
              {{ opt.label }}
            </option>
          </select>
        </div>
      </div>

      <!-- Raw Expression Input -->
      <div>
        <label class="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
          Cron Expression
        </label>
        <input
          :value="customExpression"
          @input="(e) => {
            const val = (e.target as HTMLInputElement).value
            parseExpression(val)
            emit('update:modelValue', val)
          }"
          type="text"
          class="w-full px-3 py-2 font-mono text-sm rounded-lg border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"
          placeholder="* * * * *"
        />
      </div>
    </div>

    <!-- Description -->
    <div class="flex items-center gap-2 p-3 bg-slate-50 dark:bg-slate-800 rounded-lg">
      <InformationCircleIcon class="w-5 h-5 text-violet-500" />
      <span class="text-sm text-slate-700 dark:text-slate-300">
        {{ cronDescription }}
      </span>
    </div>
  </div>
</template>
