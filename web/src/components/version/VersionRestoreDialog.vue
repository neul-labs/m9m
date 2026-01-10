<script setup lang="ts">
import { ref } from 'vue'
import type { WorkflowVersion } from '@/types/version'
import {
  XMarkIcon,
  ArrowPathIcon,
  ExclamationTriangleIcon,
  ShieldCheckIcon,
} from '@heroicons/vue/24/outline'

defineProps<{
  version: WorkflowVersion
}>()

const emit = defineEmits<{
  confirm: [createBackup: boolean, description: string]
  cancel: []
}>()

const createBackup = ref(true)
const description = ref('')

function formatDate(dateString: string): string {
  return new Date(dateString).toLocaleString()
}

function handleConfirm() {
  emit('confirm', createBackup.value, description.value)
}
</script>

<template>
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
    <div class="bg-white dark:bg-slate-800 rounded-xl shadow-2xl w-full max-w-lg overflow-hidden">
      <!-- Header -->
      <div class="px-6 py-4 border-b border-slate-200 dark:border-slate-700 flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="p-2 bg-amber-100 dark:bg-amber-900/30 rounded-lg">
            <ArrowPathIcon class="w-5 h-5 text-amber-600 dark:text-amber-400" />
          </div>
          <h2 class="text-lg font-semibold text-slate-900 dark:text-white">
            Restore Version
          </h2>
        </div>
        <button
          @click="emit('cancel')"
          class="p-2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-700"
        >
          <XMarkIcon class="w-5 h-5" />
        </button>
      </div>

      <!-- Content -->
      <div class="p-6">
        <!-- Warning -->
        <div class="flex items-start gap-3 p-4 bg-amber-50 dark:bg-amber-900/20 rounded-lg border border-amber-200 dark:border-amber-800 mb-4">
          <ExclamationTriangleIcon class="w-5 h-5 text-amber-600 dark:text-amber-400 flex-shrink-0 mt-0.5" />
          <div>
            <p class="text-sm text-amber-800 dark:text-amber-200">
              This will replace the current workflow with the state from version <strong>{{ version.versionTag }}</strong>.
            </p>
            <p class="text-xs text-amber-600 dark:text-amber-400 mt-1">
              Created on {{ formatDate(version.createdAt) }} by {{ version.author }}
            </p>
          </div>
        </div>

        <!-- Version Info -->
        <div class="p-4 bg-slate-50 dark:bg-slate-900 rounded-lg mb-4">
          <h4 class="font-medium text-slate-900 dark:text-white mb-2">
            {{ version.versionTag }}
          </h4>
          <p v-if="version.description" class="text-sm text-slate-600 dark:text-slate-400 mb-2">
            {{ version.description }}
          </p>
          <div v-if="version.changes?.length" class="text-xs text-slate-500 dark:text-slate-400">
            <span class="font-medium">Changes: </span>
            {{ version.changes.slice(0, 3).join(', ') }}
            <span v-if="version.changes.length > 3">
              and {{ version.changes.length - 3 }} more
            </span>
          </div>
        </div>

        <!-- Backup Option -->
        <div class="flex items-center justify-between p-4 bg-green-50 dark:bg-green-900/20 rounded-lg border border-green-200 dark:border-green-800 mb-4">
          <div class="flex items-center gap-3">
            <ShieldCheckIcon class="w-5 h-5 text-green-600 dark:text-green-400" />
            <div>
              <p class="text-sm font-medium text-green-800 dark:text-green-200">
                Create backup before restoring
              </p>
              <p class="text-xs text-green-600 dark:text-green-400">
                Recommended to preserve current state
              </p>
            </div>
          </div>
          <button
            @click="createBackup = !createBackup"
            class="relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-green-500 focus:ring-offset-2"
            :class="createBackup ? 'bg-green-600' : 'bg-slate-200 dark:bg-slate-700'"
          >
            <span
              class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out"
              :class="createBackup ? 'translate-x-5' : 'translate-x-0'"
            />
          </button>
        </div>

        <!-- Description -->
        <div>
          <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
            Restore note (optional)
          </label>
          <textarea
            v-model="description"
            rows="2"
            placeholder="Why are you restoring this version?"
            class="w-full px-3 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"
          />
        </div>
      </div>

      <!-- Footer -->
      <div class="px-6 py-4 border-t border-slate-200 dark:border-slate-700 flex justify-end gap-3">
        <button
          @click="emit('cancel')"
          class="px-4 py-2 text-sm font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-lg transition-colors"
        >
          Cancel
        </button>
        <button
          @click="handleConfirm"
          class="px-4 py-2 text-sm font-medium text-white bg-amber-600 hover:bg-amber-700 rounded-lg transition-colors"
        >
          Restore Version
        </button>
      </div>
    </div>
  </div>
</template>
