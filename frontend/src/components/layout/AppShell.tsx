import { useEffect, useMemo } from 'react'
import { NavLink, Outlet, useLocation } from 'react-router-dom'

import { useTheme } from '../../app/theme'
import { logClientEvent } from '../../lib/client-logger'
import { useSessionStore } from '../../store/session-store'

type NavItem = {
  to: string
  title: string
  description: string
  roles?: string[]
}

type NavGroup = {
  label: string
  items: NavItem[]
}

const navGroups: NavGroup[] = [
  {
    label: '概览',
    items: [
      {
        to: '/dashboard',
        title: '集群总览',
        description: '查看集群健康度、近期事件和核心指标。',
        roles: ['admin', 'cluster_admin'],
      },
      {
        to: '/agent',
        title: '智能体工作台',
        description: '查看对话、规划、工具执行和审批流。',
      },
    ],
  },
  {
    label: '集群运维',
    items: [
      {
        to: '/clusters',
        title: '集群看板',
        description: '查看资源、Pod 日志和运维审批。',
        roles: ['admin', 'cluster_admin'],
      },
      {
        to: '/cluster-settings',
        title: '集群配置',
        description: '维护集群接入、凭据和环境信息。',
        roles: ['admin', 'cluster_admin'],
      },
    ],
  },
  {
    label: '平台资源',
    items: [
      {
        to: '/models',
        title: '模型管理',
        description: '维护模型与默认模型配置。',
        roles: ['admin'],
      },
      {
        to: '/skills',
        title: '技能管理',
        description: '维护 Skill 元数据和扩展能力。',
        roles: ['admin', 'cluster_admin'],
      },
      {
        to: '/mcp',
        title: 'MCP 管理',
        description: '维护 MCP 服务端点与密钥。',
        roles: ['admin'],
      },
    ],
  },
  {
    label: '安全与审计',
    items: [
      {
        to: '/security',
        title: '安全规则',
        description: '维护白名单、敏感词和字段规则。',
        roles: ['admin'],
      },
      {
        to: '/audit',
        title: '审计日志',
        description: '查看是谁修改了什么数据。',
        roles: ['admin'],
      },
      {
        to: '/logs',
        title: '平台日志',
        description: '查看运行、访问、SQL 与前端日志。',
      },
    ],
  },
  {
    label: '身份组织',
    items: [
      {
        to: '/users',
        title: '用户管理',
        description: '维护用户、租户归属与平台角色。',
        roles: ['admin'],
      },
      {
        to: '/teams',
        title: '团队管理',
        description: '维护团队成员关系和协作边界。',
        roles: ['admin', 'cluster_admin'],
      },
      {
        to: '/tenants',
        title: '租户管理',
        description: '维护租户和多租户扩展入口。',
        roles: ['admin'],
      },
    ],
  },
]

const topShortcuts = ['总览', '节点', '工作负载', '配置', '告警']

export function AppShell() {
  const location = useLocation()
  const { session, setSession } = useSessionStore()
  const { mode, setMode } = useTheme()

  useEffect(() => {
    void logClientEvent('info', 'Route changed', {
      session,
      route: location.pathname,
      fields: { search: location.search },
    })
  }, [location.pathname, location.search, session])

  const visibleGroups = useMemo(() => {
    const role = session?.user.role
    return navGroups
      .map((group) => ({
        ...group,
        items: group.items.filter((item) => !item.roles || (role ? item.roles.includes(role) : false)),
      }))
      .filter((group) => group.items.length > 0)
  }, [session?.user.role])

  const currentItem = useMemo(
    () =>
      visibleGroups
        .flatMap((group) => group.items)
        .find((entry) => location.pathname === entry.to || location.pathname.startsWith(`${entry.to}/`)),
    [location.pathname, visibleGroups],
  )

  return (
    <div className="workspace-shell">
      <header className="workspace-header">
        <div className="workspace-brand">
          <div className="workspace-brand__mark" aria-hidden="true">
            KC
          </div>
          <div>
            <strong>KubeClaw</strong>
            <span>云原生运维平台</span>
          </div>
        </div>

        <div className="workspace-shortcuts">
          {topShortcuts.map((item, index) => (
            <button key={item} className={`workspace-shortcut ${index === 0 ? 'workspace-shortcut--active' : ''}`} type="button">
              {item.slice(0, 1)}
            </button>
          ))}
        </div>

        <div className="workspace-header__right">
          <div className="theme-switcher theme-switcher--compact">
            <button
              className={`chip-button ${mode === 'system' ? 'chip-button--active' : ''}`}
              onClick={() => setMode('system')}
              type="button"
            >
              系统
            </button>
            <button
              className={`chip-button ${mode === 'light' ? 'chip-button--active' : ''}`}
              onClick={() => setMode('light')}
              type="button"
            >
              浅色
            </button>
            <button
              className={`chip-button ${mode === 'dark' ? 'chip-button--active' : ''}`}
              onClick={() => setMode('dark')}
              type="button"
            >
              深色
            </button>
          </div>

          <div className="workspace-user">
            <div className="workspace-user__avatar">{(session?.user.displayName || session?.user.username || 'U').slice(0, 1)}</div>
            <div>
              <strong>{session?.user.displayName || session?.user.username || '未登录'}</strong>
              <span>{roleLabel(session?.user.role)}</span>
            </div>
          </div>

          <button className="button button--secondary" onClick={() => setSession(null)} type="button">
            退出
          </button>
        </div>
      </header>

      <div className="workspace-body">
        <aside className="workspace-sidebar">
          <div className="workspace-sidebar__panel">
            <div className="workspace-cluster-card">
              <strong>当前工作区</strong>
              <span>平台运维控制台</span>
            </div>

            <div className="sidebar-groups sidebar-groups--flat">
              {visibleGroups.map((group) => (
                <section key={group.label} className="sidebar-group sidebar-group--flat">
                  <p className="sidebar-group__label">{group.label}</p>
                  <div className="sidebar-group__items">
                    {group.items.map((item) => (
                      <NavLink
                        key={item.to}
                        className={({ isActive }) => `sidebar-link sidebar-link--flat ${isActive ? 'sidebar-link--active' : ''}`}
                        to={item.to}
                      >
                        <strong>{item.title}</strong>
                        <span>{item.description}</span>
                      </NavLink>
                    ))}
                  </div>
                </section>
              ))}
            </div>
          </div>
        </aside>

        <section className="console-main console-main--workspace">
          <div className="workspace-pagebar">
            <div>
              <p className="section-eyebrow">当前页面</p>
              <h2>{currentItem?.title ?? 'KubeClaw 控制台'}</h2>
              <p className="muted">{currentItem?.description ?? '统一的云原生运维与平台治理工作区。'}</p>
            </div>
          </div>

          <main className="console-content console-content--workspace">
            <Outlet />
          </main>
        </section>
      </div>
    </div>
  )
}

function roleLabel(role?: string): string {
  switch (role) {
    case 'admin':
      return '平台管理员'
    case 'cluster_admin':
      return '集群管理员'
    case 'readonly':
      return '只读用户'
    case 'user':
      return '普通用户'
    default:
      return '访客'
  }
}
