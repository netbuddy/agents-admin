'use client'

import { useState, useEffect, useCallback } from 'react'
import { Save, X, Pencil, Settings, Info, AlertTriangle, Lock, Database, Shield, Server, HardDrive, Key } from 'lucide-react'
import { AdminLayout } from '@/components/layout'
import { useTranslation } from 'react-i18next'

interface AgentType {
  id: string
  name: string
  image: string
  description: string
  login_methods: string[]
}

interface ConfigData {
  file_path: string
  content: string
  parsed: Record<string, any>
}

/* ── tiny YAML serializer (flat 2-level, no deps) ── */
function toYaml(obj: Record<string, any>, indent = 0): string {
  const pad = ' '.repeat(indent)
  const lines: string[] = []
  for (const [k, v] of Object.entries(obj)) {
    if (v === null || v === undefined) continue
    if (typeof v === 'object' && !Array.isArray(v)) {
      lines.push(`${pad}${k}:`)
      lines.push(toYaml(v, indent + 2))
    } else if (typeof v === 'boolean') {
      lines.push(`${pad}${k}: ${v}`)
    } else if (typeof v === 'number') {
      lines.push(`${pad}${k}: ${v}`)
    } else {
      const s = String(v)
      const needsQuote = s === '' || s.includes(':') || s.includes('#') || s.includes('"') || /^\d+$/.test(s)
      lines.push(`${pad}${k}: ${needsQuote ? `"${s}"` : s}`)
    }
  }
  return lines.join('\n')
}

/* ── form field components ── */
function Field({ label, envHint, children }: { label: string; envHint?: string; children: React.ReactNode }) {
  return (
    <div className="grid grid-cols-1 sm:grid-cols-3 gap-1 sm:gap-3 items-start py-2">
      <div>
        <label className="text-sm text-gray-700 font-medium">{label}</label>
        {envHint && (
          <span className="ml-1.5 inline-flex items-center gap-0.5 text-[10px] text-amber-600 bg-amber-50 px-1.5 py-0.5 rounded">
            <Lock className="w-2.5 h-2.5" />{envHint}
          </span>
        )}
      </div>
      <div className="sm:col-span-2">{children}</div>
    </div>
  )
}

function TextInput({ value, onChange, disabled, placeholder, mono }: {
  value: string; onChange: (v: string) => void; disabled?: boolean; placeholder?: string; mono?: boolean
}) {
  return (
    <input
      type="text"
      value={value}
      onChange={e => onChange(e.target.value)}
      disabled={disabled}
      placeholder={placeholder}
      className={`w-full px-3 py-1.5 text-sm border rounded-md disabled:bg-gray-50 disabled:text-gray-500 focus:outline-none focus:ring-1 focus:ring-blue-500 ${mono ? 'font-mono' : ''}`}
    />
  )
}

function Toggle({ checked, onChange, disabled }: { checked: boolean; onChange: (v: boolean) => void; disabled?: boolean }) {
  return (
    <button
      type="button"
      onClick={() => !disabled && onChange(!checked)}
      className={`relative w-10 h-5 rounded-full transition-colors ${checked ? 'bg-blue-600' : 'bg-gray-300'} ${disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}`}
    >
      <span className={`absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full shadow transition-transform ${checked ? 'translate-x-5' : ''}`} />
    </button>
  )
}

function SectionCard({ icon, title, children, editing }: {
  icon: React.ReactNode; title: string; children: React.ReactNode; editing: boolean
}) {
  return (
    <div className="bg-white rounded-lg border">
      <div className="px-4 py-3 border-b bg-gray-50 flex items-center gap-2">
        {icon}
        <h3 className="font-medium text-sm">{title}</h3>
      </div>
      <div className={`px-4 py-2 divide-y ${!editing ? 'opacity-80' : ''}`}>{children}</div>
    </div>
  )
}

