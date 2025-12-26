/**
 * Form Validation Framework
 * Provides validation rules and utilities for form inputs
 */

export type ValidationRule<T = unknown> = (value: T) => string | true;

export interface ValidationRules {
  [key: string]: ValidationRule | ValidationRule[];
}

export interface ValidationErrors {
  [key: string]: string[];
}

export interface ValidationResult {
  valid: boolean;
  errors: ValidationErrors;
}

// ============ Built-in Validation Rules ============

/**
 * Required field validation
 */
export const required = (message = 'This field is required'): ValidationRule => {
  return (value: unknown) => {
    if (value === null || value === undefined) return message;
    if (typeof value === 'string' && value.trim() === '') return message;
    if (Array.isArray(value) && value.length === 0) return message;
    return true;
  };
};

/**
 * Minimum length validation
 */
export const minLength = (min: number, message?: string): ValidationRule<string> => {
  return (value: string) => {
    if (!value || value.length < min) {
      return message || `Must be at least ${min} characters`;
    }
    return true;
  };
};

/**
 * Maximum length validation
 */
export const maxLength = (max: number, message?: string): ValidationRule<string> => {
  return (value: string) => {
    if (value && value.length > max) {
      return message || `Must be at most ${max} characters`;
    }
    return true;
  };
};

/**
 * Email format validation
 */
export const email = (message = 'Invalid email address'): ValidationRule<string> => {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return (value: string) => {
    if (value && !emailRegex.test(value)) return message;
    return true;
  };
};

/**
 * URL format validation
 */
export const url = (message = 'Invalid URL'): ValidationRule<string> => {
  return (value: string) => {
    if (!value) return true;
    try {
      new URL(value);
      return true;
    } catch {
      return message;
    }
  };
};

/**
 * Pattern/regex validation
 */
export const pattern = (regex: RegExp, message = 'Invalid format'): ValidationRule<string> => {
  return (value: string) => {
    if (value && !regex.test(value)) return message;
    return true;
  };
};

/**
 * Numeric value validation
 */
export const numeric = (message = 'Must be a number'): ValidationRule => {
  return (value: unknown) => {
    if (value === null || value === undefined || value === '') return true;
    if (isNaN(Number(value))) return message;
    return true;
  };
};

/**
 * Integer validation
 */
export const integer = (message = 'Must be a whole number'): ValidationRule => {
  return (value: unknown) => {
    if (value === null || value === undefined || value === '') return true;
    const num = Number(value);
    if (isNaN(num) || !Number.isInteger(num)) return message;
    return true;
  };
};

/**
 * Minimum value validation
 */
export const min = (minValue: number, message?: string): ValidationRule<number> => {
  return (value: number) => {
    if (value !== null && value !== undefined && value < minValue) {
      return message || `Must be at least ${minValue}`;
    }
    return true;
  };
};

/**
 * Maximum value validation
 */
export const max = (maxValue: number, message?: string): ValidationRule<number> => {
  return (value: number) => {
    if (value !== null && value !== undefined && value > maxValue) {
      return message || `Must be at most ${maxValue}`;
    }
    return true;
  };
};

/**
 * Range validation (inclusive)
 */
export const between = (minValue: number, maxValue: number, message?: string): ValidationRule<number> => {
  return (value: number) => {
    if (value !== null && value !== undefined && (value < minValue || value > maxValue)) {
      return message || `Must be between ${minValue} and ${maxValue}`;
    }
    return true;
  };
};

/**
 * JSON format validation
 */
export const json = (message = 'Invalid JSON'): ValidationRule<string> => {
  return (value: string) => {
    if (!value || value.trim() === '') return true;
    try {
      JSON.parse(value);
      return true;
    } catch {
      return message;
    }
  };
};

/**
 * Cron expression validation
 */
export const cronExpression = (message = 'Invalid cron expression'): ValidationRule<string> => {
  // Basic cron validation (5 or 6 fields)
  const cronRegex = /^(\*|([0-9]|1[0-9]|2[0-9]|3[0-9]|4[0-9]|5[0-9])|\*\/([0-9]|1[0-9]|2[0-9]|3[0-9]|4[0-9]|5[0-9])) (\*|([0-9]|1[0-9]|2[0-3])|\*\/([0-9]|1[0-9]|2[0-3])) (\*|([1-9]|1[0-9]|2[0-9]|3[0-1])|\*\/([1-9]|1[0-9]|2[0-9]|3[0-1])) (\*|([1-9]|1[0-2])|\*\/([1-9]|1[0-2])) (\*|([0-6])|\*\/([0-6]))$/;
  return (value: string) => {
    if (!value) return true;
    if (!cronRegex.test(value.trim())) return message;
    return true;
  };
};

/**
 * Custom validation with async support
 */
export const custom = <T>(
  validator: (value: T) => boolean | Promise<boolean>,
  message = 'Invalid value'
): ValidationRule<T> => {
  return (value: T) => {
    const result = validator(value);
    if (result instanceof Promise) {
      // For sync validation, we can't handle async
      // The useFormValidation composable handles async separately
      return true;
    }
    return result ? true : message;
  };
};

/**
 * Matches another field value
 */
export const sameAs = <T>(fieldValue: () => T, message = 'Values must match'): ValidationRule<T> => {
  return (value: T) => {
    if (value !== fieldValue()) return message;
    return true;
  };
};

// ============ Validation Utilities ============

/**
 * Validate a single value against rules
 */
export function validateValue(value: unknown, rules: ValidationRule | ValidationRule[]): string[] {
  const errors: string[] = [];
  const ruleList = Array.isArray(rules) ? rules : [rules];

  for (const rule of ruleList) {
    const result = rule(value);
    if (result !== true) {
      errors.push(result);
    }
  }

  return errors;
}

/**
 * Validate an object against a set of rules
 */
export function validateObject<T extends Record<string, unknown>>(
  data: T,
  rules: ValidationRules
): ValidationResult {
  const errors: ValidationErrors = {};
  let valid = true;

  for (const [field, fieldRules] of Object.entries(rules)) {
    const value = data[field];
    const fieldErrors = validateValue(value, fieldRules);

    if (fieldErrors.length > 0) {
      errors[field] = fieldErrors;
      valid = false;
    }
  }

  return { valid, errors };
}

/**
 * Create a composed rule from multiple rules
 */
export function composeRules(...rules: ValidationRule[]): ValidationRule {
  return (value: unknown) => {
    for (const rule of rules) {
      const result = rule(value);
      if (result !== true) return result;
    }
    return true;
  };
}

/**
 * Conditional validation - only validate if condition is true
 */
export function when<T>(
  condition: (value: T) => boolean,
  rules: ValidationRule<T> | ValidationRule<T>[]
): ValidationRule<T> {
  return (value: T) => {
    if (!condition(value)) return true;
    const ruleList = Array.isArray(rules) ? rules : [rules];
    for (const rule of ruleList) {
      const result = rule(value);
      if (result !== true) return result;
    }
    return true;
  };
}

// Export all rules as a namespace
export const rules = {
  required,
  minLength,
  maxLength,
  email,
  url,
  pattern,
  numeric,
  integer,
  min,
  max,
  between,
  json,
  cronExpression,
  custom,
  sameAs,
};

export default rules;
