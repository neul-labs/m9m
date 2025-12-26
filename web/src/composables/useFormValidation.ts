/**
 * Form Validation Composable
 * Provides reactive form validation for Vue components
 */

import { ref, reactive, computed, watch, type Ref, type ComputedRef } from 'vue';
import { validateValue, type ValidationRule, type ValidationErrors } from '@/utils/validation';

export interface FieldState {
  value: Ref<unknown>;
  errors: Ref<string[]>;
  touched: Ref<boolean>;
  dirty: Ref<boolean>;
  valid: ComputedRef<boolean>;
  validate: () => string[];
  reset: () => void;
  touch: () => void;
}

export interface FormState<T extends Record<string, unknown>> {
  values: T;
  errors: ValidationErrors;
  touched: Record<keyof T, boolean>;
  dirty: ComputedRef<boolean>;
  valid: ComputedRef<boolean>;
  validate: () => boolean;
  validateField: (field: keyof T) => string[];
  reset: () => void;
  resetField: (field: keyof T) => void;
  setFieldValue: (field: keyof T, value: unknown) => void;
  setFieldError: (field: keyof T, error: string) => void;
  clearFieldError: (field: keyof T) => void;
  touchField: (field: keyof T) => void;
  getFieldError: (field: keyof T) => string | undefined;
  hasError: (field: keyof T) => boolean;
}

export interface UseFormValidationOptions<T extends Record<string, unknown>> {
  initialValues: T;
  rules?: Partial<Record<keyof T, ValidationRule | ValidationRule[]>>;
  validateOnChange?: boolean;
  validateOnBlur?: boolean;
  onSubmit?: (values: T) => void | Promise<void>;
}

/**
 * Create a validated form state
 */
export function useFormValidation<T extends Record<string, unknown>>(
  options: UseFormValidationOptions<T>
): FormState<T> {
  const {
    initialValues,
    rules = {},
    validateOnChange = true,
    validateOnBlur = true,
  } = options;

  // Create reactive form values
  const values = reactive({ ...initialValues }) as T;

  // Track errors for each field
  const errors = reactive<ValidationErrors>({});

  // Track touched state for each field
  const touched = reactive<Record<string, boolean>>({});

  // Track initial values for dirty checking
  const initial = { ...initialValues };

  // Check if form is dirty (values changed from initial)
  const dirty = computed(() => {
    for (const key of Object.keys(initial)) {
      if (JSON.stringify(values[key as keyof T]) !== JSON.stringify(initial[key as keyof T])) {
        return true;
      }
    }
    return false;
  });

  // Check if form is valid (no errors)
  const valid = computed(() => {
    return Object.values(errors).every((fieldErrors) =>
      !Array.isArray(fieldErrors) || fieldErrors.length === 0
    );
  });

  /**
   * Validate a specific field
   */
  function validateField(field: keyof T): string[] {
    const fieldRules = rules[field];
    if (!fieldRules) {
      errors[field as string] = [];
      return [];
    }

    const fieldErrors = validateValue(values[field], fieldRules);
    errors[field as string] = fieldErrors;
    return fieldErrors;
  }

  /**
   * Validate all fields
   */
  function validate(): boolean {
    let isValid = true;

    for (const field of Object.keys(rules)) {
      const fieldErrors = validateField(field as keyof T);
      if (fieldErrors.length > 0) {
        isValid = false;
      }
    }

    return isValid;
  }

  /**
   * Reset form to initial values
   */
  function reset(): void {
    for (const key of Object.keys(initial)) {
      (values as Record<string, unknown>)[key] = initial[key as keyof T];
      errors[key] = [];
      touched[key] = false;
    }
  }

  /**
   * Reset a specific field
   */
  function resetField(field: keyof T): void {
    (values as Record<string, unknown>)[field as string] = initial[field];
    errors[field as string] = [];
    touched[field as string] = false;
  }

  /**
   * Set a field value
   */
  function setFieldValue(field: keyof T, value: unknown): void {
    (values as Record<string, unknown>)[field as string] = value;
    if (validateOnChange) {
      validateField(field);
    }
  }

  /**
   * Set a field error manually
   */
  function setFieldError(field: keyof T, error: string): void {
    if (!errors[field as string]) {
      errors[field as string] = [];
    }
    errors[field as string].push(error);
  }

  /**
   * Clear field error
   */
  function clearFieldError(field: keyof T): void {
    errors[field as string] = [];
  }

  /**
   * Mark a field as touched (for blur validation)
   */
  function touchField(field: keyof T): void {
    touched[field as string] = true;
    if (validateOnBlur) {
      validateField(field);
    }
  }

  /**
   * Get the first error for a field
   */
  function getFieldError(field: keyof T): string | undefined {
    const fieldErrors = errors[field as string];
    return Array.isArray(fieldErrors) && fieldErrors.length > 0 ? fieldErrors[0] : undefined;
  }

  /**
   * Check if a field has an error
   */
  function hasError(field: keyof T): boolean {
    const fieldErrors = errors[field as string];
    return Array.isArray(fieldErrors) && fieldErrors.length > 0;
  }

  // Watch for value changes and validate if enabled
  if (validateOnChange) {
    watch(
      () => ({ ...values }),
      (newValues, oldValues) => {
        for (const field of Object.keys(newValues)) {
          if (newValues[field as keyof T] !== oldValues?.[field as keyof T]) {
            validateField(field as keyof T);
          }
        }
      },
      { deep: true }
    );
  }

  return {
    values,
    errors,
    touched: touched as Record<keyof T, boolean>,
    dirty,
    valid,
    validate,
    validateField,
    reset,
    resetField,
    setFieldValue,
    setFieldError,
    clearFieldError,
    touchField,
    getFieldError,
    hasError,
  };
}

/**
 * Create a single validated field
 */
export function useField<T = unknown>(
  initialValue: T,
  rules: ValidationRule<T> | ValidationRule<T>[] = []
): FieldState {
  const value = ref<T>(initialValue) as Ref<T>;
  const errors = ref<string[]>([]);
  const touched = ref(false);
  const dirty = ref(false);
  const initial = initialValue;

  const valid = computed(() => errors.value.length === 0);

  function validate(): string[] {
    const fieldErrors = validateValue(value.value, rules as ValidationRule | ValidationRule[]);
    errors.value = fieldErrors;
    return fieldErrors;
  }

  function reset(): void {
    value.value = initial;
    errors.value = [];
    touched.value = false;
    dirty.value = false;
  }

  function touch(): void {
    touched.value = true;
    validate();
  }

  // Watch for changes
  watch(value, (newVal) => {
    dirty.value = JSON.stringify(newVal) !== JSON.stringify(initial);
    if (touched.value) {
      validate();
    }
  }, { deep: true });

  return {
    value: value as Ref<unknown>,
    errors,
    touched,
    dirty: computed(() => dirty.value),
    valid,
    validate,
    reset,
    touch,
  };
}

export default useFormValidation;
