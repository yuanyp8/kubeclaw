import type { Session } from '../types'

const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL ?? '').replace(/\/$/, '')

type ClientLogLevel = 'debug' | 'info' | 'warn' | 'error'

export type ClientLoggerContext = {
  session?: Session | null
  route?: string
  requestId?: string
  runId?: string
  fields?: Record<string, unknown>
}

function buildURL(path: string): string {
  return API_BASE_URL ? `${API_BASE_URL}${path}` : path
}

async function postClientLog(level: ClientLogLevel, message: string, context: ClientLoggerContext): Promise<void> {
  if (!context.session?.accessToken) {
    return
  }

  try {
    await fetch(buildURL('/api/logs/client'), {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${context.session.accessToken}`,
      },
      body: JSON.stringify({
        level,
        message,
        requestId: context.requestId,
        runId: context.runId,
        route: context.route,
        fields: context.fields ?? {},
      }),
    })
  } catch {
    // Ignore client-side log upload failures to avoid noisy loops.
  }
}

export async function logClientEvent(level: ClientLogLevel, message: string, context: ClientLoggerContext = {}): Promise<void> {
  const payload = {
    route: context.route,
    requestId: context.requestId,
    runId: context.runId,
    ...context.fields,
  }

  if (import.meta.env.DEV) {
    const target = level === 'error' ? console.error : level === 'warn' ? console.warn : console.info
    target(`[client:${level}] ${message}`, payload)
  }

  if (import.meta.env.PROD && level !== 'warn' && level !== 'error') {
    return
  }

  await postClientLog(level, message, context)
}
