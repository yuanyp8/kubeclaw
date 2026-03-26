import { useState, type FormEvent } from 'react'

import { api } from '../../lib/api'
import type { Session } from '../../types'

type LoginPageProps = {
  onLogin: (session: Session) => void
}

export function LoginPage({ onLogin }: LoginPageProps) {
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

  return (
    <div className="login-layout">
      <section className="hero-card hero-card--platform">
        <div className="hero-card__main">
          <p className="section-eyebrow">KubeClaw 智能运维平台</p>
          <h1>统一管理模型、智能体、集群与安全治理。</h1>
          <p className="muted">
            平台采用固定式工作台布局，支持浅色与深色主题，后续会围绕 Agent 编排、集群审批流和统一日志中心继续扩展。
          </p>
        </div>

        <div className="hero-card__stats">
          <article className="metric-card">
            <span>智能体工作台</span>
            <strong>会话、规划、工具执行、审批和运行轨迹集中展示。</strong>
          </article>
          <article className="metric-card">
            <span>平台观测</span>
            <strong>访问日志、SQL 日志、Agent 事件与前端日志统一纳管。</strong>
          </article>
          <article className="metric-card">
            <span>集群运维</span>
            <strong>支持 K8s 查询、审批式写操作以及 kubeconfig 外网接入处理。</strong>
          </article>
        </div>
      </section>

      <section className="form-card form-card--login">
        <div className="section-heading">
          <p className="section-eyebrow">账号认证</p>
          <h2>登录平台</h2>
          <p className="muted">使用管理员或运维账号进入控制台，继续配置模型、集群、Skill 与智能体。</p>
        </div>

        <form className="form-grid" onSubmit={handleSubmit}>
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

          <button className="button field--full" disabled={submitting} type="submit">
            {submitting ? '登录中...' : '进入工作台'}
          </button>
        </form>

        <div className="login-notes">
          <div className="login-note">
            <strong>默认账号</strong>
            <span>admin / admin123456</span>
          </div>
          <div className="login-note">
            <strong>部署方式</strong>
            <span>静态前端 + API 反向代理</span>
          </div>
          <div className="login-note">
            <strong>主题支持</strong>
            <span>跟随系统、浅色、深色</span>
          </div>
        </div>
      </section>
    </div>
  )
}
