<script setup lang="ts">
/**
 * Agent Copilot Component
 * AI-powered workflow assistance for m9m
 */
import { ref, computed, watch } from 'vue';
import {
  SparklesIcon,
  PaperAirplaneIcon,
  XMarkIcon,
  LightBulbIcon,
  WrenchScrewdriverIcon,
  ChatBubbleLeftRightIcon,
  DocumentTextIcon,
  ChevronDownIcon,
  ChevronUpIcon,
} from '@heroicons/vue/24/outline';

const props = defineProps<{
  workflow?: object;
  isOpen?: boolean;
}>();

const emit = defineEmits<{
  close: [];
  applyWorkflow: [workflow: object];
  addNode: [node: object];
}>();

// State
const activeTab = ref<'chat' | 'generate' | 'suggest' | 'explain'>('chat');
const isExpanded = ref(true);
const isLoading = ref(false);
const chatInput = ref('');
const generateInput = ref('');

// Chat messages
interface ChatMessage {
  role: 'user' | 'assistant';
  content: string;
  timestamp: Date;
  actions?: Array<{ type: string; description: string; data: object }>;
}

const chatMessages = ref<ChatMessage[]>([
  {
    role: 'assistant',
    content: 'Hi! I\'m your Agent Copilot. I can help you build workflows, suggest nodes, explain your workflow, or fix errors. What would you like to do?',
    timestamp: new Date(),
  },
]);

// Suggestions
const nodeSuggestions = ref<Array<{
  type: string;
  name: string;
  description: string;
  reason: string;
  confidence: number;
}>>([]);

// Generated workflow
const generatedWorkflow = ref<object | null>(null);
const generationExplanation = ref('');

// API base URL
const apiBase = '/api/v1';

// Send chat message
async function sendChatMessage() {
  if (!chatInput.value.trim() || isLoading.value) return;

  const userMessage = chatInput.value.trim();
  chatInput.value = '';

  // Add user message
  chatMessages.value.push({
    role: 'user',
    content: userMessage,
    timestamp: new Date(),
  });

  isLoading.value = true;

  try {
    const response = await fetch(`${apiBase}/copilot/chat`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        messages: chatMessages.value.map((m) => ({
          role: m.role,
          content: m.content,
        })),
        currentWorkflow: props.workflow,
      }),
    });

    const data = await response.json();

    chatMessages.value.push({
      role: 'assistant',
      content: data.message,
      timestamp: new Date(),
      actions: data.actions,
    });
  } catch (error) {
    chatMessages.value.push({
      role: 'assistant',
      content: 'Sorry, I encountered an error. Please try again.',
      timestamp: new Date(),
    });
  } finally {
    isLoading.value = false;
  }
}

