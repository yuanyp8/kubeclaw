import { useEffect, useMemo, useState } from 'react'

import { EmptyState, Section, StatusBadge } from '../../components/shared/Section'
import { api } from '../../lib/api'
import { useSessionStore } from '../../store/session-store'
import type { ClusterRecord, ClusterValidationRecord } from '../../types'

type ClusterForm = {
  id?: number
  name: string
  description: string
  apiServer: string
  authType: string
  environment: string
  status: string
  isPublic: boolean
  kubeConfig: string
}

const emptyForm: ClusterForm = {
  name: '',
  description: '',
  apiServer: '',
  authType: 'kubeconfig',
  environment: 'prod',
  status: 'active',
  isPublic: false,
  kubeConfig: '',
}

export function ClusterSettingsPage() {
  const { session } = useSessionStore()
  const [clusters, setClusters] = useState<ClusterRecord[]>([])
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [form, setForm] = useState<ClusterForm>(emptyForm)
  const [validation, setValidation] = useState<ClusterValidationRecord | null>(null)
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const canManageClusters = session?.user.role === 'admin' || session?.user.role === 'cluster_admin'

  useEffect(() => {
    if (!session || !canManageClusters) {
      return
    }

    void api.listClusters(session).then((items) => {
      setClusters(items)
      if (items.length > 0) {
        setSelectedId((current) => current ?? items[0].id)
      }
    })
  }, [canManageClusters, session])

  const selectedCluster = useMemo(
    () => clusters.find((item) => item.id === selectedId) ?? null,
    [clusters, selectedId],
  )

  useEffect(() => {
    if (!selectedCluster) {
      return
    }

    setForm({
      id: selectedCluster.id,
      name: selectedCluster.name,
      description: selectedCluster.description,
      apiServer: selectedCluster.apiServer,
      authType: selectedCluster.authType,
      environment: selectedCluster.environment,
      status: selectedCluster.status,
      isPublic: selectedCluster.isPublic,
      kubeConfig: '',
    })
    setValidation(null)
    setError('')
  }, [selectedCluster])

  function handleCreateMode() {
    setSelectedId(null)
    setForm(emptyForm)
    setValidation(null)
    setError('')
  }

  async function handleSave() {
    if (!session || !canManageClusters) {
      return
    }

    const payload = {
      name: form.name,
      description: form.description,
      apiServer: form.apiServer,
      authType: form.authType,
      environment: form.environment,
      status: form.status,
      isPublic: form.isPublic,
      kubeConfig: form.kubeConfig,
    }

    setSubmitting(true)
    try {
      const saved = form.id ? await api.updateCluster(session, form.id, payload) : await api.createCluster(session, payload)
      setClusters((current) => {
        const next = current.filter((item) => item.id !== saved.id)
        return [saved, ...next]
      })
      setSelectedId(saved.id)
      setError('')
      setValidation(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存集群配置失败')
    } finally {
      setSubmitting(false)
    }
  }

  async function handleDelete(id: number) {
    if (!session || !canManageClusters) {
      return
    }

    try {
      await api.deleteCluster(session, id)
      const remaining = clusters.filter((item) => item.id !== id)
      setClusters(remaining)
      if (selectedId === id) {
        setSelectedId(remaining[0]?.id ?? null)
        if (remaining.length === 0) {
          setForm(emptyForm)
        }
      }
      setValidation(null)
      setError('')
    } catch (err) {
      setError(err instanceof Error ? err.message : '删除集群失败')
    }
  }

  async function handleValidate() {
    if (!session || !selectedId) {
      return
    }

    try {
      setValidation(await api.validateCluster(session, selectedId))
      setError('')
    } catch (err) {
      setError(err instanceof Error ? err.message : '校验集群失败')
    }
  }

  if (!canManageClusters) {
    return (
      <Section eyebrow="资源中心" title="集群配置" description="当前账号没有集群配置权限。">
        <EmptyState title="无权访问集群配置" description="只有平台管理员和集群管理员可以维护集群接入配置。" />
      </Section>
    )
  }

  return (
    <div className="page-grid page-grid--two">
      <Section
        eyebrow="资源中心"
        title="集群配置"
        description="这里专门维护集群接入信息、凭据和环境属性，不再和集群看板混在同一个页面。"
        actions={
          <div className="inline-actions">
            <button className="chip-button" onClick={handleCreateMode} type="button">
              新建集群
            </button>
          </div>
        }
      >
        {clusters.length === 0 ? <EmptyState title="还没有集群配置" description="右侧表单可以录入新的 Kubernetes 集群。" /> : null}
        <div className="stack-list">
          {clusters.map((cluster) => (
            <article key={cluster.id} className={`stack-card ${selectedId === cluster.id ? 'stack-card--selected' : ''}`}>
              <div>
                <h4>{cluster.name}</h4>
                <p className="muted">{cluster.apiServer}</p>
                <p className="muted">
                  {cluster.environment} / {cluster.authType}
                </p>
              </div>
              <div className="stack-card__meta stack-card__meta--actions">
                <StatusBadge tone={cluster.status === 'active' ? 'success' : 'warning'}>{cluster.status}</StatusBadge>
                <button className="chip-button" onClick={() => setSelectedId(cluster.id)} type="button">
                  编辑
                </button>
                <button
                  className="chip-button chip-button--danger"
                  onClick={() => void handleDelete(cluster.id)}
                  type="button"
                >
                  删除
                </button>
              </div>
            </article>
          ))}
        </div>
      </Section>

      <Section
        eyebrow="配置表单"
        title={form.id ? '编辑集群配置' : '新增集群配置'}
        description="如果上传的是 kubeconfig，平台会结合外部 API Server 和 TLS ServerName 做后续处理。"
        actions={
          form.id ? (
            <div className="inline-actions">
              <button className="chip-button" onClick={handleValidate} type="button">
                校验连接
              </button>
            </div>
          ) : null
        }
      >
        {error ? <p className="form-error">{error}</p> : null}
        {validation ? (
          <div className="callout callout--success">
            <strong>{validation.message}</strong>
            <p>
              集群版本 {validation.version}，检测到 {validation.namespacesCount} 个命名空间。
            </p>
          </div>
        ) : null}

        <div className="form-grid">
          <label className="field">
            <span>集群名称</span>
            <input value={form.name} onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))} />
          </label>
          <label className="field">
            <span>API Server</span>
            <input
              value={form.apiServer}
              onChange={(event) => setForm((current) => ({ ...current, apiServer: event.target.value }))}
              placeholder="https://36.138.61.152:6443"
            />
          </label>
          <label className="field">
            <span>认证方式</span>
            <select value={form.authType} onChange={(event) => setForm((current) => ({ ...current, authType: event.target.value }))}>
              <option value="kubeconfig">kubeconfig</option>
              <option value="token">token</option>
            </select>
          </label>
          <label className="field">
            <span>环境</span>
            <select value={form.environment} onChange={(event) => setForm((current) => ({ ...current, environment: event.target.value }))}>
              <option value="dev">dev</option>
              <option value="test">test</option>
              <option value="prod">prod</option>
            </select>
          </label>
          <label className="field">
            <span>状态</span>
            <select value={form.status} onChange={(event) => setForm((current) => ({ ...current, status: event.target.value }))}>
              <option value="active">active</option>
              <option value="inactive">inactive</option>
            </select>
          </label>
          <label className="field">
            <span>共享范围</span>
            <select
              value={form.isPublic ? 'true' : 'false'}
              onChange={(event) => setForm((current) => ({ ...current, isPublic: event.target.value === 'true' }))}
            >
              <option value="false">私有</option>
              <option value="true">公开</option>
            </select>
          </label>
          <label className="field field--full">
            <span>描述</span>
            <textarea value={form.description} onChange={(event) => setForm((current) => ({ ...current, description: event.target.value }))} />
          </label>
          <label className="field field--full">
            <span>KubeConfig</span>
            <textarea
              value={form.kubeConfig}
              onChange={(event) => setForm((current) => ({ ...current, kubeConfig: event.target.value }))}
              placeholder="粘贴 kubeconfig。编辑已有集群时，如果不想替换凭据，可以保持为空。"
            />
          </label>
          <div className="field field--full inline-actions">
            <button className="button" disabled={submitting} onClick={handleSave} type="button">
              {submitting ? '保存中...' : form.id ? '保存配置' : '创建集群'}
            </button>
            <button className="button button--secondary" onClick={handleCreateMode} type="button">
              清空表单
            </button>
          </div>
        </div>
      </Section>
    </div>
  )
}
