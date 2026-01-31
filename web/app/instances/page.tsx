'use client'

import { useEffect, useState, useCallback, useMemo } from 'react'
import { Plus, Trash2, Play, Square, Terminal, Server, CheckCircle, Clock, AlertCircle, X } from 'lucide-react'
import { AdminLayout } from '@/components/layout'

interface AgentType {
  id: string
  name: string
  description: string
}

interface Account {
  id: string
  name: string
  agent_type: string
  status: string
}

interface Instance {
  id: string
  name: string
  account_id: string
  agent_type_id: string
  container_name: string | null
  status: string
  node_id: string | null
  created_at: string
}

interface TerminalSession {
  id: string
  url?: string | null
  port?: number | null
  status: string
}

export default function InstancesPage() {
  const [instances, setInstances] = useState<Instance[]>([])
  const [accounts, setAccounts] = useState<Account[]>([])
  const [agentTypes, setAgentTypes] = useState<AgentType[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreateWizard, setShowCreateWizard] = useState(false)
  const [terminalSession, setTerminalSession] = useState<TerminalSession | null>(null)

  // 固定实例展示顺序：避免状态变更导致 UI “跳动”
  const sortedInstances = useMemo(() => {
    return [...instances].sort((a, b) => {
      const byName = a.name.localeCompare(b.name, undefined, { numeric: true, sensitivity: 'base' })
      if (byName !== 0) return byName
      return a.id.localeCompare(b.id)
    })
  }, [instances])

  const fetchData = useCallback(async () => {
    try {
      const [instancesRes, accountsRes, typesRes] = await Promise.all([
        fetch('/api/v1/instances'),
        fetch('/api/v1/accounts'),
        fetch('/api/v1/agent-types')
      ])
      if (instancesRes.ok) {
        const data = await instancesRes.json()
        setInstances(data.instances || [])
      }
      if (accountsRes.ok) {
        const data = await accountsRes.json()
        setAccounts(data.accounts || [])
      }
      if (typesRes.ok) {
        const data = await typesRes.json()
        setAgentTypes(data.agent_types || [])
      }
    } catch (err) {
      console.error('Failed to fetch data:', err)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchData()
    const interval = setInterval(fetchData, 10000)
    return () => clearInterval(interval)
  }, [fetchData])

  const createInstance = async (accountId: string, name: string) => {
    try {
      const res = await fetch('/api/v1/instances', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ account_id: accountId, name }),
      })
      if (res.ok) {
        fetchData()
        setShowCreateWizard(false)
      }
    } catch (err) {
      console.error('Failed to create instance:', err)
    }
  }

  const startInstance = async (instanceId: string) => {
    try {
      await fetch(`/api/v1/instances/${instanceId}/start`, { method: 'POST' })
      fetchData()
    } catch (err) {
      console.error('Failed to start instance:', err)
    }
  }

  const stopInstance = async (instanceId: string) => {
    try {
      await fetch(`/api/v1/instances/${instanceId}/stop`, { method: 'POST' })
      fetchData()
    } catch (err) {
      console.error('Failed to stop instance:', err)
    }
  }

  const deleteInstance = async (instanceId: string) => {
    if (!confirm('确定删除此实例？')) return
    try {
      await fetch(`/api/v1/instances/${instanceId}`, { method: 'DELETE' })
      fetchData()
    } catch (err) {
      console.error('Failed to delete instance:', err)
    }
  }

  const openTerminal = async (instance: Instance) => {
    try {
      const res = await fetch('/api/v1/terminal/session', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ instance_id: instance.id }),
      })
      if (res.ok) {
        const data = await res.json()
        // 先打开弹窗（此时通常还是 pending），再轮询直到 running
        setTerminalSession({
          id: data.id,
          status: data.status || 'pending',
          port: data.port ?? null,
          url: data.url ?? null,
        })

        const startedAt = Date.now()
        const timeoutMs = 20000
        while (Date.now() - startedAt < timeoutMs) {
          const sRes = await fetch(`/api/v1/terminal/session/${data.id}`)
          if (sRes.ok) {
            const s = await sRes.json()

            if (s.status === 'running' && s.port) {
              setTerminalSession({
                id: s.id,
                status: s.status,
                port: s.port,
                url: s.url ?? null,
              })
              return
            }

            if (s.status === 'error') {
              throw new Error('terminal error')
            }

            // 逐步更新状态，让用户知道正在启动
            setTerminalSession(prev =>
              prev
                ? {
                    ...prev,
                    status: s.status || prev.status,
                    port: s.port ?? prev.port,
                    url: s.url ?? prev.url,
                  }
                : prev
            )
          }

          await new Promise<void>(resolve => setTimeout(resolve, 600))
        }

        throw new Error('terminal timeout')
      } else {
        alert('无法打开终端')
      }
    } catch (err) {
      console.error('Failed to open terminal:', err)
      alert('终端启动失败或超时，请重试')
      setTerminalSession(null)
    }
  }

  const closeTerminal = async () => {
    if (terminalSession) {
      try {
        await fetch(`/api/v1/terminal/session/${terminalSession.id}`, { method: 'DELETE' })
      } catch (err) {
        console.error('Failed to close terminal:', err)
      }
      setTerminalSession(null)
    }
  }

  const getAccountName = (accountId: string) => {
    const acc = accounts.find(a => a.id === accountId)
    return acc?.name || accountId
  }

  const getAgentTypeName = (typeId: string) => {
    const t = agentTypes.find(at => at.id === typeId)
    return t?.name || typeId
  }

  const statusIcon = (status: string) => {
    switch (status) {
      case 'running':
        return <CheckCircle className="w-5 h-5 text-green-500" />
      case 'stopped':
        return <Clock className="w-5 h-5 text-gray-400" />
      case 'error':
        return <AlertCircle className="w-5 h-5 text-red-500" />
      default:
        return <Clock className="w-5 h-5 text-yellow-500" />
    }
  }

  return (
    <AdminLayout title="实例管理" onRefresh={fetchData} loading={loading}>
      {/* Actions bar */}
      <div className="mb-4 flex items-center justify-between">
        <p className="text-sm text-gray-500">管理 AI Agent 运行实例</p>
        <button
          onClick={() => setShowCreateWizard(true)}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
        >
          <Plus className="w-4 h-4" />
          创建实例
        </button>
      </div>

      {loading ? (
          <div className="flex items-center justify-center h-64">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
          </div>
        ) : instances.length === 0 ? (
          <div className="bg-white rounded-lg border p-8 text-center">
            <Server className="w-12 h-12 mx-auto text-gray-400 mb-4" />
            <h3 className="text-lg font-medium mb-2">暂无实例</h3>
            <p className="text-gray-500 mb-4">创建一个 Agent 实例开始使用</p>
            <button
              onClick={() => setShowCreateWizard(true)}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
            >
              创建实例
            </button>
          </div>
        ) : (
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {sortedInstances.map(instance => (
              <div key={instance.id} className="bg-white rounded-lg border p-4">
                <div className="flex items-start justify-between mb-3">
                  <div className="flex items-center gap-2">
                    {statusIcon(instance.status)}
                    <div>
                      <h3 className="font-medium">{instance.name}</h3>
                      <p className="text-xs text-gray-500">{instance.container_name || '-'}</p>
                    </div>
                  </div>
                  <button
                    onClick={() => deleteInstance(instance.id)}
                    className="p-1.5 text-red-500 hover:bg-red-50 rounded"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>

                <div className="space-y-1 text-sm text-gray-600 mb-3">
                  <p><span className="text-gray-400">类型:</span> {getAgentTypeName(instance.agent_type_id)}</p>
                  <p><span className="text-gray-400">账号:</span> {getAccountName(instance.account_id)}</p>
                  <p><span className="text-gray-400">节点:</span> {instance.node_id || '-'}</p>
                </div>

                <div className="flex items-center gap-2 text-sm mb-3">
                  <span className={`px-2 py-0.5 rounded text-xs ${
                    instance.status === 'running' ? 'bg-green-100 text-green-800' : 
                    instance.status === 'stopping' ? 'bg-yellow-100 text-yellow-800' :
                    instance.status === 'pending' || instance.status === 'creating' ? 'bg-blue-100 text-blue-800' :
                    'bg-gray-100 text-gray-800'
                  }`}>
                    {instance.status === 'running' ? '运行中' : 
                     instance.status === 'stopped' ? '已停止' : 
                     instance.status === 'stopping' ? '停止中...' :
                     instance.status === 'pending' ? '等待创建...' :
                     instance.status === 'creating' ? '创建中...' :
                     instance.status}
                  </span>
                </div>

                <div className="flex gap-2">
                  {instance.status === 'running' ? (
                    <>
                      <button
                        onClick={() => openTerminal(instance)}
                        className="flex-1 flex items-center justify-center gap-1 px-3 py-2 bg-blue-50 text-blue-700 rounded-lg hover:bg-blue-100"
                      >
                        <Terminal className="w-4 h-4" />
                        终端
                      </button>
                      <button
                        onClick={() => stopInstance(instance.id)}
                        className="flex items-center justify-center gap-1 px-3 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200"
                      >
                        <Square className="w-4 h-4" />
                        停止
                      </button>
                    </>
                  ) : (
                    <button
                      onClick={() => startInstance(instance.id)}
                      className="flex-1 flex items-center justify-center gap-1 px-3 py-2 bg-green-50 text-green-700 rounded-lg hover:bg-green-100"
                    >
                      <Play className="w-4 h-4" />
                      启动
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}

      {showCreateWizard && (
        <CreateInstanceWizard
          agentTypes={agentTypes}
          accounts={accounts}
          onClose={() => setShowCreateWizard(false)}
          onCreate={createInstance}
        />
      )}

      {terminalSession && (
        <TerminalModal
          session={terminalSession}
          onClose={closeTerminal}
        />
      )}
    </AdminLayout>
  )
}

function CreateInstanceWizard({
  agentTypes,
  accounts,
  onClose,
  onCreate
}: {
  agentTypes: AgentType[]
  accounts: Account[]
  onClose: () => void
  onCreate: (accountId: string, name: string) => void
}) {
  const [step, setStep] = useState(1)
  const [selectedType, setSelectedType] = useState('')
  const [selectedAccount, setSelectedAccount] = useState('')
  const [instanceName, setInstanceName] = useState('')

  const filteredAccounts = accounts.filter(
    a => a.agent_type === selectedType && a.status === 'authenticated'
  )

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg w-full max-w-lg p-6">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-lg font-semibold">创建 Agent 实例</h2>
          <span className="text-sm text-gray-500">步骤 {step}/3</span>
        </div>

        {step === 1 && (
          <div>
            <p className="text-sm text-gray-600 mb-4">选择 Agent 类型：</p>
            <div className="grid grid-cols-2 gap-3">
              {agentTypes.map(t => (
                <button
                  key={t.id}
                  onClick={() => setSelectedType(t.id)}
                  className={`p-4 border rounded-lg text-left transition ${
                    selectedType === t.id
                      ? 'border-blue-500 bg-blue-50'
                      : 'hover:border-gray-300'
                  }`}
                >
                  <p className="font-medium">{t.name}</p>
                  <p className="text-xs text-gray-500 mt-1">{t.description}</p>
                </button>
              ))}
            </div>
          </div>
        )}

        {step === 2 && (
          <div>
            <p className="text-sm text-gray-600 mb-4">
              选择账号（{agentTypes.find(t => t.id === selectedType)?.name}）：
            </p>
            {filteredAccounts.length === 0 ? (
              <div className="text-center py-8">
                <p className="text-gray-500 mb-4">没有可用的已认证账号</p>
                <a
                  href="/accounts"
                  className="text-blue-600 hover:underline"
                >
                  前往添加账号 →
                </a>
              </div>
            ) : (
              <div className="space-y-2">
                {filteredAccounts.map(acc => (
                  <button
                    key={acc.id}
                    onClick={() => setSelectedAccount(acc.id)}
                    className={`w-full p-3 border rounded-lg text-left transition ${
                      selectedAccount === acc.id
                        ? 'border-blue-500 bg-blue-50'
                        : 'hover:border-gray-300'
                    }`}
                  >
                    <p className="font-medium">{acc.name}</p>
                    <p className="text-xs text-gray-500">已认证</p>
                  </button>
                ))}
              </div>
            )}
          </div>
        )}

        {step === 3 && (
          <div>
            <p className="text-sm text-gray-600 mb-4">配置实例：</p>
            <div>
              <label className="block text-sm font-medium mb-1">实例名称（可选）</label>
              <input
                type="text"
                value={instanceName}
                onChange={e => setInstanceName(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="留空将自动生成"
              />
            </div>
            <div className="mt-4 p-3 bg-gray-50 rounded-lg text-sm">
              <p><span className="text-gray-500">Agent 类型:</span> {agentTypes.find(t => t.id === selectedType)?.name}</p>
              <p><span className="text-gray-500">使用账号:</span> {accounts.find(a => a.id === selectedAccount)?.name}</p>
            </div>
          </div>
        )}

        <div className="flex justify-between mt-6">
          <button
            onClick={() => step > 1 ? setStep(step - 1) : onClose()}
            className="px-4 py-2 border rounded-lg hover:bg-gray-100"
          >
            {step > 1 ? '上一步' : '取消'}
          </button>
          {step < 3 ? (
            <button
              onClick={() => setStep(step + 1)}
              disabled={(step === 1 && !selectedType) || (step === 2 && !selectedAccount)}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
            >
              下一步
            </button>
          ) : (
            <button
              onClick={() => onCreate(selectedAccount, instanceName)}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
            >
              创建实例
            </button>
          )}
        </div>
      </div>
    </div>
  )
}

function TerminalModal({ session, onClose }: { session: TerminalSession, onClose: () => void }) {
  const host = typeof window !== 'undefined' ? window.location.hostname : 'localhost'
  const ready = session.status === 'running' && !!session.port
  const iframeUrl = ready ? `http://${host}:${session.port}/` : 'about:blank'

  return (
    <div className="fixed inset-0 bg-black/80 flex items-center justify-center z-50 p-4">
      <div className="bg-gray-900 rounded-lg w-full max-w-4xl h-[600px] flex flex-col">
        <div className="flex items-center justify-between px-4 py-2 border-b border-gray-700">
          <div className="flex items-center gap-2 text-white">
            <Terminal className="w-5 h-5" />
            <span className="font-medium">终端</span>
          </div>
          <button
            onClick={onClose}
            aria-label="关闭终端"
            className="p-1 text-gray-400 hover:text-white hover:bg-gray-700 rounded"
          >
            <X className="w-5 h-5" />
          </button>
        </div>
        <div className="flex-1 bg-black relative">
          {!ready && (
            <div className="absolute inset-0 flex items-center justify-center">
              <div className="text-gray-200 text-sm">
                终端启动中…（{session.status || 'pending'}）
              </div>
            </div>
          )}
          <iframe
            key={`${session.id}-${ready ? 'ready' : 'pending'}`}
            src={iframeUrl}
            className="w-full h-full border-0"
            title="Terminal"
          />
        </div>
      </div>
    </div>
  )
}
