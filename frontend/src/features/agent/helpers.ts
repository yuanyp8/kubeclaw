import type { AgentEvent, ApprovalRequest } from '../../types'

export function collectApprovals(events: AgentEvent[]): ApprovalRequest[] {
  const approvals = new Map<number, ApprovalRequest>()

  events.forEach((event) => {
    if (event.eventType !== 'approval_required') {
      return
    }

    const approvalId = Number(event.payload.approvalId)
    if (!approvalId || approvals.has(approvalId)) {
      return
    }

    const type = String(event.payload.type ?? 'approval')
    approvals.set(approvalId, {
      id: approvalId,
      runId: event.runId,
      sessionId: event.sessionId,
      userId: 0,
      type,
      title: approvalTitle(type),
      reason: approvalReason(type, event.message),
      status: 'pending',
      payload: (event.payload.payload as Record<string, unknown>) ?? {},
      createdAt: event.createdAt,
      updatedAt: event.updatedAt,
    })
  })

  return Array.from(approvals.values())
}

export function toneFromEvent(event: AgentEvent): 'neutral' | 'success' | 'warning' | 'danger' {
  if (event.eventType === 'error' || event.status === 'failed' || event.status === 'rejected') {
    return 'danger'
  }
  if (event.eventType === 'approval_required' || event.status === 'waiting_approval') {
    return 'warning'
  }
  if (event.status === 'completed') {
    return 'success'
  }
  return 'neutral'
}

export function eventName(eventType: string): string {
  switch (eventType) {
    case 'turn_start':
      return '开始执行'
    case 'planning':
      return '任务规划'
    case 'agent_spawn':
      return '分派专家'
    case 'agent_result':
      return '专家结果'
    case 'tool_start':
      return '工具开始'
    case 'tool_end':
      return '工具完成'
    case 'approval_required':
      return '等待审批'
    case 'message_delta':
      return '回复片段'
    case 'message_done':
      return '回复完成'
    case 'turn_end':
      return '执行结束'
    case 'error':
      return '执行错误'
    default:
      return eventType
  }
}

export function roleName(role: string): string {
  switch (role) {
    case 'orchestrator':
      return '主控智能体'
    case 'k8s_expert':
      return 'K8s 专家'
    case 'skill_mcp_expert':
      return 'Skill / MCP 专家'
    case 'safety_reviewer':
      return '安全审查'
    default:
      return role
  }
}

export function approvalTitle(type: string): string {
  switch (type) {
    case 'delete_resource':
      return '删除资源审批'
    case 'scale_deployment':
      return '扩缩容审批'
    case 'restart_deployment':
      return '重启部署审批'
    case 'apply_yaml':
      return '应用 YAML 审批'
    case 'manual_review':
      return '敏感内容人工复核'
    default:
      return '待审批操作'
  }
}

export function approvalReason(type: string, fallback: string): string {
  switch (type) {
    case 'delete_resource':
      return '删除资源会直接影响集群状态，需要人工确认后才能执行。'
    case 'scale_deployment':
      return '调整副本数会影响业务容量和流量承载，需要人工确认。'
    case 'restart_deployment':
      return '重启部署可能影响在线流量，需要人工确认。'
    case 'apply_yaml':
      return '应用 YAML 可能创建、修改或删除资源，需要人工确认。'
    case 'manual_review':
      return '命中了敏感规则，当前请求需要先人工复核。'
    default:
      return fallback || '当前操作需要人工审批后才能继续。'
  }
}
