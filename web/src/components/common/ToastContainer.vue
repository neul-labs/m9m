<script setup lang="ts">
/**
 * Toast Container Component
 * Renders all active toast notifications
 */
import { computed } from 'vue';
import { useNotificationStore } from '@/stores/notifications';
import Toast from './Toast.vue';

// Position options
type Position = 'top-right' | 'top-left' | 'bottom-right' | 'bottom-left' | 'top-center' | 'bottom-center';

const props = withDefaults(
  defineProps<{
    position?: Position;
  }>(),
  {
    position: 'top-right',
  }
);

const notificationStore = useNotificationStore();

const notifications = computed(() => notificationStore.activeNotifications);

// Position classes
const positionClasses = computed(() => {
  const positions: Record<Position, string> = {
    'top-right': 'top-4 right-4',
    'top-left': 'top-4 left-4',
    'bottom-right': 'bottom-4 right-4',
    'bottom-left': 'bottom-4 left-4',
    'top-center': 'top-4 left-1/2 -translate-x-1/2',
    'bottom-center': 'bottom-4 left-1/2 -translate-x-1/2',
  };

  return positions[props.position];
});

function handleDismiss(id: string) {
  notificationStore.dismiss(id);
}
</script>

<template>
  <Teleport to="body">
    <div
      :class="[
        'fixed z-50 flex flex-col gap-3 pointer-events-none',
        positionClasses,
      ]"
    >
      <TransitionGroup
        name="toast"
        tag="div"
        class="flex flex-col gap-3"
      >
        <div
          v-for="notification in notifications"
          :key="notification.id"
          class="pointer-events-auto"
        >
          <Toast
            :notification="notification"
            @dismiss="handleDismiss"
          />
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<style scoped>
.toast-enter-active {
  transition: all 0.3s ease-out;
}

.toast-leave-active {
  transition: all 0.2s ease-in;
}

.toast-enter-from {
  opacity: 0;
  transform: translateX(100%);
}

.toast-leave-to {
  opacity: 0;
  transform: translateX(100%);
}

.toast-move {
  transition: transform 0.3s ease;
}
</style>
