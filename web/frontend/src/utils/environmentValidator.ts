/**
 * Environment Configuration Validator
 * Validates that all required environment variables are properly configured
 */

import { observeLogger } from '../services/observeLogger'

interface EnvironmentConfig {
  VITE_API_BASE_URL: string
  VITE_SUPABASE_URL: string
  VITE_SUPABASE_ANON_KEY: string
  VITE_NODE_ENV: string
  VITE_OBSERVE_ENABLED: string
  VITE_OBSERVE_ENDPOINT: string
  VITE_OBSERVE_API_KEY: string
  VITE_OBSERVE_PROJECT_ID: string
}

interface ValidationResult {
  isValid: boolean
  errors: string[]
  warnings: string[]
  config: Partial<EnvironmentConfig>
}

class EnvironmentValidator {
  private requiredVars = [
    'VITE_API_BASE_URL'
  ]

  private optionalVars = [
    'VITE_SUPABASE_URL',
    'VITE_SUPABASE_ANON_KEY',
    'VITE_NODE_ENV',
    'VITE_OBSERVE_ENABLED',
    'VITE_OBSERVE_ENDPOINT',
    'VITE_OBSERVE_API_KEY',
    'VITE_OBSERVE_PROJECT_ID'
  ]

  /**
   * Validate environment configuration
   */
  validate(): ValidationResult {
    const errors: string[] = []
    const warnings: string[] = []
    const config: Partial<EnvironmentConfig> = {}

    // Check required variables
    for (const varName of this.requiredVars) {
      const value = import.meta.env[varName]
      config[varName as keyof EnvironmentConfig] = value

      if (!value) {
        errors.push(`Missing required environment variable: ${varName}`)
      } else if (this.isPlaceholder(value)) {
        errors.push(`Environment variable ${varName} contains placeholder value: ${value}`)
      }
    }

    // Check optional variables
    for (const varName of this.optionalVars) {
      const value = import.meta.env[varName]
      config[varName as keyof EnvironmentConfig] = value

      if (!value) {
        warnings.push(`Optional environment variable not set: ${varName}`)
      } else if (this.isPlaceholder(value)) {
        warnings.push(`Environment variable ${varName} contains placeholder value: ${value}`)
      }
    }

    // Validate API URL format
    if (config.VITE_API_BASE_URL) {
      if (!this.isValidUrl(config.VITE_API_BASE_URL)) {
        errors.push(`Invalid API URL format: ${config.VITE_API_BASE_URL}`)
      }
    }

    // Validate Supabase URL format
    if (config.VITE_SUPABASE_URL && !this.isValidUrl(config.VITE_SUPABASE_URL)) {
      warnings.push(`Invalid Supabase URL format: ${config.VITE_SUPABASE_URL}`)
    }

    // Check for development vs production consistency
    const nodeEnv = config.VITE_NODE_ENV || 'development'
    if (nodeEnv === 'production') {
      if (config.VITE_API_BASE_URL?.includes('localhost')) {
        warnings.push('Production environment is using localhost API URL')
      }
    }

    const isValid = errors.length === 0

    // Log validation results
    observeLogger.logEvent(
      isValid ? 'info' : 'error',
      'Environment validation completed',
      {
        isValid,
        errorCount: errors.length,
        warningCount: warnings.length,
        environment: nodeEnv,
        type: 'environment_validation'
      }
    )

    return {
      isValid,
      errors,
      warnings,
      config
    }
  }

  /**
   * Check if a value is a placeholder
   */
  private isPlaceholder(value: string): boolean {
    const placeholders = [
      'your-',
      'placeholder',
      'example',
      'localhost:3000', // Common placeholder
      'change-me',
      'replace-me',
      'todo',
      'fixme'
    ]

    const lowerValue = value.toLowerCase()
    return placeholders.some(placeholder => lowerValue.includes(placeholder))
  }

  /**
   * Validate URL format
   */
  private isValidUrl(url: string): boolean {
    try {
      new URL(url)
      return true
    } catch {
      return false
    }
  }

  /**
   * Get environment summary for debugging
   */
  getEnvironmentSummary(): Record<string, any> {
    const summary: Record<string, any> = {}

    // Add all environment variables (sanitized)
    for (const varName of [...this.requiredVars, ...this.optionalVars]) {
      const value = import.meta.env[varName]
      if (value) {
        // Sanitize sensitive values
        if (varName.includes('KEY') || varName.includes('SECRET') || varName.includes('TOKEN')) {
          summary[varName] = value.substring(0, 8) + '...[REDACTED]'
        } else {
          summary[varName] = value
        }
      } else {
        summary[varName] = '[NOT SET]'
      }
    }

    // Add runtime information
    summary['RUNTIME_INFO'] = {
      origin: window.location.origin,
      userAgent: navigator.userAgent,
      timestamp: new Date().toISOString(),
      buildMode: import.meta.env.MODE
    }

    return summary
  }

  /**
   * Generate configuration report
   */
  generateReport(): string {
    const validation = this.validate()
    const summary = this.getEnvironmentSummary()

    let report = `🔧 Environment Configuration Report\n`
    report += `=====================================\n`
    report += `Timestamp: ${new Date().toISOString()}\n`
    report += `Build Mode: ${import.meta.env.MODE}\n`
    report += `Origin: ${window.location.origin}\n\n`

    if (validation.isValid) {
      report += `✅ Configuration is valid\n\n`
    } else {
      report += `❌ Configuration has errors\n\n`
    }

    if (validation.errors.length > 0) {
      report += `Errors (${validation.errors.length}):\n`
      report += `${'-'.repeat(20)}\n`
      validation.errors.forEach(error => {
        report += `• ${error}\n`
      })
      report += `\n`
    }

    if (validation.warnings.length > 0) {
      report += `Warnings (${validation.warnings.length}):\n`
      report += `${'-'.repeat(20)}\n`
      validation.warnings.forEach(warning => {
        report += `• ${warning}\n`
      })
      report += `\n`
    }

    report += `Environment Variables:\n`
    report += `${'-'.repeat(20)}\n`
    Object.entries(summary).forEach(([key, value]) => {
      if (key !== 'RUNTIME_INFO') {
        report += `${key}: ${value}\n`
      }
    })

    report += `\nRuntime Information:\n`
    report += `${'-'.repeat(20)}\n`
    Object.entries(summary.RUNTIME_INFO).forEach(([key, value]) => {
      report += `${key}: ${value}\n`
    })

    return report
  }
}

// Create and export singleton instance
export const environmentValidator = new EnvironmentValidator()
export default environmentValidator

// Export types
export type { EnvironmentConfig, ValidationResult }