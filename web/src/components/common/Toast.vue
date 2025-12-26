<script setup lang="ts">
/**
 * Toast Component
 * Displays toast notifications with animations and actions
 */
import { computed } from 'vue';
import {
  CheckCircleIcon,
  ExclamationCircleIcon,
  ExclamationTriangleIcon,
  InformationCircleIcon,
  XMarkIcon,
} from '@heroicons/vue/24/outline';
import type { Notification, NotificationType } from '@/stores/notifications';

const props = defineProps<{
  notification: Notification;
}>();

const emit = defineEmits<{
  dismiss: [id: string];
}>();

// Icon based on notification type
const icons: Record<NotificationType, typeof CheckCircleIcon> = {
  success: CheckCircleIcon,
  error: ExclamationCircleIcon,
  warning: ExclamationTriangleIcon,
  info: InformationCircleIcon,
};

const icon = computed(() => icons[props.notification.type]);

// Styles based on notification type
const styles = computed(() => {
  const baseStyles = {
    success: {
      bg: 'bg-green-50 dark:bg-green-900/20',
      border: 'border-green-200 dark:border-green-800',
      icon: 'text-green-500 dark:text-green-400',
      title: 'text-green-800 dark:text-green-200',
      message: 'text-green-700 dark:text-green-300',
    },
    error: {
      bg: 'bg-red-50 dark:bg-red-900/20',
      border: 'border-red-200 dark:border-red-800',
      icon: 'text-red-500 dark:text-red-400',
      title: 'text-red-800 dark:text-red-200',
      message: 'text-red-700 dark:text-red-300',
    },
    warning: {
      bg: 'bg-amber-50 dark:bg-amber-900/20',
      border: 'border-amber-200 dark:border-amber-800',
      icon: 'text-amber-500 dark:text-amber-400',
      title: 'text-amber-800 dark:text-amber-200',
      message: 'text-amber-700 dark:text-amber-300',
    },
    info: {
      bg: 'bg-blue-50 dark:bg-blue-900/20',
      border: 'border-blue-200 dark:border-blue-800',
      icon: 'text-blue-500 dark:text-blue-400',
      title: 'text-blue-800 dark:text-blue-200',
      message: 'text-blue-700 dark:text-blue-300',
    },
  };

  return baseStyles[props.notification.type];
});

function handleDismiss() {
  emit('dismiss', props.notification.id);
}

function handleAction() {
  if (props.notification.action?.handler) {
    props.notification.action.handler();
  }
  handleDismiss();
}
</script>

<template>
  <div
    class="w-full max-w-sm overflow-hidden rounded-lg shadow-lg ring-1 ring-black ring-opacity-5 border transition-all duration-300 ease-out"
    :class="[styles.bg, styles.border]"
  >
    <div class="p-4">
      <div class="flex items-start">
        <!-- Icon -->
        <div class="flex-shrink-0">
          <component
            :is="icon"
            class="h-5 w-5"
            :class="styles.icon"
            aria-hidden="true"
          />
        </div>

        <!-- Content -->
        <div class="ml-3 w-0 flex-1">
          <p class="text-sm font-medium" :class="styles.title">
            {{ notification.title }}
          </p>
          <p
            v-if="notification.message"
            class="mt-1 text-sm"
            :class="styles.message"
          >
            {{ notification.message }}
          </p>

          <!-- Action button -->
          <div v-if="notification.action" class="mt-3">
            <button
              type="button"
              class="text-sm font-medium hover:underline focus:outline-none focus:ring-2 focus:ring-offset-2 rounded"
              :class="styles.title"
              @click="handleAction"
            >
              {{ notification.action.label }}
            </button>
          </div>
        </div>

        <!-- Dismiss button -->
        <div v-if="notification.dismissible" class="ml-4 flex flex-shrink-0">
          <button
            type="button"
            class="inline-flex rounded-md focus:outline-none focus:ring-2 focus:ring-offset-2 opacity-60 hover:opacity-100 transition-opacity"
            :class="styles.icon"
            @click="handleDismiss"
          >
            <span class="sr-only">Dismiss</span>
            <XMarkIcon class="h-5 w-5" aria-hidden="true" />
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
