export type ApiEnvelope<T> = {
  code: string
  message: string
  requestId: string
  data: T
}

export type AuthUser = {
  id: number
  tenantId?: number | null
  username: string
  email: string
  displayName: string
  role: string
  status: string
  lastLoginAt?: string | null
  enabled: boolean
}

export type AuthTokens = {
  accessToken: string
  refreshToken: string
}

export type LoginResult = {
  user: AuthUser
  tokens: AuthTokens
}

export type Session = {
  user: AuthUser
  accessToken: string
  refreshToken: string
}

export type UserRecord = {
  id: number
  tenantId?: number | null
  tenant?: {
    id: number
    name: string
    slug: string
  } | null
  teams: Array<{
    teamId: number
    teamName: string
    role: string
  }>
  username: string
  email: string
  displayName: string
  phone: string
  avatarUrl: string
  role: string
  status: string
  lastLoginAt?: string | null
  enabled: boolean
  createdAt: string
  updatedAt: string
}

export type TenantRecord = {
  id: number
  name: string
  slug: string
  description: string
  status: string
  isSystem: boolean
  ownerUserId?: number | null
  userCount: number
  teamCount: number
  createdAt: string
  updatedAt: string
}

export type TeamRecord = {
  id: number
  tenantId?: number | null
  tenant?: {
    id: number
    name: string
    slug: string
  } | null
  name: string
  description: string
  ownerUserId?: number | null
  visibility: string
  memberCount: number
  createdAt: string
  updatedAt: string
}

export type TeamMemberRecord = {
  id: number
  userId: number
  teamId: number
  username: string
  email: string
  displayName: string
  role: string
  createdAt: string
  updatedAt: string
}

export type ModelRecord = {
  id: number
  tenantId?: number | null
  name: string
  provider: string
  model: string
  baseUrl: string
  description: string
  capabilities: string[]
  isDefault: boolean
  isEnabled: boolean
  maxTokens: number
  temperature: number
  topP: number
  hasApiKey: boolean
  maskedApiKey: string
  createdAt: string
  updatedAt: string
}

export type ModelTestResult = {
  reachable: boolean
  model: string
  provider: string
  message: string
  checkedAt: string
}

export type ClusterRecord = {
  id: number
  tenantId?: number | null
  name: string
  description: string
  apiServer: string
  environment: string
  authType: string
  isPublic: boolean
  status: string
  ownerUserId?: number | null
  hasKubeConfig: boolean
  hasToken: boolean
  hasCaCert: boolean
  hasCredentials: boolean
  createdAt: string
  updatedAt: string
}

export type ClusterValidationRecord = {
  reachable: boolean
  version: string
  namespacesCount: number
  message: string
  checkedAt: string
}

export type NamespaceRecord = {
  name: string
  status: string
  labels: Record<string, string>
  createdAt: string
}

export type ResourceRecord = {
  type: string
  kind: string
  name: string
  namespace: string
  status: string
  labels: Record<string, string>
  createdAt: string
}

export type ResourceDetail = {
  type: string
  kind: string
  name: string
  namespace: string
  object: Record<string, unknown>
}

export type EventRecord = {
  type: string
  reason: string
  message: string
  namespace: string
  involvedObject: string
  count: number
  firstSeenAt?: string | null
  lastSeenAt?: string | null
}

export type PodHealthRecord = {
  namespace: string
  name: string
  status: string
  nodeName: string
  readyContainers: number
  totalContainers: number
  restartCount: number
}

export type DeploymentHealthRecord = {
  namespace: string
  name: string
  status: string
  readyReplicas: number
  replicas: number
  availableReplicas: number
  updatedReplicas: number
}

export type ClusterOverviewRecord = {
  clusterId: number
  clusterName: string
  namespacesCount: number
  nodeCount: number
  readyNodeCount: number
  podCount: number
  runningPodCount: number
  pendingPodCount: number
  failedPodCount: number
  deploymentCount: number
  readyDeploymentCount: number
  serviceCount: number
  problemPods: PodHealthRecord[]
  deployments: DeploymentHealthRecord[]
  recentEvents: EventRecord[]
  collectedAt: string
}

export type MCPRecord = {
  id: number
  tenantId?: number | null
  name: string
  type: string
  transport: string
  endpoint: string
  command: string
  args: string[]
  headers: Record<string, string>
  authType: string
  description: string
  healthStatus: string
  isEnabled: boolean
  hasSecret: boolean
  maskedSecret: string
  createdAt: string
  updatedAt: string
}

export type SkillRecord = {
  id: number
  tenantId?: number | null
  name: string
  description: string
  type: string
  version: number
  status: string
  definition: unknown
  isPublic: boolean
  creatorId?: number | null
  createdAt: string
  updatedAt: string
}

export type AuditRecord = {
  id: number
  tenantId?: number | null
  userId?: number | null
  action: string
  target: string
  details: string
  ip: string
  createdAt: string
  updatedAt: string
}

export type IPWhitelistRecord = {
  id: number
  name: string
  ipOrCidr: string
  scope: string
  description: string
  isEnabled: boolean
  createdAt: string
  updatedAt?: string
}

export type SensitiveWordRecord = {
  id: number
  word: string
  category: string
  level: string
  action: string
  description: string
  isEnabled: boolean
  createdAt: string
  updatedAt?: string
}

export type SensitiveFieldRuleRecord = {
  id: number
  name: string
  resource: string
  fieldPath: string
  action: string
  description: string
  isEnabled: boolean
  createdAt: string
  updatedAt?: string
}

export type AgentSessionContext = {
  modelId?: number | null
  clusterId?: number | null
  namespace?: string
}

export type AgentSession = {
  id: number
  tenantId?: number | null
  userId: number
  title: string
  context: AgentSessionContext
  createdAt: string
  updatedAt: string
}

export type AgentMessage = {
  id: number
  sessionId: number
  role: string
  content: string
  toolCalls: Array<Record<string, unknown>>
  toolCallId: string
  createdAt: string
  updatedAt: string
}

export type AgentRun = {
  id: number
  sessionId: number
  userId: number
  modelId?: number | null
  clusterId?: number | null
  status: string
  userMessageId?: number | null
  assistantMessageId?: number | null
  input: string
  output: string
  errorMessage: string
  context: Record<string, unknown>
  startedAt?: string | null
  finishedAt?: string | null
  createdAt: string
  updatedAt: string
}

export type AgentEvent = {
  id: number
  runId: number
  sessionId: number
  eventType: string
  role: string
  status: string
  message: string
  payload: Record<string, unknown>
  requestId: string
  createdAt: string
  updatedAt: string
}

export type ApprovalRequest = {
  id: number
  runId: number
  sessionId: number
  userId: number
  type: string
  title: string
  reason: string
  status: string
  payload: Record<string, unknown>
  approvedBy?: number | null
  resolvedAt?: string | null
  createdAt: string
  updatedAt: string
}

export type SendAgentMessageResult = {
  sessionId: number
  userMessageId: number
  runId: number
  status: string
}

export type PlatformLogEntry = {
  id: number
  timestamp: string
  level: string
  scope: string
  message: string
  fields: Record<string, unknown>
  requestId: string
  runId: string
}

export type PlatformLogQueryResult = {
  scope: string
  cursor: number
  nextCursor: number
  entries: PlatformLogEntry[]
}
