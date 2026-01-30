'use client'

import { useState, useEffect } from 'react'
import { Settings, Save } from 'lucide-react'
import { AdminLayout } from '@/components/layout'

interface AgentType {
  id: string
  name: string
  image: string
  description: string
  login_methods: string[]
}

export default function SettingsPage() {
  const [agentTypes, setAgentTypes] = useState<AgentType[]>([])
  const [loading, setLoading] = useState(true)

  const fetchSettings = async () => {
    setLoading(true)
    try {
      const res = await fetch('/api/v1/agent-types')
      if (res.ok) {
        const data = await res.json()
        setAgentTypes(data.agent_types || [])
      }
    } catch (err) {
      console.error('Failed to fetch settings:', err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchSettings()
  }, [])

  return (
    <AdminLayout title="系统设置" onRefresh={fetchSettings} loading={loading}>
      <div className="space-y-6">
        {/* Agent Types */}
        <div className="bg-white rounded-lg border">
          <div className="px-4 py-3 border-b bg-gray-50">
            <h2 className="font-medium">Agent 类型配置</h2>
            <p className="text-sm text-gray-500">查看已注册的 AI Agent 类型</p>
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
                      <p className="text-xs text-gray-400">
                        登录方式: {type.login_methods?.join(', ') || 'N/A'}
                      </p>
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
            <h2 className="font-medium">系统信息</h2>
          </div>
          <div className="px-4 py-3 space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-gray-500">版本</span>
              <span className="font-mono">v0.1.0 (Phase 1.5)</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-500">API 地址</span>
              <span className="font-mono">http://localhost:8080</span>
            </div>
          </div>
        </div>
      </div>
    </AdminLayout>
  )
}
