'use client'

import { useEffect, useState, useCallback } from 'react'
import {
  Bot, Plus, Search, Filter, ChevronRight, X, Trash2, Play, Square,
  Terminal, Cpu, Sparkles, BookOpen, Shield, Tag, RefreshCw, Zap,
  Server, CheckCircle, Clock, AlertCircle, Settings2, Pencil
} from 'lucide-react'
import { AdminLayout } from '@/components/layout'
import { AGENT_TYPE_CONFIGS, getAgentTypeConfig, getAgentTypeOptions } from '@/lib/agent-models'

// ============================================================================
// Types
// ============================================================================

interface AgentTemplate {
  id: string
  name: string
  type: string
  role?: string
  description?: string
  personality?: string[]
  system_prompt?: string
  skills?: string[]
  tools?: any
  mcp_servers?: any
  model?: string
  temperature?: number
  max_context?: number
  is_builtin: boolean
  category?: string
  tags?: string[]
  created_at: string
  updated_at: string
}

interface AgentType {
  id: string
  name: string
  description: string
  image?: string
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
  node_id: string | null
  status: string
  created_at: string
}

interface TerminalSession {
  id: string
  url?: string | null
  port?: number | null
  status: string
}

// ============================================================================
// 类型颜色配置
// ============================================================================

const typeColorMap: Record<string, { bg: string; text: string; icon: string; border: string }> = {
  claude:  { bg: 'bg-purple-100', text: 'text-purple-700', icon: 'bg-purple-500', border: 'border-purple-200' },
  gemini:  { bg: 'bg-blue-100',   text: 'text-blue-700',   icon: 'bg-blue-500',   border: 'border-blue-200' },
  qwen:    { bg: 'bg-orange-100', text: 'text-orange-700', icon: 'bg-orange-500', border: 'border-orange-200' },
  codex:   { bg: 'bg-green-100',  text: 'text-green-700',  icon: 'bg-green-500',  border: 'border-green-200' },
  custom:  { bg: 'bg-gray-100',   text: 'text-gray-700',   icon: 'bg-gray-500',   border: 'border-gray-200' },
}

function getTypeColor(type: string) {
  return typeColorMap[type] || typeColorMap.custom
}

// ============================================================================
// Main Page
// ============================================================================

export default function AgentsPage() {
  const [activeTab, setActiveTab] = useState<'templates' | 'agents'>('agents')
  const [templates, setTemplates] = useState<AgentTemplate[]>([])
  const [instances, setInstances] = useState<Instance[]>([])
  const [accounts, setAccounts] = useState<Account[]>([])
  const [agentTypes, setAgentTypes] = useState<AgentType[]>([])
  const [loading, setLoading] = useState(true)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const [tmplRes, instRes, accRes, typesRes] = await Promise.all([
        fetch('/api/v1/agent-templates'),
        fetch('/api/v1/instances'),
        fetch('/api/v1/accounts'),
        fetch('/api/v1/agent-types'),
      ])
      if (tmplRes.ok) {
        const data = await tmplRes.json()
        setTemplates(data.templates || data.agent_templates || [])
      }
      if (instRes.ok) {
        const data = await instRes.json()
        setInstances(data.instances || [])
      }
      if (accRes.ok) {
        const data = await accRes.json()
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
  }, [fetchData])

  return (
    <AdminLayout title="智能体" onRefresh={fetchData} loading={loading}>
      {/* Tab 切换 */}
      <div className="flex items-center gap-1 mb-5 border-b border-gray-200">
        <button
          onClick={() => setActiveTab('agents')}
          className={`px-4 py-2.5 text-sm font-medium border-b-2 transition-colors ${
            activeTab === 'agents'
              ? 'border-blue-600 text-blue-600'
              : 'border-transparent text-gray-500 hover:text-gray-700'
          }`}
        >
          <div className="flex items-center gap-2">
            <Bot className="w-4 h-4" />
            智能体实例
          </div>
        </button>
        <button
          onClick={() => setActiveTab('templates')}
          className={`px-4 py-2.5 text-sm font-medium border-b-2 transition-colors ${
            activeTab === 'templates'
              ? 'border-blue-600 text-blue-600'
              : 'border-transparent text-gray-500 hover:text-gray-700'
          }`}
        >
          <div className="flex items-center gap-2">
            <BookOpen className="w-4 h-4" />
            模板库
          </div>
        </button>
      </div>

      {activeTab === 'agents' ? (
        <AgentsTab
          instances={instances}
          accounts={accounts}
          agentTypes={agentTypes}
          templates={templates}
          loading={loading}
          onRefresh={fetchData}
        />
      ) : (
        <TemplatesTab
          templates={templates}
          loading={loading}
          onRefresh={fetchData}
        />
      )}
    </AdminLayout>
  )
}

