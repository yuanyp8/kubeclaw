import { useEffect, useMemo, useState } from 'react'

import { EmptyState, Section, StatusBadge } from '../../components/shared/Section'
import { api } from '../../lib/api'
import { useSessionStore } from '../../store/session-store'
import type { TeamMemberRecord, TeamRecord, TenantRecord, UserRecord } from '../../types'

export function TeamsPage() {
  const { session } = useSessionStore()
  const [teams, setTeams] = useState<TeamRecord[]>([])
  const [users, setUsers] = useState<UserRecord[]>([])
  const [tenants, setTenants] = useState<TenantRecord[]>([])
  const [members, setMembers] = useState<TeamMemberRecord[]>([])
  const [selectedTeamId, setSelectedTeamId] = useState<number | null>(null)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [visibility, setVisibility] = useState('private')
  const [tenantId, setTenantId] = useState('')
  const [memberUserId, setMemberUserId] = useState('')
  const [memberRole, setMemberRole] = useState('member')
  const [error, setError] = useState('')

  const role = session?.user.role
  const canManageTeams = role === 'admin' || role === 'cluster_admin'
  const canManageTenants = role === 'admin'

  useEffect(() => {
    if (!session || !canManageTeams) {
      return
    }

    void (async () => {
      try {
        const teamItems = await api.listTeams(session)
        setTeams(teamItems)
        if (teamItems.length > 0) {
          setSelectedTeamId((current) => current ?? teamItems[0].id)
        }
        setError('')
      } catch (err) {
        setError(err instanceof Error ? err.message : '加载团队数据失败')
      }
    })()
  }, [canManageTeams, session])

  useEffect(() => {
    if (!session || !selectedTeamId || !canManageTeams) {
      return
    }

    void api
      .listTeamMembers(session, selectedTeamId)
      .then((items) => {
        setMembers(items)
        setError('')
      })
      .catch((err) => {
        setError(err instanceof Error ? err.message : '加载团队成员失败')
      })
  }, [canManageTeams, selectedTeamId, session])

  useEffect(() => {
    if (!session || !canManageTenants) {
      return
    }

    void api
      .listUsers(session)
      .then(setUsers)
      .catch((err) => setError(err instanceof Error ? err.message : '加载用户列表失败'))

    void api
      .listTenants(session)
      .then(setTenants)
      .catch((err) => setError(err instanceof Error ? err.message : '加载租户列表失败'))
  }, [canManageTenants, session])

  const selectedTeam = useMemo(() => teams.find((item) => item.id === selectedTeamId) ?? null, [selectedTeamId, teams])

  async function handleCreate() {
    if (!session || !canManageTeams) {
      return
    }
    try {
      const created = await api.createTeam(session, {
        tenantId: canManageTenants ? (tenantId ? Number(tenantId) : null) : session.user.tenantId ?? null,
        name,
        description,
        visibility,
      })
      setTeams((current) => [created, ...current])
      setSelectedTeamId(created.id)
      setName('')
      setDescription('')
      setTenantId('')
      setError('')
    } catch (err) {
      setError(err instanceof Error ? err.message : '创建团队失败')
    }
  }

  async function handleAddMember() {
    if (!session || !selectedTeamId || !memberUserId || !canManageTenants) {
      return
    }
    try {
      const created = await api.addTeamMember(session, selectedTeamId, {
        userId: Number(memberUserId),
        role: memberRole,
      })
      setMembers((current) => {
        const next = current.filter((item) => item.userId !== created.userId)
        return [...next, created]
      })
      setTeams((current) =>
        current.map((item) =>
          item.id === selectedTeamId ? { ...item, memberCount: Math.max(0, item.memberCount + (members.some((member) => member.userId === created.userId) ? 0 : 1)) } : item,
        ),
      )
      setMemberUserId('')
      setMemberRole('member')
      setError('')
    } catch (err) {
      setError(err instanceof Error ? err.message : '添加团队成员失败')
    }
  }

  async function handleRemoveMember(userId: number) {
    if (!session || !selectedTeamId || !canManageTeams) {
      return
    }
    try {
      await api.removeTeamMember(session, selectedTeamId, userId)
      const freshMembers = await api.listTeamMembers(session, selectedTeamId)
      setMembers(freshMembers)
      setTeams((current) =>
        current.map((item) => (item.id === selectedTeamId ? { ...item, memberCount: freshMembers.length } : item)),
      )
      setError('')
    } catch (err) {
      setError(err instanceof Error ? err.message : '移除团队成员失败')
    }
  }

  if (!canManageTeams) {
    return (
      <Section eyebrow="身份组织" title="团队管理" description="当前账号没有团队管理权限。">
        <EmptyState title="无权访问团队管理" description="只有平台管理员和集群管理员可以查看团队与成员关系。" />
      </Section>
    )
  }

  return (
    <div className="page-grid page-grid--wide">
      <Section eyebrow="身份组织" title="团队管理" description="团队承载协作边界和成员关系。移除成员后会立即刷新团队成员列表与计数。">
        {error ? <p className="form-error">{error}</p> : null}
        {teams.length === 0 ? <EmptyState title="还没有团队" description="先创建一个团队，再把用户加入进去。" /> : null}
        <div className="stack-list">
          {teams.map((team) => (
            <button
              key={team.id}
              className={`stack-card stack-card--button ${selectedTeamId === team.id ? 'stack-card--selected' : ''}`}
              onClick={() => setSelectedTeamId(team.id)}
              type="button"
            >
              <div>
                <h4>{team.name}</h4>
                <p className="muted">{team.description || '暂无描述'}</p>
                <p className="muted">
                  租户：{team.tenant?.name ?? '未绑定租户'} / 成员数：{team.memberCount}
                </p>
              </div>
              <div className="stack-card__meta">
                <StatusBadge>{visibilityLabel(team.visibility)}</StatusBadge>
              </div>
            </button>
          ))}
        </div>
      </Section>

      <Section eyebrow="团队详情" title={selectedTeam?.name ?? '成员与归属'} description="右侧用于创建团队、查看成员，并在有权限时维护团队成员。">
        <div className="form-grid">
          <label className="field">
            <span>所属租户</span>
            <select value={tenantId} disabled={!canManageTenants} onChange={(event) => setTenantId(event.target.value)}>
              <option value="">不绑定租户</option>
              {tenants.map((item) => (
                <option key={item.id} value={item.id}>
                  {item.name}
                </option>
              ))}
            </select>
          </label>
          <label className="field">
            <span>可见范围</span>
            <select value={visibility} onChange={(event) => setVisibility(event.target.value)}>
              <option value="private">私有</option>
              <option value="internal">组织内</option>
              <option value="public">公开</option>
            </select>
          </label>
          <label className="field field--full">
            <span>团队名称</span>
            <input value={name} onChange={(event) => setName(event.target.value)} />
          </label>
          <label className="field field--full">
            <span>描述</span>
            <textarea value={description} onChange={(event) => setDescription(event.target.value)} />
          </label>
          <button className="button field--full" onClick={handleCreate} type="button">
            创建团队
          </button>
        </div>

        {selectedTeam ? (
          <>
            {canManageTenants ? (
              <div className="form-grid">
                <label className="field">
                  <span>选择用户</span>
                  <select value={memberUserId} onChange={(event) => setMemberUserId(event.target.value)}>
                    <option value="">请选择用户</option>
                    {users.map((item) => (
                      <option key={item.id} value={item.id}>
                        {item.displayName || item.username}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="field">
                  <span>团队角色</span>
                  <select value={memberRole} onChange={(event) => setMemberRole(event.target.value)}>
                    <option value="member">成员</option>
                    <option value="admin">管理员</option>
                    <option value="owner">负责人</option>
                  </select>
                </label>
                <button className="button field--full" onClick={handleAddMember} type="button">
                  添加团队成员
                </button>
              </div>
            ) : (
              <div className="callout">
                <strong>当前账号可查看和移除成员</strong>
                <p>如需新增成员，请使用平台管理员账号进入此页面操作。</p>
              </div>
            )}

            {members.length === 0 ? <EmptyState title="还没有成员" description="先选择一个用户并添加到当前团队。" /> : null}
            <div className="stack-list">
              {members.map((member) => (
                <article key={member.id} className="stack-card">
                  <div>
                    <h4>{member.displayName || member.username}</h4>
                    <p className="muted">{member.email}</p>
                  </div>
                  <div className="stack-card__meta stack-card__meta--actions">
                    <StatusBadge>{memberRoleLabel(member.role)}</StatusBadge>
                    <button className="chip-button chip-button--danger" onClick={() => void handleRemoveMember(member.userId)} type="button">
                      移除
                    </button>
                  </div>
                </article>
              ))}
            </div>
          </>
        ) : (
          <EmptyState title="先选择一个团队" description="左侧选择团队后，这里会显示成员列表和成员管理入口。" />
        )}
      </Section>
    </div>
  )
}

function visibilityLabel(value: string): string {
  switch (value) {
    case 'private':
      return '私有'
    case 'internal':
      return '组织内'
    case 'public':
      return '公开'
    default:
      return value
  }
}

function memberRoleLabel(value: string): string {
  switch (value) {
    case 'owner':
      return '负责人'
    case 'admin':
      return '管理员'
    default:
      return '成员'
  }
}
