/**
 * Notifications Store
 * Manages toast notifications and alerts
 */

import { defineStore } from 'pinia';
import { ref, computed } from 'vue';

export type NotificationType = 'success' | 'error' | 'warning' | 'info';

export interface Notification {
  id: string;
  type: NotificationType;
  title: string;
  message?: string;
  duration?: number;
  dismissible?: boolean;
  action?: {
    label: string;
    handler: () => void;
  };
  createdAt: Date;
}

export interface NotificationOptions {
  title: string;
  message?: string;
  type?: NotificationType;
  duration?: number;
  dismissible?: boolean;
  action?: {
    label: string;
    handler: () => void;
  };
}

const DEFAULT_DURATION = 5000; // 5 seconds
const MAX_NOTIFICATIONS = 5;

export const useNotificationStore = defineStore('notifications', () => {
  // State
  const notifications = ref<Notification[]>([]);
  const timers = new Map<string, ReturnType<typeof setTimeout>>();

  // Getters
  const activeNotifications = computed(() => notifications.value);
  const hasNotifications = computed(() => notifications.value.length > 0);

  /**
   * Generate unique ID
   */
  function generateId(): string {
    return `notification-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
  }

  /**
   * Add a notification
   */
  function add(options: NotificationOptions): string {
    const id = generateId();

    const notification: Notification = {
      id,
      type: options.type || 'info',
      title: options.title,
      message: options.message,
      duration: options.duration ?? DEFAULT_DURATION,
      dismissible: options.dismissible ?? true,
      action: options.action,
      createdAt: new Date(),
    };

    // Add to list
    notifications.value.push(notification);

    // Remove oldest if exceeding max
    while (notifications.value.length > MAX_NOTIFICATIONS) {
      const oldest = notifications.value[0];
      dismiss(oldest.id);
    }

    // Set auto-dismiss timer if duration is set
    if (notification.duration && notification.duration > 0) {
      const timer = setTimeout(() => {
        dismiss(id);
      }, notification.duration);
      timers.set(id, timer);
    }

    return id;
  }

  /**
   * Dismiss a notification
   */
  function dismiss(id: string): void {
    const index = notifications.value.findIndex((n) => n.id === id);
    if (index !== -1) {
      notifications.value.splice(index, 1);
    }

    // Clear timer if exists
    const timer = timers.get(id);
    if (timer) {
      clearTimeout(timer);
      timers.delete(id);
    }
  }

  /**
   * Dismiss all notifications
   */
  function dismissAll(): void {
    // Clear all timers
    for (const timer of timers.values()) {
      clearTimeout(timer);
    }
    timers.clear();

    // Clear notifications
    notifications.value = [];
  }

  /**
   * Show success notification
   */
  function success(title: string, message?: string, options?: Partial<NotificationOptions>): string {
    return add({
      ...options,
      type: 'success',
      title,
      message,
    });
  }

  /**
   * Show error notification
   */
  function error(title: string, message?: string, options?: Partial<NotificationOptions>): string {
    return add({
      ...options,
      type: 'error',
      title,
      message,
      duration: options?.duration ?? 8000, // Errors stay longer
    });
  }

  /**
   * Show warning notification
   */
  function warning(title: string, message?: string, options?: Partial<NotificationOptions>): string {
    return add({
      ...options,
      type: 'warning',
      title,
      message,
    });
  }

  /**
   * Show info notification
   */
  function info(title: string, message?: string, options?: Partial<NotificationOptions>): string {
    return add({
      ...options,
      type: 'info',
      title,
      message,
    });
  }

  /**
   * Update an existing notification
   */
  function update(id: string, updates: Partial<NotificationOptions>): void {
    const notification = notifications.value.find((n) => n.id === id);
    if (notification) {
      if (updates.title !== undefined) notification.title = updates.title;
      if (updates.message !== undefined) notification.message = updates.message;
      if (updates.type !== undefined) notification.type = updates.type;
      if (updates.action !== undefined) notification.action = updates.action;

      // Update timer if duration changed
      if (updates.duration !== undefined) {
        notification.duration = updates.duration;

        // Clear existing timer
        const existingTimer = timers.get(id);
        if (existingTimer) {
          clearTimeout(existingTimer);
          timers.delete(id);
        }

        // Set new timer if duration > 0
        if (updates.duration > 0) {
          const timer = setTimeout(() => {
            dismiss(id);
          }, updates.duration);
          timers.set(id, timer);
        }
      }
    }
  }

  return {
    // State
    notifications,

    // Getters
    activeNotifications,
    hasNotifications,

    // Actions
    add,
    dismiss,
    dismissAll,
    success,
    error,
    warning,
    info,
    update,
  };
});

export default useNotificationStore;
