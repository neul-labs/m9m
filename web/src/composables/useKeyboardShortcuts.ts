/**
 * Keyboard Shortcuts Composable
 * Provides global keyboard shortcut handling for Vue applications
 */

import { onMounted, onUnmounted, ref, type Ref } from 'vue';

export interface KeyboardShortcut {
  key: string;
  modifiers?: {
    ctrl?: boolean;
    alt?: boolean;
    shift?: boolean;
    meta?: boolean;
  };
  handler: (event: KeyboardEvent) => void;
  description?: string;
  category?: string;
  enabled?: boolean | (() => boolean);
  preventDefault?: boolean;
  stopPropagation?: boolean;
}

export interface ShortcutRegistry {
  [key: string]: KeyboardShortcut;
}

// Global shortcut registry
const globalShortcuts: Ref<ShortcutRegistry> = ref({});
let isInitialized = false;

/**
 * Create a unique key for a shortcut
 */
function createShortcutKey(shortcut: Omit<KeyboardShortcut, 'handler'>): string {
  const parts: string[] = [];
  if (shortcut.modifiers?.ctrl) parts.push('ctrl');
  if (shortcut.modifiers?.alt) parts.push('alt');
  if (shortcut.modifiers?.shift) parts.push('shift');
  if (shortcut.modifiers?.meta) parts.push('meta');
  parts.push(shortcut.key.toLowerCase());
  return parts.join('+');
}

/**
 * Parse a shortcut string like "Ctrl+S" into components
 */
export function parseShortcut(shortcutString: string): Omit<KeyboardShortcut, 'handler'> {
  const parts = shortcutString.toLowerCase().split('+');
  const key = parts.pop() || '';

  return {
    key,
    modifiers: {
      ctrl: parts.includes('ctrl') || parts.includes('control'),
      alt: parts.includes('alt'),
      shift: parts.includes('shift'),
      meta: parts.includes('meta') || parts.includes('cmd') || parts.includes('command'),
    },
  };
}

/**
 * Format a shortcut for display
 */
export function formatShortcut(shortcut: Omit<KeyboardShortcut, 'handler'>): string {
  const parts: string[] = [];
  const isMac = navigator.platform.toLowerCase().includes('mac');

  if (shortcut.modifiers?.ctrl) parts.push(isMac ? '⌃' : 'Ctrl');
  if (shortcut.modifiers?.alt) parts.push(isMac ? '⌥' : 'Alt');
  if (shortcut.modifiers?.shift) parts.push(isMac ? '⇧' : 'Shift');
  if (shortcut.modifiers?.meta) parts.push(isMac ? '⌘' : 'Win');

  // Format special keys
  const keyDisplay = {
    'escape': isMac ? '⎋' : 'Esc',
    'enter': isMac ? '↩' : 'Enter',
    'backspace': isMac ? '⌫' : 'Backspace',
    'delete': isMac ? '⌦' : 'Delete',
    'arrowup': '↑',
    'arrowdown': '↓',
    'arrowleft': '←',
    'arrowright': '→',
    'tab': isMac ? '⇥' : 'Tab',
    ' ': 'Space',
  }[shortcut.key.toLowerCase()] || shortcut.key.toUpperCase();

  parts.push(keyDisplay);
  return parts.join(isMac ? '' : '+');
}

/**
 * Check if a keyboard event matches a shortcut
 */
function matchesShortcut(event: KeyboardEvent, shortcut: KeyboardShortcut): boolean {
  // Check key match
  if (event.key.toLowerCase() !== shortcut.key.toLowerCase()) {
    return false;
  }

  // Check modifiers
  const modifiers = shortcut.modifiers || {};
  if (!!modifiers.ctrl !== event.ctrlKey) return false;
  if (!!modifiers.alt !== event.altKey) return false;
  if (!!modifiers.shift !== event.shiftKey) return false;
  if (!!modifiers.meta !== event.metaKey) return false;

  return true;
}

/**
 * Check if the event target is an input element
 */
