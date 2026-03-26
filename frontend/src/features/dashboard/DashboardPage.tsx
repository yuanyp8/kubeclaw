import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'

import { EmptyState, Section, StatusBadge } from '../../components/shared/Section'
import { api } from '../../lib/api'
import { useSessionStore } from '../../store/session-store'
import type { ClusterOverviewRecord, ClusterRecord } from '../../types'

export function DashboardPage() {
  const { session } = useSessionStore()
  const [clusters, setClusters] = useState<ClusterRecord[]>([])
  const [selectedClusterId, setSelectedClusterId] = useState<number | null>(null)
  const [overview, setOverview] = useState<ClusterOverviewRecord | null>(null)

  useEffect(() => {
    if (!session) {
      return
    }

    void api.listClusters(session).then((items) => {
      setClusters(items)
      if (items.length > 0) {
        setSelectedClusterId((current) => current ?? items[0].id)
      }
    })
  }, [session])

  useEffect(() => {
    if (!session || !selectedClusterId) {
      return
    }

    void api.getClusterOverview(session, selectedClusterId).then(setOverview)
  }, [selectedClusterId, session])

  const selectedCluster = useMemo(
    () => clusters.find((item) => item.id === selectedClusterId) ?? null,
    [clusters, selectedClusterId],
  )
  const loading = Boolean(selectedClusterId && overview?.clusterId !== selectedClusterId)

  return (
    <div className="page-stack">
      <Section
        eyebrow="工作台"
        title="集群总览 Dashboard"
        description="参考 KubeSphere 的首页思路，先把最常用的集群观测能力放到一屏：健康概况、异常 Pod、Deployment 状态和近期事件。"
        actions={
          <div className="inline-actions">
            <select
              value={selectedClusterId ?? ''}
              onChange={(event) => setSelectedClusterId(event.target.value ? Number(event.target.value) : null)}
            >
              <option value="">选择集群</option>
              {clusters.map((item) => (
                <option key={item.id} value={item.id}>
                  {item.name}
                </option>
              ))}
            </select>
            <Link className="button button--secondary" to="/clusters">
              打开集群工作台
            </Link>
          </div>
        }
      >
        {!selectedCluster ? (
          <EmptyState title="还没有可用集群" description="先到集群工作台录入一个 Kubernetes 集群，Dashboard 才能展示实时概况。" />
        ) : null}

        {selectedCluster && overview ? (
          <>
            <div className="dashboard-hero">
              <div className="dashboard-hero__main">
                <p className="section-eyebrow">当前集群</p>
                <h2>{selectedCluster.name}</h2>
                <p className="muted">
                  {selectedCluster.apiServer} · 最近采样于 {new Date(overview.collectedAt).toLocaleString()}
                </p>
              </div>
              <div className="dashboard-hero__meta">
                <StatusBadge tone={selectedCluster.status === 'active' ? 'success' : 'warning'}>
                  {selectedCluster.status}
                </StatusBadge>
                <StatusBadge tone={overview.failedPodCount > 0 ? 'danger' : 'success'}>
                  异常 Pod {overview.failedPodCount + overview.pendingPodCount}
                </StatusBadge>
              </div>
            </div>

            <div className="metric-grid metric-grid--five">
              <MetricCard label="命名空间" value={overview.namespacesCount} helper="租户与业务隔离范围" />
              <MetricCard
                label="节点"
                value={`${overview.readyNodeCount}/${overview.nodeCount}`}
                helper="Ready / Total"
              />
              <MetricCard
                label="Pod"
                value={`${overview.runningPodCount}/${overview.podCount}`}
                helper={`Pending ${overview.pendingPodCount} · Failed ${overview.failedPodCount}`}
              />
              <MetricCard
                label="Deployment"
                value={`${overview.readyDeploymentCount}/${overview.deploymentCount}`}
                helper="Ready / Total"
              />
              <MetricCard label="Service" value={overview.serviceCount} helper="服务暴露与流量入口" />
            </div>
          </>
        ) : null}

        {loading ? <p className="muted">正在拉取集群总览...</p> : null}
      </Section>

      {overview ? (
        <div className="page-grid page-grid--dashboard">
          <Section eyebrow="异常排查" title="重点 Pod" description="优先看不健康 Pod、未 Ready 容器和重启次数高的实例。">
            {overview.problemPods.length === 0 ? (
              <EmptyState title="当前没有明显异常 Pod" description="所有容器都比较健康时，这里会保持干净。" />
            ) : (
              <div className="stack-list">
                {overview.problemPods.map((item) => (
                  <article key={`${item.namespace}-${item.name}`} className="stack-card">
                    <div>
                      <h4>{item.name}</h4>
                      <p className="muted">
                        {item.namespace} · {item.nodeName || '未调度节点'}
                      </p>
                      <p className="muted">
                        Ready {item.readyContainers}/{item.totalContainers} · Restart {item.restartCount}
                      </p>
                    </div>
                    <div className="stack-card__meta">
                      <StatusBadge tone={podTone(item.status)}>{item.status}</StatusBadge>
                    </div>
                  </article>
                ))}
              </div>
            )}
          </Section>

          <Section eyebrow="工作负载" title="Deployment 状态" description="快速看发布面是否健康，后续可以继续扩成滚动发布、灰度和回滚入口。">
            <div className="stack-list">
              {overview.deployments.map((item) => (
                <article key={`${item.namespace}-${item.name}`} className="stack-card">
                  <div>
                    <h4>{item.name}</h4>
                    <p className="muted">
                      {item.namespace} · Ready {item.readyReplicas}/{item.replicas}
                    </p>
                    <p className="muted">
                      Available {item.availableReplicas} · Updated {item.updatedReplicas}
                    </p>
                  </div>
                  <div className="stack-card__meta">
                    <StatusBadge tone={item.status === 'Healthy' ? 'success' : 'warning'}>{item.status}</StatusBadge>
                  </div>
                </article>
              ))}
            </div>
          </Section>
        </div>
      ) : null}

      {overview ? (
        <Section eyebrow="事件" title="近期事件" description="这里适合先看调度失败、镜像拉取失败、探针失败等集群噪声。">
          <div className="stack-list">
            {overview.recentEvents.map((item, index) => (
              <article key={`${item.reason}-${item.involvedObject}-${index}`} className="stack-card">
                <div>
                  <h4>{item.reason}</h4>
                  <p className="muted">
                    {item.namespace} · {item.involvedObject}
                  </p>
                  <p>{item.message}</p>
                </div>
                <div className="stack-card__meta">
                  <StatusBadge tone={item.type === 'Warning' ? 'danger' : 'success'}>{item.type}</StatusBadge>
                </div>
              </article>
            ))}
          </div>
        </Section>
      ) : null}
    </div>
  )
}

function MetricCard({ label, value, helper }: { label: string; value: string | number; helper: string }) {
  return (
    <article className="metric-card">
      <span>{label}</span>
      <strong>{value}</strong>
      <p className="muted">{helper}</p>
    </article>
  )
}

function podTone(status: string): 'neutral' | 'success' | 'warning' | 'danger' {
  switch (status) {
    case 'Running':
      return 'success'
    case 'Pending':
      return 'warning'
    case 'Failed':
      return 'danger'
    default:
      return 'neutral'
  }
}
