<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useVersionStore } from '@/stores'
import type { WorkflowVersion, VersionCreateRequest } from '@/types/version'
import {
  ClockIcon,
  PlusIcon,
  ArrowPathIcon,
  TrashIcon,
  DocumentDuplicateIcon,
  ChevronDownIcon,
  ChevronUpIcon,
  TagIcon,
  UserIcon,
} from '@heroicons/vue/24/outline'
import { CheckCircleIcon } from '@heroicons/vue/24/solid'

const props = defineProps<{
  workflowId: string
}>()

const emit = defineEmits<{
  restore: [version: WorkflowVersion]
  compare: [fromVersion: WorkflowVersion, toVersion: WorkflowVersion]
  viewVersion: [version: WorkflowVersion]
}>()

const versionStore = useVersionStore()

// UI State
const showCreateForm = ref(false)
const expandedVersionId = ref<string | null>(null)
const selectedVersions = ref<string[]>([])

// Create form data
const newVersionTag = ref('')
const newVersionDescription = ref('')

onMounted(() => {
  versionStore.fetchVersions(props.workflowId)
})

watch(() => props.workflowId, (newId) => {
  if (newId) {
    versionStore.fetchVersions(newId)
  }
})

function formatDate(dateString: string): string {
  const date = new Date(dateString)
  return date.toLocaleDateString(undefined, {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function formatRelativeDate(dateString: string): string {
  const date = new Date(dateString)
  const now = new Date()
  const diff = now.getTime() - date.getTime()

  if (diff < 60000) return 'Just now'
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`
  if (diff < 604800000) return `${Math.floor(diff / 86400000)}d ago`
  return formatDate(dateString)
}

function toggleExpand(versionId: string) {
  expandedVersionId.value = expandedVersionId.value === versionId ? null : versionId
}

function toggleSelect(versionId: string) {
  const index = selectedVersions.value.indexOf(versionId)
  if (index === -1) {
    if (selectedVersions.value.length < 2) {
      selectedVersions.value.push(versionId)
    }
  } else {
    selectedVersions.value.splice(index, 1)
  }
}

async function handleCreateVersion() {
  const data: VersionCreateRequest = {
    versionTag: newVersionTag.value || undefined,
    description: newVersionDescription.value || undefined,
  }

  try {
    await versionStore.createVersion(props.workflowId, data)
    showCreateForm.value = false
    newVersionTag.value = ''
    newVersionDescription.value = ''
  } catch (e) {
    console.error('Failed to create version:', e)
  }
}

async function handleRestore(version: WorkflowVersion) {
  emit('restore', version)
}

async function handleDelete(version: WorkflowVersion) {
  if (confirm(`Delete version ${version.versionTag}? This cannot be undone.`)) {
    try {
      await versionStore.deleteVersion(props.workflowId, version.id)
    } catch (e) {
      console.error('Failed to delete version:', e)
    }
  }
}

function handleCompare() {
  if (selectedVersions.value.length !== 2) return
  const versions = versionStore.versions
  const fromVersion = versions.find((v) => v.id === selectedVersions.value[0])
  const toVersion = versions.find((v) => v.id === selectedVersions.value[1])
  if (fromVersion && toVersion) {
    emit('compare', fromVersion, toVersion)
  }
}
</script>

<template>
  <div class="flex flex-col h-full bg-white dark:bg-slate-800 border-l border-slate-200 dark:border-slate-700">
    <!-- Header -->
    <div class="px-4 py-3 border-b border-slate-200 dark:border-slate-700">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <ClockIcon class="w-5 h-5 text-violet-500" />
          <h3 class="font-semibold text-slate-900 dark:text-white">Version History</h3>
        </div>
        <button
          @click="showCreateForm = !showCreateForm"
          class="p-1.5 text-slate-400 hover:text-violet-600 dark:hover:text-violet-400 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-700"
          title="Create new version"
        >
          <PlusIcon class="w-5 h-5" />
        </button>
      </div>

      <!-- Create Form -->
      <div v-if="showCreateForm" class="mt-3 p-3 bg-slate-50 dark:bg-slate-900 rounded-lg">
        <input
          v-model="newVersionTag"
          type="text"
          placeholder="Version tag (e.g., v1.0.0)"
          class="w-full px-3 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500 mb-2"
        />
        <textarea
          v-model="newVersionDescription"
          placeholder="Description (optional)"
          rows="2"
          class="w-full px-3 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500 mb-2"
        />
        <div class="flex justify-end gap-2">
          <button
            @click="showCreateForm = false"
            class="px-3 py-1.5 text-sm text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700 rounded"
          >
            Cancel
          </button>
          <button
            @click="handleCreateVersion"
            :disabled="versionStore.loading"
            class="px-3 py-1.5 text-sm bg-violet-600 hover:bg-violet-700 text-white rounded disabled:opacity-50"
          >
            Create
          </button>
        </div>
      </div>

      <!-- Compare Button -->
      <div v-if="selectedVersions.length === 2" class="mt-3">
        <button
          @click="handleCompare"
          class="w-full flex items-center justify-center gap-2 px-3 py-2 text-sm bg-blue-600 hover:bg-blue-700 text-white rounded-lg"
        >
          <DocumentDuplicateIcon class="w-4 h-4" />
          Compare Selected Versions
        </button>
      </div>
    </div>

    <!-- Version List -->
    <div class="flex-1 overflow-y-auto">
      <!-- Loading -->
      <div v-if="versionStore.loading" class="p-4 text-center text-slate-500">
        <div class="animate-spin inline-block w-6 h-6 border-2 border-violet-500 border-t-transparent rounded-full"></div>
      </div>

      <!-- Empty State -->
      <div v-else-if="versionStore.versions.length === 0" class="p-6 text-center text-slate-500 dark:text-slate-400">
        <ClockIcon class="w-10 h-10 mx-auto mb-2 opacity-50" />
        <p class="text-sm">No versions yet</p>
        <p class="text-xs mt-1">Create a version to track changes</p>
      </div>

      <!-- Versions -->
      <div v-else class="divide-y divide-slate-100 dark:divide-slate-700">
        <div
          v-for="version in versionStore.sortedVersions"
          :key="version.id"
          class="group"
        >
          <div
            class="px-4 py-3 hover:bg-slate-50 dark:hover:bg-slate-700/50 cursor-pointer"
            @click="toggleExpand(version.id)"
          >
            <div class="flex items-start gap-3">
              <!-- Checkbox for comparison -->
              <button
                @click.stop="toggleSelect(version.id)"
                class="mt-0.5 w-4 h-4 rounded border transition-colors"
                :class="selectedVersions.includes(version.id)
                  ? 'bg-violet-600 border-violet-600 text-white'
                  : 'border-slate-300 dark:border-slate-600 hover:border-violet-400'"
              >
                <svg v-if="selectedVersions.includes(version.id)" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
                </svg>
              </button>

              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2">
                  <span class="font-medium text-sm text-slate-900 dark:text-white">
                    {{ version.versionTag }}
                  </span>
                  <CheckCircleIcon
                    v-if="version.isCurrent"
                    class="w-4 h-4 text-green-500"
                    title="Current version"
                  />
                </div>
                <p class="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                  {{ formatRelativeDate(version.createdAt) }}
                </p>
              </div>

              <component
                :is="expandedVersionId === version.id ? ChevronUpIcon : ChevronDownIcon"
                class="w-4 h-4 text-slate-400"
              />
            </div>
          </div>

          <!-- Expanded Details -->
          <div
            v-if="expandedVersionId === version.id"
            class="px-4 py-3 bg-slate-50 dark:bg-slate-900/50 border-t border-slate-100 dark:border-slate-700"
          >
            <!-- Author -->
            <div class="flex items-center gap-2 text-xs text-slate-500 dark:text-slate-400 mb-2">
              <UserIcon class="w-3.5 h-3.5" />
              <span>{{ version.author }}</span>
              <span class="text-slate-300 dark:text-slate-600">|</span>
              <span>{{ formatDate(version.createdAt) }}</span>
            </div>

            <!-- Description -->
            <p v-if="version.description" class="text-sm text-slate-700 dark:text-slate-300 mb-2">
              {{ version.description }}
            </p>

            <!-- Changes -->
            <div v-if="version.changes?.length" class="mb-3">
              <h4 class="text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">Changes:</h4>
              <ul class="text-xs text-slate-600 dark:text-slate-400 space-y-0.5">
                <li v-for="(change, i) in version.changes.slice(0, 5)" :key="i" class="flex items-start gap-1">
                  <span class="text-slate-400">-</span>
                  {{ change }}
                </li>
                <li v-if="version.changes.length > 5" class="text-slate-400 italic">
                  +{{ version.changes.length - 5 }} more changes
                </li>
              </ul>
            </div>

            <!-- Tags -->
            <div v-if="version.tags?.length" class="flex flex-wrap gap-1 mb-3">
              <span
                v-for="tag in version.tags"
                :key="tag"
                class="inline-flex items-center gap-1 px-2 py-0.5 text-xs bg-slate-200 dark:bg-slate-700 text-slate-600 dark:text-slate-300 rounded"
              >
                <TagIcon class="w-3 h-3" />
                {{ tag }}
              </span>
            </div>

            <!-- Actions -->
            <div class="flex items-center gap-2">
              <button
                v-if="!version.isCurrent"
                @click.stop="handleRestore(version)"
                class="flex items-center gap-1.5 px-2.5 py-1.5 text-xs bg-violet-100 dark:bg-violet-900/30 text-violet-700 dark:text-violet-300 hover:bg-violet-200 dark:hover:bg-violet-900/50 rounded transition-colors"
              >
                <ArrowPathIcon class="w-3.5 h-3.5" />
                Restore
              </button>
              <button
                @click.stop="emit('viewVersion', version)"
                class="flex items-center gap-1.5 px-2.5 py-1.5 text-xs bg-slate-100 dark:bg-slate-700 text-slate-700 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-600 rounded transition-colors"
              >
                <DocumentDuplicateIcon class="w-3.5 h-3.5" />
                View
              </button>
              <button
                v-if="!version.isCurrent"
                @click.stop="handleDelete(version)"
                class="flex items-center gap-1.5 px-2.5 py-1.5 text-xs text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded transition-colors"
              >
                <TrashIcon class="w-3.5 h-3.5" />
                Delete
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Footer -->
    <div class="px-4 py-2 border-t border-slate-200 dark:border-slate-700 text-xs text-slate-500 dark:text-slate-400">
      {{ versionStore.total }} version{{ versionStore.total !== 1 ? 's' : '' }}
    </div>
  </div>
</template>
