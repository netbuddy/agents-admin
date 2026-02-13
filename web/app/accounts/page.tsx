'use client'

import { useEffect, useState } from 'react'
import { Plus, Trash2, Key, CheckCircle, Clock, AlertCircle, User } from 'lucide-react'
import { AdminLayout } from '@/components/layout'
import { useTranslation } from 'react-i18next'
import { useFormatDate } from '@/i18n/useFormatDate'

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
  volume: string
  status: string
  created_at: string
  last_used_at?: string
}

interface Node {
  id: string
  display_name?: string
  hostname?: string
  status: string
  last_heartbeat?: string
}

interface AuthOperation {
  operation_id: string
  action_id: string
  type: string
  status: string
  phase?: string
  message?: string
  progress?: number
  result?: {
    verify_url?: string
    user_code?: string
    device_code?: string
    volume_name?: string
    container_name?: string
  }
  error?: string
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
  const { t } = useTranslation('accounts')
  const { formatDate } = useFormatDate()
  const [accounts, setAccounts] = useState<Account[]>([])
  const [agentTypes, setAgentTypes] = useState<AgentType[]>([])
  const [nodes, setNodes] = useState<Node[]>([])
  const [proxies, setProxies] = useState<Proxy[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [authOp, setAuthOp] = useState<AuthOperation | null>(null)

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
        const activeProxies = (data.proxies || []).filter((p: Proxy) => p.status !== 'inactive')
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
    if (authOp && !['success', 'failed', 'timeout'].includes(authOp.status)) {
      const interval = setInterval(async () => {
        try {
          const res = await fetch(`/api/v1/actions/${authOp.action_id}`)
          if (res.ok) {
            const data = await res.json()
            setAuthOp(prev => ({
              ...prev!,
              status: data.status,
              phase: data.phase,
              message: data.message,
              progress: data.progress,
              result: data.result ? (typeof data.result === 'string' ? JSON.parse(data.result) : data.result) : prev?.result,
              error: data.error,
            }))
            if (data.status === 'success' || data.status === 'failed' || data.status === 'timeout') {
              fetchData()
            }
          }
        } catch (err) {
          console.error('Failed to check auth status:', err)
        }
      }, 2000)
      return () => clearInterval(interval)
    }
  }, [authOp?.status, authOp?.action_id])

  const startAuthOperation = async (name: string, agentType: string, nodeId: string, proxyId?: string) => {
    try {
      const config: Record<string, string> = { name, agent_type: agentType }
      if (proxyId) {
        config.proxy_id = proxyId
      }
      const res = await fetch('/api/v1/operations', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ type: 'oauth', config, node_id: nodeId }),
      })
      if (res.ok) {
        const data = await res.json()
        setShowCreateModal(false)
        setAuthOp({
          operation_id: data.operation_id,
          action_id: data.action_id,
          type: data.type,
          status: data.status,
        })
      } else {
        const err = await res.json().catch(() => ({ error: res.statusText }))
        console.error('Failed to create auth operation:', err)
        alert(err.error || 'Failed to create auth operation')
      }
    } catch (err) {
      console.error('Failed to start auth:', err)
    }
  }

  const retryAuth = async (accountName: string, agentType: string) => {
    // 重新发起认证：使用第一个在线节点
    if (nodes.length === 0) {
      alert(t('create.noOnlineNodes'))
      return
    }
    startAuthOperation(accountName, agentType, nodes[0].id)
  }

  const deleteAccount = async (accountId: string) => {
    if (!confirm(t('confirmDelete'))) return
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
      case 'authenticated': return t('statusAuthenticated')
      case 'pending': return t('statusPending')
      case 'expired': return t('statusExpired')
      default: return status
    }
  }

  return (
    <AdminLayout title={t('title')} onRefresh={fetchData} loading={loading}>
      {/* Actions bar */}
      <div className="mb-4 flex items-center justify-between">
        <p className="text-sm text-gray-500">{t('subtitle')}</p>
        <button
          onClick={() => setShowCreateModal(true)}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
        >
          <Plus className="w-4 h-4" />
          {t('addAccount')}
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
            <h3 className="text-lg font-medium mb-2">{t('noAccounts')}</h3>
            <p className="text-gray-500 mb-4">{t('noAccountsHint')}</p>
            <button
              onClick={() => setShowCreateModal(true)}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
            >
              {t('addAccount')}
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
                      <div key={account.id} className="px-3 sm:px-4 py-3 flex flex-col sm:flex-row sm:items-center justify-between gap-2">
                        <div className="flex items-center gap-3 min-w-0">
                          {statusIcon(account.status)}
                          <div className="min-w-0">
                            <p className="font-medium truncate">{account.name}</p>
                            <p className="text-sm text-gray-500 truncate">
                              {statusText(account.status)}
                              {account.last_used_at && ` · ${t('lastUsed')}: ${formatDate(account.last_used_at)}`}
                            </p>
                          </div>
                        </div>
                        <div className="flex items-center gap-2 flex-shrink-0 ml-8 sm:ml-0">
                          {account.status !== 'authenticated' && (
                            <button
                              onClick={() => retryAuth(account.name, account.agent_type)}
                              className="flex items-center gap-1 px-3 py-2 sm:py-1.5 text-sm bg-blue-50 text-blue-700 rounded hover:bg-blue-100"
                            >
                              <Key className="w-4 h-4" />
                              {t('authenticate')}
                            </button>
                          )}
                          <button
                            onClick={() => deleteAccount(account.id)}
                            className="p-2 sm:p-1.5 text-red-500 hover:bg-red-50 rounded"
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
          onStartAuth={startAuthOperation}
        />
      )}

      {authOp && !['success'].includes(authOp.status) && (
        <AuthModal
          authOp={authOp}
          onClose={() => { setAuthOp(null); fetchData() }}
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
  onStartAuth 
}: { 
  agentTypes: AgentType[]
  nodes: Node[]
  proxies: Proxy[]
  onClose: () => void
  onStartAuth: (name: string, agentType: string, nodeId: string, proxyId?: string) => void 
}) {
  const { t } = useTranslation('accounts')
  const [name, setName] = useState('')
  const [agentType, setAgentType] = useState(agentTypes[0]?.id || '')
  const [nodeId, setNodeId] = useState(nodes[0]?.id || '')
  // 默认选中默认代理，如果没有则不选择
  const defaultProxy = proxies.find(p => p.is_default)
  const [proxyId, setProxyId] = useState(defaultProxy?.id || '')

  return (
    <div className="fixed inset-0 bg-black/50 flex items-end sm:items-center justify-center z-50">
      <div className="bg-white rounded-t-2xl sm:rounded-lg w-full sm:max-w-md p-4 sm:p-6 max-h-[90vh] overflow-y-auto touch-scroll">
        <h2 className="text-lg font-semibold mb-4">{t('create.title')}</h2>
        
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">{t('create.agentType')}</label>
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
            <label className="block text-sm font-medium mb-1">{t('create.node')}</label>
            {nodes.length === 0 ? (
              <p className="text-sm text-red-500">{t('create.noOnlineNodes')}</p>
            ) : (
              <select
                value={nodeId}
                onChange={e => setNodeId(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                {nodes.map(n => (
                  <option key={n.id} value={n.id}>{n.display_name || n.hostname || n.id}</option>
                ))}
              </select>
            )}
            <p className="text-xs text-gray-500 mt-1">{t('create.nodeHint')}</p>
          </div>
          
          <div>
            <label className="block text-sm font-medium mb-1">{t('create.nameLabel')}</label>
            <input
              type="text"
              value={name}
              onChange={e => setName(e.target.value)}
              className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="user@example.com"
            />
            <p className="text-xs text-gray-500 mt-1">{t('create.nameHint')}</p>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">{t('create.proxy')}</label>
            <select
              value={proxyId}
              onChange={e => setProxyId(e.target.value)}
              className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="">{t('create.noProxy')}</option>
              {proxies.map(p => (
                <option key={p.id} value={p.id}>
                  {p.name} ({p.host}:{p.port}){p.is_default ? ` [${t('create.defaultProxy')}]` : ''}
                </option>
              ))}
            </select>
            <p className="text-xs text-gray-500 mt-1">
              {t('create.proxyHint')}
              {proxies.length === 0 && (
                <a href="/proxies" className="text-blue-600 hover:underline ml-1">{t('create.addProxy')}</a>
              )}
            </p>
          </div>
        </div>

        <div className="flex justify-end gap-2 mt-6">
          <button onClick={onClose} className="px-4 py-2 border rounded-lg hover:bg-gray-100">
            {t('action.cancel', { ns: 'common' })}
          </button>
          <button
            onClick={() => onStartAuth(name, agentType, nodeId, proxyId || undefined)}
            disabled={!name || !agentType || !nodeId}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
          >
            {t('create.createAndAuth')}
          </button>
        </div>
      </div>
    </div>
  )
}

function AuthModal({ authOp, onClose }: { 
  authOp: AuthOperation, 
  onClose: () => void,
}) {
  const { t } = useTranslation('accounts')
  
  const isWaiting = authOp.status === 'waiting'
  const isRunning = authOp.status === 'assigned' || authOp.status === 'running'
  const isFailed = authOp.status === 'failed' || authOp.status === 'timeout'
  const isSuccess = authOp.status === 'success'
  const verifyUrl = authOp.result?.verify_url
  const userCode = authOp.result?.user_code || authOp.result?.device_code

  return (
    <div className="fixed inset-0 bg-black/50 flex items-end sm:items-center justify-center z-50">
      <div className="bg-white rounded-t-2xl sm:rounded-lg w-full sm:max-w-lg p-4 sm:p-6 max-h-[90vh] overflow-y-auto touch-scroll">
        <h2 className="text-lg font-semibold mb-4">{t('auth.title')}</h2>
        
        {isRunning && (
          <div className="text-center py-8">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto mb-4" />
            <p className="text-gray-600">{t('auth.startingAuth')}</p>
            {authOp.message && (
              <p className="text-xs text-gray-500 mt-2">{authOp.message}</p>
            )}
          </div>
        )}

        {isWaiting && (
          <div className="space-y-4">
            {verifyUrl && (
              <div className="bg-blue-50 rounded-lg p-4">
                <p className="text-sm font-medium text-blue-800 mb-2">{t('auth.oauthTitle')}</p>
                <p className="text-sm text-gray-600 mb-3">
                  {t('auth.oauthDesc')}
                </p>
                <a
                  href={verifyUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 text-sm"
                >
                  {t('auth.openAuthPage')}
                </a>
                <p className="text-xs text-gray-500 mt-3 break-all">
                  {t('auth.linkAddress')}: <code className="bg-white px-1 rounded">{verifyUrl}</code>
                </p>
              </div>
            )}

            {userCode && (
              <div className="bg-blue-50 rounded-lg p-4 text-center">
                <p className="text-sm text-gray-600 mb-2">{t('auth.deviceCode')}</p>
                <p className="text-3xl font-mono font-bold text-blue-700 tracking-widest">
                  {userCode}
                </p>
                <p className="text-xs text-gray-500 mt-2">{t('auth.deviceCodeHint')}</p>
              </div>
            )}

            <div className="flex items-center justify-center gap-2 text-sm text-gray-500">
              <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-blue-600" />
              {t('auth.waitingAuth')}
            </div>

            {authOp.message && (
              <p className="text-xs text-center text-gray-500">{authOp.message}</p>
            )}
          </div>
        )}

        {isFailed && (
          <div className="space-y-4">
            <div className="text-center py-4">
              <AlertCircle className="w-12 h-12 text-red-500 mx-auto mb-4" />
              <p className="text-lg font-medium text-red-600">{t('auth.authFailed')}</p>
            </div>
            
            <div className="bg-red-50 rounded-lg p-4">
              <p className="text-sm font-medium text-red-800 mb-2">{t('auth.failReason')}</p>
              <p className="text-sm text-red-700">{authOp.error || authOp.message || t('error.unknown', { ns: 'common' })}</p>
            </div>
          </div>
        )}

        {isSuccess && (
          <div className="text-center py-8">
            <CheckCircle className="w-12 h-12 text-green-500 mx-auto mb-4" />
            <p className="text-green-600">{t('auth.authSuccess')}</p>
          </div>
        )}

        <div className="flex justify-end mt-6">
          <button onClick={onClose} className="px-4 py-2 border rounded-lg hover:bg-gray-100">
            {t('action.close', { ns: 'common' })}
          </button>
        </div>
      </div>
    </div>
  )
}
