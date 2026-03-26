import { useEffect, useState } from 'react'

import { EmptyState, Section, StatusBadge } from '../../components/shared/Section'
import { api } from '../../lib/api'
import { useSessionStore } from '../../store/session-store'
import type { TenantRecord } from '../../types'

export function TenantsPage() {
  const { session } = useSessionStore()
  const [tenants, setTenants] = useState<TenantRecord[]>([])
  const [name, setName] = useState('')
  const [slug, setSlug] = useState('')
  const [description, setDescription] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    if (!session) {
      return
    }
    void api.listTenants(session).then(setTenants).catch((err) => {
      setError(err instanceof Error ? err.message : '加载租户失败')
    })
  }, [session])

  async function handleCreate() {
    if (!session) {
      return
    }
    try {
      const created = await api.createTenant(session, { name, slug, description, status: 'active' })
      setTenants((current) => [created, ...current])
      setName('')
      setSlug('')
      setDescription('')
    } catch (err) {
      setError(err instanceof Error ? err.message : '创建租户失败')
    }
  }

  return (
    <div className="page-grid page-grid--two">
      <Section eyebrow="身份组织" title="租户管理" description="租户记录现在会统计关联用户和团队数量，便于观察组织边界。">
        {error ? <p className="form-error">{error}</p> : null}
        {tenants.length === 0 ? <EmptyState title="还没有租户" description="创建租户后，可以在用户和团队页面中继续绑定关系。" /> : null}
        <div className="stack-list">
          {tenants.map((tenant) => (
            <article key={tenant.id} className="stack-card">
              <div>
                <h4>{tenant.name}</h4>
                <p className="muted">{tenant.slug}</p>
                <p className="muted">
                  用户 {tenant.userCount} / 团队 {tenant.teamCount}
                </p>
              </div>
              <div className="stack-card__meta">
                <StatusBadge tone={tenant.status === 'active' ? 'success' : 'warning'}>{tenant.status}</StatusBadge>
              </div>
            </article>
          ))}
        </div>
      </Section>

      <Section eyebrow="创建租户" title="新增租户" description="这里保留租户主数据，后续可以继续扩展策略、存储和隔离配置。">
        <div className="form-grid">
          <label className="field field--full">
            <span>租户名称</span>
            <input value={name} onChange={(event) => setName(event.target.value)} />
          </label>
          <label className="field field--full">
            <span>租户标识</span>
            <input value={slug} onChange={(event) => setSlug(event.target.value)} />
          </label>
          <label className="field field--full">
            <span>描述</span>
            <textarea value={description} onChange={(event) => setDescription(event.target.value)} />
          </label>
          <button className="button field--full" onClick={handleCreate} type="button">
            创建租户
          </button>
        </div>
      </Section>
    </div>
  )
}
