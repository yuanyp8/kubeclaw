import { useEffect, useMemo, useRef, useState } from 'react'

import { EmptyState, Section, StatusBadge } from '../../components/shared/Section'
import { api, streamAgentRun, streamPodLogs } from '../../lib/api'
import { useSessionStore } from '../../store/session-store'
import type {
  AgentEvent,
  ClusterOverviewRecord,
  ClusterRecord,
  ClusterValidationRecord,
  EventRecord,
  NamespaceRecord,
  ResourceDetail,
  ResourceRecord,
  SendAgentMessageResult,
} from '../../types'
import { collectApprovals, eventName, roleName, toneFromEvent } from '../agent/helpers'

type PodContainer = {
  name: string
}

const resourceKinds = [
  { value: 'pods', label: 'Pod' },
  { value: 'deployments', label: 'Deployment' },
  { value: 'services', label: 'Service' },
  { value: 'configmaps', label: 'ConfigMap' },
  { value: 'secrets', label: 'Secret' },
]

export function ClustersPage() {
  const { session } = useSessionStore()
  const [clusters, setClusters] = useState<ClusterRecord[]>([])
  const [selectedClusterId, setSelectedClusterId] = useState<number | null>(null)
  const [overview, setOverview] = useState<ClusterOverviewRecord | null>(null)
  const [validation, setValidation] = useState<ClusterValidationRecord | null>(null)
  const [namespaces, setNamespaces] = useState<NamespaceRecord[]>([])
  const [resources, setResources] = useState<ResourceRecord[]>([])
  const [events, setEvents] = useState<EventRecord[]>([])
  const [resourceDetail, setResourceDetail] = useState<ResourceDetail | null>(null)
  const [selectedResource, setSelectedResource] = useState<ResourceRecord | null>(null)
  const [resourceType, setResourceType] = useState('pods')
  const [namespace, setNamespace] = useState('default')
  const [scaleReplicas, setScaleReplicas] = useState('2')
  const [applyManifest, setApplyManifest] = useState('')
  const [actionRunId, setActionRunId] = useState<number | null>(null)
  const [actionEvents, setActionEvents] = useState<AgentEvent[]>([])
  const [actionError, setActionError] = useState('')
  const [workspaceError, setWorkspaceError] = useState('')
  const [podContainers, setPodContainers] = useState<PodContainer[]>([])
  const [selectedContainer, setSelectedContainer] = useState('')
  const [logLines, setLogLines] = useState<string[]>([])
  const [logError, setLogError] = useState('')
  const [logFollowing, setLogFollowing] = useState(false)
  const actionStreamCleanupRef = useRef<null | (() => void)>(null)
  const podLogCleanupRef = useRef<null | (() => void)>(null)

  const canManageClusters = session?.user.role === 'admin' || session?.user.role === 'cluster_admin'

  useEffect(() => {
    if (!session || !canManageClusters) {
      return
    }

    void api.listClusters(session).then((items) => {
      setClusters(items)
      if (items.length > 0) {
        setSelectedClusterId((current) => current ?? items[0].id)
      }
    })
  }, [canManageClusters, session])

  useEffect(() => {
    if (!session || !selectedClusterId || !canManageClusters) {
      return
    }

    void Promise.all([api.listNamespaces(session, selectedClusterId), api.getClusterOverview(session, selectedClusterId)])
      .then(([namespaceItems, overviewRecord]) => {
        setNamespaces(namespaceItems)
        setOverview(overviewRecord)
        setWorkspaceError('')
        if (namespaceItems.length > 0) {
          setNamespace((current) => (namespaceItems.some((item) => item.name === current) ? current : namespaceItems[0].name))
        }
      })
      .catch((error) => {
        setWorkspaceError(error instanceof Error ? error.message : '加载集群工作台失败')
      })
  }, [canManageClusters, selectedClusterId, session])

  useEffect(() => {
    if (!session || !selectedClusterId || !canManageClusters) {
      return
    }

    void Promise.all([
      api.listResources(session, selectedClusterId, resourceType, namespace),
      api.listEvents(session, selectedClusterId, namespace),
    ])
      .then(([resourceItems, eventItems]) => {
        setResources(resourceItems)
        setEvents(eventItems)
        setWorkspaceError('')
      })
      .catch((error) => {
        setWorkspaceError(error instanceof Error ? error.message : '加载资源列表失败')
      })
  }, [canManageClusters, namespace, resourceType, selectedClusterId, session])

  useEffect(() => {
    return () => {
      actionStreamCleanupRef.current?.()
      podLogCleanupRef.current?.()
    }
  }, [])

  const selectedCluster = useMemo(
    () => clusters.find((item) => item.id === selectedClusterId) ?? null,
    [clusters, selectedClusterId],
  )

  const approvals = useMemo(() => collectApprovals(actionEvents), [actionEvents])

  function resetPodLogs() {
    podLogCleanupRef.current?.()
    podLogCleanupRef.current = null
    setLogLines([])
    setLogError('')
    setLogFollowing(false)
    setPodContainers([])
    setSelectedContainer('')
  }

  function selectCluster(clusterId: number) {
    actionStreamCleanupRef.current?.()
    setSelectedClusterId(clusterId)
    setValidation(null)
    setSelectedResource(null)
    setResourceDetail(null)
    setActionRunId(null)
    setActionEvents([])
    setActionError('')
    setWorkspaceError('')
    resetPodLogs()
  }

  async function handleValidate() {
    if (!session || !selectedClusterId) {
      return
    }

    try {
      setValidation(await api.validateCluster(session, selectedClusterId))
      setWorkspaceError('')
    } catch (error) {
      setWorkspaceError(error instanceof Error ? error.message : '校验集群失败')
    }
  }

  async function refreshWorkspace() {
    if (!session || !selectedClusterId) {
      return
    }

    try {
      const [resourceItems, eventItems, overviewRecord] = await Promise.all([
        api.listResources(session, selectedClusterId, resourceType, namespace),
        api.listEvents(session, selectedClusterId, namespace),
        api.getClusterOverview(session, selectedClusterId),
      ])
      setResources(resourceItems)
      setEvents(eventItems)
      setOverview(overviewRecord)
      setWorkspaceError('')

      if (selectedResource) {
        try {
          const detail = await api.getResource(session, selectedClusterId, selectedResource.type, selectedResource.name, selectedResource.namespace)
          setResourceDetail(detail)
          syncPodContainers(detail)
        } catch {
          setResourceDetail(null)
          resetPodLogs()
        }
      }
    } catch (error) {
      setWorkspaceError(error instanceof Error ? error.message : '刷新集群工作台失败')
    }
  }

  function syncPodContainers(detail: ResourceDetail | null) {
    if (!detail || detail.type !== 'pods') {
      resetPodLogs()
      return
    }

    const containers = extractPodContainers(detail)
    setPodContainers(containers)
    setSelectedContainer((current) => current || containers[0]?.name || '')
  }

  async function handleInspectResource(item: ResourceRecord) {
    if (!session || !selectedClusterId) {
      return
    }

    try {
      setSelectedResource(item)
      const detail = await api.getResource(session, selectedClusterId, item.type, item.name, item.namespace)
      setResourceDetail(detail)
      syncPodContainers(detail)
      setWorkspaceError('')

      if (item.type === 'pods') {
        const containers = extractPodContainers(detail)
        await startLogStream(containers[0]?.name || '', item)
      }

      const replicas = Number((detail.object?.spec as Record<string, unknown> | undefined)?.replicas)
      if (item.type === 'deployments' && Number.isFinite(replicas) && replicas > 0) {
        setScaleReplicas(String(replicas))
      }
    } catch (error) {
      setWorkspaceError(error instanceof Error ? error.message : '加载资源详情失败')
    }
  }

  async function startLogStream(containerName?: string, targetResource?: ResourceRecord) {
    const podResource = targetResource ?? selectedResource
    if (!session || !selectedClusterId || !podResource || podResource.type !== 'pods') {
      return
    }

    podLogCleanupRef.current?.()
    setLogLines([])
    setLogError('')
    setLogFollowing(true)

    try {
      podLogCleanupRef.current = await streamPodLogs(
        session,
        selectedClusterId,
        podResource.name,
        {
          namespace: podResource.namespace,
          container: containerName || selectedContainer,
          follow: true,
          tailLines: 200,
        },
        {
          onLine: (line) => {
            setLogLines((current) => {
              const next = [...current, line]
              return next.slice(-600)
            })
          },
          onError: (error) => {
            setLogFollowing(false)
            setLogError(error.message)
          },
          onDone: () => {
            setLogFollowing(false)
          },
        },
      )
    } catch (error) {
      setLogFollowing(false)
      setLogError(error instanceof Error ? error.message : '打开 Pod 日志失败')
    }
  }

  function stopLogStream() {
    podLogCleanupRef.current?.()
    podLogCleanupRef.current = null
    setLogFollowing(false)
  }

  async function handleDeleteResource() {
    if (!session || !selectedClusterId || !selectedResource) {
      return
    }

    const result = await api.requestDeleteResource(session, selectedClusterId, {
      type: selectedResource.type,
      name: selectedResource.name,
      namespace: selectedResource.namespace,
    })
    await startActionRun(result)
  }

  async function handleScaleDeployment() {
    if (!session || !selectedClusterId || !selectedResource) {
      return
    }

    const result = await api.requestScaleDeployment(session, selectedClusterId, {
      name: selectedResource.name,
      namespace: selectedResource.namespace,
      replicas: Number(scaleReplicas),
    })
    await startActionRun(result)
  }

  async function handleRestartDeployment() {
    if (!session || !selectedClusterId || !selectedResource) {
      return
    }

    const result = await api.requestRestartDeployment(session, selectedClusterId, {
      name: selectedResource.name,
      namespace: selectedResource.namespace,
    })
    await startActionRun(result)
  }

  async function handleApplyManifest() {
    if (!session || !selectedClusterId || !applyManifest.trim()) {
      return
    }

    const result = await api.requestApplyYAML(session, selectedClusterId, {
      namespace,
      manifest: applyManifest.trim(),
    })
    await startActionRun(result)
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

    if (actionRunId) {
      setActionEvents(await api.listAgentRunEvents(session, actionRunId))
      await refreshWorkspace()
    }
  }

  async function startActionRun(result: SendAgentMessageResult) {
    if (!session) {
      return
    }

    actionStreamCleanupRef.current?.()
    setActionRunId(result.runId)
    setActionError('')
    setActionEvents(await api.listAgentRunEvents(session, result.runId))

    actionStreamCleanupRef.current = await streamAgentRun(session, result.runId, {
      onEvent: (event) => {
        setActionEvents((current) => mergeEvent(current, event))
        if (event.eventType === 'tool_end' || event.eventType === 'turn_end' || event.eventType === 'message_done') {
          void refreshWorkspace()
        }
      },
      onError: (error) => {
        setActionError(error.message)
      },
    })
  }

  if (!canManageClusters) {
    return (
      <Section eyebrow="资源中心" title="集群看板" description="当前账号没有集群运维工作台权限。">
        <EmptyState title="无权访问集群看板" description="只有平台管理员和集群管理员可以查看集群资源、日志与审批操作。" />
      </Section>
    )
  }

  return (
    <div className="page-stack">
      <Section
        eyebrow="资源中心"
        title="集群看板"
        description="这里专注于集群观测、资源查看、Pod 日志与变更审批；集群接入配置请到“集群配置”页面单独维护。"
        actions={
          <div className="inline-actions">
            <select value={selectedClusterId ?? ''} onChange={(event) => selectCluster(Number(event.target.value))}>
              <option value="">选择集群</option>
              {clusters.map((item) => (
                <option key={item.id} value={item.id}>
                  {item.name}
                </option>
              ))}
            </select>
            <button className="chip-button" onClick={() => void refreshWorkspace()} type="button">
              刷新
            </button>
            <button className="button" onClick={handleValidate} type="button" disabled={!selectedClusterId}>
              校验连接
            </button>
          </div>
        }
      >
        {!selectedCluster ? <EmptyState title="还没有选中集群" description="先选择一个集群，再开始查看资源和日志。" /> : null}
        {workspaceError ? <p className="form-error">{workspaceError}</p> : null}

        {selectedCluster && overview ? (
          <>
            <div className="cluster-summary-bar">
              <MetricPill label="命名空间" value={overview.namespacesCount} />
              <MetricPill label="节点" value={`${overview.readyNodeCount}/${overview.nodeCount}`} />
              <MetricPill label="Pod" value={`${overview.runningPodCount}/${overview.podCount}`} />
              <MetricPill label="Deployment" value={`${overview.readyDeploymentCount}/${overview.deploymentCount}`} />
              <MetricPill label="Service" value={overview.serviceCount} />
            </div>

            {validation ? (
              <div className="callout callout--success">
                <strong>{validation.message}</strong>
                <p>
                  Kubernetes 版本 {validation.version}，共探测到 {validation.namespacesCount} 个命名空间。
                </p>
              </div>
            ) : null}

            <div className="inline-form">
              <select value={resourceType} onChange={(event) => setResourceType(event.target.value)}>
                {resourceKinds.map((item) => (
                  <option key={item.value} value={item.value}>
                    {item.label}
                  </option>
                ))}
              </select>
              <select value={namespace} onChange={(event) => setNamespace(event.target.value)}>
                {namespaces.map((item) => (
                  <option key={item.name} value={item.name}>
                    {item.name}
                  </option>
                ))}
              </select>
            </div>
          </>
        ) : null}
      </Section>

      <div className="page-grid page-grid--cluster-main">
        <Section eyebrow="资源浏览" title="资源列表" description="优先支持 Pod、Deployment、Service、ConfigMap 和 Secret。">
          <div className="stack-list">
            {resources.map((item) => (
              <button
                key={`${item.type}-${item.namespace}-${item.name}`}
                className={`stack-card stack-card--button ${selectedResource?.name === item.name && selectedResource?.type === item.type ? 'stack-card--selected' : ''}`}
                onClick={() => void handleInspectResource(item)}
                type="button"
              >
                <div>
                  <h4>{item.name}</h4>
                  <p className="muted">
                    {item.kind} / {item.namespace}
                  </p>
                </div>
                <div className="stack-card__meta">
                  <StatusBadge tone={resourceTone(item.status)}>{item.status || 'Unknown'}</StatusBadge>
                </div>
              </button>
            ))}
          </div>
        </Section>

        <Section eyebrow="资源详情" title={selectedResource ? selectedResource.name : '选择一个资源'} description="查看资源原始对象，并在这里发起删除、扩缩容、重启等操作。">
          {!selectedResource || !resourceDetail ? (
            <EmptyState title="还没有资源详情" description="点击左侧任意资源后，这里会显示对象详情和可执行操作。" />
          ) : (
            <div className="detail-grid">
              <div className="detail-block">
                <h4>基础信息</h4>
                <p className="muted">
                  {resourceDetail.kind} / {resourceDetail.namespace} / {resourceDetail.name}
                </p>
                <p className="muted">资源类型：{resourceDetail.type}</p>
              </div>

              {selectedResource.type === 'deployments' ? (
                <div className="detail-block">
                  <h4>Deployment 操作</h4>
                  <div className="inline-actions">
                    <input value={scaleReplicas} onChange={(event) => setScaleReplicas(event.target.value)} placeholder="副本数" />
                    <button className="button" onClick={handleScaleDeployment} type="button">
                      发起扩缩容审批
                    </button>
                    <button className="button button--secondary" onClick={handleRestartDeployment} type="button">
                      发起重启审批
                    </button>
                  </div>
                </div>
              ) : (
                <div className="detail-block">
                  <h4>资源操作</h4>
                  <div className="inline-actions">
                    <button className="button button--secondary" onClick={handleDeleteResource} type="button">
                      发起删除审批
                    </button>
                  </div>
                </div>
              )}

              <div className="detail-block detail-block--full">
                <h4>原始对象</h4>
                <pre className="code-block">{JSON.stringify(resourceDetail.object, null, 2)}</pre>
              </div>
            </div>
          )}
        </Section>
      </div>

      <div className="page-grid page-grid--cluster-main">
        <Section eyebrow="实时日志" title="Pod 日志流" description="选择一个 Pod 后，可以直接在这里实时查看日志，并对关键字做高亮。">
          {selectedResource?.type !== 'pods' ? (
            <EmptyState title="当前不是 Pod" description="先在资源列表里选择一个 Pod，这里才会显示容器日志。" />
          ) : (
            <>
              <div className="inline-actions">
                <select value={selectedContainer} onChange={(event) => setSelectedContainer(event.target.value)}>
                  {podContainers.map((item) => (
                    <option key={item.name} value={item.name}>
                      {item.name}
                    </option>
                  ))}
                </select>
                <button className="button" onClick={() => void startLogStream(selectedContainer)} type="button">
                  {logFollowing ? '重新拉取日志' : '开始跟随'}
                </button>
                <button className="button button--secondary" onClick={stopLogStream} type="button">
                  停止
                </button>
                <button className="chip-button" onClick={() => setLogLines([])} type="button">
                  清空
                </button>
              </div>

              {logError ? <p className="form-error">{logError}</p> : null}

              <div className="log-console">
                {logLines.length === 0 ? <p className="muted">还没有日志内容，点击“开始跟随”后会实时显示。</p> : null}
                {logLines.map((line, index) => (
                  <div key={`${index}-${line.slice(0, 12)}`} className={`log-line log-line--${logLineTone(line)}`}>
                    <span className="log-line__index">{index + 1}</span>
                    <code>{line || ' '}</code>
                  </div>
                ))}
              </div>
            </>
          )}
        </Section>

        <Section eyebrow="变更与事件" title="近期事件与审批时间线" description="一边看资源事件，一边看你发起的审批和执行进度。">
          <div className="stack-list">
            {events.slice(0, 8).map((item, index) => (
              <article key={`${item.reason}-${item.involvedObject}-${index}`} className="stack-card">
                <div>
                  <h4>{item.reason}</h4>
                  <p className="muted">
                    {item.namespace} / {item.involvedObject}
                  </p>
                  <p>{item.message}</p>
                </div>
                <div className="stack-card__meta">
                  <StatusBadge tone={item.type === 'Warning' ? 'danger' : 'success'}>{item.type}</StatusBadge>
                </div>
              </article>
            ))}
          </div>

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

          {actionError ? <p className="form-error">{actionError}</p> : null}
          <div className="timeline">
            {actionEvents.map((event) => (
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

      <Section eyebrow="YAML 变更" title="应用资源清单" description="初期先走审批式 Apply，后续再补 diff 预览和 dry-run。">
        <div className="form-grid">
          <label className="field field--full">
            <span>YAML Manifest</span>
            <textarea
              value={applyManifest}
              onChange={(event) => setApplyManifest(event.target.value)}
              placeholder={'apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: demo\n  namespace: default\ndata:\n  hello: world'}
            />
          </label>
          <button className="button field--full" onClick={handleApplyManifest} type="button" disabled={!selectedClusterId}>
            发起 YAML 应用审批
          </button>
        </div>
      </Section>
    </div>
  )
}

function MetricPill({ label, value }: { label: string; value: number | string }) {
  return (
    <div className="metric-pill">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  )
}

function mergeEvent(events: AgentEvent[], event: AgentEvent): AgentEvent[] {
  const next = events.filter((item) => item.id !== event.id)
  next.push(event)
  next.sort((left, right) => left.id - right.id)
  return next
}

function resourceTone(status: string): 'neutral' | 'success' | 'warning' | 'danger' {
  if (/running|ready|healthy|active/i.test(status)) {
    return 'success'
  }
  if (/pending|creating|notready|paused/i.test(status)) {
    return 'warning'
  }
  if (/failed|error|crash|backoff/i.test(status)) {
    return 'danger'
  }
  return 'neutral'
}

function logLineTone(line: string): 'default' | 'info' | 'warn' | 'error' {
  if (/\b(error|fatal|panic|exception|failed)\b/i.test(line)) {
    return 'error'
  }
  if (/\b(warn|warning|backoff|timeout)\b/i.test(line)) {
    return 'warn'
  }
  if (/\b(info|started|listening|ready|success)\b/i.test(line)) {
    return 'info'
  }
  return 'default'
}

function extractPodContainers(detail: ResourceDetail): PodContainer[] {
  const spec = detail.object?.spec as { containers?: Array<{ name?: string }> } | undefined
  return (spec?.containers ?? []).map((item) => ({ name: item.name ?? '' })).filter((item) => item.name)
}
