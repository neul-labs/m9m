<script setup lang="ts">
/**
 * Execution Output Component
 * Displays workflow execution results with node-by-node data inspection
 */
import { computed, ref } from 'vue';
import {
  ChevronRightIcon,
  ChevronDownIcon,
  DocumentTextIcon,
  PhotoIcon,
  ClipboardDocumentIcon,
  CheckIcon,
  ExclamationCircleIcon,
  ClockIcon,
  PlayCircleIcon,
} from '@heroicons/vue/24/outline';
import type { WorkflowExecution, DataItem } from '@/types/workflow';

const props = defineProps<{
  execution: WorkflowExecution;
}>();

// Expanded nodes
const expandedNodes = ref<Set<string>>(new Set());

// Selected tab per node
const selectedTabs = ref<Record<string, 'json' | 'table' | 'binary'>>({});

// Copy success state
const copiedNodes = ref<Set<string>>(new Set());

// Get node execution data
const nodeResults = computed(() => {
  if (!props.execution.data) return [];

  const results: Array<{
    nodeId: string;
    nodeName: string;
    status: 'success' | 'error' | 'running';
    data: DataItem[];
    error?: string;
    startTime?: Date;
    endTime?: Date;
    executionTime?: number;
  }> = [];

  // Parse execution data - adapt based on actual data structure
  const executionData = props.execution.data;

  if (Array.isArray(executionData)) {
    // Simple array of results
    results.push({
      nodeId: 'output',
      nodeName: 'Output',
      status: 'success',
      data: executionData,
    });
  } else if (typeof executionData === 'object' && executionData !== null) {
    // Object with node-keyed results
    for (const [nodeId, nodeData] of Object.entries(executionData)) {
      const data = nodeData as { data?: DataItem[]; error?: string };
      results.push({
        nodeId,
        nodeName: nodeId,
        status: data.error ? 'error' : 'success',
        data: data.data || [],
        error: data.error,
      });
    }
  }

  return results;
});

// Status icon
function getStatusIcon(status: string) {
  switch (status) {
    case 'success':
    case 'completed':
      return CheckIcon;
    case 'error':
    case 'failed':
      return ExclamationCircleIcon;
    case 'running':
      return PlayCircleIcon;
    default:
      return ClockIcon;
  }
}

// Status colors
function getStatusColor(status: string) {
  switch (status) {
    case 'success':
    case 'completed':
      return 'text-green-500';
    case 'error':
    case 'failed':
      return 'text-red-500';
    case 'running':
      return 'text-blue-500';
    default:
      return 'text-gray-500';
  }
}

// Toggle node expansion
function toggleNode(nodeId: string) {
  if (expandedNodes.value.has(nodeId)) {
    expandedNodes.value.delete(nodeId);
  } else {
    expandedNodes.value.add(nodeId);
  }
}

// Get selected tab
function getSelectedTab(nodeId: string): 'json' | 'table' | 'binary' {
  return selectedTabs.value[nodeId] || 'json';
}

// Set selected tab
function setSelectedTab(nodeId: string, tab: 'json' | 'table' | 'binary') {
  selectedTabs.value[nodeId] = tab;
}

// Format JSON for display
function formatJSON(data: unknown): string {
  try {
    return JSON.stringify(data, null, 2);
  } catch {
    return String(data);
  }
}

// Copy data to clipboard
async function copyToClipboard(nodeId: string, data: unknown) {
  try {
    await navigator.clipboard.writeText(formatJSON(data));
    copiedNodes.value.add(nodeId);
    setTimeout(() => {
      copiedNodes.value.delete(nodeId);
    }, 2000);
  } catch (err) {
    console.error('Failed to copy:', err);
  }
}

// Check if data has binary content
function hasBinaryData(items: DataItem[]): boolean {
  return items.some((item) => item.binary && Object.keys(item.binary).length > 0);
}

