<script setup lang="ts">
/**
 * Templates Hub Component
 * Curated workflow templates for quick start
 */
import { ref, computed, onMounted } from 'vue';
import {
  MagnifyingGlassIcon,
  FunnelIcon,
  RocketLaunchIcon,
  ClockIcon,
  GlobeAltIcon,
  CircleStackIcon,
  ChatBubbleLeftRightIcon,
  EnvelopeIcon,
  CpuChipIcon,
  CloudIcon,
  DocumentDuplicateIcon,
  ArrowTopRightOnSquareIcon,
} from '@heroicons/vue/24/outline';

const emit = defineEmits<{
  useTemplate: [template: Template];
  close: [];
}>();

interface Template {
  id: string;
  name: string;
  description: string;
  category: string;
  tags: string[];
  nodeCount: number;
  usageCount: number;
  author: string;
  featured: boolean;
}

// State
const searchQuery = ref('');
const selectedCategory = ref('all');
const isLoading = ref(false);

// Categories
const categories = [
  { id: 'all', name: 'All Templates', icon: DocumentDuplicateIcon },
  { id: 'trigger', name: 'Triggers', icon: RocketLaunchIcon },
  { id: 'schedule', name: 'Scheduled', icon: ClockIcon },
  { id: 'api', name: 'API & HTTP', icon: GlobeAltIcon },
  { id: 'database', name: 'Database', icon: CircleStackIcon },
  { id: 'messaging', name: 'Messaging', icon: ChatBubbleLeftRightIcon },
  { id: 'email', name: 'Email', icon: EnvelopeIcon },
  { id: 'ai', name: 'AI & ML', icon: CpuChipIcon },
  { id: 'cloud', name: 'Cloud', icon: CloudIcon },
];

// Sample templates (would be fetched from API in production)
const templates = ref<Template[]>([
  {
    id: 'http-webhook',
    name: 'HTTP Webhook Handler',
    description: 'Receive and process incoming HTTP webhooks with validation and routing.',
    category: 'trigger',
    tags: ['webhook', 'http', 'api', 'automation'],
    nodeCount: 3,
    usageCount: 12500,
    author: 'm9m',
    featured: true,
  },
  {
    id: 'scheduled-backup',
    name: 'Scheduled Database Backup',
    description: 'Automatically backup your database on a schedule and upload to cloud storage.',
    category: 'schedule',
    tags: ['backup', 'database', 'cron', 's3'],
    nodeCount: 5,
    usageCount: 8700,
    author: 'm9m',
    featured: true,
  },
  {
    id: 'slack-notifications',
    name: 'Slack Alert Pipeline',
    description: 'Send custom notifications to Slack channels based on workflow events.',
    category: 'messaging',
    tags: ['slack', 'notifications', 'alerts'],
    nodeCount: 4,
    usageCount: 15200,
    author: 'm9m',
    featured: true,
  },
  {
    id: 'email-automation',
    name: 'Email Marketing Automation',
    description: 'Automated email sequences with personalization and tracking.',
    category: 'email',
    tags: ['email', 'marketing', 'automation'],
    nodeCount: 6,
    usageCount: 9300,
    author: 'm9m',
    featured: false,
  },
  {
    id: 'ai-content',
    name: 'AI Content Generator',
    description: 'Generate content using AI with custom prompts and templates.',
    category: 'ai',
    tags: ['ai', 'gpt', 'content', 'generation'],
    nodeCount: 4,
    usageCount: 21000,
    author: 'm9m',
    featured: true,
  },
  {
    id: 'api-aggregator',
    name: 'Multi-API Data Aggregator',
    description: 'Fetch data from multiple APIs, merge, and transform for unified output.',
    category: 'api',
    tags: ['api', 'aggregation', 'transform'],
    nodeCount: 7,
    usageCount: 6800,
    author: 'm9m',
    featured: false,
  },
  {
    id: 'postgres-sync',
    name: 'PostgreSQL Data Sync',
    description: 'Sync data between PostgreSQL databases with conflict resolution.',
    category: 'database',
    tags: ['postgres', 'sync', 'database'],
    nodeCount: 5,
    usageCount: 4200,
    author: 'm9m',
    featured: false,
  },
  {
    id: 'cloud-monitor',
    name: 'Cloud Resource Monitor',
    description: 'Monitor cloud resources and trigger alerts based on thresholds.',
    category: 'cloud',
    tags: ['aws', 'monitoring', 'alerts'],
    nodeCount: 6,
    usageCount: 3900,
    author: 'm9m',
    featured: false,
  },
  {
    id: 'discord-bot',
    name: 'Discord Bot Commands',
    description: 'Handle Discord bot commands with custom responses and actions.',
    category: 'messaging',
    tags: ['discord', 'bot', 'commands'],
    nodeCount: 4,
    usageCount: 11500,
    author: 'm9m',
    featured: false,
  },
  {
    id: 'ai-chatbot',
    name: 'AI Customer Support Bot',
    description: 'AI-powered chatbot for customer support with context memory.',
    category: 'ai',
    tags: ['ai', 'chatbot', 'support', 'claude'],
    nodeCount: 8,
    usageCount: 18700,
    author: 'm9m',
    featured: true,
  },
]);

