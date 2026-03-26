import { useEffect, useState } from 'react'

import { EmptyState, Section, StatusBadge } from '../../components/shared/Section'
import { api } from '../../lib/api'
import { useSessionStore } from '../../store/session-store'
import type { MCPRecord } from '../../types'

export function MCPPage() {
  const { session } = useSessionStore()
  const [items, setItems] = useState<MCPRecord[]>([])
  const [name, setName] = useState('')
  const [endpoint, setEndpoint] = useState('')

  useEffect(() => {
    if (!session) {
      return
    }
    void api.listMCPServers(session).then(setItems)
  }, [session])

  async function handleCreate() {
    if (!session) {
      return
    }
    const created = await api.createMCPServer(session, {
      name,
      endpoint,
      type: 'custom',
      transport: 'http',
      authType: 'none',
      description: '',
      healthStatus: 'unknown',
      isEnabled: true,
      args: [],
      headers: {},
    })
    setItems((current) => [created, ...current])
    setName('')
    setEndpoint('')
  }

  return (
    <div className="page-grid page-grid--two">
      <Section eyebrow="扩展能力" title="MCP 管理" description="注册 MCP 服务端点，为后续 Agent 工具路由和扩展能力做准备。">
        {items.length === 0 ? <EmptyState title="还没有 MCP 服务" description="先登记一个 MCP 服务地址，后续可以逐步接入外部工具。" /> : null}
        <div className="stack-list">
          {items.map((item) => (
            <article key={item.id} className="stack-card">
              <div>
                <h4>{item.name}</h4>
                <p className="muted">{item.endpoint || item.command || '尚未配置端点'}</p>
              </div>
              <div className="stack-card__meta">
                <StatusBadge tone={item.isEnabled ? 'success' : 'warning'}>{item.healthStatus}</StatusBadge>
                <span>{item.transport}</span>
              </div>
            </article>
          ))}
        </div>
      </Section>

      <Section eyebrow="创建 MCP" title="新增服务" description="当前先保留连接元数据，密钥轮换和工具浏览会在后续补齐。">
        <div className="form-grid">
          <label className="field field--full">
            <span>名称</span>
            <input value={name} onChange={(event) => setName(event.target.value)} />
          </label>
          <label className="field field--full">
            <span>服务地址</span>
            <input value={endpoint} onChange={(event) => setEndpoint(event.target.value)} placeholder="https://..." />
          </label>
          <button className="button field--full" onClick={handleCreate} type="button">
            创建 MCP 服务
          </button>
        </div>
      </Section>
    </div>
  )
}
