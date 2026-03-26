import { logClientEvent } from './client-logger'
import type {
  AgentEvent,
  AgentMessage,
  AgentSession,
  ApiEnvelope,
  AuditRecord,
  ClusterRecord,
  ClusterOverviewRecord,
  ClusterValidationRecord,
  EventRecord,
  IPWhitelistRecord,
  LoginResult,
  MCPRecord,
  ModelRecord,
  ModelTestResult,
  NamespaceRecord,
  PlatformLogQueryResult,
  ResourceDetail,
  ResourceRecord,
  SendAgentMessageResult,
  SensitiveFieldRuleRecord,
  SensitiveWordRecord,
  Session,
  SkillRecord,
  TeamMemberRecord,
  TeamRecord,
  TenantRecord,
  UserRecord,
} from '../types'

const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL ?? '').replace(/\/$/, '')
const SESSION_KEY = 'kubeclaw-console-session'

type RequestOptions = RequestInit & {
  token?: string | null
  session?: Session | null
}

type EventStreamHandlers = {
  onEvent: (event: AgentEvent) => void
  onError?: (error: Error) => void
  onDone?: () => void
}

type LogStreamHandlers = {
  onLine: (line: string) => void
  onError?: (error: Error) => void
  onDone?: () => void
}

function buildURL(path: string): string {
  return API_BASE_URL ? `${API_BASE_URL}${path}` : path
}

