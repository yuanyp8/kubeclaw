import { useEffect, useState } from 'react'

import { EmptyState, Section, StatusBadge } from '../../components/shared/Section'
import { api } from '../../lib/api'
import { useSessionStore } from '../../store/session-store'
import type { SkillRecord } from '../../types'

export function SkillsPage() {
  const { session } = useSessionStore()
  const [items, setItems] = useState<SkillRecord[]>([])
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')

  useEffect(() => {
    if (!session) {
      return
    }
    void api.listSkills(session).then(setItems)
  }, [session])

  async function handleCreate() {
    if (!session) {
      return
    }
    const created = await api.createSkill(session, {
      name,
      description,
      type: 'http',
      version: 1,
      status: 'draft',
      definition: {},
      isPublic: false,
    })
    setItems((current) => [created, ...current])
    setName('')
    setDescription('')
  }

  return (
    <div className="page-grid page-grid--two">
      <Section eyebrow="扩展能力" title="技能管理" description="Skill 元数据既面向运维人员，也会供后续 Agent 运行时读取。">
        {items.length === 0 ? <EmptyState title="还没有 Skill" description="先创建一个 Skill 占位，后续再补真实执行定义和调度能力。" /> : null}
        <div className="stack-list">
          {items.map((item) => (
            <article key={item.id} className="stack-card">
              <div>
                <h4>{item.name}</h4>
                <p className="muted">{item.description || '暂无描述'}</p>
              </div>
              <div className="stack-card__meta">
                <StatusBadge>{item.status}</StatusBadge>
                <span>v{item.version}</span>
              </div>
            </article>
          ))}
        </div>
      </Section>

      <Section eyebrow="创建 Skill" title="新增 Skill" description="当前先对接后端 CRUD，后续再补执行器、参数编辑器和版本管理。">
        <div className="form-grid">
          <label className="field field--full">
            <span>名称</span>
            <input value={name} onChange={(event) => setName(event.target.value)} />
          </label>
          <label className="field field--full">
            <span>描述</span>
            <textarea value={description} onChange={(event) => setDescription(event.target.value)} />
          </label>
          <button className="button field--full" onClick={handleCreate} type="button">
            创建 Skill
          </button>
        </div>
      </Section>
    </div>
  )
}
