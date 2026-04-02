export interface NodeType {
  name: string
  displayName: string
  description: string
  version: number
  defaults: NodeDefaults
  inputs: string[]
  outputs: string[]
  properties?: NodeProperty[]
  credentials?: NodeCredentialType[]
  icon?: string
  iconColor?: string
  group?: string[]
  subtitle?: string
}

export interface NodeDefaults {
  name: string
  color?: string
}

export interface NodeProperty {
  displayName: string
  name: string
  type: NodePropertyType
  default?: unknown
  required?: boolean
  description?: string
  placeholder?: string
  options?: NodePropertyOption[]
  displayOptions?: NodePropertyDisplayOptions
  typeOptions?: NodePropertyTypeOptions
}

export type NodePropertyType =
  | 'string'
  | 'number'
  | 'boolean'
  | 'options'
  | 'multiOptions'
  | 'collection'
  | 'fixedCollection'
  | 'json'
  | 'dateTime'
  | 'color'

export interface NodePropertyOption {
  name: string
  value: string | number | boolean
  description?: string
  action?: string
}

export interface NodePropertyDisplayOptions {
  show?: Record<string, unknown[]>
  hide?: Record<string, unknown[]>
}

export interface NodePropertyTypeOptions {
  multipleValues?: boolean
  multipleValueButtonText?: string
  maxValue?: number
  minValue?: number
  numberPrecision?: number
  password?: boolean
  rows?: number
  alwaysOpenEditWindow?: boolean
}

export interface NodeCredentialType {
  name: string
  required?: boolean
  displayOptions?: NodePropertyDisplayOptions
}

export type NodeCategory =
  | 'trigger'
  | 'action'
  | 'transform'
  | 'flow'
  | 'core'
  | 'data'
  | 'communication'
  | 'marketing'
  | 'productivity'
  | 'sales'
  | 'development'
  | 'utility'

export interface NodeCategoryInfo {
  name: NodeCategory
  displayName: string
  icon: string
  color: string
}

export const NODE_CATEGORIES: NodeCategoryInfo[] = [
  { name: 'trigger', displayName: 'Triggers', icon: 'bolt', color: 'green' },
  { name: 'action', displayName: 'Actions', icon: 'play', color: 'indigo' },
  { name: 'transform', displayName: 'Transform', icon: 'arrows-right-left', color: 'amber' },
  { name: 'flow', displayName: 'Flow', icon: 'share', color: 'purple' },
  { name: 'core', displayName: 'Core', icon: 'cube', color: 'slate' },
  { name: 'data', displayName: 'Data', icon: 'database', color: 'blue' },
  { name: 'communication', displayName: 'Communication', icon: 'chat-bubble-left-right', color: 'pink' },
  { name: 'utility', displayName: 'Utility', icon: 'wrench', color: 'gray' },
]

export function getNodeCategory(nodeType: string): NodeCategory {
  // Triggers
  if (nodeType.includes('trigger') || nodeType.includes('webhook') || nodeType.includes('cron') || nodeType.includes('errorTrigger')) {
    return 'trigger'
  }
  // Transform
  if (nodeType.includes('function') || nodeType.includes('code') || nodeType.includes('set') || nodeType.includes('filter')) {
    return 'transform'
  }
  // Flow control
  if (nodeType.includes('if') || nodeType.includes('switch') || nodeType.includes('merge') || nodeType.includes('split') || nodeType.includes('wait') || nodeType.includes('loop') || nodeType.includes('noOp')) {
    return 'flow'
  }
  // Core
  if (nodeType.includes('start') || nodeType.includes('executeWorkflow')) {
    return 'core'
  }
  // Data / Database
  if (nodeType.includes('postgres') || nodeType.includes('mysql') || nodeType.includes('sqlite') || nodeType.includes('mongo') || nodeType.includes('redis') || nodeType.includes('elastic')) {
    return 'data'
  }
  // Communication
  if (nodeType.includes('slack') || nodeType.includes('discord') || nodeType.includes('twilio') || nodeType.includes('sendGrid') || nodeType.includes('teams') || nodeType.includes('email')) {
    return 'communication'
  }
  // Productivity
  if (nodeType.includes('notion') || nodeType.includes('stripe') || nodeType.includes('googleSheets')) {
    return 'productivity'
  }
  return 'action'
}
