<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  PlusIcon,
  KeyIcon,
  PencilIcon,
  TrashIcon,
} from '@heroicons/vue/24/outline'
import { Dialog, DialogPanel, DialogTitle, TransitionRoot, TransitionChild } from '@headlessui/vue'
import { useCredentialsStore } from '@/stores'
import type { Credential, CredentialCreate } from '@/types/api'

const credentialsStore = useCredentialsStore()

const showModal = ref(false)
const editingCredential = ref<Credential | null>(null)
const form = ref<CredentialCreate>({
  name: '',
  type: '',
  data: {},
})

const credentialTypes = [
  { value: 'httpBasicAuth', label: 'HTTP Basic Auth' },
  { value: 'httpHeaderAuth', label: 'HTTP Header Auth' },
  { value: 'oAuth2Api', label: 'OAuth2' },
  { value: 'apiKey', label: 'API Key' },
  { value: 'postgres', label: 'PostgreSQL' },
  { value: 'mysql', label: 'MySQL' },
  { value: 'slack', label: 'Slack' },
  { value: 'discord', label: 'Discord' },
  { value: 'openai', label: 'OpenAI' },
  { value: 'anthropic', label: 'Anthropic' },
]

onMounted(async () => {
  await credentialsStore.fetchCredentials()
})

const openCreateModal = () => {
  editingCredential.value = null
  form.value = { name: '', type: '', data: {} }
  showModal.value = true
}

const openEditModal = (credential: Credential) => {
  editingCredential.value = credential
  form.value = {
    name: credential.name,
    type: credential.type,
    data: {},
  }
  showModal.value = true
}

const closeModal = () => {
  showModal.value = false
  editingCredential.value = null
  form.value = { name: '', type: '', data: {} }
}

const saveCredential = async () => {
  try {
    if (editingCredential.value) {
      await credentialsStore.updateCredential(editingCredential.value.id, form.value)
    } else {
      await credentialsStore.createCredential(form.value)
    }
    closeModal()
  } catch (e) {
    console.error('Failed to save credential:', e)
    alert('Failed to save credential. Please try again.')
  }
}

const deleteCredential = async (credential: Credential) => {
  if (confirm(`Are you sure you want to delete "${credential.name}"?`)) {
    try {
      await credentialsStore.deleteCredential(credential.id)
    } catch (e) {
      console.error('Failed to delete credential:', e)
      alert('Failed to delete credential. Please try again.')
    }
  }
}

const formatDate = (dateStr: string) => {
  return new Date(dateStr).toLocaleDateString()
}
</script>

<template>
  <div class="p-6">
    <!-- Header -->
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold text-slate-900 dark:text-white">Credentials</h1>
        <p class="text-slate-500 dark:text-slate-400">
          Manage your authentication credentials securely
        </p>
      </div>
      <button @click="openCreateModal" class="btn-primary flex items-center gap-2">
        <PlusIcon class="w-5 h-5" />
        <span>Add Credential</span>
      </button>
    </div>

    <!-- Credentials Grid -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <div
        v-for="credential in credentialsStore.credentials"
        :key="credential.id"
        class="card p-4"
      >
        <div class="flex items-start justify-between">
          <div class="flex items-center gap-3">
            <div class="p-2 bg-primary-100 dark:bg-primary-900/30 rounded-lg">
              <KeyIcon class="w-6 h-6 text-primary-600 dark:text-primary-400" />
            </div>
            <div>
              <h3 class="font-semibold text-slate-900 dark:text-white">
                {{ credential.name }}
              </h3>
              <p class="text-sm text-slate-500 dark:text-slate-400">
                {{ credential.type }}
              </p>
            </div>
          </div>
          <div class="flex items-center gap-1">
            <button
              @click="openEditModal(credential)"
              class="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700"
            >
              <PencilIcon class="w-4 h-4" />
            </button>
            <button
              @click="deleteCredential(credential)"
              class="p-1.5 rounded-lg text-slate-400 hover:text-red-600 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20"
            >
              <TrashIcon class="w-4 h-4" />
            </button>
          </div>
        </div>
        <div class="mt-4 text-xs text-slate-500 dark:text-slate-400">
          Created {{ formatDate(credential.createdAt) }}
        </div>
      </div>
    </div>

    <!-- Empty State -->
    <div
      v-if="credentialsStore.credentials.length === 0 && !credentialsStore.loading"
      class="text-center py-16"
    >
      <KeyIcon class="w-16 h-16 mx-auto text-slate-300 dark:text-slate-600" />
      <h3 class="mt-4 text-lg font-medium text-slate-900 dark:text-white">
        No credentials yet
      </h3>
      <p class="mt-2 text-slate-500 dark:text-slate-400">
        Add credentials to use in your workflows
      </p>
      <button @click="openCreateModal" class="mt-6 btn-primary">
        Add Credential
      </button>
    </div>

    <!-- Loading -->
    <div v-if="credentialsStore.loading" class="text-center py-16">
      <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600 mx-auto" />
    </div>

    <!-- Modal -->
    <TransitionRoot :show="showModal" as="template">
      <Dialog @close="closeModal" class="relative z-50">
        <TransitionChild
          enter="ease-out duration-300"
          enter-from="opacity-0"
          enter-to="opacity-100"
          leave="ease-in duration-200"
          leave-from="opacity-100"
          leave-to="opacity-0"
        >
          <div class="fixed inset-0 bg-black/30 dark:bg-black/50" />
        </TransitionChild>

        <div class="fixed inset-0 flex items-center justify-center p-4">
          <TransitionChild
            enter="ease-out duration-300"
            enter-from="opacity-0 scale-95"
            enter-to="opacity-100 scale-100"
            leave="ease-in duration-200"
            leave-from="opacity-100 scale-100"
            leave-to="opacity-0 scale-95"
          >
            <DialogPanel class="w-full max-w-md bg-white dark:bg-slate-800 rounded-xl shadow-xl">
              <div class="p-6">
                <DialogTitle class="text-lg font-semibold text-slate-900 dark:text-white">
                  {{ editingCredential ? 'Edit Credential' : 'New Credential' }}
                </DialogTitle>

                <form @submit.prevent="saveCredential" class="mt-4 space-y-4">
                  <div>
                    <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                      Name
                    </label>
                    <input
                      v-model="form.name"
                      type="text"
                      class="input"
                      placeholder="My API Key"
                      required
                    />
                  </div>

                  <div>
                    <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                      Type
                    </label>
                    <select v-model="form.type" class="input" required>
                      <option value="">Select type...</option>
                      <option
                        v-for="type in credentialTypes"
                        :key="type.value"
                        :value="type.value"
                      >
                        {{ type.label }}
                      </option>
                    </select>
                  </div>

                  <div class="flex justify-end gap-3 pt-4">
                    <button type="button" @click="closeModal" class="btn-secondary">
                      Cancel
                    </button>
                    <button type="submit" class="btn-primary">
                      {{ editingCredential ? 'Save' : 'Create' }}
                    </button>
                  </div>
                </form>
              </div>
            </DialogPanel>
          </TransitionChild>
        </div>
      </Dialog>
    </TransitionRoot>
  </div>
</template>
