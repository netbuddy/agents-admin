'use client'

import { useEffect, useState } from 'react'
import { Plus, Trash2, TestTube, Edit2, CheckCircle, XCircle, Loader2, X, Globe } from 'lucide-react'
import { AdminLayout } from '@/components/layout'
import { useTranslation } from 'react-i18next'

interface Proxy {
  id: string
  name: string
  type: string
  host: string
  port: number
  username?: string
  no_proxy?: string
  is_default: boolean
  status: string
  created_at: string
}

export default function ProxiesPage() {
  const { t } = useTranslation('proxies')
  const [proxies, setProxies] = useState<Proxy[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [editingProxy, setEditingProxy] = useState<Proxy | null>(null)
  const [testingProxy, setTestingProxy] = useState<string | null>(null)
  const [testResult, setTestResult] = useState<{id: string, success: boolean, message: string, headers?: Record<string, string>, page_title?: string, status_code?: number, latency_ms?: number} | null>(null)
  const [testDialogProxy, setTestDialogProxy] = useState<Proxy | null>(null)

  const fetchProxies = async () => {
    try {
      const res = await fetch('/api/v1/proxies')
      if (res.ok) {
        const data = await res.json()
        setProxies(data.proxies || [])
      }
    } catch (err) {
      console.error('Failed to fetch proxies:', err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchProxies()
  }, [])

  const createProxy = async (data: Partial<Proxy> & { password?: string }) => {
    try {
      const res = await fetch('/api/v1/proxies', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      })
      if (res.ok) {
        fetchProxies()
        setShowCreateModal(false)
      }
    } catch (err) {
      console.error('Failed to create proxy:', err)
    }
  }

  const updateProxy = async (id: string, data: Partial<Proxy> & { password?: string }) => {
    try {
      const res = await fetch(`/api/v1/proxies/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      })
      if (res.ok) {
        fetchProxies()
        setEditingProxy(null)
      }
    } catch (err) {
      console.error('Failed to update proxy:', err)
    }
  }

  const deleteProxy = async (id: string) => {
    if (!confirm(t('confirmDelete'))) return
    try {
      await fetch(`/api/v1/proxies/${id}`, { method: 'DELETE' })
      fetchProxies()
    } catch (err) {
      console.error('Failed to delete proxy:', err)
    }
  }


  const testProxy = async (id: string, targetUrl?: string) => {
    setTestingProxy(id)
    setTestResult(null)
    try {
      const body = targetUrl ? JSON.stringify({ target_url: targetUrl }) : '{}'
      const res = await fetch(`/api/v1/proxies/${id}/test`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body,
      })
      if (res.ok) {
        const data = await res.json()
        setTestResult({
          id, success: data.success, message: data.message,
          headers: data.headers, page_title: data.page_title,
          status_code: data.status_code, latency_ms: data.latency_ms,
        })
      }
    } catch (err) {
      setTestResult({ id, success: false, message: t('testFailed') })
    } finally {
      setTestingProxy(null)
    }
  }

  const proxyTypeLabel = (type: string) => {
    switch (type) {
      case 'http': return 'HTTP'
      case 'https': return 'HTTPS'
      case 'socks5': return 'SOCKS5'
      default: return type.toUpperCase()
    }
  }

  return (
    <AdminLayout title={t('title')} onRefresh={fetchProxies} loading={loading}>
      <div className="mb-4 flex items-center justify-between">
        <p className="text-sm text-gray-500">{t('subtitle')}</p>
        <button
          onClick={() => setShowCreateModal(true)}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
        >
          <Plus className="w-4 h-4" />
          {t('addProxy')}
        </button>
      </div>

      {loading ? (
        <div className="flex items-center justify-center h-64">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
        </div>
      ) : proxies.length === 0 ? (
        <div className="bg-white rounded-lg border p-8 text-center">
          <div className="w-12 h-12 mx-auto text-gray-400 mb-4 flex items-center justify-center">
            <svg className="w-12 h-12" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
            </svg>
          </div>
          <h3 className="text-lg font-medium mb-2">{t('noProxies')}</h3>
          <p className="text-gray-500 mb-4">{t('noProxiesHint')}</p>
          <button
            onClick={() => setShowCreateModal(true)}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
          >
            {t('addProxy')}
          </button>
        </div>
      ) : (
        <div className="bg-white rounded-lg border divide-y">
          {proxies.map(proxy => (
            <div key={proxy.id} className="px-3 sm:px-4 py-3 sm:py-4 flex flex-col sm:flex-row sm:items-center justify-between gap-2">
              <div className="flex items-center gap-3 min-w-0">
                <div className="min-w-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    <p className="font-medium truncate">{proxy.name}</p>
                    <span className="px-2 py-0.5 text-xs bg-gray-100 rounded">
                      {proxyTypeLabel(proxy.type)}
                    </span>
                    {proxy.status === 'inactive' && (
                      <span className="px-2 py-0.5 text-xs bg-red-100 text-red-700 rounded">
                        {t('disabled')}
                      </span>
                    )}
                  </div>
                  <p className="text-sm text-gray-500 truncate">
                    {proxy.host}:{proxy.port}
                    {proxy.username && ` (${t('authUser')}: ${proxy.username})`}
                  </p>
                </div>
              </div>

              <div className="flex items-center gap-1 sm:gap-2 flex-shrink-0 ml-8 sm:ml-0">
                {testResult?.id === proxy.id && (
                  <span className={`flex items-center gap-1 text-xs sm:text-sm ${testResult.success ? 'text-green-600' : 'text-red-600'}`}>
                    {testResult.success ? <CheckCircle className="w-4 h-4" /> : <XCircle className="w-4 h-4" />}
                    <span className="hidden sm:inline">{testResult.message}</span>
                  </span>
                )}
                
                <button
                  onClick={() => setTestDialogProxy(proxy)}
                  disabled={testingProxy === proxy.id}
                  className="flex items-center gap-1 px-2 sm:px-3 py-2 sm:py-1.5 text-sm text-gray-600 hover:bg-gray-100 rounded disabled:opacity-50"
                >
                  {testingProxy === proxy.id ? (
                    <Loader2 className="w-4 h-4 animate-spin" />
                  ) : (
                    <TestTube className="w-4 h-4" />
                  )}
                  <span className="hidden sm:inline">{t('action.test', { ns: 'common' })}</span>
                </button>

                <button
                  onClick={() => setEditingProxy(proxy)}
                  className="p-2 sm:p-1.5 text-gray-500 hover:bg-gray-100 rounded"
                >
                  <Edit2 className="w-4 h-4" />
                </button>

                <button
                  onClick={() => deleteProxy(proxy.id)}
                  className="p-2 sm:p-1.5 text-red-500 hover:bg-red-50 rounded"
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {showCreateModal && (
        <ProxyFormModal
          onClose={() => setShowCreateModal(false)}
          onSave={createProxy}
        />
      )}

      {editingProxy && (
        <ProxyFormModal
          proxy={editingProxy}
          onClose={() => setEditingProxy(null)}
          onSave={(data) => updateProxy(editingProxy.id, data)}
        />
      )}

      {testDialogProxy && (
        <ProxyTestDialog
          proxy={testDialogProxy}
          testing={testingProxy === testDialogProxy.id}
          result={testResult?.id === testDialogProxy.id ? testResult : null}
          onTest={(targetUrl) => testProxy(testDialogProxy.id, targetUrl)}
          onClose={() => { setTestDialogProxy(null); setTestResult(null) }}
        />
      )}
    </AdminLayout>
  )
}

function ProxyFormModal({
  proxy,
  onClose,
  onSave,
}: {
  proxy?: Proxy
  onClose: () => void
  onSave: (data: Partial<Proxy> & { password?: string }) => void
}) {
  const { t } = useTranslation('proxies')
  const [name, setName] = useState(proxy?.name || '')
  const [type, setType] = useState(proxy?.type || 'http')
  const [host, setHost] = useState(proxy?.host || '')
  const [port, setPort] = useState(proxy?.port?.toString() || '')
  const [username, setUsername] = useState(proxy?.username || '')
  const [password, setPassword] = useState('')
  // 将逗号分隔转换为多行显示
  const [noProxy, setNoProxy] = useState(
    proxy?.no_proxy ? proxy.no_proxy.split(',').map(s => s.trim()).join('\n') : ''
  )
  const [status, setStatus] = useState(proxy?.status || 'active')

  const handleSubmit = () => {
    // 将多行转换为逗号分隔格式
    const noProxyValue = noProxy
      .split('\n')
      .map(s => s.trim())
      .filter(s => s)
      .join(',')
    
    const data: Partial<Proxy> & { password?: string } = {
      name,
      type,
      host,
      port: parseInt(port, 10),
      status,
    }
    if (username) data.username = username
    if (password) data.password = password
    if (noProxyValue) data.no_proxy = noProxyValue
    onSave(data)
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-end sm:items-center justify-center z-50">
      <div className="bg-white rounded-t-2xl sm:rounded-lg w-full sm:max-w-md p-4 sm:p-6 max-h-[90vh] overflow-y-auto touch-scroll">
        <h2 className="text-lg font-semibold mb-4">
          {proxy ? t('form.editTitle') : t('form.addTitle')}
        </h2>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">{t('form.name')}</label>
            <input
              type="text"
              value={name}
              onChange={e => setName(e.target.value)}
              className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder={t('form.namePlaceholder')}
            />
          </div>

          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1">{t('form.type')}</label>
              <select
                value={type}
                onChange={e => setType(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="http">HTTP</option>
                <option value="https">HTTPS</option>
                <option value="socks5">SOCKS5</option>
              </select>
            </div>
            <div className="col-span-2">
              <label className="block text-sm font-medium mb-1">{t('form.host')}</label>
              <input
                type="text"
                value={host}
                onChange={e => setHost(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="192.168.1.1"
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">{t('form.port')}</label>
            <input
              type="number"
              value={port}
              onChange={e => setPort(e.target.value)}
              className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="8080"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1">{t('form.username')}</label>
              <input
                type="text"
                value={username}
                onChange={e => setUsername(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">{t('form.password')}</label>
              <input
                type="password"
                value={password}
                onChange={e => setPassword(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder={proxy ? t('form.passwordKeep') : ''}
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">{t('form.noProxyList')}</label>
            <textarea
              value={noProxy}
              onChange={e => setNoProxy(e.target.value)}
              className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono text-sm"
              placeholder="127.0.0.1&#10;::1&#10;localhost&#10;192.168.213.*"
              rows={4}
            />
            <p className="text-xs text-gray-500 mt-1">{t('form.noProxyHint')}</p>
          </div>

          {proxy && (
            <div className="flex items-center gap-4">
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={status === 'active'}
                  onChange={e => setStatus(e.target.checked ? 'active' : 'inactive')}
                  className="rounded"
                />
                <span className="text-sm">{t('form.enabled')}</span>
              </label>
            </div>
          )}
        </div>

        <div className="flex justify-end gap-2 mt-6">
          <button onClick={onClose} className="px-4 py-2 border rounded-lg hover:bg-gray-100">
            {t('action.cancel', { ns: 'common' })}
          </button>
          <button
            onClick={handleSubmit}
            disabled={!name || !host || !port}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
          >
            {proxy ? t('action.save', { ns: 'common' }) : t('action.create', { ns: 'common' })}
          </button>
        </div>
      </div>
    </div>
  )
}

const QUICK_TARGETS = [
  { label: 'Google', url: 'https://www.google.com' },
  { label: 'GitHub', url: 'https://github.com' },
  { label: 'OpenAI', url: 'https://api.openai.com' },
  { label: 'Anthropic', url: 'https://api.anthropic.com' },
]

function ProxyTestDialog({
  proxy,
  testing,
  result,
  onTest,
  onClose,
}: {
  proxy: Proxy
  testing: boolean
  result: { success: boolean; message: string; headers?: Record<string, string>; page_title?: string; status_code?: number; latency_ms?: number } | null
  onTest: (targetUrl: string) => void
  onClose: () => void
}) {
  const { t } = useTranslation('proxies')
  const [targetUrl, setTargetUrl] = useState('https://www.google.com')

  return (
    <div className="fixed inset-0 bg-black/50 flex items-end sm:items-center justify-center z-50">
      <div className="bg-white rounded-t-2xl sm:rounded-xl w-full sm:max-w-md p-5 sm:p-6 max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <Globe className="w-5 h-5 text-blue-600" />
            <h2 className="text-lg font-semibold">{t('test.title')}</h2>
          </div>
          <button onClick={onClose} className="p-1 hover:bg-gray-100 rounded">
            <X className="w-5 h-5 text-gray-400" />
          </button>
        </div>

        <p className="text-sm text-gray-500 mb-4">
          {t('test.desc', { name: proxy.name, host: proxy.host, port: proxy.port }).replace(/<\/?strong>/g, '')}
        </p>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">{t('test.targetUrl')}</label>
            <input
              type="url"
              value={targetUrl}
              onChange={e => setTargetUrl(e.target.value)}
              className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
              placeholder="https://www.google.com"
            />
          </div>

          <div>
            <label className="block text-xs text-gray-500 mb-2">{t('test.quickTargets')}</label>
            <div className="flex flex-wrap gap-2">
              {QUICK_TARGETS.map(t => (
                <button
                  key={t.url}
                  onClick={() => setTargetUrl(t.url)}
                  className={`px-3 py-1.5 text-xs rounded-full border transition-colors ${
                    targetUrl === t.url
                      ? 'bg-blue-50 border-blue-300 text-blue-700'
                      : 'bg-gray-50 border-gray-200 text-gray-600 hover:bg-gray-100'
                  }`}
                >
                  {t.label}
                </button>
              ))}
            </div>
          </div>

          {result && (
            <div className={`p-3 rounded-lg border ${
              result.success
                ? 'bg-green-50 border-green-200'
                : 'bg-red-50 border-red-200'
            }`}>
              <div className="flex items-center gap-2 mb-1">
                {result.success
                  ? <CheckCircle className="w-4 h-4 text-green-600" />
                  : <XCircle className="w-4 h-4 text-red-600" />
                }
                <span className={`text-sm font-medium ${result.success ? 'text-green-700' : 'text-red-700'}`}>
                  {result.success ? t('test.proxyAvailable') : t('test.testFailed')}
                </span>
              </div>
              <p className={`text-xs ${result.success ? 'text-green-600' : 'text-red-600'}`}>
                {result.message}
              </p>
              {result.success && (
                <div className="mt-2 pt-2 border-t border-green-200 space-y-1">
                  <p className="text-xs text-gray-500 font-medium">{t('test.evidence')}</p>
                  {result.page_title && (
                    <p className="text-xs text-gray-600">{t('test.pageTitle')}: <span className="font-mono bg-white/60 px-1 rounded">{result.page_title}</span></p>
                  )}
                  {result.status_code && (
                    <p className="text-xs text-gray-600">{t('test.statusCode')}: <span className="font-mono">{result.status_code}</span></p>
                  )}
                  {result.headers && Object.keys(result.headers).length > 0 && (
                    <div className="text-xs text-gray-600">
                      <p>{t('test.responseHeaders')}:</p>
                      <div className="font-mono text-[11px] bg-white/60 rounded p-1.5 mt-0.5 space-y-0.5">
                        {Object.entries(result.headers).map(([k, v]) => (
                          <p key={k}>{k}: {v}</p>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>
          )}
        </div>

        <div className="flex justify-end gap-3 mt-6">
          <button onClick={onClose} className="px-4 py-2 border rounded-lg hover:bg-gray-100 text-sm">
            {t('action.close', { ns: 'common' })}
          </button>
          <button
            onClick={() => onTest(targetUrl)}
            disabled={testing || !targetUrl}
            className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 text-sm"
          >
            {testing ? (
              <>
                <Loader2 className="w-4 h-4 animate-spin" />
                {t('test.testing')}
              </>
            ) : (
              <>
                <TestTube className="w-4 h-4" />
                {t('test.startTest')}
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  )
}
