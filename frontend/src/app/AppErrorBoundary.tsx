import { Component, type ErrorInfo, type ReactNode } from 'react'

import { logClientEvent } from '../lib/client-logger'

type AppErrorBoundaryProps = {
  children: ReactNode
}

type AppErrorBoundaryState = {
  hasError: boolean
}

export class AppErrorBoundary extends Component<AppErrorBoundaryProps, AppErrorBoundaryState> {
  public constructor(props: AppErrorBoundaryProps) {
    super(props)
    this.state = { hasError: false }
  }

  public static getDerivedStateFromError(): AppErrorBoundaryState {
    return { hasError: true }
  }

  public componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    void logClientEvent('error', 'React tree crashed', {
      fields: {
        message: error.message,
        stack: error.stack,
        componentStack: errorInfo.componentStack,
      },
    })
  }

  public render() {
    if (this.state.hasError) {
      return (
        <div className="app-error">
          <div className="app-error__card">
            <p className="section-eyebrow">前端异常</p>
            <h1>控制台界面出现了未预期错误。</h1>
            <p className="muted">系统已经记录客户端日志，请刷新页面，或前往平台日志页查看详细信息。</p>
          </div>
        </div>
      )
    }

    return this.props.children
  }
}
