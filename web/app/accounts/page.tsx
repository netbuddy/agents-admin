'use client'

import { useEffect, useState } from 'react'
import { Plus, Trash2, Key, CheckCircle, Clock, AlertCircle, User, RefreshCw } from 'lucide-react'
import { AdminLayout } from '@/components/layout'

interface AgentType {
  id: string
  name: string
  description: string
  login_methods: string[]
}

interface Account {
  id: string
  name: string
  agent_type: string
  node_id?: string
  volume: string
  status: string
  created_at: string
  last_used_at?: string
}

interface Node {
  id: string
  status: string
  last_heartbeat?: string
}

interface AuthSession {
  id: string
  account_id: string
  device_code?: string
  verify_url?: string
  callback_port?: number
  status: string
  message?: string
  executed?: boolean
  executed_at?: string
  can_retry?: boolean
}

interface Proxy {
  id: string
  name: string
  type: string
  host: string
  port: number
  is_default: boolean
  status: string
}

export default function AccountsPage() {
  const [accounts, setAccounts] = useState<Account[]>([])
  const [agentTypes, setAgentTypes] = useState<AgentType[]>([])
  const [nodes, setNodes] = useState<Node[]>([])
  const [proxies, setProxies] = useState<Proxy[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [authSession, setAuthSession] = useState<AuthSession | null>(null)

  const fetchData = async () => {
    try {
      const [accountsRes, typesRes, nodesRes, proxiesRes] = await Promise.all([
        fetch('/api/v1/accounts'),
        fetch('/api/v1/agent-types'),
        fetch('/api/v1/nodes'),
        fetch('/api/v1/proxies')
      ])
      if (accountsRes.ok) {
        const data = await accountsRes.json()
        setAccounts(data.accounts || [])
      }
      if (typesRes.ok) {
        const data = await typesRes.json()
        setAgentTypes(data.agent_types || [])
      }
      if (nodesRes.ok) {
        const data = await nodesRes.json()
        // 只显示在线节点（后端通过 etcd 判断状态）
        const onlineNodes = (data.nodes || []).filter((n: Node) => n.status === 'online')
        setNodes(onlineNodes)
      }
      if (proxiesRes.ok) {
        const data = await proxiesRes.json()
        // 只显示活跃的代理
        const activeProxies = (data.proxies || []).filter((p: Proxy) => p.status === 'active')
        setProxies(activeProxies)
      }
    } catch (err) {
      console.error('Failed to fetch data:', err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [])

  useEffect(() => {
    if (authSession && (authSession.status === 'pending' || authSession.status === 'waiting')) {
      const interval = setInterval(async () => {
        try {
          const res = await fetch(`/api/v1/accounts/${authSession.account_id}/auth/status`)
          if (res.ok) {
            const data = await res.json()
            setAuthSession(data)
            if (data.status === 'success' || data.status === 'failed') {
              fetchData()
            }
          }
        } catch (err) {
          console.error('Failed to check auth status:', err)
        }
      }, 2000)
      return () => clearInterval(interval)
    }
  }, [authSession?.status, authSession?.account_id])

  const createAccount = async (name: string, agentType: string, nodeId: string, proxyId?: string) => {
    try {
      const res = await fetch('/api/v1/accounts', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, agent_type: agentType, node_id: nodeId }),
      })
      if (res.ok) {
        const account = await res.json()
        fetchData()
        setShowCreateModal(false)
        // 自动开始认证，传递代理ID
        startAuth(account.id, proxyId)
      }
    } catch (err) {
      console.error('Failed to create account:', err)
    }
  }

  const startAuth = async (accountId: string, proxyId?: string) => {
    try {
      const body: { method: string; proxy_id?: string } = { method: 'oauth' }
      if (proxyId) {
        body.proxy_id = proxyId
      }
      const res = await fetch(`/api/v1/accounts/${accountId}/auth`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      if (res.ok) {
        const data = await res.json()
        setAuthSession(data)
      }
    } catch (err) {
      console.error('Failed to start auth:', err)
    }
  }

  const deleteAccount = async (accountId: string) => {
    if (!confirm('确定删除此账号？认证数据将被清除。')) return
    try {
      await fetch(`/api/v1/accounts/${accountId}?purge=true`, { method: 'DELETE' })
      fetchData()
    } catch (err) {
      console.error('Failed to delete account:', err)
    }
  }

  const getAgentTypeName = (typeId: string) => {
    const t = agentTypes.find(at => at.id === typeId)
    return t?.name || typeId
  }

  const statusIcon = (status: string) => {
    switch (status) {
      case 'authenticated':
        return <CheckCircle className="w-5 h-5 text-green-500" />
      case 'pending':
        return <Clock className="w-5 h-5 text-yellow-500" />
      case 'expired':
        return <AlertCircle className="w-5 h-5 text-red-500" />
      default:
        return <Clock className="w-5 h-5 text-gray-400" />
    }
  }

  const statusText = (status: string) => {
    switch (status) {
      case 'authenticated': return '已认证'
      case 'pending': return '待认证'
      case 'expired': return '已过期'
      default: return status
    }
  }

  return (
    <AdminLayout title="账号管理" onRefresh={fetchData} loading={loading}>
      {/* Actions bar */}
      <div className="mb-4 flex items-center justify-between">
        <p className="text-sm text-gray-500">管理 AI Agent 认证账号</p>
        <button
          onClick={() => setShowCreateModal(true)}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
        >
          <Plus className="w-4 h-4" />
          添加账号
        </button>
      </div>

      {/* Content */}
      {loading ? (
          <div className="flex items-center justify-center h-64">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
          </div>
        ) : accounts.length === 0 ? (
          <div className="bg-white rounded-lg border p-8 text-center">
            <User className="w-12 h-12 mx-auto text-gray-400 mb-4" />
            <h3 className="text-lg font-medium mb-2">暂无账号</h3>
            <p className="text-gray-500 mb-4">添加一个 AI Agent 账号开始使用</p>
            <button
              onClick={() => setShowCreateModal(true)}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
            >
              添加账号
            </button>
          </div>
        ) : (
          <div className="space-y-6">
            {agentTypes.map(agentType => {
              const typeAccounts = accounts.filter(a => a.agent_type === agentType.id)
              if (typeAccounts.length === 0) return null
              
              return (
                <div key={agentType.id} className="bg-white rounded-lg border">
                  <div className="px-4 py-3 border-b bg-gray-50">
                    <h2 className="font-medium">{agentType.name}</h2>
                    <p className="text-sm text-gray-500">{agentType.description}</p>
                  </div>
                  <div className="divide-y">
                    {typeAccounts.map(account => (
                      <div key={account.id} className="px-4 py-3 flex items-center justify-between">
                        <div className="flex items-center gap-3">
                          {statusIcon(account.status)}
                          <div>
                            <p className="font-medium">{account.name}</p>
                            <p className="text-sm text-gray-500">
                              {statusText(account.status)}
                              {account.last_used_at && ` · 上次使用: ${new Date(account.last_used_at).toLocaleDateString()}`}
                            </p>
                          </div>
                        </div>
                        <div className="flex items-center gap-2">
                          {account.status !== 'authenticated' && (
                            <button
                              onClick={() => startAuth(account.id)}
                              className="flex items-center gap-1 px-3 py-1.5 text-sm bg-blue-50 text-blue-700 rounded hover:bg-blue-100"
                            >
                              <Key className="w-4 h-4" />
                              认证
                            </button>
                          )}
                          <button
                            onClick={() => deleteAccount(account.id)}
                            className="p-1.5 text-red-500 hover:bg-red-50 rounded"
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )
            })}
          </div>
        )}
      {showCreateModal && (
        <CreateAccountModal
          agentTypes={agentTypes}
          nodes={nodes}
          proxies={proxies}
          onClose={() => setShowCreateModal(false)}
          onCreate={createAccount}
        />
      )}

      {authSession && authSession.status !== 'success' && authSession.status !== 'not_started' && (
        <AuthModal
          session={authSession}
          onClose={() => setAuthSession(null)}
          onRetry={() => {
            // 重试认证：关闭当前对话框，重新发起认证
            setAuthSession(null)
            startAuth(authSession.account_id)
          }}
        />
      )}
    </AdminLayout>
  )
}

function CreateAccountModal({ 
  agentTypes, 
  nodes,
  proxies,
  onClose, 
  onCreate 
}: { 
  agentTypes: AgentType[]
  nodes: Node[]
  proxies: Proxy[]
  onClose: () => void
  onCreate: (name: string, agentType: string, nodeId: string, proxyId?: string) => void 
}) {
  const [name, setName] = useState('')
  const [agentType, setAgentType] = useState(agentTypes[0]?.id || '')
  const [nodeId, setNodeId] = useState(nodes[0]?.id || '')
  // 默认选中默认代理，如果没有则不选择
  const defaultProxy = proxies.find(p => p.is_default)
  const [proxyId, setProxyId] = useState(defaultProxy?.id || '')

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg w-full max-w-md p-6">
        <h2 className="text-lg font-semibold mb-4">添加账号</h2>
        
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">Agent 类型</label>
            <select
              value={agentType}
              onChange={e => setAgentType(e.target.value)}
              className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              {agentTypes.map(t => (
                <option key={t.id} value={t.id}>{t.name}</option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">节点</label>
            {nodes.length === 0 ? (
              <p className="text-sm text-red-500">没有在线节点，请先启动 Node Agent</p>
            ) : (
              <select
                value={nodeId}
                onChange={e => setNodeId(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                {nodes.map(n => (
                  <option key={n.id} value={n.id}>{n.id}</option>
                ))}
              </select>
            )}
            <p className="text-xs text-gray-500 mt-1">账号将绑定到此节点（认证数据存储在节点本地）</p>
          </div>
          
          <div>
            <label className="block text-sm font-medium mb-1">账号名称/邮箱</label>
            <input
              type="text"
              value={name}
              onChange={e => setName(e.target.value)}
              className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="user@example.com"
            />
            <p className="text-xs text-gray-500 mt-1">用于标识账号，建议使用登录邮箱</p>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">网络代理</label>
            <select
              value={proxyId}
              onChange={e => setProxyId(e.target.value)}
              className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="">不使用代理</option>
              {proxies.map(p => (
                <option key={p.id} value={p.id}>
                  {p.name} ({p.host}:{p.port}){p.is_default ? ' [默认]' : ''}
                </option>
              ))}
            </select>
            <p className="text-xs text-gray-500 mt-1">
              选择代理用于认证过程中访问外部网络
              {proxies.length === 0 && (
                <a href="/proxies" className="text-blue-600 hover:underline ml-1">添加代理</a>
              )}
            </p>
          </div>
        </div>

        <div className="flex justify-end gap-2 mt-6">
          <button onClick={onClose} className="px-4 py-2 border rounded-lg hover:bg-gray-100">
            取消
          </button>
          <button
            onClick={() => onCreate(name, agentType, nodeId, proxyId || undefined)}
            disabled={!name || !agentType || !nodeId}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
          >
            创建并认证
          </button>
        </div>
      </div>
    </div>
  )
}

function AuthModal({ session, onClose, onRetry }: { 
  session: AuthSession, 
  onClose: () => void,
  onRetry?: () => void
}) {
  const host = typeof window !== 'undefined' ? window.location.hostname : 'localhost'
  
  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg w-full max-w-lg p-6">
        <h2 className="text-lg font-semibold mb-4">账号认证</h2>
        
        {session.status === 'pending' && (
          <div className="text-center py-8">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto mb-4" />
            <p className="text-gray-600">正在启动认证流程...</p>
          </div>
        )}

        {session.status === 'waiting' && (
          <div className="space-y-4">
            {/* OAuth 认证方式 - Web 终端 */}
            {session.callback_port && (
              <div className="bg-blue-50 rounded-lg p-4">
                <p className="text-sm font-medium text-blue-800 mb-2">Web 终端认证</p>
                <p className="text-sm text-gray-600 mb-3">
                  请点击下方链接打开 Web 终端，在终端中完成 OAuth 登录流程。
                </p>
                <a
                  href={`http://${host}:${session.callback_port}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 text-sm"
                >
                  打开 Web 终端
                </a>
                <p className="text-xs text-gray-500 mt-3">
                  终端地址: <code className="bg-white px-1 rounded">{host}:{session.callback_port}</code>
                </p>
              </div>
            )}

            {/* OAuth 认证方式 - 点击链接完成验证 */}
            {session.verify_url && (
              <div className="bg-blue-50 rounded-lg p-4">
                <p className="text-sm font-medium text-blue-800 mb-2">OAuth 认证</p>
                <p className="text-sm text-gray-600 mb-3">
                  请点击下方按钮，在新标签页中完成 OAuth 登录验证。
                </p>
                <a
                  href={session.verify_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 text-sm"
                >
                  打开认证页面
                </a>
                <p className="text-xs text-gray-500 mt-3 break-all">
                  链接地址: <code className="bg-white px-1 rounded">{session.verify_url}</code>
                </p>
              </div>
            )}

            {session.device_code && (
              <div className="bg-blue-50 rounded-lg p-4 text-center">
                <p className="text-sm text-gray-600 mb-2">授权代码（用于确认）：</p>
                <p className="text-3xl font-mono font-bold text-blue-700 tracking-widest">
                  {session.device_code}
                </p>
                <p className="text-xs text-gray-500 mt-2">此代码已包含在链接中，无需手动输入</p>
              </div>
            )}

            <div className="flex items-center justify-center gap-2 text-sm text-gray-500">
              <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-blue-600" />
              等待认证完成...
            </div>

            {session.message && (
              <p className="text-xs text-center text-gray-500">{session.message}</p>
            )}
          </div>
        )}

        {session.status === 'failed' && (
          <div className="space-y-4">
            <div className="text-center py-4">
              <AlertCircle className="w-12 h-12 text-red-500 mx-auto mb-4" />
              <p className="text-lg font-medium text-red-600">认证失败</p>
            </div>
            
            <div className="bg-red-50 rounded-lg p-4">
              <p className="text-sm font-medium text-red-800 mb-2">失败原因：</p>
              <p className="text-sm text-red-700">{session.message || '未知错误'}</p>
              {session.executed_at && (
                <p className="text-xs text-gray-500 mt-2">
                  执行时间: {new Date(session.executed_at).toLocaleString()}
                </p>
              )}
            </div>

            {session.can_retry && onRetry && (
              <div className="bg-yellow-50 rounded-lg p-4">
                <p className="text-sm text-yellow-800 mb-3">
                  您可以点击下方按钮重新尝试认证。建议先检查网络和代理配置。
                </p>
                <button
                  onClick={onRetry}
                  className="inline-flex items-center gap-2 px-4 py-2 bg-yellow-600 text-white rounded-lg hover:bg-yellow-700 text-sm"
                >
                  <RefreshCw className="w-4 h-4" />
                  重试认证
                </button>
              </div>
            )}
          </div>
        )}

        {session.status === 'success' && (
          <div className="text-center py-8">
            <CheckCircle className="w-12 h-12 text-green-500 mx-auto mb-4" />
            <p className="text-green-600">认证成功！</p>
          </div>
        )}

        <div className="flex justify-end mt-6">
          <button onClick={onClose} className="px-4 py-2 border rounded-lg hover:bg-gray-100">
            关闭
          </button>
        </div>
      </div>
    </div>
  )
}