// Filtered templates
const filteredTemplates = computed(() => {
  return templates.value.filter((template) => {
    const matchesCategory =
      selectedCategory.value === 'all' || template.category === selectedCategory.value;
    const matchesSearch =
      !searchQuery.value ||
      template.name.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
      template.description.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
      template.tags.some((tag) => tag.toLowerCase().includes(searchQuery.value.toLowerCase()));

    return matchesCategory && matchesSearch;
  });
});

// Featured templates
const featuredTemplates = computed(() => {
  return templates.value.filter((t) => t.featured).slice(0, 4);
});

// Use template
async function useTemplate(template: Template) {
  emit('useTemplate', template);
}

// Format usage count
function formatUsageCount(count: number): string {
  if (count >= 1000) {
    return `${(count / 1000).toFixed(1)}k`;
  }
  return count.toString();
}
</script>

<template>
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
    <div class="bg-white dark:bg-slate-800 rounded-xl shadow-2xl w-full max-w-5xl max-h-[90vh] overflow-hidden flex flex-col">
      <!-- Header -->
      <div class="px-6 py-4 border-b border-slate-200 dark:border-slate-700">
        <div class="flex items-center justify-between">
          <div>
            <h2 class="text-xl font-semibold text-slate-900 dark:text-white">
              Templates Hub
            </h2>
            <p class="text-sm text-slate-500 dark:text-slate-400 mt-1">
              Start faster with pre-built workflow templates
            </p>
          </div>
          <button
            @click="emit('close')"
            class="p-2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-700"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <!-- Search -->
        <div class="mt-4 flex items-center gap-4">
          <div class="flex-1 relative">
            <MagnifyingGlassIcon class="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-400" />
            <input
              v-model="searchQuery"
              type="text"
              placeholder="Search templates..."
              class="w-full pl-10 pr-4 py-2 rounded-lg border border-slate-200 dark:border-slate-600 bg-slate-50 dark:bg-slate-900 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"
            />
          </div>
        </div>
      </div>

      <div class="flex flex-1 overflow-hidden">
        <!-- Sidebar Categories -->
        <div class="w-48 border-r border-slate-200 dark:border-slate-700 p-4 overflow-y-auto">
          <h3 class="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider mb-3">
            Categories
          </h3>
          <nav class="space-y-1">
            <button
              v-for="category in categories"
              :key="category.id"
              @click="selectedCategory = category.id"
              class="w-full flex items-center gap-2 px-3 py-2 text-sm rounded-lg transition-colors"
              :class="
                selectedCategory === category.id
                  ? 'bg-violet-100 dark:bg-violet-900/30 text-violet-700 dark:text-violet-300'
                  : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700'
              "
            >
              <component :is="category.icon" class="w-4 h-4" />
              {{ category.name }}
            </button>
          </nav>
        </div>

        <!-- Templates Grid -->
        <div class="flex-1 overflow-y-auto p-6">
          <!-- Featured Section -->
          <div v-if="selectedCategory === 'all' && !searchQuery" class="mb-8">
            <h3 class="text-lg font-semibold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
              <RocketLaunchIcon class="w-5 h-5 text-violet-500" />
              Featured Templates
            </h3>
            <div class="grid grid-cols-2 gap-4">
              <div
                v-for="template in featuredTemplates"
                :key="template.id"
                class="p-4 bg-gradient-to-br from-violet-50 to-indigo-50 dark:from-violet-900/20 dark:to-indigo-900/20 rounded-xl border border-violet-200 dark:border-violet-800 hover:shadow-lg transition-shadow cursor-pointer"
                @click="useTemplate(template)"
              >
                <div class="flex items-start justify-between">
                  <div>
                    <h4 class="font-semibold text-slate-900 dark:text-white">
                      {{ template.name }}
                    </h4>
                    <p class="text-sm text-slate-600 dark:text-slate-400 mt-1 line-clamp-2">
                      {{ template.description }}
                    </p>
                  </div>
                  <span class="text-xs bg-violet-100 dark:bg-violet-800 text-violet-700 dark:text-violet-300 px-2 py-0.5 rounded-full">
                    Featured
                  </span>
                </div>
                <div class="mt-3 flex items-center gap-4 text-xs text-slate-500 dark:text-slate-400">
                  <span>{{ template.nodeCount }} nodes</span>
                  <span>{{ formatUsageCount(template.usageCount) }} uses</span>
                </div>
              </div>
            </div>
          </div>

          <!-- All Templates -->
          <div>
            <h3 class="text-lg font-semibold text-slate-900 dark:text-white mb-4">
              {{ selectedCategory === 'all' ? 'All Templates' : categories.find(c => c.id === selectedCategory)?.name }}
              <span class="text-sm font-normal text-slate-500 dark:text-slate-400 ml-2">
                ({{ filteredTemplates.length }})
              </span>
            </h3>

            <div v-if="filteredTemplates.length" class="grid grid-cols-2 gap-4">
              <div
                v-for="template in filteredTemplates"
                :key="template.id"
                class="p-4 bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-700 hover:shadow-lg hover:border-violet-300 dark:hover:border-violet-700 transition-all cursor-pointer group"
                @click="useTemplate(template)"
              >
                <div class="flex items-start justify-between">
                  <h4 class="font-semibold text-slate-900 dark:text-white group-hover:text-violet-600 dark:group-hover:text-violet-400 transition-colors">
                    {{ template.name }}
                  </h4>
                  <ArrowTopRightOnSquareIcon class="w-4 h-4 text-slate-400 opacity-0 group-hover:opacity-100 transition-opacity" />
                </div>
                <p class="text-sm text-slate-600 dark:text-slate-400 mt-1 line-clamp-2">
                  {{ template.description }}
                </p>
                <div class="mt-3 flex flex-wrap gap-1">
                  <span
                    v-for="tag in template.tags.slice(0, 3)"
                    :key="tag"
                    class="text-xs bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400 px-2 py-0.5 rounded"
                  >
                    {{ tag }}
                  </span>
                </div>
                <div class="mt-3 flex items-center justify-between text-xs text-slate-500 dark:text-slate-400">
                  <span>{{ template.nodeCount }} nodes</span>
                  <span>{{ formatUsageCount(template.usageCount) }} uses</span>
                </div>
              </div>
            </div>

            <div v-else class="text-center py-12 text-slate-500 dark:text-slate-400">
              <DocumentDuplicateIcon class="w-12 h-12 mx-auto mb-3 opacity-50" />
              <p class="font-medium">No templates found</p>
              <p class="text-sm mt-1">Try adjusting your search or category</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