export default function SettingsPage() {
  const { t } = useTranslation('settings')
  const [agentTypes, setAgentTypes] = useState<AgentType[]>([])
  const [loading, setLoading] = useState(true)
  const [configData, setConfigData] = useState<ConfigData | null>(null)
  const [configError, setConfigError] = useState('')
  const [editing, setEditing] = useState(false)
  const [form, setForm] = useState<Record<string, any>>({})
  const [saving, setSaving] = useState(false)
  const [saveMsg, setSaveMsg] = useState<{ type: 'success' | 'error'; text: string } | null>(null)

  const fetchSettings = useCallback(async () => {
    setLoading(true)
    try {
      const [agentRes, configRes] = await Promise.all([
        fetch('/api/v1/agent-types'),
        fetch('/api/v1/config'),
      ])
      if (agentRes.ok) {
        const data = await agentRes.json()
        setAgentTypes(data.agent_types || [])
      }
      if (configRes.ok) {
        const data: ConfigData = await configRes.json()
        setConfigData(data)
        setForm(structuredClone(data.parsed || {}))
        setConfigError('')
      } else {
        const err = await configRes.json().catch(() => ({}))
        setConfigError(err.error || t('configLoadFailed'))
      }
    } catch (err) {
      console.error('Failed to fetch settings:', err)
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => { fetchSettings() }, [fetchSettings])

  const startEditing = () => { setEditing(true); setSaveMsg(null) }
  const cancelEditing = () => {
    setEditing(false)
    if (configData) setForm(structuredClone(configData.parsed || {}))
    setSaveMsg(null)
  }

  const set = (section: string, key: string, val: any) => {
    setForm(prev => ({ ...prev, [section]: { ...(prev[section] || {}), [key]: val } }))
  }
  const get = (section: string, key: string, fallback: any = '') => form?.[section]?.[key] ?? fallback

  const saveConfig = async () => {
    setSaving(true); setSaveMsg(null)
    try {
      const yamlStr = toYaml(form)
      const res = await fetch('/api/v1/config', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content: yamlStr }),
      })
      const data = await res.json()
      if (res.ok && data.success) {
        setSaveMsg({ type: 'success', text: t('configSaved') })
        setEditing(false)
        fetchSettings()
      } else {
        setSaveMsg({ type: 'error', text: data.error || t('configSaveFailed') })
      }
    } catch { setSaveMsg({ type: 'error', text: t('configSaveFailed') }) }
    finally { setSaving(false) }
  }

  const envTag = t('fromEnv')

  return (
    <AdminLayout title={t('title')} onRefresh={fetchSettings} loading={loading}>
      <div className="space-y-4">

        {/* Edit / Save bar */}
        {configData && (
          <div className="flex items-center justify-between">
            <p className="text-xs text-gray-500 font-mono">{configData.file_path}</p>
            {editing ? (
              <div className="flex items-center gap-2">
                <p className="text-xs text-amber-600 flex items-center gap-1 mr-2">
                  <AlertTriangle className="w-3.5 h-3.5" />{t('configRestartHint')}
                </p>
                <button onClick={cancelEditing} className="flex items-center gap-1 px-3 py-1.5 text-sm border rounded-md hover:bg-gray-100">
                  <X className="w-3.5 h-3.5" />{t('configCancel')}
                </button>
                <button onClick={saveConfig} disabled={saving}
                  className="flex items-center gap-1 px-3 py-1.5 text-sm bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50">
                  <Save className="w-3.5 h-3.5" />{saving ? t('configSaving') : t('configSave')}
                </button>
              </div>
            ) : (
              <button onClick={startEditing} className="flex items-center gap-1 px-3 py-1.5 text-sm bg-blue-600 text-white rounded-md hover:bg-blue-700">
                <Pencil className="w-3.5 h-3.5" />{t('configEdit')}
              </button>
            )}
          </div>
        )}

        {saveMsg && (
          <div className={`px-3 py-2 rounded text-sm flex items-center gap-2 ${saveMsg.type === 'success' ? 'bg-green-50 text-green-700 border border-green-200' : 'bg-red-50 text-red-700 border border-red-200'}`}>
            {saveMsg.type === 'success' ? <Info className="w-4 h-4" /> : <AlertTriangle className="w-4 h-4" />}{saveMsg.text}
          </div>
        )}

        {loading ? (
          <div className="flex items-center justify-center h-64">
            <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600" />
          </div>
        ) : configError ? (
          <div className="bg-white rounded-lg border px-4 py-8 text-center text-gray-500">
            <Settings className="w-8 h-8 mx-auto mb-2 text-gray-300" /><p>{configError}</p>
          </div>
        ) : configData ? (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            {/* API Server */}
            <SectionCard icon={<Server className="w-4 h-4 text-blue-600" />} title={t('sectionApiServer')} editing={editing}>
              <Field label={t('fieldPort')}>
                <TextInput value={String(get('api_server', 'port'))} onChange={v => set('api_server', 'port', v)} disabled={!editing} mono />
              </Field>
              <Field label={t('fieldUrl')}>
                <TextInput value={get('api_server', 'url')} onChange={v => set('api_server', 'url', v)} disabled={!editing} mono />
              </Field>
            </SectionCard>

            {/* Database */}
            <SectionCard icon={<Database className="w-4 h-4 text-green-600" />} title={t('sectionDatabase')} editing={editing}>
              <Field label={t('fieldDriver')}>
                <TextInput value={get('database', 'driver')} onChange={v => set('database', 'driver', v)} disabled={!editing} />
              </Field>
              <Field label={t('fieldHost')}>
                <TextInput value={get('database', 'host')} onChange={v => set('database', 'host', v)} disabled={!editing} mono />
              </Field>
              <Field label={t('fieldSslMode')}>
                <TextInput value={get('database', 'sslmode')} onChange={v => set('database', 'sslmode', v)} disabled={!editing} />
              </Field>
              <Field label={t('fieldPort')} envHint={envTag}>
                <TextInput value={String(get('database', 'port', ''))} onChange={() => {}} disabled placeholder="MONGO_PORT / POSTGRES_PORT" mono />
              </Field>
            </SectionCard>

            {/* Redis */}
            <SectionCard icon={<HardDrive className="w-4 h-4 text-red-500" />} title={t('sectionRedis')} editing={editing}>
              <Field label={t('fieldHost')}>
                <TextInput value={get('redis', 'host')} onChange={v => set('redis', 'host', v)} disabled={!editing} mono />
              </Field>
              <Field label={t('fieldDb')}>
                <TextInput value={String(get('redis', 'db', 0))} onChange={v => set('redis', 'db', parseInt(v) || 0)} disabled={!editing} mono />
              </Field>
              <Field label={t('fieldPort')} envHint={envTag}>
                <TextInput value={String(get('redis', 'port', ''))} onChange={() => {}} disabled placeholder="REDIS_PORT" mono />
              </Field>
            </SectionCard>

            {/* MinIO */}
            <SectionCard icon={<HardDrive className="w-4 h-4 text-orange-500" />} title={t('sectionMinio')} editing={editing}>
              <Field label={t('fieldEndpoint')}>
                <TextInput value={get('minio', 'endpoint')} onChange={v => set('minio', 'endpoint', v)} disabled={!editing} mono />
              </Field>
              <Field label={t('fieldUseSsl')}>
                <Toggle checked={get('minio', 'use_ssl', false)} onChange={v => set('minio', 'use_ssl', v)} disabled={!editing} />
              </Field>
              <Field label={t('fieldBucket')}>
                <TextInput value={get('minio', 'bucket')} onChange={v => set('minio', 'bucket', v)} disabled={!editing} />
              </Field>
            </SectionCard>

            {/* TLS */}
            <SectionCard icon={<Shield className="w-4 h-4 text-purple-600" />} title={t('sectionTls')} editing={editing}>
              <Field label={t('fieldEnabled')}>
                <Toggle checked={get('tls', 'enabled', false)} onChange={v => set('tls', 'enabled', v)} disabled={!editing} />
              </Field>
              <Field label={t('fieldAutoGenerate')}>
                <Toggle checked={get('tls', 'auto_generate', false)} onChange={v => set('tls', 'auto_generate', v)} disabled={!editing} />
              </Field>
              <Field label={t('fieldCertDir')}>
                <TextInput value={get('tls', 'cert_dir')} onChange={v => set('tls', 'cert_dir', v)} disabled={!editing} mono />
              </Field>
              <Field label={t('fieldCaFile')}>
                <TextInput value={get('tls', 'ca_file')} onChange={v => set('tls', 'ca_file', v)} disabled={!editing} mono />
              </Field>
              <Field label={t('fieldHosts')}>
                <TextInput value={get('tls', 'hosts')} onChange={v => set('tls', 'hosts', v)} disabled={!editing} mono />
              </Field>
            </SectionCard>

            {/* Auth */}
            <SectionCard icon={<Key className="w-4 h-4 text-yellow-600" />} title={t('sectionAuth')} editing={editing}>
              <Field label={t('fieldAccessTokenTtl')}>
                <TextInput value={get('auth', 'access_token_ttl')} onChange={v => set('auth', 'access_token_ttl', v)} disabled={!editing} mono />
              </Field>
              <Field label={t('fieldRefreshTokenTtl')}>
                <TextInput value={get('auth', 'refresh_token_ttl')} onChange={v => set('auth', 'refresh_token_ttl', v)} disabled={!editing} mono />
              </Field>
            </SectionCard>

            {/* Scheduler */}
            <SectionCard icon={<Settings className="w-4 h-4 text-gray-600" />} title={t('sectionScheduler')} editing={editing}>
              <Field label={t('fieldNodeId')}>
                <TextInput value={get('scheduler', 'node_id')} onChange={v => set('scheduler', 'node_id', v)} disabled={!editing} />
              </Field>
            </SectionCard>
          </div>
        ) : null}

        {/* Agent Types */}
        <div className="bg-white rounded-lg border">
          <div className="px-4 py-3 border-b bg-gray-50">
            <h2 className="font-medium">{t('agentTypeConfig')}</h2>
            <p className="text-sm text-gray-500">{t('agentTypeDesc')}</p>
          </div>
          {loading ? (
            <div className="flex items-center justify-center h-32">
              <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600" />
            </div>
          ) : (
            <div className="divide-y">
              {agentTypes.map(type => (
                <div key={type.id} className="px-4 py-3">
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="font-medium">{type.name}</p>
                      <p className="text-sm text-gray-500">{type.description}</p>
                    </div>
                    <div className="text-right">
                      <p className="text-sm font-mono text-gray-600">{type.image}</p>
                      <p className="text-xs text-gray-400">{t('loginMethods')}: {type.login_methods?.join(', ') || 'N/A'}</p>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* System Info */}
        <div className="bg-white rounded-lg border">
          <div className="px-4 py-3 border-b bg-gray-50">
            <h2 className="font-medium">{t('systemInfo')}</h2>
          </div>
          <div className="px-4 py-3 space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-gray-500">{t('version')}</span>
              <span className="font-mono">v0.1.0 (Phase 1.5)</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-500">{t('apiAddress')}</span>
              <span className="font-mono">http://localhost:8080</span>
            </div>
          </div>
        </div>
      </div>
    </AdminLayout>
  )
}
