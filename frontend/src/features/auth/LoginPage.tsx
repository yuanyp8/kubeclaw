import { useState, type FormEvent } from 'react'

import { useTheme } from '../../app/theme'
import { api } from '../../lib/api'
import type { Session } from '../../types'

type LoginPageProps = {
  onLogin: (session: Session) => void
}

const quickAccounts = [
  {
    label: '默认管理员',
    login: 'admin',
    password: 'admin123456',
    hint: '拥有平台配置、模型、审计与安全治理权限',
  },
]

const featureCards = [
  {
    eyebrow: 'Agent Runtime',
    title: '把问答、计划、审批和执行轨迹放进同一条运行链路',
  },
  {
    eyebrow: 'Cluster Ops',
    title: '用模型辅助定位资源、查看日志，再通过审批执行变更',
  },
  {
    eyebrow: 'Platform Guardrails',
    title: '统一纳管模型、日志、安全规则和多租户边界',
  },
]

const controlSignals = [
  { label: '响应路径', value: '会话 -> Run -> Event -> Approval' },
  { label: '交互方式', value: 'Chat + Console + Stream' },
  { label: '主题状态', value: 'System / Light / Dark' },
]

export function LoginPage({ onLogin }: LoginPageProps) {
  const { mode, resolvedTheme, setMode } = useTheme()
  const [login, setLogin] = useState('admin')
  const [password, setPassword] = useState('admin123456')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSubmitting(true)
    setError('')

    try {
      const session = await api.login(login, password)
      onLogin(session)
    } catch (err) {
      setError(err instanceof Error ? err.message : '登录失败，请检查账号或密码。')
    } finally {
      setSubmitting(false)
    }
  }

  function applyQuickAccount(nextLogin: string, nextPassword: string) {
    setLogin(nextLogin)
    setPassword(nextPassword)
    setError('')
  }

  return (
    <div className="login-layout">
      <section className="login-stage">
        <div className="login-stage__topbar">
          <div className="login-brand">
            <div className="login-brand__mark" aria-hidden="true">
              KC
            </div>
            <div>
              <p className="login-brand__eyebrow">KubeClaw Control Surface</p>
              <strong>KubeClaw</strong>
            </div>
          </div>

          <div className="theme-switcher login-theme-switcher">
            <button
              className={`chip-button ${mode === 'system' ? 'chip-button--active' : ''}`}
              onClick={() => setMode('system')}
              type="button"
            >
              System
            </button>
            <button
              className={`chip-button ${mode === 'light' ? 'chip-button--active' : ''}`}
              onClick={() => setMode('light')}
              type="button"
            >
              Light
            </button>
            <button
              className={`chip-button ${mode === 'dark' ? 'chip-button--active' : ''}`}
              onClick={() => setMode('dark')}
              type="button"
            >
              Dark
            </button>
          </div>
        </div>

        <div className="login-stage__hero">
          <p className="login-stage__kicker">Cloud-native operations cockpit</p>
          <h1>把模型、集群、日志与审批流程收束到一张更可靠的控制台。</h1>
          <p className="login-stage__body">
            这不是一个普通登录页，而是进入运维工作面的入口。你会在这里接管 Agent Run、集群变更、模型治理与平台观测。
          </p>

          <div className="login-signal-strip">
            {controlSignals.map((item) => (
              <article key={item.label} className="login-signal">
                <span>{item.label}</span>
                <strong>{item.value}</strong>
              </article>
            ))}
          </div>
        </div>

        <div className="login-stage__grid">
          {featureCards.map((item) => (
            <article key={item.eyebrow} className="login-feature-card">
              <p>{item.eyebrow}</p>
              <h2>{item.title}</h2>
            </article>
          ))}
        </div>

        <div className="login-stage__footer">
          <div className="login-status-pill">
            <span>Resolved theme</span>
            <strong>{resolvedTheme}</strong>
          </div>
          <div className="login-status-pill">
            <span>Recommended flow</span>
            <strong>{'Login -> Inspect -> Plan -> Approve -> Execute'}</strong>
          </div>
        </div>
      </section>

      <section className="login-panel">
        <div className="login-panel__header">
          <p className="section-eyebrow">Account Access</p>
          <h2>登录工作台</h2>
          <p className="muted">使用管理员或运维账号进入平台，继续处理模型配置、Agent 会话、集群巡检与审计追踪。</p>
        </div>

        <div className="login-quick-accounts">
          {quickAccounts.map((account) => (
            <button
              key={account.label}
              className="login-quick-account"
              onClick={() => applyQuickAccount(account.login, account.password)}
              type="button"
            >
              <strong>{account.label}</strong>
              <span>{account.hint}</span>
              <code>
                {account.login} / {account.password}
              </code>
            </button>
          ))}
        </div>

        <form className="login-form" onSubmit={handleSubmit}>
          <label className="field field--full">
            <span>登录账号</span>
            <input value={login} onChange={(event) => setLogin(event.target.value)} placeholder="用户名或邮箱" />
          </label>

          <label className="field field--full">
            <span>登录密码</span>
            <input
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              placeholder="请输入密码"
            />
          </label>

          {error ? <p className="form-error field--full">{error}</p> : null}

          <button className="button field--full login-submit" disabled={submitting} type="submit">
            {submitting ? '正在验证身份...' : '进入控制台'}
          </button>
        </form>

        <div className="login-panel__meta">
          <article className="login-meta-card">
            <span>默认入口</span>
            <strong>管理员演示账号已预填</strong>
          </article>
          <article className="login-meta-card">
            <span>安全基线</span>
            <strong>请求链路、审批记录与平台日志统一可追踪</strong>
          </article>
          <article className="login-meta-card">
            <span>界面模式</span>
            <strong>主题切换已和登录页、工作台共用同一套变量</strong>
          </article>
        </div>
      </section>
    </div>
  )
}
