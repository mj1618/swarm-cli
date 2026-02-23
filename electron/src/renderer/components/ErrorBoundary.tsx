import { Component } from 'react'
import type { ErrorInfo, ReactNode } from 'react'

interface ErrorBoundaryProps {
  children: ReactNode
  name?: string
  fallback?: ReactNode
}

interface ErrorBoundaryState {
  hasError: boolean
  error: Error | null
}

class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error(`[ErrorBoundary]${this.props.name ? ` ${this.props.name}` : ''}`, error, errorInfo)
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: null })
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback
      }

      return (
        <div className="flex flex-col items-center justify-center h-full p-6 bg-slate-800 text-slate-300">
          <div className="text-sm font-medium text-slate-400 mb-2">
            {this.props.name ? `${this.props.name} crashed` : 'Something went wrong'}
          </div>
          <div className="text-xs text-slate-500 mb-4 max-w-xs text-center break-words">
            {this.state.error?.message || 'An unexpected error occurred'}
          </div>
          <button
            onClick={this.handleRetry}
            className="px-3 py-1.5 text-xs font-medium rounded bg-slate-700 hover:bg-slate-600 text-slate-300 transition-colors"
          >
            Retry
          </button>
        </div>
      )
    }

    return this.props.children
  }
}

export default ErrorBoundary