function isInputElement(target: EventTarget | null): boolean {
  if (!target || !(target instanceof HTMLElement)) return false;

  const tagName = target.tagName.toLowerCase();
  if (tagName === 'input' || tagName === 'textarea' || tagName === 'select') {
    return true;
  }

  // Check for contenteditable
  if (target.isContentEditable) {
    return true;
  }

  return false;
}

/**
 * Global keyboard event handler
 */
function handleKeyDown(event: KeyboardEvent): void {
  // Skip if inside input and not a global shortcut
  const inInput = isInputElement(event.target);

  for (const shortcut of Object.values(globalShortcuts.value)) {
    // Skip disabled shortcuts
    if (shortcut.enabled !== undefined) {
      const isEnabled = typeof shortcut.enabled === 'function'
        ? shortcut.enabled()
        : shortcut.enabled;
      if (!isEnabled) continue;
    }

    if (matchesShortcut(event, shortcut)) {
      // For most shortcuts, skip if in input (unless it's Escape)
      if (inInput && shortcut.key.toLowerCase() !== 'escape') {
        continue;
      }

      if (shortcut.preventDefault !== false) {
        event.preventDefault();
      }
      if (shortcut.stopPropagation) {
        event.stopPropagation();
      }

      shortcut.handler(event);
      return;
    }
  }
}

/**
 * Initialize global keyboard listener
 */
function initGlobalListener(): void {
  if (isInitialized) return;

  document.addEventListener('keydown', handleKeyDown);
  isInitialized = true;
}

/**
 * Cleanup global keyboard listener
 */
function cleanupGlobalListener(): void {
  if (!isInitialized) return;
  if (Object.keys(globalShortcuts.value).length > 0) return;

  document.removeEventListener('keydown', handleKeyDown);
  isInitialized = false;
}

/**
 * Main composable for keyboard shortcuts
 */
export function useKeyboardShortcuts() {
  const localShortcuts: string[] = [];

  /**
   * Register a keyboard shortcut
   */
  function register(
    shortcutString: string,
    handler: (event: KeyboardEvent) => void,
    options: Partial<Omit<KeyboardShortcut, 'key' | 'modifiers' | 'handler'>> = {}
  ): void {
    const parsed = parseShortcut(shortcutString);
    const key = createShortcutKey(parsed);

    globalShortcuts.value[key] = {
      ...parsed,
      handler,
      ...options,
    };

    localShortcuts.push(key);
    initGlobalListener();
  }

  /**
   * Unregister a keyboard shortcut
   */
  function unregister(shortcutString: string): void {
    const parsed = parseShortcut(shortcutString);
    const key = createShortcutKey(parsed);

    delete globalShortcuts.value[key];
    const idx = localShortcuts.indexOf(key);
    if (idx !== -1) {
      localShortcuts.splice(idx, 1);
    }
  }

  /**
   * Get all registered shortcuts
   */
  function getShortcuts(): ShortcutRegistry {
    return { ...globalShortcuts.value };
  }

  /**
   * Get shortcuts by category
   */
  function getShortcutsByCategory(): Record<string, KeyboardShortcut[]> {
    const byCategory: Record<string, KeyboardShortcut[]> = {};

    for (const shortcut of Object.values(globalShortcuts.value)) {
      const category = shortcut.category || 'General';
      if (!byCategory[category]) {
        byCategory[category] = [];
      }
      byCategory[category].push(shortcut);
    }

    return byCategory;
  }

  /**
   * Cleanup all locally registered shortcuts
   */
  function cleanup(): void {
    for (const key of localShortcuts) {
      delete globalShortcuts.value[key];
    }
    localShortcuts.length = 0;
    cleanupGlobalListener();
  }

  // Auto-cleanup on unmount
  onMounted(() => {
    initGlobalListener();
  });

  onUnmounted(() => {
    cleanup();
  });

  return {
    register,
    unregister,
    getShortcuts,
    getShortcutsByCategory,
    cleanup,
    formatShortcut,
    parseShortcut,
  };
}

// ============ Preset Shortcuts for Workflow Editor ============

