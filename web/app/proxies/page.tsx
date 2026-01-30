'use client'

import { useEffect, useState } from 'react'
import { Plus, Trash2, Star, TestTube, Edit2, CheckCircle, XCircle, Loader2 } from 'lucide-react'
import { AdminLayout } from '@/components/layout'

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
  const [proxies, setProxies] = useState<Proxy[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [editingProxy, setEditingProxy] = useState<Proxy | null>(null)
  const [testingProxy, setTestingProxy] = useState<string | null>(null)
  const [testResult, setTestResult] = useState<{id: string, success: boolean, message: string} | null>(null)

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
    if (!confirm('确定删除此代理配置？')) return
    try {
      await fetch(`/api/v1/proxies/${id}`, { method: 'DELETE' })
      fetchProxies()
    } catch (err) {
      console.error('Failed to delete proxy:', err)
    }
  }

  const setDefaultProxy = async (id: string) => {
    try {
      await fetch(`/api/v1/proxies/${id}/set-default`, { method: 'POST' })
      fetchProxies()
    } catch (err) {
      console.error('Failed to set default proxy:', err)
    }
  }

  const testProxy = async (id: string) => {
    setTestingProxy(id)
    setTestResult(null)
    try {
      const res = await fetch(`/api/v1/proxies/${id}/test`, { method: 'POST' })
      if (res.ok) {
        const data = await res.json()
        setTestResult({ id, success: data.success, message: data.message })
      }
    } catch (err) {
      setTestResult({ id, success: false, message: '测试请求失败' })
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
    <AdminLayout title="代理管理" onRefresh={fetchProxies} loading={loading}>
      <div className="mb-4 flex items-center justify-between">
        <p className="text-sm text-gray-500">管理网络代理配置，用于Agent访问外部网络</p>
        <button
          onClick={() => setShowCreateModal(true)}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
        >
          <Plus className="w-4 h-4" />
          添加代理
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
          <h3 className="text-lg font-medium mb-2">暂无代理配置</h3>
          <p className="text-gray-500 mb-4">添加代理配置以便Agent访问需要代理的网络</p>
          <button
            onClick={() => setShowCreateModal(true)}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
          >
            添加代理
          </button>
        </div>
      ) : (
        <div className="bg-white rounded-lg border divide-y">
          {proxies.map(proxy => (
            <div key={proxy.id} className="px-4 py-4 flex items-center justify-between">
              <div className="flex items-center gap-4">
                <div className="flex items-center gap-2">
                  {proxy.is_default && (
                    <Star className="w-5 h-5 text-yellow-500 fill-yellow-500" />
                  )}
                  <div>
                    <div className="flex items-center gap-2">
                      <p className="font-medium">{proxy.name}</p>
                      <span className="px-2 py-0.5 text-xs bg-gray-100 rounded">
                        {proxyTypeLabel(proxy.type)}
                      </span>
                      {proxy.status === 'inactive' && (
                        <span className="px-2 py-0.5 text-xs bg-red-100 text-red-700 rounded">
                          已禁用
                        </span>
                      )}
                    </div>
                    <p className="text-sm text-gray-500">
                      {proxy.host}:{proxy.port}
                      {proxy.username && ` (认证: ${proxy.username})`}
                    </p>
                  </div>
                </div>
              </div>

              <div className="flex items-center gap-2">
                {testResult?.id === proxy.id && (
                  <span className={`flex items-center gap-1 text-sm ${testResult.success ? 'text-green-600' : 'text-red-600'}`}>
                    {testResult.success ? <CheckCircle className="w-4 h-4" /> : <XCircle className="w-4 h-4" />}
                    {testResult.message}
                  </span>
                )}
                
                <button
                  onClick={() => testProxy(proxy.id)}
                  disabled={testingProxy === proxy.id}
                  className="flex items-center gap-1 px-3 py-1.5 text-sm text-gray-600 hover:bg-gray-100 rounded disabled:opacity-50"
                >
                  {testingProxy === proxy.id ? (
                    <Loader2 className="w-4 h-4 animate-spin" />
                  ) : (
                    <TestTube className="w-4 h-4" />
                  )}
                  测试
                </button>

                {!proxy.is_default && (
                  <button
                    onClick={() => setDefaultProxy(proxy.id)}
                    className="flex items-center gap-1 px-3 py-1.5 text-sm text-yellow-600 hover:bg-yellow-50 rounded"
                  >
                    <Star className="w-4 h-4" />
                    设为默认
                  </button>
                )}

                <button
                  onClick={() => setEditingProxy(proxy)}
                  className="p-1.5 text-gray-500 hover:bg-gray-100 rounded"
                >
                  <Edit2 className="w-4 h-4" />
                </button>

                <button
                  onClick={() => deleteProxy(proxy.id)}
                  className="p-1.5 text-red-500 hover:bg-red-50 rounded"
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
  const [isDefault, setIsDefault] = useState(proxy?.is_default || false)
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
      is_default: isDefault,
      status,
    }
    if (username) data.username = username
    if (password) data.password = password
    if (noProxyValue) data.no_proxy = noProxyValue
    onSave(data)
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg w-full max-w-md p-6">
        <h2 className="text-lg font-semibold mb-4">
          {proxy ? '编辑代理' : '添加代理'}
        </h2>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">名称</label>
            <input
              type="text"
              value={name}
              onChange={e => setName(e.target.value)}
              className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="公司代理"
            />
          </div>

          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1">类型</label>
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
              <label className="block text-sm font-medium mb-1">主机</label>
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
            <label className="block text-sm font-medium mb-1">端口</label>
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
              <label className="block text-sm font-medium mb-1">用户名（可选）</label>
              <input
                type="text"
                value={username}
                onChange={e => setUsername(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">密码（可选）</label>
              <input
                type="password"
                value={password}
                onChange={e => setPassword(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder={proxy ? '留空保持不变' : ''}
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">不代理的地址列表（可选）</label>
            <textarea
              value={noProxy}
              onChange={e => setNoProxy(e.target.value)}
              className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono text-sm"
              placeholder="127.0.0.1&#10;::1&#10;localhost&#10;192.168.213.*"
              rows={4}
            />
            <p className="text-xs text-gray-500 mt-1">每行一个地址，可使用通配符匹配规则（如 192.168.213.*）</p>
          </div>

          <div className="flex items-center gap-4">
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={isDefault}
                onChange={e => setIsDefault(e.target.checked)}
                className="rounded"
              />
              <span className="text-sm">设为默认代理</span>
            </label>

            {proxy && (
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={status === 'active'}
                  onChange={e => setStatus(e.target.checked ? 'active' : 'inactive')}
                  className="rounded"
                />
                <span className="text-sm">启用</span>
              </label>
            )}
          </div>
        </div>

        <div className="flex justify-end gap-2 mt-6">
          <button onClick={onClose} className="px-4 py-2 border rounded-lg hover:bg-gray-100">
            取消
          </button>
          <button
            onClick={handleSubmit}
            disabled={!name || !host || !port}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
          >
            {proxy ? '保存' : '创建'}
          </button>
        </div>
      </div>
    </div>
  )
}
