import { useEffect, useMemo, useRef, useState } from 'react'

import { EmptyState, Section, StatusBadge } from '../../components/shared/Section'
import { api, streamAgentRun } from '../../lib/api'
import { logClientEvent } from '../../lib/client-logger'
import { useSessionStore } from '../../store/session-store'
import type { AgentEvent, AgentMessage, AgentSession, ClusterRecord, ModelRecord } from '../../types'
import { collectApprovals, eventName, roleName, toneFromEvent } from './helpers'

type SessionDraft = {
  title: string
  modelId: string
  clusterId: string
  namespace: string
}

const initialDraft: SessionDraft = {
  title: '',
  modelId: '',
  clusterId: '',
  namespace: 'default',
}

export function AgentPage() {
  const { session } = useSessionStore()
  const [models, setModels] = useState<ModelRecord[]>([])
  const [clusters, setClusters] = useState<ClusterRecord[]>([])
  const [sessions, setSessions] = useState<AgentSession[]>([])
  const [messages, setMessages] = useState<AgentMessage[]>([])
  const [events, setEvents] = useState<AgentEvent[]>([])
  const [selectedSessionId, setSelectedSessionId] = useState<number | null>(null)
  const [draft, setDraft] = useState<SessionDraft>(initialDraft)
  const [composer, setComposer] = useState('')
  const [activeRunId, setActiveRunId] = useState<number | null>(null)
  const [error, setError] = useState('')
  const streamCleanupRef = useRef<null | (() => void)>(null)

  useEffect(() => {
    if (!session) {
      return
    }

    void Promise.all([api.listModels(session), api.listClusters(session), api.listAgentSessions(session)]).then(
      ([modelItems, clusterItems, sessionItems]) => {
        setModels(modelItems)
        setClusters(clusterItems)
        setSessions(sessionItems)
        if (sessionItems.length > 0) {
          setSelectedSessionId((current) => current ?? sessionItems[0].id)
        }
      },
    )
  }, [session])

  useEffect(() => {
    if (!session || !selectedSessionId) {
      return
    }

    void api.listAgentMessages(session, selectedSessionId).then(setMessages)
  }, [selectedSessionId, session])

  useEffect(() => {
    return () => streamCleanupRef.current?.()
  }, [])

  const selectedSession = useMemo(
    () => sessions.find((item) => item.id === selectedSessionId) ?? null,
    [selectedSessionId, sessions],
  )

  const approvals = useMemo(() => collectApprovals(events), [events])
  const hasAvailableModels = useMemo(() => models.some((item) => item.isEnabled), [models])

  async function handleCreateSession() {
    if (!session) {
      return
    }

    const created = await api.createAgentSession(session, {
      title: draft.title || '新的智能体会话',
      modelId: draft.modelId ? Number(draft.modelId) : null,
      clusterId: draft.clusterId ? Number(draft.clusterId) : null,
      namespace: draft.namespace,
    })

    setSessions((current) => [created, ...current])
    setSelectedSessionId(created.id)
    setDraft(initialDraft)
    setMessages([])
    setEvents([])
    setError('')
  }

  async function handleSendMessage() {
    if (!session || !selectedSessionId || !composer.trim()) {
      return
    }

    setError('')
    streamCleanupRef.current?.()

    try {
      const result = await api.sendAgentMessage(session, selectedSessionId, composer.trim())
      setComposer('')
      setActiveRunId(result.runId)
      setMessages(await api.listAgentMessages(session, selectedSessionId))
      setEvents(await api.listAgentRunEvents(session, result.runId))

      streamCleanupRef.current = await streamAgentRun(session, result.runId, {
        onEvent: (event) => {
          setEvents((current) => mergeEvent(current, event))

          if (event.eventType === 'error' || event.status === 'failed') {
            setError(toUserFriendlyAgentError(event.message))
          }

          if (event.eventType === 'message_done' || event.eventType === 'turn_end' || event.eventType === 'error') {
            void api.listAgentMessages(session, selectedSessionId).then(setMessages)
          }
        },
        onError: (streamError) => {
          setError(streamError.message)
        },
      })
    } catch (err) {
      const nextError = err instanceof Error ? err.message : '发送消息失败，请稍后重试。'
      setError(nextError)
      await logClientEvent('error', 'Agent message send failed', {
        session,
        fields: { message: nextError, sessionId: selectedSessionId },
      })
    }
  }

  async function handleDeleteSession(id: number) {
    if (!session) {
      return
    }

    await api.deleteAgentSession(session, id)
    setSessions((current) => current.filter((item) => item.id !== id))
    if (selectedSessionId === id) {
      setSelectedSessionId(null)
      setMessages([])
      setEvents([])
      setActiveRunId(null)
      setError('')
    }
  }

  async function handleApproval(approvalId: number, action: 'approve' | 'reject') {
    if (!session) {
      return
    }

    if (action === 'approve') {
      await api.approveAgentAction(session, approvalId)
    } else {
      await api.rejectAgentAction(session, approvalId)
    }

    if (activeRunId) {
      setEvents(await api.listAgentRunEvents(session, activeRunId))
      if (selectedSessionId) {
        setMessages(await api.listAgentMessages(session, selectedSessionId))
      }
    }
  }

  return (
    <div className="agent-layout">
      <Section eyebrow="智能体" title="会话列表" description="左侧负责新建会话、绑定模型和集群，上下文会直接影响问答与工具调度。">
        {!hasAvailableModels ? (
          <div className="callout">
            <strong>当前没有可用模型</strong>
            <p>请先到模型管理页创建并测试一个可用模型，再回来发起问答。</p>
          </div>
        ) : null}

        <div className="form-grid">
          <label className="field field--full">
            <span>会话标题</span>
            <input
              value={draft.title}
              onChange={(event) => setDraft((current) => ({ ...current, title: event.target.value }))}
              placeholder="例如：生产集群巡检"
            />
          </label>
          <label className="field">
            <span>绑定模型</span>
            <select value={draft.modelId} onChange={(event) => setDraft((current) => ({ ...current, modelId: event.target.value }))}>
              <option value="">默认模型</option>
              {models.map((item) => (
                <option key={item.id} value={item.id}>
                  {item.name}
                </option>
              ))}
            </select>
          </label>
          <label className="field">
            <span>绑定集群</span>
            <select value={draft.clusterId} onChange={(event) => setDraft((current) => ({ ...current, clusterId: event.target.value }))}>
              <option value="">暂不绑定</option>
              {clusters.map((item) => (
                <option key={item.id} value={item.id}>
                  {item.name}
                </option>
              ))}
            </select>
          </label>
          <label className="field field--full">
            <span>默认命名空间</span>
            <input value={draft.namespace} onChange={(event) => setDraft((current) => ({ ...current, namespace: event.target.value }))} />
          </label>
          <button className="button field--full" onClick={handleCreateSession} type="button">
            新建会话
          </button>
        </div>

        <div className="stack-list">
          {sessions.map((item) => (
            <div key={item.id} className={`stack-card ${selectedSessionId === item.id ? 'stack-card--selected' : ''}`}>
              <button className="stack-card__main" onClick={() => setSelectedSessionId(item.id)} type="button">
                <div>
                  <h4>{item.title}</h4>
                  <p className="muted">
                    模型 {item.context.modelId ?? '默认'} / 集群 {item.context.clusterId ?? '未绑定'}
                  </p>
                </div>
              </button>
              <div className="stack-card__meta">
                <button className="chip-button chip-button--danger" onClick={() => void handleDeleteSession(item.id)} type="button">
                  删除
                </button>
              </div>
            </div>
          ))}
        </div>
      </Section>

      <Section eyebrow="问答区" title={selectedSession?.title ?? '智能体会话'} description="中间区域展示问答内容；失败时也会写回一条友好的助手消息。">
        {!selectedSession ? <EmptyState title="还没有选中会话" description="先在左侧新建或选择一个会话，再开始问答。" /> : null}
        <div className="chat-thread">
          {messages.map((message) => (
            <article key={message.id} className={`chat-bubble chat-bubble--${message.role}`}>
              <header>
                <strong>{message.role === 'user' ? '用户' : message.role === 'assistant' ? '智能体' : message.role}</strong>
                <span>{new Date(message.createdAt).toLocaleTimeString()}</span>
              </header>
              <pre className="chat-bubble__content">{message.content}</pre>
            </article>
          ))}
        </div>

        {selectedSession ? (
          <div className="composer">
            <textarea
              value={composer}
              onChange={(event) => setComposer(event.target.value)}
              placeholder="请输入问题，例如：列出 default 命名空间中的 Pod，或帮我分析最近的事件。"
            />
            {error ? <p className="form-error">{error}</p> : null}
            <div className="composer__actions">
              <button className="button" onClick={handleSendMessage} type="button">
                发送消息
              </button>
            </div>
          </div>
        ) : null}
      </Section>

      <Section eyebrow="执行面板" title="运行时间线" description="右侧显示规划、专家调度、工具调用、审批卡片和失败原因。">
        <div className="run-summary">
          <StatusBadge tone={activeRunId ? 'success' : 'neutral'}>{activeRunId ? `运行 #${activeRunId}` : '当前空闲'}</StatusBadge>
          {selectedSession ? (
            <p className="muted">
              模型 {selectedSession.context.modelId ?? '默认'} / 集群 {selectedSession.context.clusterId ?? '未绑定'} / 命名空间{' '}
              {selectedSession.context.namespace || 'default'}
            </p>
          ) : null}
        </div>

        {approvals.length > 0 ? (
          <div className="approval-list">
            {approvals.map((approval) => (
              <article key={approval.id} className="approval-card">
                <div>
                  <p className="section-eyebrow">待审批</p>
                  <h4>{approval.title}</h4>
                  <p className="muted">{approval.reason}</p>
                </div>
                <div className="inline-actions">
                  <button className="button" onClick={() => void handleApproval(approval.id, 'approve')} type="button">
                    通过
                  </button>
                  <button className="button button--secondary" onClick={() => void handleApproval(approval.id, 'reject')} type="button">
                    驳回
                  </button>
                </div>
              </article>
            ))}
          </div>
        ) : null}

        {events.length === 0 ? <EmptyState title="暂无运行事件" description="发送一次消息后，这里会展示规划、执行和审批过程。" /> : null}
        <div className="timeline">
          {events.map((event) => (
            <article key={event.id} className="timeline-item">
              <div className="timeline-item__header">
                <StatusBadge tone={toneFromEvent(event)}>{eventName(event.eventType)}</StatusBadge>
                <span>{roleName(event.role)}</span>
              </div>
              <strong>{event.message}</strong>
              {Object.keys(event.payload ?? {}).length > 0 ? (
                <pre className="code-block code-block--inline">{JSON.stringify(event.payload, null, 2)}</pre>
              ) : null}
            </article>
          ))}
        </div>
      </Section>
    </div>
  )
}

function mergeEvent(events: AgentEvent[], event: AgentEvent): AgentEvent[] {
  const next = events.filter((item) => item.id !== event.id)
  next.push(event)
  next.sort((left, right) => left.id - right.id)
  return next
}

function toUserFriendlyAgentError(message: string): string {
  if (message.includes('Model not found')) {
    return '当前模型不可用，模型服务返回 Model not found。请到模型管理页测试并修正模型名称或 Base URL。'
  }
  return message
}
