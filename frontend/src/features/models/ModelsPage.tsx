import { useEffect, useMemo, useState } from 'react'

import { EmptyState, Section, StatusBadge } from '../../components/shared/Section'
import { api } from '../../lib/api'
import { useSessionStore } from '../../store/session-store'
import type { ModelRecord, ModelTestResult } from '../../types'

type ModelForm = {
  id?: number
  name: string
  provider: string
  model: string
  baseUrl: string
  apiKey: string
  description: string
  capabilities: string
  maxTokens: string
  temperature: string
  topP: string
  isEnabled: boolean
}

type ModelPreset = {
  label: string
  name: string
  model: string
  baseUrl: string
  description: string
}

const emptyForm: ModelForm = {
  name: '',
  provider: 'openai-compatible',
  model: '',
  baseUrl: '',
  apiKey: '',
  description: '',
  capabilities: 'chat,reasoning,stream',
  maxTokens: '4096',
  temperature: '0.2',
  topP: '1',
  isEnabled: true,
}

const presets: ModelPreset[] = [
  {
    label: '填入 32B 示例',
    name: 'DeepSeek 32B 示例',
    model: 'DeepSeek-R1-Distill-Qwen-32B',
    baseUrl: 'http://10.55.139.5:38888/v1',
    description: 'OpenAI 兼容的 32B 推理模型示例。',
  },
  {
    label: '填入本地流式示例',
    name: 'DeepSeek R1 本地流式示例',
    model: 'DeepSeek-R1-0528',
    baseUrl: 'http://localhost:31026/v1',
    description: '本地流式模型示例，适合联调 Agent 场景。',
  },
]

