import { useEffect, useMemo, useState } from 'react'

import { EmptyState, Section, StatusBadge } from '../../components/shared/Section'
import { api } from '../../lib/api'
import { useSessionStore } from '../../store/session-store'
import type { TenantRecord, UserRecord } from '../../types'

export function UsersPage() {
  const { session } = useSessionStore()
  const [users, setUsers] = useState<UserRecord[]>([])
  const [tenants, setTenants] = useState<TenantRecord[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [form, setForm] = useState({
    tenantId: '',
    username: '',
    email: '',
    displayName: '',
    password: '',
    role: 'user',
  })

  const role = session?.user.role
  const canViewUsers = role === 'admin' || role === 'cluster_admin'
  const canManageUsers = role === 'admin'

  useEffect(() => {
    if (!session || !canViewUsers) {
      setLoading(false)
      return
    }

    void (async () => {
      try {
        setLoading(true)
        const userItems = await api.listUsers(session)
        setUsers(userItems)

        if (canManageUsers) {
          try {
            const tenantItems = await api.listTenants(session)
            setTenants(tenantItems)
          } catch {
            setTenants([])
          }
        } else {
          setTenants([])
        }

        setError('')
      } catch (err) {
        setUsers([])
        setError(err instanceof Error ? err.message : '加载用户数据失败')
      } finally {
        setLoading(false)
      }
    })()
  }, [canManageUsers, canViewUsers, session])

  const tenantOptions = useMemo(
    () => tenants.map((item) => ({ value: String(item.id), label: `${item.name} (${item.slug})` })),
    [tenants],
  )

  async function handleCreate() {
    if (!session || !canManageUsers) {
      return
    }

    try {
      const next = await api.createUser(session, {
        tenantId: form.tenantId ? Number(form.tenantId) : null,
        username: form.username,
        email: form.email,
        displayName: form.displayName,
        password: form.password,
        role: form.role,
      })
      setUsers((current) => [next, ...current])
      setForm({ tenantId: '', username: '', email: '', displayName: '', password: '', role: 'user' })
      setError('')
    } catch (err) {
      setError(err instanceof Error ? err.message : '创建用户失败')
    }
  }

  async function handleDelete(id: number) {
    if (!session || !canManageUsers) {
      return
    }

    try {
      await api.deleteUser(session, id)
      setUsers((current) => current.filter((item) => item.id !== id))
      setError('')
    } catch (err) {
      setError(err instanceof Error ? err.message : '删除用户失败')
    }
  }

  if (!canViewUsers) {
    return (
      <Section eyebrow="Identity" title="用户管理" description="当前账号没有用户查看权限。">
        <EmptyState title="无权访问用户管理" description="目前只有平台管理员和集群管理员可以查看用户列表。" />
      </Section>
    )
  }

  return (
    <div className="page-grid page-grid--two">
      <Section
        eyebrow="Identity"
        title="用户管理"
        description={canManageUsers ? '查看并维护平台用户、租户归属和角色分配。' : '当前账号为只读模式，仅展示你所在租户下的用户。'}
      >
        {loading ? <p className="muted">正在加载用户列表...</p> : null}
        {error ? <p className="form-error">{error}</p> : null}
        {!canManageUsers ? (
          <div className="callout">
            <strong>只读查看</strong>
            <p className="muted">Cluster Admin 只能查看本租户用户列表；创建、删除和角色调整仍需管理员处理。</p>
          </div>
        ) : null}
        {!loading && users.length === 0 ? (
          <EmptyState
            title="当前没有可展示的用户"
            description={canManageUsers ? '可以在右侧创建平台用户。' : '如果数据库里已有用户但这里为空，请确认这些用户绑定到了你当前租户。'}
          />
        ) : null}

        {users.length > 0 ? (
          <div className="stack-list">
            {users.map((user) => (
              <article key={user.id} className="stack-card">
                <div>
                  <h4>{user.displayName || user.username}</h4>
                  <p className="muted">{user.email}</p>
                  <p className="muted">租户：{user.tenant?.name ?? '未绑定租户'}</p>
                  <p className="muted">
                    团队：
                    {user.teams.length > 0 ? user.teams.map((item) => `${item.teamName}(${item.role})`).join('，') : '未加入团队'}
                  </p>
                </div>
                <div className="stack-card__meta stack-card__meta--actions">
                  <StatusBadge tone={user.status === 'active' ? 'success' : 'warning'}>{roleLabel(user.role)}</StatusBadge>
                  {canManageUsers ? (
                    <button className="chip-button chip-button--danger" onClick={() => void handleDelete(user.id)} type="button">
                      删除用户
                    </button>
                  ) : (
                    <span className="muted">只读</span>
                  )}
                </div>
              </article>
            ))}
          </div>
        ) : null}
      </Section>

      <Section
        eyebrow={canManageUsers ? 'Create' : 'Notice'}
        title={canManageUsers ? '新增平台用户' : '管理权限说明'}
        description={
          canManageUsers
            ? '管理员可以创建用户并直接绑定租户，团队关系可后续在团队页面维护。'
            : '当前账号无法新建或删除用户。如果需要修改用户信息，请使用管理员账号。'
        }
      >
        {canManageUsers ? (
          <div className="form-grid">
            <label className="field">
              <span>所属租户</span>
              <select value={form.tenantId} onChange={(event) => setForm((current) => ({ ...current, tenantId: event.target.value }))}>
                <option value="">不绑定租户</option>
                {tenantOptions.map((item) => (
                  <option key={item.value} value={item.value}>
                    {item.label}
                  </option>
                ))}
              </select>
            </label>
            <label className="field">
              <span>平台角色</span>
              <select value={form.role} onChange={(event) => setForm((current) => ({ ...current, role: event.target.value }))}>
                <option value="user">普通用户</option>
                <option value="cluster_admin">集群管理员</option>
                <option value="admin">平台管理员</option>
              </select>
            </label>
            <label className="field">
              <span>用户名</span>
              <input value={form.username} onChange={(event) => setForm((current) => ({ ...current, username: event.target.value }))} />
            </label>
            <label className="field">
              <span>邮箱</span>
              <input value={form.email} onChange={(event) => setForm((current) => ({ ...current, email: event.target.value }))} />
            </label>
            <label className="field field--full">
              <span>显示名称</span>
              <input value={form.displayName} onChange={(event) => setForm((current) => ({ ...current, displayName: event.target.value }))} />
            </label>
            <label className="field field--full">
              <span>登录密码</span>
              <input
                type="password"
                value={form.password}
                onChange={(event) => setForm((current) => ({ ...current, password: event.target.value }))}
              />
            </label>
            <button className="button field--full" onClick={handleCreate} type="button">
              创建用户
            </button>
          </div>
        ) : (
          <EmptyState title="当前为只读模式" description="你可以在左侧查看本租户用户信息，但创建、删除和角色修改需要管理员账号。" />
        )}
      </Section>
    </div>
  )
}

function roleLabel(role: string): string {
  switch (role) {
    case 'admin':
      return '平台管理员'
    case 'cluster_admin':
      return '集群管理员'
    case 'readonly':
      return '只读用户'
    default:
      return '普通用户'
  }
}