function resolveToken(options: RequestOptions): string | null {
  return options.token ?? options.session?.accessToken ?? null
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers = new Headers(options.headers)

  if (!headers.has('Content-Type') && options.body && !(options.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json')
  }

  const token = resolveToken(options)
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  const response = await fetch(buildURL(path), {
    ...options,
    headers,
  })

  const raw = await response.text()
  const payload = raw ? (JSON.parse(raw) as ApiEnvelope<T>) : null

  if (!response.ok) {
    const errorMessage = payload?.message ?? `Request failed (${response.status})`
    await logClientEvent('error', 'API request failed', {
      session: options.session ?? null,
      fields: {
        path,
        status: response.status,
        message: errorMessage,
      },
      requestId: payload?.requestId,
    })
    throw new Error(errorMessage)
  }

  return payload?.data as T
}

export function loadSession(): Session | null {
  const raw = window.localStorage.getItem(SESSION_KEY)
  if (!raw) {
    return null
  }

  try {
    return JSON.parse(raw) as Session
  } catch {
    return null
  }
}

export function saveSession(session: Session): void {
  window.localStorage.setItem(SESSION_KEY, JSON.stringify(session))
}

export function clearSession(): void {
  window.localStorage.removeItem(SESSION_KEY)
}

export const api = {
  login: async (login: string, password: string): Promise<Session> => {
    const result = await request<LoginResult>('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify({ login, password }),
    })

    return {
      user: result.user,
      accessToken: result.tokens.accessToken,
      refreshToken: result.tokens.refreshToken,
    }
  },

  me: (session: Session) => request<UserRecord>('/api/users/me', { session }),

  listUsers: (session: Session) => request<UserRecord[]>('/api/users', { session }),
  createUser: (session: Session, body: Record<string, unknown>) =>
    request<UserRecord>('/api/users', { method: 'POST', session, body: JSON.stringify(body) }),
  updateUser: (session: Session, id: number, body: Record<string, unknown>) =>
    request<UserRecord>(`/api/users/${id}`, { method: 'PUT', session, body: JSON.stringify(body) }),
  deleteUser: (session: Session, id: number) =>
    request<{ id: number }>(`/api/users/${id}`, { method: 'DELETE', session }),

  listTenants: (session: Session) => request<TenantRecord[]>('/api/tenants', { session }),
  createTenant: (session: Session, body: Record<string, unknown>) =>
    request<TenantRecord>('/api/tenants', { method: 'POST', session, body: JSON.stringify(body) }),

  listTeams: (session: Session) => request<TeamRecord[]>('/api/teams', { session }),
  createTeam: (session: Session, body: Record<string, unknown>) =>
    request<TeamRecord>('/api/teams', { method: 'POST', session, body: JSON.stringify(body) }),
  listTeamMembers: (session: Session, id: number) => request<TeamMemberRecord[]>(`/api/teams/${id}/members`, { session }),
  addTeamMember: (session: Session, id: number, body: Record<string, unknown>) =>
    request<TeamMemberRecord>(`/api/teams/${id}/members`, { method: 'POST', session, body: JSON.stringify(body) }),
  removeTeamMember: (session: Session, id: number, userId: number) =>
    request<{ teamId: number; userId: number }>(`/api/teams/${id}/members/${userId}`, { method: 'DELETE', session }),

  listModels: (session: Session) => request<ModelRecord[]>('/api/models', { session }),
  createModel: (session: Session, body: Record<string, unknown>) =>
    request<ModelRecord>('/api/models', { method: 'POST', session, body: JSON.stringify(body) }),
  updateModel: (session: Session, id: number, body: Record<string, unknown>) =>
    request<ModelRecord>(`/api/models/${id}`, { method: 'PUT', session, body: JSON.stringify(body) }),
  deleteModel: (session: Session, id: number) =>
    request<{ id: number }>(`/api/models/${id}`, { method: 'DELETE', session }),
  testModel: (session: Session, id: number) =>
    request<ModelTestResult>(`/api/models/${id}/test`, { method: 'POST', session }),
  setDefaultModel: (session: Session, id: number) =>
    request<ModelRecord>(`/api/models/${id}/set-default`, { method: 'POST', session }),

  listClusters: (session: Session) => request<ClusterRecord[]>('/api/clusters', { session }),
  createCluster: (session: Session, body: Record<string, unknown>) =>
    request<ClusterRecord>('/api/clusters', { method: 'POST', session, body: JSON.stringify(body) }),
  updateCluster: (session: Session, id: number, body: Record<string, unknown>) =>
    request<ClusterRecord>(`/api/clusters/${id}`, { method: 'PUT', session, body: JSON.stringify(body) }),
  deleteCluster: (session: Session, id: number) =>
    request<{ id: number }>(`/api/clusters/${id}`, { method: 'DELETE', session }),
  validateCluster: (session: Session, id: number) =>
    request<ClusterValidationRecord>(`/api/clusters/${id}/validate`, { method: 'POST', session }),
  getClusterOverview: (session: Session, id: number, namespace?: string) => {
    const query = new URLSearchParams()
    if (namespace) {
      query.set('namespace', namespace)
    }
    const suffix = query.toString() ? `?${query.toString()}` : ''
    return request<ClusterOverviewRecord>(`/api/clusters/${id}/overview${suffix}`, { session })
  },
  listNamespaces: (session: Session, id: number) =>
    request<NamespaceRecord[]>(`/api/clusters/${id}/namespaces`, { session }),
  listResources: (session: Session, id: number, type: string, namespace?: string) => {
    const query = new URLSearchParams({ type })
    if (namespace) {
      query.set('namespace', namespace)
    }
    return request<ResourceRecord[]>(`/api/clusters/${id}/resources?${query.toString()}`, { session })
  },
  getResource: (session: Session, id: number, type: string, name: string, namespace?: string) => {
    const query = new URLSearchParams()
    if (namespace) {
      query.set('namespace', namespace)
    }
    const suffix = query.toString() ? `?${query.toString()}` : ''
    return request<ResourceDetail>(`/api/clusters/${id}/resources/${type}/${name}${suffix}`, { session })
  },
  listEvents: (session: Session, id: number, namespace?: string) => {
    const query = new URLSearchParams()
    if (namespace) {
      query.set('namespace', namespace)
    }
    const suffix = query.toString() ? `?${query.toString()}` : ''
    return request<EventRecord[]>(`/api/clusters/${id}/events${suffix}`, { session })
  },
  requestDeleteResource: (session: Session, id: number, body: Record<string, unknown>) =>
    request<SendAgentMessageResult>(`/api/clusters/${id}/actions/delete-resource`, {
      method: 'POST',
      session,
      body: JSON.stringify(body),
    }),
  requestScaleDeployment: (session: Session, id: number, body: Record<string, unknown>) =>
    request<SendAgentMessageResult>(`/api/clusters/${id}/actions/scale-deployment`, {
      method: 'POST',
      session,
      body: JSON.stringify(body),
    }),
  requestRestartDeployment: (session: Session, id: number, body: Record<string, unknown>) =>
    request<SendAgentMessageResult>(`/api/clusters/${id}/actions/restart-deployment`, {
      method: 'POST',
      session,
      body: JSON.stringify(body),
    }),
  requestApplyYAML: (session: Session, id: number, body: Record<string, unknown>) =>
    request<SendAgentMessageResult>(`/api/clusters/${id}/actions/apply-yaml`, {
      method: 'POST',
      session,
      body: JSON.stringify(body),
    }),

  listMCPServers: (session: Session) => request<MCPRecord[]>('/api/mcp/servers', { session }),
  createMCPServer: (session: Session, body: Record<string, unknown>) =>
    request<MCPRecord>('/api/mcp/servers', { method: 'POST', session, body: JSON.stringify(body) }),

  listSkills: (session: Session) => request<SkillRecord[]>('/api/skills', { session }),
  createSkill: (session: Session, body: Record<string, unknown>) =>
    request<SkillRecord>('/api/skills', { method: 'POST', session, body: JSON.stringify(body) }),

  listAudit: (session: Session) => request<AuditRecord[]>('/api/audit', { session }),

  listIPWhitelists: (session: Session) => request<IPWhitelistRecord[]>('/api/security/ip-whitelists', { session }),
  createIPWhitelist: (session: Session, body: Record<string, unknown>) =>
    request<IPWhitelistRecord>('/api/security/ip-whitelists', {
      method: 'POST',
      session,
      body: JSON.stringify(body),
    }),
  listSensitiveWords: (session: Session) => request<SensitiveWordRecord[]>('/api/security/sensitive-words', { session }),
  createSensitiveWord: (session: Session, body: Record<string, unknown>) =>
    request<SensitiveWordRecord>('/api/security/sensitive-words', {
      method: 'POST',
      session,
      body: JSON.stringify(body),
    }),
  listSensitiveFieldRules: (session: Session) =>
    request<SensitiveFieldRuleRecord[]>('/api/security/sensitive-field-rules', { session }),
  createSensitiveFieldRule: (session: Session, body: Record<string, unknown>) =>
    request<SensitiveFieldRuleRecord>('/api/security/sensitive-field-rules', {
      method: 'POST',
      session,
      body: JSON.stringify(body),
    }),

  listLogScopes: (session: Session) => request<string[]>('/api/logs/scopes', { session }),
  listLogs: (session: Session, scope: string, cursor = 0, limit = 50) =>
    request<PlatformLogQueryResult>(`/api/logs?scope=${encodeURIComponent(scope)}&cursor=${cursor}&limit=${limit}`, { session }),

  listAgentSessions: (session: Session) => request<AgentSession[]>('/api/agent/sessions', { session }),
  createAgentSession: (session: Session, body: Record<string, unknown>) =>
    request<AgentSession>('/api/agent/sessions', { method: 'POST', session, body: JSON.stringify(body) }),
  getAgentSession: (session: Session, id: number) => request<AgentSession>(`/api/agent/sessions/${id}`, { session }),
  listAgentMessages: (session: Session, id: number) =>
    request<AgentMessage[]>(`/api/agent/sessions/${id}/messages`, { session }),
  sendAgentMessage: (session: Session, id: number, content: string) =>
    request<SendAgentMessageResult>(`/api/agent/sessions/${id}/messages`, {
      method: 'POST',
      session,
      body: JSON.stringify({ content }),
    }),
  deleteAgentSession: (session: Session, id: number) =>
    request<{ id: number }>(`/api/agent/sessions/${id}`, { method: 'DELETE', session }),
  listAgentRunEvents: (session: Session, runId: number) =>
    request<AgentEvent[]>(`/api/agent/runs/${runId}/events`, { session }),
  approveAgentAction: (session: Session, approvalId: number) =>
    request(`/api/agent/approvals/${approvalId}/approve`, { method: 'POST', session }),
  rejectAgentAction: (session: Session, approvalId: number) =>
    request(`/api/agent/approvals/${approvalId}/reject`, { method: 'POST', session }),
}

