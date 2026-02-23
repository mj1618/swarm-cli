import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import ErrorBoundary from '../ErrorBoundary'

// Component that throws an error on demand
const ThrowingComponent = ({ shouldThrow }: { shouldThrow: boolean }) => {
  if (shouldThrow) {
    throw new Error('Test error message')
  }
  return <div>Child content</div>
}

describe('ErrorBoundary', () => {
  let consoleErrorSpy: ReturnType<typeof vi.spyOn>

  beforeEach(() => {
    // Suppress console.error output during tests
    consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
  })

  afterEach(() => {
    consoleErrorSpy.mockRestore()
  })

  it('renders children normally when no error occurs', () => {
    render(
      <ErrorBoundary>
        <ThrowingComponent shouldThrow={false} />
      </ErrorBoundary>
    )

    expect(screen.getByText('Child content')).toBeInTheDocument()
  })

  it('catches errors and displays default fallback UI', () => {
    render(
      <ErrorBoundary>
        <ThrowingComponent shouldThrow={true} />
      </ErrorBoundary>
    )

    expect(screen.getByText('Something went wrong')).toBeInTheDocument()
    expect(screen.getByText('Test error message')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument()
  })

  it('displays custom name prop in error message when provided', () => {
    render(
      <ErrorBoundary name="TestComponent">
        <ThrowingComponent shouldThrow={true} />
      </ErrorBoundary>
    )

    expect(screen.getByText('TestComponent crashed')).toBeInTheDocument()
    expect(screen.queryByText('Something went wrong')).not.toBeInTheDocument()
  })

  it('displays custom fallback prop when provided instead of default UI', () => {
    const customFallback = <div>Custom error fallback</div>

    render(
      <ErrorBoundary fallback={customFallback}>
        <ThrowingComponent shouldThrow={true} />
      </ErrorBoundary>
    )

    expect(screen.getByText('Custom error fallback')).toBeInTheDocument()
    expect(screen.queryByText('Something went wrong')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Retry' })).not.toBeInTheDocument()
  })

  it('Retry button resets the error state and re-renders children', () => {
    // Use a stateful wrapper to control when the child throws
    let shouldThrow = true
    const ControlledThrowingComponent = () => {
      if (shouldThrow) {
        throw new Error('Test error')
      }
      return <div>Recovered content</div>
    }

    render(
      <ErrorBoundary>
        <ControlledThrowingComponent />
      </ErrorBoundary>
    )

    // Should show error UI
    expect(screen.getByText('Something went wrong')).toBeInTheDocument()

    // Fix the "error" before clicking retry
    shouldThrow = false

    // Click retry
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }))

    // Should now show recovered content
    expect(screen.getByText('Recovered content')).toBeInTheDocument()
    expect(screen.queryByText('Something went wrong')).not.toBeInTheDocument()
  })

  it('logs error to console via componentDidCatch', () => {
    render(
      <ErrorBoundary>
        <ThrowingComponent shouldThrow={true} />
      </ErrorBoundary>
    )

    // componentDidCatch should have been called and logged the error
    expect(consoleErrorSpy).toHaveBeenCalled()
    
    // Find the call from our ErrorBoundary (not React's internal error logging)
    const errorBoundaryCall = consoleErrorSpy.mock.calls.find(
      (call: unknown[]) => typeof call[0] === 'string' && call[0].includes('[ErrorBoundary]')
    )
    
    expect(errorBoundaryCall).toBeDefined()
    expect(errorBoundaryCall![0]).toContain('[ErrorBoundary]')
    
    // Second argument should be the error
    expect(errorBoundaryCall![1]).toBeInstanceOf(Error)
    expect((errorBoundaryCall![1] as Error).message).toBe('Test error message')
  })

  it('logs error with name in console when name prop is provided', () => {
    render(
      <ErrorBoundary name="MyComponent">
        <ThrowingComponent shouldThrow={true} />
      </ErrorBoundary>
    )

    expect(consoleErrorSpy).toHaveBeenCalled()
    
    // Find the call from our ErrorBoundary (not React's internal error logging)
    const errorBoundaryCall = consoleErrorSpy.mock.calls.find(
      (call: unknown[]) => typeof call[0] === 'string' && call[0].includes('[ErrorBoundary]')
    )
    
    expect(errorBoundaryCall).toBeDefined()
    expect(errorBoundaryCall![0]).toContain('[ErrorBoundary]')
    expect(errorBoundaryCall![0]).toContain('MyComponent')
  })

  it('handles error without message gracefully', () => {
    const ThrowWithoutMessage = () => {
      throw new Error()
    }

    render(
      <ErrorBoundary>
        <ThrowWithoutMessage />
      </ErrorBoundary>
    )

    expect(screen.getByText('Something went wrong')).toBeInTheDocument()
    expect(screen.getByText('An unexpected error occurred')).toBeInTheDocument()
  })
})
