import type { DataItem } from './workflow'

export interface Schedule {
  id: string
  workflowId: string
  workflowName?: string
  cronExpression: string
  timezone: string
  enabled: boolean
  lastRun?: string
  nextRun?: string
  maxRuns: number
  runCount: number
  maxDuration?: number
  inputData?: DataItem[]
  parameters?: Record<string, unknown>
  createdAt: string
  updatedAt: string
  createdBy?: string
}

export interface ScheduleFilters {
  workflowId?: string
  enabled?: boolean
  search?: string
  offset?: number
  limit?: number
}

export interface ExecutionRecord {
  id: string
  scheduleId: string
  workflowId: string
  startTime: string
  endTime?: string
  duration: number
  status: 'pending' | 'running' | 'success' | 'failed' | 'timeout'
  error?: string
  resultData?: unknown
  metrics?: ExecutionMetrics
}

export interface ExecutionMetrics {
  nodesExecuted: number
  dataProcessed: number
  memoryUsed: number
  cpuTime: number
}

export interface ExecutionHistory {
  scheduleId: string
  executions: ExecutionRecord[]
  successCount: number
  failureCount: number
  lastSuccess?: string
  lastFailure?: string
  averageTime: number
}

export interface ScheduleCreateRequest {
  workflowId: string
  cronExpression: string
  timezone?: string
  enabled?: boolean
  maxRuns?: number
  maxDuration?: number
  inputData?: DataItem[]
  parameters?: Record<string, unknown>
}

export interface ScheduleUpdateRequest {
  cronExpression?: string
  timezone?: string
  enabled?: boolean
  maxRuns?: number
  maxDuration?: number
  inputData?: DataItem[]
  parameters?: Record<string, unknown>
}

// Common cron presets for the UI
export const CRON_PRESETS = [
  { label: 'Every minute', value: '* * * * *' },
  { label: 'Every 5 minutes', value: '*/5 * * * *' },
  { label: 'Every 15 minutes', value: '*/15 * * * *' },
  { label: 'Every 30 minutes', value: '*/30 * * * *' },
  { label: 'Every hour', value: '0 * * * *' },
  { label: 'Every 6 hours', value: '0 */6 * * *' },
  { label: 'Every 12 hours', value: '0 */12 * * *' },
  { label: 'Daily at midnight', value: '0 0 * * *' },
  { label: 'Daily at 9am', value: '0 9 * * *' },
  { label: 'Weekly on Monday', value: '0 0 * * 1' },
  { label: 'Monthly on the 1st', value: '0 0 1 * *' },
] as const

// Common timezones
export const TIMEZONES = [
  'UTC',
  'America/New_York',
  'America/Chicago',
  'America/Denver',
  'America/Los_Angeles',
  'Europe/London',
  'Europe/Paris',
  'Europe/Berlin',
  'Asia/Tokyo',
  'Asia/Shanghai',
  'Asia/Singapore',
  'Australia/Sydney',
] as const
