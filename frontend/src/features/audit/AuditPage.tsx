import { useEffect, useState } from 'react'

import { EmptyState, Section, StatusBadge } from '../../components/shared/Section'
import { api } from '../../lib/api'
import { useSessionStore } from '../../store/session-store'
import type { AuditRecord } from '../../types'

export function AuditPage() {
  const { session } = useSessionStore()
  const [items, setItems] = useState<AuditRecord[]>([])

  useEffect(() => {
    if (!session) {
      return
    }
    void api.listAudit(session).then(setItems)
  }, [session])

  return (
    <Section eyebrow="治理审计" title="审计日志" description="这里展示 MySQL 持久化的审计记录，与平台实时日志分开管理。">
      {items.length === 0 ? <EmptyState title="当前没有审计记录" description="当访问、写操作或治理动作进入后端后，这里会出现对应审计信息。" /> : null}
      <div className="stack-list">
        {items.map((item) => (
          <article key={item.id} className="stack-card">
            <div>
              <h4>{item.action}</h4>
              <p className="muted">{item.target || item.details}</p>
            </div>
            <div className="stack-card__meta">
              <StatusBadge>{item.ip}</StatusBadge>
              <span>{new Date(item.createdAt).toLocaleString()}</span>
            </div>
          </article>
        ))}
      </div>
    </Section>
  )
}