function getItemJson(item: DataItem): Record<string, unknown> | null {
  const normalized = item.json ?? (item as DataItem & { JSON?: Record<string, unknown> }).JSON;
  return normalized && typeof normalized === 'object' ? normalized : null;
}

// Get table columns from data
function getTableColumns(items: DataItem[]): string[] {
  const columns = new Set<string>();
  for (const item of items) {
    const jsonValue = getItemJson(item);
    if (!jsonValue) {
      continue;
    }
    for (const key of Object.keys(jsonValue)) {
      columns.add(key);
    }
  }
  return Array.from(columns);
}

// Format cell value
function formatCellValue(value: unknown): string {
  if (value === null || value === undefined) return '-';
  if (typeof value === 'object') return JSON.stringify(value);
  return String(value);
}

// Format execution time
function formatDuration(ms?: number): string {
  if (!ms) return '-';
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
}

function formatBinarySize(fileSize?: string): string {
  const bytes = Number(fileSize);
  if (!Number.isFinite(bytes) || bytes <= 0) return 'Unknown size';
  return `${(bytes / 1024).toFixed(1)} KB`;
}
</script>

<template>
  <div class="execution-output">
    <!-- Header -->
    <div class="border-b border-gray-200 dark:border-gray-700 pb-4 mb-4">
      <div class="flex items-center justify-between">
        <h3 class="text-lg font-medium text-gray-900 dark:text-gray-100">
          Execution Output
        </h3>
        <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
          <component
            :is="getStatusIcon(execution.status)"
            class="h-5 w-5"
            :class="getStatusColor(execution.status)"
          />
          <span class="capitalize">{{ execution.status }}</span>
        </div>
      </div>
      <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
        Started: {{ new Date(execution.startedAt).toLocaleString() }}
        <template v-if="execution.finishedAt">
          | Duration: {{ formatDuration(new Date(execution.finishedAt).getTime() - new Date(execution.startedAt).getTime()) }}
        </template>
      </p>
    </div>

    <!-- No results -->
    <div
      v-if="nodeResults.length === 0"
      class="text-center py-8 text-gray-500 dark:text-gray-400"
    >
      <DocumentTextIcon class="h-12 w-12 mx-auto mb-3 opacity-50" />
      <p>No execution data available</p>
    </div>

    <!-- Node results -->
    <div v-else class="space-y-3">
      <div
        v-for="node in nodeResults"
        :key="node.nodeId"
        class="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden"
      >
        <!-- Node header -->
        <button
          class="w-full px-4 py-3 flex items-center justify-between bg-gray-50 dark:bg-gray-800 hover:bg-gray-100 dark:hover:bg-gray-750 transition-colors"
          @click="toggleNode(node.nodeId)"
        >
          <div class="flex items-center gap-3">
            <component
              :is="expandedNodes.has(node.nodeId) ? ChevronDownIcon : ChevronRightIcon"
              class="h-5 w-5 text-gray-400"
            />
            <component
              :is="getStatusIcon(node.status)"
              class="h-5 w-5"
              :class="getStatusColor(node.status)"
            />
            <span class="font-medium text-gray-900 dark:text-gray-100">
              {{ node.nodeName }}
            </span>
          </div>
          <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
            <span>{{ node.data.length }} items</span>
            <span v-if="node.executionTime">{{ formatDuration(node.executionTime) }}</span>
          </div>
        </button>

        <!-- Node content (expanded) -->
        <div v-if="expandedNodes.has(node.nodeId)" class="border-t border-gray-200 dark:border-gray-700">
          <!-- Error message -->
          <div
            v-if="node.error"
            class="p-4 bg-red-50 dark:bg-red-900/20 border-b border-red-200 dark:border-red-800"
          >
            <p class="text-sm text-red-600 dark:text-red-400 font-mono">
              {{ node.error }}
            </p>
          </div>

          <!-- Tabs -->
          <div v-if="node.data.length > 0" class="border-b border-gray-200 dark:border-gray-700">
            <div class="flex gap-1 p-2">
              <button
                class="px-3 py-1.5 text-sm font-medium rounded-md transition-colors"
                :class="getSelectedTab(node.nodeId) === 'json'
                  ? 'bg-primary-100 dark:bg-primary-900 text-primary-700 dark:text-primary-300'
                  : 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'"
                @click="setSelectedTab(node.nodeId, 'json')"
              >
                JSON
              </button>
              <button
                class="px-3 py-1.5 text-sm font-medium rounded-md transition-colors"
                :class="getSelectedTab(node.nodeId) === 'table'
                  ? 'bg-primary-100 dark:bg-primary-900 text-primary-700 dark:text-primary-300'
                  : 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'"
                @click="setSelectedTab(node.nodeId, 'table')"
              >
                Table
              </button>
              <button
                v-if="hasBinaryData(node.data)"
                class="px-3 py-1.5 text-sm font-medium rounded-md transition-colors"
                :class="getSelectedTab(node.nodeId) === 'binary'
                  ? 'bg-primary-100 dark:bg-primary-900 text-primary-700 dark:text-primary-300'
                  : 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'"
                @click="setSelectedTab(node.nodeId, 'binary')"
              >
                Binary
              </button>

              <!-- Copy button -->
              <button
                class="ml-auto px-3 py-1.5 text-sm font-medium rounded-md text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors flex items-center gap-1"
                @click="copyToClipboard(node.nodeId, node.data)"
              >
                <component
                  :is="copiedNodes.has(node.nodeId) ? CheckIcon : ClipboardDocumentIcon"
                  class="h-4 w-4"
                />
                {{ copiedNodes.has(node.nodeId) ? 'Copied!' : 'Copy' }}
              </button>
            </div>
          </div>

          <!-- JSON view -->
          <div
            v-if="getSelectedTab(node.nodeId) === 'json' && node.data.length > 0"
            class="p-4 max-h-96 overflow-auto"
          >
            <pre class="text-sm font-mono text-gray-700 dark:text-gray-300 whitespace-pre-wrap">{{ formatJSON(node.data) }}</pre>
          </div>

          <!-- Table view -->
          <div
            v-else-if="getSelectedTab(node.nodeId) === 'table' && node.data.length > 0"
            class="overflow-x-auto"
          >
            <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
              <thead class="bg-gray-50 dark:bg-gray-800">
                <tr>
                  <th
                    v-for="col in getTableColumns(node.data)"
                    :key="col"
                    class="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"
                  >
                    {{ col }}
                  </th>
                </tr>
              </thead>
              <tbody class="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
                <tr v-for="(item, idx) in node.data" :key="idx">
                  <td
                    v-for="col in getTableColumns(node.data)"
                    :key="col"
                    class="px-4 py-2 text-sm text-gray-900 dark:text-gray-100 whitespace-nowrap"
                  >
                    {{ formatCellValue(getItemJson(item)?.[col]) }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- Binary view -->
          <div
            v-else-if="getSelectedTab(node.nodeId) === 'binary'"
            class="p-4"
          >
            <div
              v-for="(item, idx) in node.data"
              :key="idx"
              class="mb-4"
            >
              <template v-if="item.binary">
                <div
                  v-for="(binary, key) in item.binary"
                  :key="key"
                  class="flex items-center gap-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg"
                >
                  <PhotoIcon class="h-8 w-8 text-gray-400" />
                  <div>
                    <p class="font-medium text-gray-900 dark:text-gray-100">{{ key }}</p>
                    <p class="text-sm text-gray-500 dark:text-gray-400">
                      {{ binary.mimeType }} | {{ formatBinarySize(binary.fileSize) }}
                    </p>
                  </div>
                </div>
              </template>
            </div>
          </div>

          <!-- Empty data -->
          <div
            v-if="node.data.length === 0 && !node.error"
            class="p-8 text-center text-gray-500 dark:text-gray-400"
          >
            <p>No output data</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
pre {
  background: transparent;
}
</style>