export async function streamAgentRun(session: Session, runId: number, handlers: EventStreamHandlers): Promise<() => void> {
  const controller = new AbortController()

  try {
    const response = await fetch(buildURL(`/api/agent/runs/${runId}/stream`), {
      headers: {
        Authorization: `Bearer ${session.accessToken}`,
      },
      signal: controller.signal,
    })

    if (!response.ok || !response.body) {
      throw new Error(`Unable to open event stream (${response.status})`)
    }

    const reader = response.body.getReader()
    const decoder = new TextDecoder()
    let buffer = ''

    ;(async () => {
      try {
        while (true) {
          const { done, value } = await reader.read()
          if (done) {
            handlers.onDone?.()
            return
          }

          buffer += decoder.decode(value, { stream: true })
          const chunks = buffer.split('\n\n')
          buffer = chunks.pop() ?? ''

          for (const chunk of chunks) {
            const dataLine = chunk
              .split('\n')
              .find((line) => line.startsWith('data: '))

            if (!dataLine) {
              continue
            }

            const payload = dataLine.replace(/^data:\s*/, '')
            const event = JSON.parse(payload) as AgentEvent
            handlers.onEvent(event)
          }
        }
      } catch (error) {
        if (controller.signal.aborted) {
          return
        }
        const streamError = error instanceof Error ? error : new Error('Agent stream failed')
        handlers.onError?.(streamError)
        await logClientEvent('warn', 'Agent SSE stream disconnected', {
          session,
          runId: String(runId),
          fields: { message: streamError.message },
        })
      }
    })()
  } catch (error) {
    const streamError = error instanceof Error ? error : new Error('Unable to open agent stream')
    handlers.onError?.(streamError)
    throw streamError
  }

  return () => controller.abort()
}

