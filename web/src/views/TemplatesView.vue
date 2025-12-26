<script setup lang="ts">
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import TemplatesHub from '@/components/templates/TemplatesHub.vue';

const router = useRouter();
const showHub = ref(true);

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

async function handleUseTemplate(template: Template) {
  // Apply template via API and navigate to editor
  try {
    const response = await fetch(`/api/v1/templates/${template.id}/apply`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
    });

    if (response.ok) {
      const data = await response.json();
      if (data.workflowId) {
        router.push(`/workflows/${data.workflowId}`);
      } else {
        router.push('/workflows/new');
      }
    }
  } catch (error) {
    console.error('Failed to apply template:', error);
    router.push('/workflows/new');
  }
}

function handleClose() {
  router.push('/workflows');
}
</script>

<template>
  <div class="h-full">
    <TemplatesHub
      v-if="showHub"
      @use-template="handleUseTemplate"
      @close="handleClose"
    />
  </div>
</template>
