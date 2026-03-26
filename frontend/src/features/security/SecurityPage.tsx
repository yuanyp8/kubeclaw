import { useEffect, useState } from 'react'

import { Section, StatusBadge } from '../../components/shared/Section'
import { api } from '../../lib/api'
import { useSessionStore } from '../../store/session-store'
import type { IPWhitelistRecord, SensitiveFieldRuleRecord, SensitiveWordRecord } from '../../types'

export function SecurityPage() {
  const { session } = useSessionStore()
  const [ipRules, setIpRules] = useState<IPWhitelistRecord[]>([])
  const [sensitiveWords, setSensitiveWords] = useState<SensitiveWordRecord[]>([])
  const [fieldRules, setFieldRules] = useState<SensitiveFieldRuleRecord[]>([])
  const [ipOrCidr, setIpOrCidr] = useState('')
  const [word, setWord] = useState('')
  const [fieldPath, setFieldPath] = useState('')

  useEffect(() => {
    if (!session) {
      return
    }

    void Promise.all([
      api.listIPWhitelists(session),
      api.listSensitiveWords(session),
      api.listSensitiveFieldRules(session),
    ]).then(([ipItems, wordItems, fieldItems]) => {
      setIpRules(ipItems)
      setSensitiveWords(wordItems)
      setFieldRules(fieldItems)
    })
  }, [session])

  async function handleCreateIPRule() {
    if (!session) {
      return
    }
    const created = await api.createIPWhitelist(session, {
      name: ipOrCidr,
      ipOrCidr,
      scope: 'global',
      description: '',
      isEnabled: true,
    })
    setIpRules((current) => [created, ...current])
    setIpOrCidr('')
  }

  async function handleCreateWord() {
    if (!session) {
      return
    }
    const created = await api.createSensitiveWord(session, {
      word,
      category: 'command',
      level: 'high',
      action: 'review',
      description: '',
      isEnabled: true,
    })
    setSensitiveWords((current) => [created, ...current])
    setWord('')
  }

  async function handleCreateFieldRule() {
    if (!session) {
      return
    }
    const created = await api.createSensitiveFieldRule(session, {
      name: fieldPath,
      resource: 'secret',
      fieldPath,
      action: 'mask',
      description: '',
      isEnabled: true,
    })
    setFieldRules((current) => [created, ...current])
    setFieldPath('')
  }

  return (
    <div className="page-grid page-grid--three">
      <Section eyebrow="安全治理" title="IP 白名单" description="维护可信来源，后续可继续扩展到接口访问控制与集群接入边界。">
        <div className="inline-form">
          <input value={ipOrCidr} onChange={(event) => setIpOrCidr(event.target.value)} placeholder="10.0.0.0/24" />
          <button className="button" onClick={handleCreateIPRule} type="button">
            添加
          </button>
        </div>
        <div className="stack-list">
          {ipRules.map((item) => (
            <article key={item.id} className="stack-card">
              <div>
                <h4>{item.name}</h4>
                <p className="muted">{item.ipOrCidr}</p>
              </div>
              <div className="stack-card__meta">
                <StatusBadge tone={item.isEnabled ? 'success' : 'warning'}>{item.scope}</StatusBadge>
              </div>
            </article>
          ))}
        </div>
      </Section>

      <Section eyebrow="安全治理" title="敏感词规则" description="命中后可触发智能体的确定性审核或人工审批。">
        <div className="inline-form">
          <input value={word} onChange={(event) => setWord(event.target.value)} placeholder="delete" />
          <button className="button" onClick={handleCreateWord} type="button">
            添加
          </button>
        </div>
        <div className="stack-list">
          {sensitiveWords.map((item) => (
            <article key={item.id} className="stack-card">
              <div>
                <h4>{item.word}</h4>
                <p className="muted">{item.description || `${item.category} / ${item.level}`}</p>
              </div>
              <div className="stack-card__meta">
                <StatusBadge tone={item.isEnabled ? 'success' : 'warning'}>{item.action}</StatusBadge>
              </div>
            </article>
          ))}
        </div>
      </Section>

      <Section eyebrow="安全治理" title="敏感字段规则" description="用于控制高风险资源字段的脱敏、遮罩与拦截策略。">
        <div className="inline-form">
          <input value={fieldPath} onChange={(event) => setFieldPath(event.target.value)} placeholder="data.password" />
          <button className="button" onClick={handleCreateFieldRule} type="button">
            添加
          </button>
        </div>
        <div className="stack-list">
          {fieldRules.map((item) => (
            <article key={item.id} className="stack-card">
              <div>
                <h4>{item.name}</h4>
                <p className="muted">{item.fieldPath}</p>
              </div>
              <div className="stack-card__meta">
                <StatusBadge tone={item.isEnabled ? 'success' : 'warning'}>{item.action}</StatusBadge>
              </div>
            </article>
          ))}
        </div>
      </Section>
    </div>
  )
}
