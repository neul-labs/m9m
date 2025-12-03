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
  if (nodeType.includes('trigger') || nodeType.includes('webhook') || nodeType.includes('cron')) {
    return 'trigger'
  }
  if (nodeType.includes('function') || nodeType.includes('code') || nodeType.includes('set') || nodeType.includes('filter')) {
    return 'transform'
  }
  if (nodeType.includes('if') || nodeType.includes('switch') || nodeType.includes('merge') || nodeType.includes('split')) {
    return 'flow'
  }
  return 'action'
}