// ============================================================================
// Agents Tab
// ============================================================================

function AgentsTab({
  instances, accounts, agentTypes, templates, loading, onRefresh
}: {
  instances: Instance[]
  accounts: Account[]
  agentTypes: AgentType[]
  templates: AgentTemplate[]
  loading: boolean
  onRefresh: () => void
}) {
  const [showCreate, setShowCreate] = useState(false)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [terminalSession, setTerminalSession] = useState<TerminalSession | null>(null)

  const getAccountName = (id: string) => accounts.find(a => a.id === id)?.name || id
  const getTypeName = (id: string) => agentTypes.find(t => t.id === id)?.name || id

  const startInstance = async (id: string) => {
    await fetch(`/api/v1/instances/${id}/start`, { method: 'POST' })
    onRefresh()
  }

  const stopInstance = async (id: string) => {
    await fetch(`/api/v1/instances/${id}/stop`, { method: 'POST' })
    onRefresh()
  }

  const deleteInstance = async (id: string) => {
    if (!confirm('确定要删除此智能体？')) return
    await fetch(`/api/v1/instances/${id}`, { method: 'DELETE' })
    onRefresh()
  }

  const openTerminal = async (inst: Instance) => {
    try {
      const res = await fetch('/api/v1/terminal/session', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ instance_id: inst.id }),
      })
      if (!res.ok) { alert('无法打开终端'); return }
      const data = await res.json()
      setTerminalSession({ id: data.id, status: data.status || 'pending', port: data.port ?? null, url: data.url ?? null })

      const startedAt = Date.now()
      while (Date.now() - startedAt < 20000) {
        const sRes = await fetch(`/api/v1/terminal/session/${data.id}`)
        if (sRes.ok) {
          const s = await sRes.json()
          if (s.status === 'running' && s.port) {
            setTerminalSession({ id: s.id, status: s.status, port: s.port, url: s.url ?? null })
            return
          }
          if (s.status === 'error') throw new Error('terminal error')
          setTerminalSession(prev => prev ? { ...prev, status: s.status || prev.status, port: s.port ?? prev.port, url: s.url ?? prev.url } : prev)
        }
        await new Promise<void>(r => setTimeout(r, 600))
      }
      throw new Error('terminal timeout')
    } catch {
      alert('终端启动失败或超时')
      setTerminalSession(null)
    }
  }

  const closeTerminal = async () => {
    if (terminalSession) {
      try { await fetch(`/api/v1/terminal/session/${terminalSession.id}`, { method: 'DELETE' }) } catch {}
      setTerminalSession(null)
    }
  }

  const runningCount = instances.filter(i => i.status === 'running').length
  const stoppedCount = instances.filter(i => i.status === 'stopped' || i.status === 'error').length
  const pendingCount = instances.filter(i => i.status === 'pending' || i.status === 'creating').length

  const selected = selectedId ? instances.find(i => i.id === selectedId) : null

  const createInstance = async (accountId: string, name: string) => {
    try {
      const res = await fetch('/api/v1/instances', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ account_id: accountId, name }),
      })
      if (res.ok) {
        onRefresh()
        setShowCreate(false)
      }
    } catch (err) {
      console.error('Failed to create instance:', err)
    }
  }

  return (
    <>
      {/* 统计 + 创建按钮 */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-3">
          <StatBadge label="总计" value={instances.length} color="text-gray-900" />
          <StatBadge label="运行中" value={runningCount} color="text-green-600" />
          <StatBadge label="已停止" value={stoppedCount} color="text-gray-500" />
          {pendingCount > 0 && <StatBadge label="创建中" value={pendingCount} color="text-blue-600" />}
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 text-sm font-medium"
        >
          <Plus className="w-4 h-4" />
          创建智能体
        </button>
      </div>

      {loading ? (
        <div className="flex items-center justify-center h-64">
          <RefreshCw className="w-8 h-8 text-blue-500 animate-spin" />
        </div>
      ) : instances.length === 0 ? (
        <EmptyState
          icon={Bot}
          title="暂无智能体"
          description="创建一个智能体开始使用，它将基于 AI Agent 模板运行"
          action={<button onClick={() => setShowCreate(true)} className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 text-sm">创建智能体</button>}
        />
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
          {instances.map(inst => (
            <AgentCard
              key={inst.id}
              instance={inst}
              typeName={getTypeName(inst.agent_type_id)}
              accountName={getAccountName(inst.account_id)}
              onStart={() => startInstance(inst.id)}
              onStop={() => stopInstance(inst.id)}
              onTerminal={() => openTerminal(inst)}
              onDelete={() => deleteInstance(inst.id)}
              onClick={() => setSelectedId(inst.id)}
            />
          ))}
        </div>
      )}

      {/* 详情弹窗 */}
      {selected && (
        <AgentDetail
          instance={selected}
          typeName={getTypeName(selected.agent_type_id)}
          accountName={getAccountName(selected.account_id)}
          onClose={() => setSelectedId(null)}
          onStart={() => startInstance(selected.id)}
          onStop={() => stopInstance(selected.id)}
          onTerminal={() => openTerminal(selected)}
          onDelete={() => { deleteInstance(selected.id); setSelectedId(null) }}
        />
      )}

      {/* 创建弹窗 */}
      {showCreate && (
        <CreateAgentWizard
          agentTypes={agentTypes}
          accounts={accounts}
          onClose={() => setShowCreate(false)}
          onCreate={createInstance}
        />
      )}

      {/* 终端弹窗 */}
      {terminalSession && (
        <TerminalModal session={terminalSession} onClose={closeTerminal} />
      )}
    </>
  )
}

// ============================================================================
// Agent Card
// ============================================================================

function AgentCard({
  instance, typeName, accountName, onStart, onStop, onTerminal, onDelete, onClick
}: {
  instance: Instance
  typeName: string
  accountName: string
  onStart: () => void
  onStop: () => void
  onTerminal: () => void
  onDelete: () => void
  onClick: () => void
}) {
  const isRunning = instance.status === 'running'
  const isPending = instance.status === 'pending' || instance.status === 'creating'
  const statusColor = isRunning ? 'bg-green-500' : isPending ? 'bg-blue-500' : instance.status === 'error' ? 'bg-red-500' : 'bg-gray-400'
  const statusLabel = isRunning ? '运行中' : isPending ? '创建中' : instance.status === 'stopping' ? '停止中' : instance.status === 'error' ? '错误' : '已停止'
  const typeColor = getTypeColor(instance.agent_type_id.includes('qwen') ? 'qwen' : instance.agent_type_id.includes('codex') ? 'codex' : 'custom')

  return (
    <div className="bg-white rounded-xl border shadow-sm hover:shadow-md hover:border-blue-300 transition-all p-4 sm:p-5">
      {/* Header */}
      <div className="flex items-start justify-between mb-3 cursor-pointer" onClick={onClick}>
        <div className="flex items-center gap-3 min-w-0">
          <div className={`p-2 rounded-lg ${typeColor.bg}`}>
            <Bot className={`w-5 h-5 ${typeColor.text}`} />
          </div>
          <div className="min-w-0">
            <h3 className="font-semibold text-gray-900 truncate">{instance.name}</h3>
            <p className="text-xs text-gray-500 mt-0.5">{typeName}</p>
          </div>
        </div>
        <div className="flex items-center gap-1.5 flex-shrink-0">
          <span className={`w-2 h-2 rounded-full ${statusColor} ${isRunning ? 'animate-pulse' : ''}`} />
          <span className={`text-xs font-medium ${isRunning ? 'text-green-700' : isPending ? 'text-blue-700' : 'text-gray-500'}`}>
            {statusLabel}
          </span>
        </div>
      </div>

      {/* Info */}
      <div className="space-y-1 text-xs text-gray-500 mb-3">
        <p>账号: {accountName}</p>
        <p>节点: {instance.node_id || '-'}</p>
      </div>

      {/* Actions */}
      <div className="flex gap-2">
        {isRunning ? (
          <>
            <button onClick={onTerminal} className="flex-1 flex items-center justify-center gap-1 px-3 py-2 bg-blue-50 text-blue-700 rounded-lg hover:bg-blue-100 text-sm">
              <Terminal className="w-4 h-4" />
              终端
            </button>
            <button onClick={onStop} className="flex items-center justify-center gap-1 px-3 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 text-sm">
              <Square className="w-4 h-4" />
              停止
            </button>
          </>
        ) : isPending ? (
          <div className="flex-1 flex items-center justify-center gap-2 px-3 py-2 bg-blue-50 text-blue-600 rounded-lg text-sm">
            <RefreshCw className="w-4 h-4 animate-spin" />
            正在创建...
          </div>
        ) : (
          <>
            <button onClick={onStart} className="flex-1 flex items-center justify-center gap-1 px-3 py-2 bg-green-50 text-green-700 rounded-lg hover:bg-green-100 text-sm">
              <Play className="w-4 h-4" />
              启动
            </button>
            <button onClick={onDelete} className="flex items-center justify-center gap-1 px-3 py-2 bg-red-50 text-red-600 rounded-lg hover:bg-red-100 text-sm">
              <Trash2 className="w-4 h-4" />
            </button>
          </>
        )}
      </div>
    </div>
  )
}

// ============================================================================
// Agent Detail
// ============================================================================

function AgentDetail({
  instance, typeName, accountName, onClose, onStart, onStop, onTerminal, onDelete
}: {
  instance: Instance
  typeName: string
  accountName: string
  onClose: () => void
  onStart: () => void
  onStop: () => void
  onTerminal: () => void
  onDelete: () => void
}) {
  const isRunning = instance.status === 'running'

  return (
    <div className="fixed inset-0 z-50 flex items-end sm:items-center justify-center">
      <div className="fixed inset-0 bg-black/40" onClick={onClose} />
      <div className="relative bg-white rounded-t-2xl sm:rounded-xl shadow-xl w-full sm:max-w-lg max-h-[85vh] overflow-y-auto z-10">
        <div className="sticky top-0 bg-white border-b px-5 py-4 flex items-center justify-between rounded-t-2xl sm:rounded-t-xl">
          <div className="flex items-center gap-3 min-w-0">
            <div className="p-2 rounded-lg bg-blue-100">
              <Bot className="w-5 h-5 text-blue-700" />
            </div>
            <div className="min-w-0">
              <h2 className="font-bold text-gray-900 truncate">{instance.name}</h2>
              <p className="text-xs text-gray-500">{typeName}</p>
            </div>
          </div>
          <button onClick={onClose} className="p-2 hover:bg-gray-100 rounded-lg">
            <X className="w-5 h-5 text-gray-500" />
          </button>
        </div>

        <div className="p-5 space-y-5">
          <div>
            <h3 className="text-sm font-semibold text-gray-700 mb-3">基本信息</h3>
            <div className="grid grid-cols-2 gap-3">
              <InfoItem label="ID" value={instance.id} />
              <InfoItem label="状态" value={instance.status} />
              <InfoItem label="账号" value={accountName} />
              <InfoItem label="节点" value={instance.node_id || '-'} />
              <InfoItem label="容器" value={instance.container_name || '-'} />
              <InfoItem label="创建时间" value={new Date(instance.created_at).toLocaleString('zh-CN')} />
            </div>
          </div>

          <div className="border-t pt-4 flex gap-2">
            {isRunning ? (
              <>
                <button onClick={onTerminal} className="flex-1 flex items-center justify-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 text-sm">
                  <Terminal className="w-4 h-4" />
                  打开终端
                </button>
                <button onClick={onStop} className="flex items-center justify-center gap-2 px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 text-sm">
                  <Square className="w-4 h-4" />
                  停止
                </button>
              </>
            ) : (
              <>
                <button onClick={onStart} className="flex-1 flex items-center justify-center gap-2 px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 text-sm">
                  <Play className="w-4 h-4" />
                  启动
                </button>
                <button onClick={onDelete} className="flex items-center justify-center gap-2 px-4 py-2 text-red-600 hover:bg-red-50 rounded-lg text-sm">
                  <Trash2 className="w-4 h-4" />
                  删除
                </button>
              </>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

// ============================================================================
// Templates Tab
// ============================================================================

function TemplatesTab({
  templates, loading, onRefresh
}: {
  templates: AgentTemplate[]
  loading: boolean
  onRefresh: () => void
}) {
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [showCreate, setShowCreate] = useState(false)
  const [editingTemplate, setEditingTemplate] = useState<AgentTemplate | null>(null)
  const [filterType, setFilterType] = useState('')

  const typeOptions = getAgentTypeOptions()
  const filtered = templates.filter(t => {
    if (filterType && t.type !== filterType) return false
    return true
  })

  const builtinCount = templates.filter(t => t.is_builtin).length
  const customCount = templates.filter(t => !t.is_builtin).length
  const selected = selectedId ? templates.find(t => t.id === selectedId) : null

  const deleteTemplate = async (id: string) => {
    if (!confirm('确定要删除此模板？')) return
    await fetch(`/api/v1/agent-templates/${id}`, { method: 'DELETE' })
    onRefresh()
    setSelectedId(null)
  }

  return (
    <>
      {/* 统计 + 操作 */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-3">
          <StatBadge label="总计" value={templates.length} color="text-gray-900" />
          <StatBadge label="内置" value={builtinCount} color="text-blue-600" />
          <StatBadge label="自定义" value={customCount} color="text-purple-600" />
        </div>
        <div className="flex items-center gap-2">
          <select
            value={filterType}
            onChange={e => setFilterType(e.target.value)}
            className="px-3 py-2 border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="">所有类型</option>
            {typeOptions.map(opt => (
              <option key={opt.value} value={opt.value}>{opt.label}</option>
            ))}
          </select>
          <button
            onClick={() => setShowCreate(true)}
            className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 text-sm font-medium"
          >
            <Plus className="w-4 h-4" />
            创建模板
          </button>
        </div>
      </div>

      {loading ? (
        <div className="flex items-center justify-center h-64">
          <RefreshCw className="w-8 h-8 text-blue-500 animate-spin" />
        </div>
      ) : filtered.length === 0 ? (
        <EmptyState
          icon={BookOpen}
          title="暂无模板"
          description="系统预置了内置模板，首次使用请导入数据库初始化脚本"
        />
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
          {filtered.map(tmpl => (
            <TemplateCard key={tmpl.id} template={tmpl} onClick={() => setSelectedId(tmpl.id)} />
          ))}
        </div>
      )}

      {/* 模板详情弹窗 */}
      {selected && (
        <TemplateDetail
          template={selected}
          onClose={() => setSelectedId(null)}
          onDelete={() => deleteTemplate(selected.id)}
          onEdit={() => { setEditingTemplate(selected); setSelectedId(null) }}
        />
      )}

      {/* 创建模板弹窗 */}
      {showCreate && (
        <CreateTemplateModal
          onClose={() => setShowCreate(false)}
          onCreated={() => { setShowCreate(false); onRefresh() }}
        />
      )}

      {/* 编辑模板弹窗 */}
      {editingTemplate && (
        <TemplateFormModal
          mode="edit"
          initial={editingTemplate}
          onClose={() => setEditingTemplate(null)}
          onSaved={() => { setEditingTemplate(null); onRefresh() }}
        />
      )}
    </>
  )
}

// ============================================================================
// Template Card
// ============================================================================

function TemplateCard({ template, onClick }: { template: AgentTemplate; onClick: () => void }) {
  const color = getTypeColor(template.type)
  const skillCount = template.skills?.length || 0

  return (
    <div
      onClick={onClick}
      className={`bg-white rounded-xl border ${color.border} shadow-sm hover:shadow-md transition-all cursor-pointer p-4 sm:p-5`}
    >
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-3 min-w-0">
          <div className={`p-2 rounded-lg ${color.bg}`}>
            <Sparkles className={`w-5 h-5 ${color.text}`} />
          </div>
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <h3 className="font-semibold text-gray-900 truncate">{template.name}</h3>
              {template.is_builtin && (
                <span className="px-1.5 py-0.5 bg-blue-100 text-blue-700 text-[10px] font-medium rounded">内置</span>
              )}
            </div>
            <p className="text-xs text-gray-500 mt-0.5">{template.role || template.type}</p>
          </div>
        </div>
      </div>

      {template.description && (
        <p className="text-sm text-gray-600 mb-3 line-clamp-2">{template.description}</p>
      )}

      <div className="flex items-center gap-3 text-xs text-gray-500">
        {template.model && (
          <div className="flex items-center gap-1">
            <Cpu className="w-3 h-3" />
            <span>{template.model}</span>
          </div>
        )}
        {skillCount > 0 && (
          <div className="flex items-center gap-1">
            <Zap className="w-3 h-3" />
            <span>{skillCount} 技能</span>
          </div>
        )}
        {template.temperature !== undefined && template.temperature > 0 && (
          <span>T:{template.temperature}</span>
        )}
        <ChevronRight className="w-4 h-4 ml-auto text-gray-400" />
      </div>
    </div>
  )
}

// ============================================================================
// Template Detail
// ============================================================================

function TemplateDetail({ template, onClose, onDelete, onEdit }: {
  template: AgentTemplate
  onClose: () => void
  onDelete: () => void
  onEdit: () => void
}) {
  const color = getTypeColor(template.type)

  return (
    <div className="fixed inset-0 z-50 flex items-end sm:items-center justify-center">
      <div className="fixed inset-0 bg-black/40" onClick={onClose} />
      <div className="relative bg-white rounded-t-2xl sm:rounded-xl shadow-xl w-full sm:max-w-lg max-h-[85vh] overflow-y-auto z-10">
        <div className="sticky top-0 bg-white border-b px-5 py-4 flex items-center justify-between rounded-t-2xl sm:rounded-t-xl">
          <div className="flex items-center gap-3 min-w-0">
            <div className={`p-2 rounded-lg ${color.bg}`}>
              <Sparkles className={`w-5 h-5 ${color.text}`} />
            </div>
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <h2 className="font-bold text-gray-900 truncate">{template.name}</h2>
                {template.is_builtin && <span className="px-1.5 py-0.5 bg-blue-100 text-blue-700 text-[10px] font-medium rounded">内置</span>}
              </div>
              <p className="text-xs text-gray-500">{template.role || template.type}</p>
            </div>
          </div>
          <button onClick={onClose} className="p-2 hover:bg-gray-100 rounded-lg">
            <X className="w-5 h-5 text-gray-500" />
          </button>
        </div>

        <div className="p-5 space-y-5">
          {template.description && (
            <p className="text-sm text-gray-600">{template.description}</p>
          )}

          {/* 身份与性格 */}
          <div>
            <h3 className="text-sm font-semibold text-gray-700 mb-3">身份与运行参数</h3>
            <div className="grid grid-cols-2 gap-3">
              <InfoItem label="模型类型" value={template.type} />
              <InfoItem label="模型" value={template.model || '-'} />
              <InfoItem label="温度" value={template.temperature?.toString() || '-'} />
              <InfoItem label="上下文" value={template.max_context ? `${(template.max_context / 1000).toFixed(0)}K` : '-'} />
            </div>
          </div>

          {/* 性格 */}
          {template.personality && template.personality.length > 0 && (
            <div>
              <h3 className="text-sm font-semibold text-gray-700 mb-3">性格特征</h3>
              <div className="flex flex-wrap gap-2">
                {template.personality.map((p, i) => (
                  <span key={i} className="px-2.5 py-1 rounded-full text-xs font-medium bg-purple-50 text-purple-700 border border-purple-200">
                    {p}
                  </span>
                ))}
              </div>
            </div>
          )}

          {/* System Prompt */}
          {template.system_prompt && (
            <div>
              <h3 className="text-sm font-semibold text-gray-700 mb-3">系统提示词</h3>
              <div className="bg-gray-50 rounded-lg p-3">
                <pre className="text-xs text-gray-600 whitespace-pre-wrap">{template.system_prompt}</pre>
              </div>
            </div>
          )}

          {/* Skills */}
          {template.skills && template.skills.length > 0 && (
            <div>
              <h3 className="text-sm font-semibold text-gray-700 mb-3">技能</h3>
              <div className="flex flex-wrap gap-2">
                {template.skills.map((s, i) => (
                  <span key={i} className="px-2.5 py-1 rounded-full text-xs font-medium bg-green-50 text-green-700 border border-green-200">
                    <Zap className="w-3 h-3 inline mr-1" />{s}
                  </span>
                ))}
              </div>
            </div>
          )}

          {/* Tags */}
          {template.tags && template.tags.length > 0 && (
            <div>
              <h3 className="text-sm font-semibold text-gray-700 mb-3">标签</h3>
              <div className="flex flex-wrap gap-2">
                {template.tags.map((tag, i) => (
                  <span key={i} className="px-2.5 py-1 rounded-full text-xs bg-gray-100 text-gray-600">
                    <Tag className="w-3 h-3 inline mr-1" />{tag}
                  </span>
                ))}
              </div>
            </div>
          )}

          {/* Actions */}
          <div className="border-t pt-4 flex items-center justify-between">
            <p className="text-xs text-gray-400">
              创建于 {new Date(template.created_at).toLocaleString('zh-CN')}
            </p>
            {!template.is_builtin && (
              <div className="flex items-center gap-2">
                <button onClick={onEdit} className="flex items-center gap-2 px-4 py-2 text-sm text-blue-600 hover:bg-blue-50 rounded-lg">
                  <Pencil className="w-4 h-4" />
                  编辑
                </button>
                <button onClick={onDelete} className="flex items-center gap-2 px-4 py-2 text-sm text-red-600 hover:bg-red-50 rounded-lg">
                  <Trash2 className="w-4 h-4" />
                  删除
                </button>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

// ============================================================================
// Create Agent Wizard (复用 Instance 创建流程)
// ============================================================================

function CreateAgentWizard({
  agentTypes, accounts, onClose, onCreate
}: {
  agentTypes: AgentType[]
  accounts: Account[]
  onClose: () => void
  onCreate: (accountId: string, name: string) => void
}) {
  const [step, setStep] = useState(1)
  const [selectedType, setSelectedType] = useState('')
  const [selectedAccount, setSelectedAccount] = useState('')
  const [name, setName] = useState('')

  const filteredAccounts = accounts.filter(
    a => a.agent_type === selectedType && a.status === 'authenticated'
  )

  return (
    <div className="fixed inset-0 bg-black/50 flex items-end sm:items-center justify-center z-50">
      <div className="bg-white rounded-t-2xl sm:rounded-xl w-full sm:max-w-lg p-5 sm:p-6 max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-lg font-semibold">创建智能体</h2>
          <div className="flex items-center gap-3">
            <span className="text-sm text-gray-500">步骤 {step}/3</span>
            <button onClick={onClose} className="p-1 hover:bg-gray-100 rounded">
              <X className="w-5 h-5 text-gray-400" />
            </button>
          </div>
        </div>

        {step === 1 && (
          <div>
            <p className="text-sm text-gray-600 mb-4">选择 Agent 类型：</p>
            <div className="grid grid-cols-2 gap-3">
              {agentTypes.map(t => (
                <button
                  key={t.id}
                  onClick={() => setSelectedType(t.id)}
                  className={`p-4 border rounded-xl text-left transition-all ${
                    selectedType === t.id ? 'border-blue-500 bg-blue-50 shadow-sm' : 'hover:border-gray-300'
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
                <a href="/accounts" className="text-blue-600 hover:underline text-sm">前往添加账号 →</a>
              </div>
            ) : (
              <div className="space-y-2">
                {filteredAccounts.map(acc => (
                  <button
                    key={acc.id}
                    onClick={() => setSelectedAccount(acc.id)}
                    className={`w-full p-3 border rounded-xl text-left transition-all ${
                      selectedAccount === acc.id ? 'border-blue-500 bg-blue-50' : 'hover:border-gray-300'
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
            <p className="text-sm text-gray-600 mb-4">配置智能体：</p>
            <div>
              <label className="block text-sm font-medium mb-1">名称（可选）</label>
              <input
                type="text"
                value={name}
                onChange={e => setName(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="留空将自动生成"
              />
            </div>
            <div className="mt-4 p-3 bg-gray-50 rounded-lg text-sm space-y-1">
              <p><span className="text-gray-500">类型:</span> {agentTypes.find(t => t.id === selectedType)?.name}</p>
              <p><span className="text-gray-500">账号:</span> {accounts.find(a => a.id === selectedAccount)?.name}</p>
            </div>
          </div>
        )}

        <div className="flex justify-between mt-6">
          <button
            onClick={() => step > 1 ? setStep(step - 1) : onClose()}
            className="px-4 py-2 border rounded-lg hover:bg-gray-100 text-sm"
          >
            {step > 1 ? '上一步' : '取消'}
          </button>
          {step < 3 ? (
            <button
              onClick={() => setStep(step + 1)}
              disabled={(step === 1 && !selectedType) || (step === 2 && !selectedAccount)}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 text-sm"
            >
              下一步
            </button>
          ) : (
            <button
              onClick={() => onCreate(selectedAccount, name)}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 text-sm"
            >
              创建智能体
            </button>
          )}
        </div>
      </div>
    </div>
  )
}

// ============================================================================
// Create Template Modal
// ============================================================================

function TemplateFormModal({
  mode, initial, onClose, onSaved
}: {
  mode: 'create' | 'edit'
  initial?: AgentTemplate
  onClose: () => void
  onSaved: () => void
}) {
  const typeOptions = getAgentTypeOptions()
  const [name, setName] = useState(initial?.name || '')
  const [type, setType] = useState(initial?.type || 'claude')
  const [role, setRole] = useState(initial?.role || '')
  const [description, setDescription] = useState(initial?.description || '')
  const [model, setModel] = useState(initial?.model || getAgentTypeConfig(initial?.type || 'claude').defaultModel)
  const [customModel, setCustomModel] = useState('')
  const [temperature, setTemperature] = useState(initial?.temperature ?? getAgentTypeConfig(initial?.type || 'claude').defaultTemperature)
  const [systemPrompt, setSystemPrompt] = useState(initial?.system_prompt || '')
  const [submitting, setSubmitting] = useState(false)

  const typeConfig = getAgentTypeConfig(type)
  const hasModelList = typeConfig.models.length > 0

  const handleTypeChange = (newType: string) => {
    setType(newType)
    const cfg = getAgentTypeConfig(newType)
    setModel(cfg.defaultModel)
    setTemperature(cfg.defaultTemperature)
    setCustomModel('')
  }

  const handleSubmit = async () => {
    if (!name) return
    setSubmitting(true)
    try {
      const finalModel = hasModelList ? model : customModel
      const body: any = { name, type, role, description, model: finalModel, temperature }
      if (systemPrompt) body.system_prompt = systemPrompt

      const url = mode === 'edit' && initial
        ? `/api/v1/agent-templates/${initial.id}`
        : '/api/v1/agent-templates'
      const method = mode === 'edit' ? 'PATCH' : 'POST'

      const res = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      if (res.ok) onSaved()
    } catch (err) {
      console.error(`Failed to ${mode} template:`, err)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-end sm:items-center justify-center z-50">
      <div className="bg-white rounded-t-2xl sm:rounded-xl w-full sm:max-w-lg p-5 sm:p-6 max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-lg font-semibold">{mode === 'edit' ? '编辑模板' : '创建模板'}</h2>
          <button onClick={onClose} className="p-1 hover:bg-gray-100 rounded">
            <X className="w-5 h-5 text-gray-400" />
          </button>
        </div>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">名称 *</label>
            <input value={name} onChange={e => setName(e.target.value)}
              className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500" placeholder="例：代码审查助手" />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium mb-1">类型</label>
              <select value={type} onChange={e => handleTypeChange(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500">
                {typeOptions.map(opt => (
                  <option key={opt.value} value={opt.value}>{opt.label}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">角色</label>
              <input value={role} onChange={e => setRole(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500" placeholder="例：代码助手" />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">描述</label>
            <input value={description} onChange={e => setDescription(e.target.value)}
              className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500" placeholder="模板的简要描述" />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium mb-1">模型</label>
              {hasModelList ? (
                <select value={model} onChange={e => setModel(e.target.value)}
                  className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500">
                  {typeConfig.models.map(m => (
                    <option key={m.value} value={m.value}>{m.label}</option>
                  ))}
                </select>
              ) : (
                <input value={customModel} onChange={e => setCustomModel(e.target.value)}
                  className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="输入模型名称" />
              )}
              {hasModelList && (
                <p className="text-xs text-gray-400 mt-1">
                  {typeConfig.models.find(m => m.value === model)?.description}
                </p>
              )}
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">温度 ({temperature})</label>
              <input type="range" min="0" max="1" step="0.1" value={temperature}
                onChange={e => setTemperature(parseFloat(e.target.value))}
                className="w-full mt-2" />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">系统提示词</label>
            <textarea value={systemPrompt} onChange={e => setSystemPrompt(e.target.value)}
              rows={4} className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
              placeholder="可选：定义智能体的核心使命和行为规范" />
          </div>
        </div>

        <div className="flex justify-end gap-3 mt-6">
          <button onClick={onClose} className="px-4 py-2 border rounded-lg hover:bg-gray-100 text-sm">取消</button>
          <button onClick={handleSubmit} disabled={!name || submitting}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 text-sm">
            {submitting ? (mode === 'edit' ? '保存中...' : '创建中...') : (mode === 'edit' ? '保存' : '创建模板')}
          </button>
        </div>
      </div>
    </div>
  )
}

function CreateTemplateModal({ onClose, onCreated }: { onClose: () => void; onCreated: () => void }) {
  return <TemplateFormModal mode="create" onClose={onClose} onSaved={onCreated} />
}

// ============================================================================
// Terminal Modal
// ============================================================================

function TerminalModal({ session, onClose }: { session: TerminalSession; onClose: () => void }) {
  const host = typeof window !== 'undefined' ? window.location.hostname : 'localhost'
  const ready = session.status === 'running' && !!session.port
  const iframeUrl = ready ? `http://${host}:${session.port}/` : 'about:blank'

  return (
    <div className="fixed inset-0 bg-black/80 flex items-center justify-center z-50 p-2 sm:p-4">
      <div className="bg-gray-900 rounded-lg w-full max-w-4xl h-[85vh] sm:h-[600px] flex flex-col">
        <div className="flex items-center justify-between px-4 py-2 border-b border-gray-700">
          <div className="flex items-center gap-2 text-white">
            <Terminal className="w-5 h-5" />
            <span className="font-medium">终端</span>
          </div>
          <button onClick={onClose} className="p-1 text-gray-400 hover:text-white hover:bg-gray-700 rounded">
            <X className="w-5 h-5" />
          </button>
        </div>
        <div className="flex-1 bg-black relative">
          {!ready && (
            <div className="absolute inset-0 flex items-center justify-center">
              <div className="text-gray-200 text-sm">终端启动中…（{session.status || 'pending'}）</div>
            </div>
          )}
          <iframe key={`${session.id}-${ready ? 'ready' : 'pending'}`} src={iframeUrl} className="w-full h-full border-0" title="Terminal" />
        </div>
      </div>
    </div>
  )
}

// ============================================================================
// Shared Components
// ============================================================================

function InfoItem({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-gray-50 rounded-lg p-3">
      <p className="text-xs text-gray-500">{label}</p>
      <p className="text-sm font-medium text-gray-900 mt-0.5 truncate" title={value}>{value}</p>
    </div>
  )
}

function StatBadge({ label, value, color }: { label: string; value: number; color: string }) {
  return (
    <div className="flex items-center gap-1.5">
      <span className={`text-lg font-bold ${color}`}>{value}</span>
      <span className="text-xs text-gray-500">{label}</span>
    </div>
  )
}

function EmptyState({ icon: Icon, title, description, action }: {
  icon: React.ComponentType<{ className?: string }>
  title: string
  description: string
  action?: React.ReactNode
}) {
  return (
    <div className="bg-white rounded-xl border p-8 text-center">
      <Icon className="w-12 h-12 mx-auto text-gray-400 mb-4" />
      <h3 className="text-lg font-medium mb-2">{title}</h3>
      <p className="text-gray-500 text-sm mb-4">{description}</p>
      {action}
    </div>
  )
}