export const WORKFLOW_SHORTCUTS = {
  SAVE: 'Ctrl+S',
  EXECUTE: 'Ctrl+Enter',
  UNDO: 'Ctrl+Z',
  REDO: 'Ctrl+Shift+Z',
  DELETE: 'Delete',
  SELECT_ALL: 'Ctrl+A',
  DESELECT: 'Escape',
  COPY: 'Ctrl+C',
  PASTE: 'Ctrl+V',
  CUT: 'Ctrl+X',
  DUPLICATE: 'Ctrl+D',
  ZOOM_IN: 'Ctrl+=',
  ZOOM_OUT: 'Ctrl+-',
  ZOOM_RESET: 'Ctrl+0',
  TOGGLE_ACTIVE: 'Ctrl+Shift+A',
  NEW_WORKFLOW: 'Ctrl+N',
  SEARCH: 'Ctrl+F',
  HELP: 'Ctrl+/',
} as const;

/**
 * Setup default workflow editor shortcuts
 */
export function useWorkflowShortcuts(handlers: {
  onSave?: () => void;
  onExecute?: () => void;
  onUndo?: () => void;
  onRedo?: () => void;
  onDelete?: () => void;
  onSelectAll?: () => void;
  onDeselect?: () => void;
  onCopy?: () => void;
  onPaste?: () => void;
  onCut?: () => void;
  onDuplicate?: () => void;
  onZoomIn?: () => void;
  onZoomOut?: () => void;
  onZoomReset?: () => void;
  onToggleActive?: () => void;
  onNewWorkflow?: () => void;
  onSearch?: () => void;
  onHelp?: () => void;
}) {
  const { register } = useKeyboardShortcuts();

  onMounted(() => {
    if (handlers.onSave) {
      register(WORKFLOW_SHORTCUTS.SAVE, handlers.onSave, {
        description: 'Save workflow',
        category: 'File',
      });
    }

    if (handlers.onExecute) {
      register(WORKFLOW_SHORTCUTS.EXECUTE, handlers.onExecute, {
        description: 'Execute workflow',
        category: 'Execution',
      });
    }

    if (handlers.onUndo) {
      register(WORKFLOW_SHORTCUTS.UNDO, handlers.onUndo, {
        description: 'Undo',
        category: 'Edit',
      });
    }

    if (handlers.onRedo) {
      register(WORKFLOW_SHORTCUTS.REDO, handlers.onRedo, {
        description: 'Redo',
        category: 'Edit',
      });
    }

    if (handlers.onDelete) {
      register(WORKFLOW_SHORTCUTS.DELETE, handlers.onDelete, {
        description: 'Delete selected nodes',
        category: 'Edit',
      });
    }

    if (handlers.onDeselect) {
      register(WORKFLOW_SHORTCUTS.DESELECT, handlers.onDeselect, {
        description: 'Deselect all',
        category: 'Selection',
      });
    }

    if (handlers.onCopy) {
      register(WORKFLOW_SHORTCUTS.COPY, handlers.onCopy, {
        description: 'Copy selected nodes',
        category: 'Edit',
      });
    }

    if (handlers.onPaste) {
      register(WORKFLOW_SHORTCUTS.PASTE, handlers.onPaste, {
        description: 'Paste nodes',
        category: 'Edit',
      });
    }

    if (handlers.onDuplicate) {
      register(WORKFLOW_SHORTCUTS.DUPLICATE, handlers.onDuplicate, {
        description: 'Duplicate selected nodes',
        category: 'Edit',
      });
    }

    if (handlers.onZoomIn) {
      register(WORKFLOW_SHORTCUTS.ZOOM_IN, handlers.onZoomIn, {
        description: 'Zoom in',
        category: 'View',
      });
    }

    if (handlers.onZoomOut) {
      register(WORKFLOW_SHORTCUTS.ZOOM_OUT, handlers.onZoomOut, {
        description: 'Zoom out',
        category: 'View',
      });
    }

    if (handlers.onZoomReset) {
      register(WORKFLOW_SHORTCUTS.ZOOM_RESET, handlers.onZoomReset, {
        description: 'Reset zoom',
        category: 'View',
      });
    }

    if (handlers.onHelp) {
      register(WORKFLOW_SHORTCUTS.HELP, handlers.onHelp, {
        description: 'Show keyboard shortcuts',
        category: 'Help',
      });
    }
  });
}

export default useKeyboardShortcuts;
