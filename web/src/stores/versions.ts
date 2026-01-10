import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type {
  WorkflowVersion,
  VersionComparison,
  VersionCreateRequest,
  VersionRestoreRequest,
} from '@/types/version'
import * as versionApi from '@/api/versions'

export const useVersionStore = defineStore('versions', () => {
  // State
  const versions = ref<WorkflowVersion[]>([])
  const currentVersion = ref<WorkflowVersion | null>(null)
  const comparison = ref<VersionComparison | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)
  const total = ref(0)
  const currentWorkflowId = ref<string | null>(null)

  // Getters
  const latestVersion = computed(() =>
    versions.value.length > 0
      ? versions.value.reduce((latest, v) =>
          v.versionNum > latest.versionNum ? v : latest
        )
      : null
  )

  const activeVersion = computed(() =>
    versions.value.find((v) => v.isCurrent) || null
  )

  const sortedVersions = computed(() =>
    [...versions.value].sort((a, b) => b.versionNum - a.versionNum)
  )

  // Actions
  async function fetchVersions(workflowId: string, limit = 50, offset = 0) {
    loading.value = true
    error.value = null
    currentWorkflowId.value = workflowId
    try {
      const response = await versionApi.listVersions(workflowId, limit, offset)
      versions.value = response.data
      total.value = response.total
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch versions'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function fetchVersion(workflowId: string, versionId: string) {
    loading.value = true
    error.value = null
    try {
      currentVersion.value = await versionApi.getVersion(workflowId, versionId)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch version'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function createVersion(workflowId: string, data: VersionCreateRequest) {
    loading.value = true
    error.value = null
    try {
      const version = await versionApi.createVersion(workflowId, data)
      // Mark previous current version as not current
      versions.value.forEach((v) => {
        if (v.isCurrent) v.isCurrent = false
      })
      versions.value.unshift(version)
      total.value++
      return version
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to create version'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function deleteVersion(workflowId: string, versionId: string) {
    loading.value = true
    error.value = null
    try {
      await versionApi.deleteVersion(workflowId, versionId)
      versions.value = versions.value.filter((v) => v.id !== versionId)
      total.value--
      if (currentVersion.value?.id === versionId) {
        currentVersion.value = null
      }
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to delete version'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function restoreVersion(
    workflowId: string,
    versionId: string,
    data?: VersionRestoreRequest
  ) {
    loading.value = true
    error.value = null
    try {
      const result = await versionApi.restoreVersion(workflowId, versionId, data)
      // Refresh versions after restore
      await fetchVersions(workflowId)
      return result
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to restore version'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function compareVersions(
    workflowId: string,
    fromVersionId: string,
    toVersionId: string
  ) {
    loading.value = true
    error.value = null
    try {
      comparison.value = await versionApi.compareVersions(
        workflowId,
        fromVersionId,
        toVersionId
      )
      return comparison.value
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to compare versions'
      throw e
    } finally {
      loading.value = false
    }
  }

  function clearVersions() {
    versions.value = []
    currentVersion.value = null
    comparison.value = null
    currentWorkflowId.value = null
    total.value = 0
  }

  function clearError() {
    error.value = null
  }

  return {
    // State
    versions,
    currentVersion,
    comparison,
    loading,
    error,
    total,
    currentWorkflowId,

    // Getters
    latestVersion,
    activeVersion,
    sortedVersions,

    // Actions
    fetchVersions,
    fetchVersion,
    createVersion,
    deleteVersion,
    restoreVersion,
    compareVersions,
    clearVersions,
    clearError,
  }
})
