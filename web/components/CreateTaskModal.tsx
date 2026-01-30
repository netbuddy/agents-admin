'use client'

import { useState, useEffect } from 'react'
import { X, AlertCircle, CheckCircle, Box, Server } from 'lucide-react'
import Link from 'next/link'

interface AgentType {
  id: string
  name: string
  description: string
}

interface Instance {
  id: string
  name: string
  account_id: string
  agent_type: string
  container: string
  status: string
  node: string
}

interface Props {
  onClose: () => void
  onCreated: () => void
}

export default function CreateTaskModal({ onClose, onCreated }: Props) {
  const [name, setName] = useState('')
  const [prompt, setPrompt] = useState('')
  const [agentType, setAgentType] = useState('')
  const [instanceId, setInstanceId] = useState('')
  const [loading, setLoading] = useState(false)
  
  // 数据加载状态
  const [agentTypes, setAgentTypes] = useState<AgentType[]>([])
  const [instances, setInstances] = useState<Instance[]>([])
  const [loadingData, setLoadingData] = useState(true)

  // 获取 Agent 类型和实例列表
  useEffect(() => {
    const fetchData = async () => {
      try {
        const [typesRes, instancesRes] = await Promise.all([
          fetch('/api/v1/agent-types'),
          fetch('/api/v1/instances')
        ])
        
        if (typesRes.ok) {
          const data = await typesRes.json()
          const types = data.agent_types || []
          setAgentTypes(types)
          // 默认选择第一个类型
          if (types.length > 0 && !agentType) {
            setAgentType(types[0].id)
          }
        }
        
        if (instancesRes.ok) {
          const data = await instancesRes.json()
          setInstances(data.instances || [])
        }
      } catch (err) {
        console.error('Failed to fetch data:', err)
      } finally {
        setLoadingData(false)
      }
    }
    
    fetchData()
  }, [])

  // 根据选择的 Agent 类型过滤运行中的实例
  const filteredInstances = instances.filter(
    inst => inst.agent_type === agentType && inst.status === 'running'
  )

  // 当 Agent 类型改变时，重置实例选择
  useEffect(() => {
    const available = instances.filter(
      inst => inst.agent_type === agentType && inst.status === 'running'
    )
    if (available.length > 0) {
      setInstanceId(available[0].id)
    } else {
      setInstanceId('')
    }
  }, [agentType, instances])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!instanceId) {
      alert('请选择一个运行中的实例')
      return
    }
    
    // 获取选中的实例信息
    const selectedInstance = instances.find(inst => inst.id === instanceId)
    
    setLoading(true)

    try {
      const res = await fetch('/api/v1/tasks', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name,
          spec: {
            prompt,
            agent: { 
              type: agentType,
              instance_id: instanceId,
              account_id: selectedInstance?.account_id // 保留账号信息
            },
          },
        }),
      })

      if (res.ok) {
        onCreated()
      }
    } catch (err) {
      console.error('Failed to create task:', err)
    } finally {
      setLoading(false)
    }
  }

  const getAgentTypeName = (typeId: string) => {
    const t = agentTypes.find(at => at.id === typeId)
    return t?.name || typeId
  }
  
  // 从 account_id 中提取用户名显示
  const getAccountName = (accountId: string) => {
    // account_id 格式: qwen-code_user_at_email_com
    const parts = accountId.split('_')
    if (parts.length > 1) {
      // 移除第一部分（agent 类型），将剩余部分还原
      const emailPart = parts.slice(1).join('_')
      return emailPart.replace('_at_', '@').replace(/_/g, '.')
    }
    return accountId
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg w-full max-w-lg p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold">新建任务</h2>
          <button onClick={onClose} className="p-1 hover:bg-gray-100 rounded">
            <X className="w-5 h-5" />
          </button>
        </div>

        {loadingData ? (
          <div className="flex items-center justify-center py-12">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
          </div>
        ) : (
          <form onSubmit={handleSubmit}>
            <div className="mb-4">
              <label className="block text-sm font-medium mb-1">任务名称</label>
              <input
                type="text"
                value={name}
                onChange={e => setName(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="例如：修复登录页面 bug"
                required
              />
            </div>

            <div className="mb-4">
              <label className="block text-sm font-medium mb-1">Agent 类型</label>
              <select
                value={agentType}
                onChange={e => setAgentType(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                {agentTypes.map(t => (
                  <option key={t.id} value={t.id}>
                    {t.name}
                  </option>
                ))}
              </select>
            </div>

            <div className="mb-4">
              <label className="block text-sm font-medium mb-1">选择实例</label>
              {filteredInstances.length === 0 ? (
                <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
                  <div className="flex items-start gap-3">
                    <AlertCircle className="w-5 h-5 text-yellow-600 flex-shrink-0 mt-0.5" />
                    <div className="flex-1">
                      <p className="text-sm text-yellow-800">
                        没有可用的 {getAgentTypeName(agentType)} 实例
                      </p>
                      <p className="text-xs text-yellow-600 mt-1">
                        需要先创建一个运行中的实例才能创建任务
                      </p>
                      <Link 
                        href="/instances" 
                        className="inline-flex items-center gap-1 mt-2 text-sm text-blue-600 hover:underline"
                      >
                        <Box className="w-4 h-4" />
                        前往创建实例
                      </Link>
                    </div>
                  </div>
                </div>
              ) : (
                <div className="space-y-2">
                  {filteredInstances.map(inst => (
                    <label
                      key={inst.id}
                      className={`flex items-center gap-3 p-3 border rounded-lg cursor-pointer transition-colors ${
                        instanceId === inst.id 
                          ? 'border-blue-500 bg-blue-50' 
                          : 'border-gray-200 hover:bg-gray-50'
                      }`}
                    >
                      <input
                        type="radio"
                        name="instance"
                        value={inst.id}
                        checked={instanceId === inst.id}
                        onChange={e => setInstanceId(e.target.value)}
                        className="sr-only"
                      />
                      <div className={`w-4 h-4 rounded-full border-2 flex items-center justify-center ${
                        instanceId === inst.id ? 'border-blue-500' : 'border-gray-300'
                      }`}>
                        {instanceId === inst.id && (
                          <div className="w-2 h-2 rounded-full bg-blue-500" />
                        )}
                      </div>
                      <Box className={`w-5 h-5 flex-shrink-0 ${
                        inst.status === 'running' ? 'text-green-500' : 'text-gray-400'
                      }`} />
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="font-medium text-sm truncate">{inst.name}</span>
                          {inst.status === 'running' && (
                            <CheckCircle className="w-4 h-4 text-green-500 flex-shrink-0" />
                          )}
                        </div>
                        <p className="text-xs text-gray-500 flex items-center gap-2">
                          <span>账号: {getAccountName(inst.account_id)}</span>
                          <span className="flex items-center gap-1">
                            <Server className="w-3 h-3" />
                            {inst.node}
                          </span>
                        </p>
                      </div>
                    </label>
                  ))}
                </div>
              )}
            </div>

            <div className="mb-4">
              <label className="block text-sm font-medium mb-1">任务提示词</label>
              <textarea
                value={prompt}
                onChange={e => setPrompt(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 h-32"
                placeholder="描述你希望 AI Agent 完成的任务..."
                required
              />
            </div>

            <div className="flex justify-end gap-2">
              <button
                type="button"
                onClick={onClose}
                className="px-4 py-2 border rounded-lg hover:bg-gray-100"
              >
                取消
              </button>
              <button
                type="submit"
                disabled={loading || !instanceId}
                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {loading ? '创建中...' : '创建任务'}
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  )
}