export function ModelsPage() {
  const { session } = useSessionStore()
  const [models, setModels] = useState<ModelRecord[]>([])
  const [form, setForm] = useState<ModelForm>(emptyForm)
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [lastTest, setLastTest] = useState<ModelTestResult | null>(null)
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const isAdmin = session?.user.role === 'admin'

  useEffect(() => {
    if (!session || !isAdmin) {
      return
    }
    void api.listModels(session).then(setModels)
  }, [isAdmin, session])

  const grouped = useMemo(() => {
    return models.reduce<Record<string, ModelRecord[]>>((accumulator, item) => {
      accumulator[item.provider] ??= []
      accumulator[item.provider].push(item)
      return accumulator
    }, {})
  }, [models])

  function selectModel(model: ModelRecord | null) {
    if (!model) {
      setSelectedId(null)
      setForm(emptyForm)
      setLastTest(null)
      setError('')
      return
    }

    setSelectedId(model.id)
    setForm({
      id: model.id,
      name: model.name,
      provider: model.provider,
      model: model.model,
      baseUrl: model.baseUrl,
      apiKey: '',
      description: model.description,
      capabilities: model.capabilities.join(','),
      maxTokens: String(model.maxTokens || 4096),
      temperature: String(model.temperature ?? 0.2),
      topP: String(model.topP ?? 1),
      isEnabled: model.isEnabled,
    })
    setLastTest(null)
    setError('')
  }

  function applyPreset(preset: ModelPreset) {
    setSelectedId(null)
    setForm((current) => ({
      ...current,
      id: undefined,
      name: preset.name,
      provider: 'openai-compatible',
      model: preset.model,
      baseUrl: preset.baseUrl,
      description: preset.description,
      apiKey: '',
      isEnabled: true,
    }))
    setLastTest(null)
    setError('')
  }

  async function handleSave() {
    if (!session || !isAdmin) {
      return
    }

    const payload = {
      name: form.name,
      provider: form.provider,
      model: form.model,
      baseUrl: form.baseUrl,
      apiKey: form.apiKey,
      description: form.description,
      capabilities: form.capabilities
        .split(',')
        .map((item) => item.trim())
        .filter(Boolean),
      maxTokens: Number(form.maxTokens),
      temperature: Number(form.temperature),
      topP: Number(form.topP),
      isEnabled: form.isEnabled,
      isDefault: false,
      testBeforeSave: !form.id,
    }

    setSubmitting(true)
    try {
      const saved = form.id ? await api.updateModel(session, form.id, payload) : await api.createModel(session, payload)
      setModels((current) => {
        const next = current.filter((item) => item.id !== saved.id)
        return [saved, ...next]
      })
      selectModel(saved)
      setError('')

      if (form.id) {
        setLastTest({
          reachable: true,
          model: saved.model,
          provider: saved.provider,
          message: '模型配置已保存。编辑时不会强制重新测试，避免因为暂时不可达而卡住保存。',
          checkedAt: new Date().toISOString(),
        })
      } else {
        setLastTest({
          reachable: true,
          model: saved.model,
          provider: saved.provider,
          message: '模型创建成功，且保存前测试已经通过。',
          checkedAt: new Date().toISOString(),
        })
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存模型失败，请检查模型配置。')
    } finally {
      setSubmitting(false)
    }
  }

  async function handleDelete(id: number) {
    if (!session || !isAdmin) {
      return
    }

    try {
      await api.deleteModel(session, id)
      setModels((current) => current.filter((item) => item.id !== id))
      if (selectedId === id) {
        selectModel(null)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '删除模型失败')
    }
  }

  async function handleTest(id: number) {
    if (!session || !isAdmin) {
      return
    }

    try {
      setLastTest(await api.testModel(session, id))
      setError('')
    } catch (err) {
      setError(err instanceof Error ? err.message : '模型连通性测试失败')
    }
  }

  async function handleSetDefault(id: number) {
    if (!session || !isAdmin) {
      return
    }

    try {
      const updated = await api.setDefaultModel(session, id)
      setModels((current) => current.map((item) => ({ ...item, isDefault: item.id === updated.id })))
      setError('')
    } catch (err) {
      setError(err instanceof Error ? err.message : '设置默认模型失败')
    }
  }

  if (!isAdmin) {
    return (
      <Section eyebrow="资源中心" title="模型管理" description="当前账号没有模型管理权限。">
        <EmptyState title="无权访问模型管理" description="只有平台管理员可以维护模型配置、测试模型和设置默认模型。" />
      </Section>
    )
  }

  return (
    <div className="page-grid page-grid--wide">
      <Section eyebrow="资源中心" title="模型管理" description="新增模型时会先做可用性测试；编辑现有模型时直接保存，不再因为测试而卡住。">
        {models.length === 0 ? <EmptyState title="还没有模型配置" description="先创建一个可用模型，Agent 才能进行问答和规划。" /> : null}
        <div className="provider-groups">
          {Object.entries(grouped).map(([provider, items]) => (
            <article key={provider} className="provider-group">
              <div className="provider-group__header">
                <h4>{provider}</h4>
                <span>{items.length} 个配置</span>
              </div>
              <div className="stack-list">
                {items.map((item) => (
                  <div key={item.id} className={`stack-card ${selectedId === item.id ? 'stack-card--selected' : ''}`}>
                    <div>
                      <h4>{item.name}</h4>
                      <p className="muted">
                        {item.model}
                        {item.baseUrl ? ` / ${item.baseUrl}` : ''}
                      </p>
                      <p className="muted">{item.maskedApiKey || '未保存 API Key'}</p>
                    </div>
                    <div className="stack-card__meta stack-card__meta--actions">
                      <StatusBadge tone={item.isDefault ? 'success' : 'neutral'}>{item.isDefault ? '默认模型' : '候选模型'}</StatusBadge>
                      <div className="inline-actions">
                        <button className="chip-button" onClick={() => selectModel(item)} type="button">
                          编辑
                        </button>
                        <button className="chip-button" onClick={() => void handleTest(item.id)} type="button">
                          测试
                        </button>
                        <button className="chip-button" onClick={() => void handleSetDefault(item.id)} type="button">
                          设为默认
                        </button>
                        <button className="chip-button chip-button--danger" onClick={() => void handleDelete(item.id)} type="button">
                          删除
                        </button>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </article>
          ))}
        </div>
      </Section>

      <Section eyebrow="编辑器" title={form.id ? '编辑模型' : '新建模型'} description="编辑时保留已有密钥；如果要验证连通性，先保存再点“测试”更直观。">
        {error ? <p className="form-error">{error}</p> : null}
        {lastTest ? (
          <div className="callout callout--success">
            <strong>最近一次结果</strong>
            <p>{lastTest.message}</p>
          </div>
        ) : null}

        <div className="inline-actions">
          {presets.map((preset) => (
            <button key={preset.label} className="chip-button" onClick={() => applyPreset(preset)} type="button">
              {preset.label}
            </button>
          ))}
        </div>

        <div className="form-grid">
          <label className="field">
            <span>名称</span>
            <input value={form.name} onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))} />
          </label>
          <label className="field">
            <span>Provider</span>
            <input value={form.provider} onChange={(event) => setForm((current) => ({ ...current, provider: event.target.value }))} />
          </label>
          <label className="field">
            <span>模型名称</span>
            <input value={form.model} onChange={(event) => setForm((current) => ({ ...current, model: event.target.value }))} />
          </label>
          <label className="field">
            <span>Base URL</span>
            <input value={form.baseUrl} onChange={(event) => setForm((current) => ({ ...current, baseUrl: event.target.value }))} />
          </label>
          <label className="field field--full">
            <span>API Key</span>
            <input
              value={form.apiKey}
              onChange={(event) => setForm((current) => ({ ...current, apiKey: event.target.value }))}
              placeholder={form.id ? '留空表示继续沿用已有密钥' : '没有密钥可留空'}
            />
          </label>
          <label className="field field--full">
            <span>描述</span>
            <textarea value={form.description} onChange={(event) => setForm((current) => ({ ...current, description: event.target.value }))} />
          </label>
          <label className="field field--full">
            <span>能力标签</span>
            <input
              value={form.capabilities}
              onChange={(event) => setForm((current) => ({ ...current, capabilities: event.target.value }))}
              placeholder="chat,reasoning,stream"
            />
          </label>
          <label className="field">
            <span>最大 Token</span>
            <input value={form.maxTokens} onChange={(event) => setForm((current) => ({ ...current, maxTokens: event.target.value }))} />
          </label>
          <label className="field">
            <span>Temperature</span>
            <input value={form.temperature} onChange={(event) => setForm((current) => ({ ...current, temperature: event.target.value }))} />
          </label>
          <label className="field">
            <span>Top P</span>
            <input value={form.topP} onChange={(event) => setForm((current) => ({ ...current, topP: event.target.value }))} />
          </label>
          <label className="field">
            <span>启用状态</span>
            <select
              value={form.isEnabled ? 'true' : 'false'}
              onChange={(event) => setForm((current) => ({ ...current, isEnabled: event.target.value === 'true' }))}
            >
              <option value="true">启用</option>
              <option value="false">停用</option>
            </select>
          </label>
          <div className="field field--full inline-actions">
            <button className="button" disabled={submitting} onClick={handleSave} type="button">
              {submitting ? '保存中...' : form.id ? '保存修改' : '创建并测试'}
            </button>
            <button className="button button--secondary" onClick={() => selectModel(null)} type="button">
              重置表单
            </button>
          </div>
        </div>
      </Section>
    </div>
  )
}