export async function streamPodLogs(
  session: Session,
  clusterId: number,
  podName: string,
  options: {
    namespace?: string
    container?: string
    follow?: boolean
    tailLines?: number
    sinceSeconds?: number
  },
  handlers: LogStreamHandlers,
): Promise<() => void> {
  const controller = new AbortController()
  const query = new URLSearchParams()

  if (options.namespace) {
    query.set('namespace', options.namespace)
  }
  if (options.container) {
    query.set('container', options.container)
  }
  if (options.follow !== undefined) {
    query.set('follow', String(options.follow))
  }
  if (options.tailLines !== undefined) {
    query.set('tailLines', String(options.tailLines))
  }
  if (options.sinceSeconds !== undefined) {
    query.set('sinceSeconds', String(options.sinceSeconds))
  }

  const suffix = query.toString() ? `?${query.toString()}` : ''
  const url = buildURL(`/api/clusters/${clusterId}/pods/${encodeURIComponent(podName)}/logs${suffix}`)

  try {
    const response = await fetch(url, {
      headers: {
        Authorization: `Bearer ${session.accessToken}`,
      },
      signal: controller.signal,
    })

    if (!response.ok || !response.body) {
      throw new Error(`Unable to open pod log stream (${response.status})`)
    }

    const reader = response.body.getReader()
    const decoder = new TextDecoder()
    let buffer = ''

    ;(async () => {
      try {
        while (true) {
          const { done, value } = await reader.read()
          if (done) {
            if (buffer.trim()) {
              handlers.onLine(buffer)
            }
            handlers.onDone?.()
            return
          }

          buffer += decoder.decode(value, { stream: true })
          const lines = buffer.split(/\r?\n/)
          buffer = lines.pop() ?? ''

          for (const line of lines) {
            handlers.onLine(line)
          }
        }
      } catch (error) {
        if (controller.signal.aborted) {
          return
        }
        const streamError = error instanceof Error ? error : new Error('Pod log stream failed')
        handlers.onError?.(streamError)
        await logClientEvent('warn', 'Pod log stream disconnected', {
          session,
          fields: {
            clusterId,
            podName,
            message: streamError.message,
          },
        })
      }
    })()
  } catch (error) {
    const streamError = error instanceof Error ? error : new Error('Unable to open pod log stream')
    handlers.onError?.(streamError)
    throw streamError
  }

  return () => controller.abort()
}