// Generate workflow
async function generateWorkflow() {
  if (!generateInput.value.trim() || isLoading.value) return;

  isLoading.value = true;

  try {
    const response = await fetch(`${apiBase}/copilot/generate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        description: generateInput.value,
      }),
    });

    const data = await response.json();
    generatedWorkflow.value = data.workflow;
    generationExplanation.value = data.explanation;
  } catch (error) {
    generationExplanation.value = 'Failed to generate workflow. Please try again.';
  } finally {
    isLoading.value = false;
  }
}

// Get node suggestions
async function getSuggestions() {
  isLoading.value = true;

  try {
    const response = await fetch(`${apiBase}/copilot/suggest`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        currentWorkflow: props.workflow,
        userQuery: 'Suggest next nodes',
      }),
    });

    const data = await response.json();
    nodeSuggestions.value = data.suggestions || [];
  } catch (error) {
    nodeSuggestions.value = [];
  } finally {
    isLoading.value = false;
  }
}

// Apply generated workflow
function applyGeneratedWorkflow() {
  if (generatedWorkflow.value) {
    emit('applyWorkflow', generatedWorkflow.value);
  }
}

// Add suggested node
function addSuggestedNode(suggestion: typeof nodeSuggestions.value[0]) {
  emit('addNode', {
    type: suggestion.type,
    name: suggestion.name,
    parameters: {},
  });
}

// Execute action from chat
function executeAction(action: { type: string; description: string; data: object }) {
  switch (action.type) {
    case 'add_node':
      emit('addNode', action.data);
      break;
    case 'apply_workflow':
      emit('applyWorkflow', action.data);
      break;
  }
}

// Load suggestions when tab changes
watch(activeTab, (tab) => {
  if (tab === 'suggest') {
    getSuggestions();
  }
});

// Keyboard shortcut
function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault();
    if (activeTab.value === 'chat') {
      sendChatMessage();
    } else if (activeTab.value === 'generate') {
      generateWorkflow();
    }
  }
}
</script>

<template>
  <div
    class="fixed bottom-4 right-4 w-96 bg-white dark:bg-slate-800 rounded-xl shadow-2xl border border-slate-200 dark:border-slate-700 overflow-hidden z-50"
    :class="{ 'h-[600px]': isExpanded, 'h-14': !isExpanded }"
  >
    <!-- Header -->
    <div
      class="flex items-center justify-between px-4 py-3 bg-gradient-to-r from-violet-600 to-indigo-600 text-white cursor-pointer"
      @click="isExpanded = !isExpanded"
    >
      <div class="flex items-center gap-2">
        <SparklesIcon class="w-5 h-5" />
        <span class="font-semibold">Agent Copilot</span>
        <span class="text-xs bg-white/20 px-2 py-0.5 rounded-full">AI</span>
      </div>
      <div class="flex items-center gap-2">
        <button
          @click.stop="isExpanded = !isExpanded"
          class="p-1 hover:bg-white/20 rounded"
        >
          <component :is="isExpanded ? ChevronDownIcon : ChevronUpIcon" class="w-4 h-4" />
        </button>
        <button
          @click.stop="emit('close')"
          class="p-1 hover:bg-white/20 rounded"
        >
          <XMarkIcon class="w-4 h-4" />
        </button>
      </div>
    </div>

    <!-- Content -->
    <div v-if="isExpanded" class="flex flex-col h-[calc(100%-56px)]">
      <!-- Tabs -->
      <div class="flex border-b border-slate-200 dark:border-slate-700">
        <button
          v-for="tab in [
            { id: 'chat', label: 'Chat', icon: ChatBubbleLeftRightIcon },
            { id: 'generate', label: 'Generate', icon: SparklesIcon },
            { id: 'suggest', label: 'Suggest', icon: LightBulbIcon },
            { id: 'explain', label: 'Explain', icon: DocumentTextIcon },
          ]"
          :key="tab.id"
          @click="activeTab = tab.id as typeof activeTab"
          class="flex-1 flex items-center justify-center gap-1.5 py-2.5 text-sm font-medium transition-colors"
          :class="
            activeTab === tab.id
              ? 'text-violet-600 dark:text-violet-400 border-b-2 border-violet-600 dark:border-violet-400'
              : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300'
          "
        >
          <component :is="tab.icon" class="w-4 h-4" />
          {{ tab.label }}
        </button>
      </div>

      <!-- Tab Content -->
      <div class="flex-1 overflow-hidden">
        <!-- Chat Tab -->
        <div v-if="activeTab === 'chat'" class="flex flex-col h-full">
          <!-- Messages -->
          <div class="flex-1 overflow-y-auto p-4 space-y-4">
            <div
              v-for="(message, index) in chatMessages"
              :key="index"
              class="flex"
              :class="message.role === 'user' ? 'justify-end' : 'justify-start'"
            >
              <div
                class="max-w-[85%] rounded-lg px-3 py-2 text-sm"
                :class="
                  message.role === 'user'
                    ? 'bg-violet-600 text-white'
                    : 'bg-slate-100 dark:bg-slate-700 text-slate-900 dark:text-slate-100'
                "
              >
                <p class="whitespace-pre-wrap">{{ message.content }}</p>

                <!-- Actions -->
                <div v-if="message.actions?.length" class="mt-2 space-y-1">
                  <button
                    v-for="(action, i) in message.actions"
                    :key="i"
                    @click="executeAction(action)"
                    class="w-full text-left px-2 py-1 rounded bg-violet-500/20 hover:bg-violet-500/30 text-xs"
                  >
                    {{ action.description }}
                  </button>
                </div>
              </div>
            </div>

            <!-- Loading indicator -->
            <div v-if="isLoading" class="flex justify-start">
              <div class="bg-slate-100 dark:bg-slate-700 rounded-lg px-4 py-2">
                <div class="flex items-center gap-2 text-sm text-slate-500">
                  <div class="animate-pulse">Thinking...</div>
                </div>
              </div>
            </div>
          </div>

          <!-- Input -->
          <div class="p-3 border-t border-slate-200 dark:border-slate-700">
            <div class="flex items-center gap-2">
              <input
                v-model="chatInput"
                @keydown="handleKeydown"
                type="text"
                placeholder="Ask me anything about workflows..."
                class="flex-1 px-3 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-900 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-violet-500"
              />
              <button
                @click="sendChatMessage"
                :disabled="isLoading || !chatInput.trim()"
                class="p-2 rounded-lg bg-violet-600 text-white hover:bg-violet-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <PaperAirplaneIcon class="w-4 h-4" />
              </button>
            </div>
          </div>
        </div>

        <!-- Generate Tab -->
        <div v-else-if="activeTab === 'generate'" class="flex flex-col h-full p-4">
          <div class="space-y-4">
            <div>
              <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                Describe your workflow
              </label>
              <textarea
                v-model="generateInput"
                @keydown="handleKeydown"
                rows="4"
                placeholder="e.g., When I receive a webhook, send the data to Slack and save it to a database..."
                class="w-full px-3 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-900 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-violet-500 resize-none"
              ></textarea>
            </div>

            <button
              @click="generateWorkflow"
              :disabled="isLoading || !generateInput.trim()"
              class="w-full py-2 rounded-lg bg-violet-600 text-white font-medium hover:bg-violet-700 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
            >
              <SparklesIcon class="w-4 h-4" />
              {{ isLoading ? 'Generating...' : 'Generate Workflow' }}
            </button>

            <!-- Generated Result -->
            <div v-if="generatedWorkflow" class="mt-4 p-3 bg-slate-50 dark:bg-slate-900 rounded-lg">
              <p class="text-sm text-slate-600 dark:text-slate-400 mb-3">
                {{ generationExplanation }}
              </p>
              <button
                @click="applyGeneratedWorkflow"
                class="w-full py-2 rounded-lg bg-green-600 text-white font-medium hover:bg-green-700 flex items-center justify-center gap-2"
              >
                Apply Workflow
              </button>
            </div>
          </div>
        </div>

        <!-- Suggest Tab -->
        <div v-else-if="activeTab === 'suggest'" class="h-full overflow-y-auto p-4">
          <div v-if="isLoading" class="flex items-center justify-center h-32">
            <div class="text-slate-500">Loading suggestions...</div>
          </div>

          <div v-else-if="nodeSuggestions.length" class="space-y-3">
            <div
              v-for="(suggestion, index) in nodeSuggestions"
              :key="index"
              class="p-3 bg-slate-50 dark:bg-slate-900 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 cursor-pointer transition-colors"
              @click="addSuggestedNode(suggestion)"
            >
              <div class="flex items-start justify-between">
                <div>
                  <h4 class="font-medium text-slate-900 dark:text-slate-100">
                    {{ suggestion.name }}
                  </h4>
                  <p class="text-sm text-slate-500 dark:text-slate-400 mt-0.5">
                    {{ suggestion.description }}
                  </p>
                  <p class="text-xs text-violet-600 dark:text-violet-400 mt-1">
                    {{ suggestion.reason }}
                  </p>
                </div>
                <span
                  class="text-xs font-medium px-2 py-0.5 rounded-full"
                  :class="
                    suggestion.confidence > 0.8
                      ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                      : 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
                  "
                >
                  {{ Math.round(suggestion.confidence * 100) }}%
                </span>
              </div>
            </div>
          </div>

          <div v-else class="text-center py-8 text-slate-500">
            <LightBulbIcon class="w-12 h-12 mx-auto mb-3 opacity-50" />
            <p>No suggestions available</p>
            <p class="text-sm">Build more of your workflow to get suggestions</p>
          </div>
        </div>

        <!-- Explain Tab -->
        <div v-else-if="activeTab === 'explain'" class="h-full overflow-y-auto p-4">
          <div class="text-center py-8 text-slate-500">
            <DocumentTextIcon class="w-12 h-12 mx-auto mb-3 opacity-50" />
            <p>Select a workflow to explain</p>
            <p class="text-sm mt-1">The copilot will analyze and explain your workflow</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
