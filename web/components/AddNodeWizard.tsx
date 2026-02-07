'use client'

import { useState, useEffect } from 'react'
import {
  X, Server, Key, Lock, Globe, Download,
  CheckCircle, XCircle, Loader2, ChevronRight, ChevronLeft
} from 'lucide-react'

interface ProvisionRequest {
  node_id: string
  host: string
  port: number
  ssh_user: string
  auth_method: 'password' | 'pubkey'
  password?: string
  private_key?: string
  version: string
  github_repo: string
  api_server_url: string
}

interface Provision {
  id: string
  node_id: string
  host: string
  status: string
  error_message?: string
  version: string
  created_at: string
  updated_at: string
}

const statusSteps = [
  { key: 'pending', label: '准备中' },
  { key: 'connecting', label: 'SSH 连接' },
  { key: 'downloading', label: '下载 deb' },
  { key: 'installing', label: '安装' },
  { key: 'configuring', label: '配置' },
  { key: 'completed', label: '完成' },
]

function StepIndicator({ currentStatus }: { currentStatus: string }) {
  const failed = currentStatus === 'failed'
  const currentIdx = failed
    ? statusSteps.length - 1
    : statusSteps.findIndex(s => s.key === currentStatus)

  return (
    <div className="flex items-center gap-1 overflow-x-auto pb-2">
      {statusSteps.map((step, i) => {
        const done = !failed && i < currentIdx
        const active = !failed && i === currentIdx
        const isFailed = failed && i === currentIdx

        return (
          <div key={step.key} className="flex items-center gap-1 flex-shrink-0">
            {i > 0 && <div className={`w-4 h-0.5 ${done ? 'bg-green-400' : 'bg-gray-200'}`} />}
            <div className="flex flex-col items-center gap-0.5">
              <div className={`w-6 h-6 rounded-full flex items-center justify-center text-xs
                ${done ? 'bg-green-100 text-green-600' : ''}
                ${active ? 'bg-blue-100 text-blue-600 ring-2 ring-blue-300' : ''}
                ${isFailed ? 'bg-red-100 text-red-600' : ''}
                ${!done && !active && !isFailed ? 'bg-gray-100 text-gray-400' : ''}
              `}>
                {done ? <CheckCircle className="w-3.5 h-3.5" /> :
                 isFailed ? <XCircle className="w-3.5 h-3.5" /> :
                 active ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> :
                 <span>{i + 1}</span>}
              </div>
              <span className={`text-[10px] whitespace-nowrap
                ${done ? 'text-green-600' : active ? 'text-blue-600' : isFailed ? 'text-red-600' : 'text-gray-400'}
              `}>{step.label}</span>
            </div>
          </div>
        )
      })}
    </div>
  )
}

export default function AddNodeWizard({ onClose, onSuccess }: { onClose: () => void; onSuccess?: () => void }) {
  const [step, setStep] = useState(1)
  const [submitting, setSubmitting] = useState(false)
  const [provision, setProvision] = useState<Provision | null>(null)
  const [error, setError] = useState('')

  // Step 1: Connection
  const [host, setHost] = useState('')
  const [port, setPort] = useState(22)
  const [sshUser, setSshUser] = useState('root')
  const [authMethod, setAuthMethod] = useState<'password' | 'pubkey'>('password')
  const [password, setPassword] = useState('')
  const [privateKey, setPrivateKey] = useState('')
  const [nodeId, setNodeId] = useState('')

  // Step 2: Version
  const [version, setVersion] = useState('')
  const [githubRepo, setGithubRepo] = useState('netbuddy/agents-admin')
  const [apiServerUrl, setApiServerUrl] = useState('')

  useEffect(() => {
    if (typeof window !== 'undefined') {
      setApiServerUrl(`${window.location.protocol}//${window.location.hostname}:8080`)
    }
  }, [])

  // Auto-generate node_id from host
  useEffect(() => {
    if (!nodeId && host) {
      setNodeId(`node-${host.replace(/\./g, '-')}`)
    }
  }, [host, nodeId])

  const canGoStep2 = host && sshUser && (authMethod === 'password' ? password : privateKey)
  const canGoStep3 = version && apiServerUrl

  const handleSubmit = async () => {
    setSubmitting(true)
    setError('')
    try {
      const body: ProvisionRequest = {
        node_id: nodeId || `node-${host}`,
        host,
        port,
        ssh_user: sshUser,
        auth_method: authMethod,
        version,
        github_repo: githubRepo,
        api_server_url: apiServerUrl,
      }
      if (authMethod === 'password') body.password = password
      else body.private_key = privateKey

      const res = await fetch('/api/v1/node-provisions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      if (!res.ok) {
        const data = await res.json()
        throw new Error(data.error || 'provision failed')
      }
      const prov = await res.json()
      setProvision(prov)
      setStep(4) // Progress view
    } catch (err: any) {
      setError(err.message)
    } finally {
      setSubmitting(false)
    }
  }

  // Poll provision status
  useEffect(() => {
    if (!provision || provision.status === 'completed' || provision.status === 'failed') return
    const interval = setInterval(async () => {
      try {
        const res = await fetch(`/api/v1/node-provisions/${provision.id}`)
        if (res.ok) {
          const data = await res.json()
          setProvision(data)
          if (data.status === 'completed' || data.status === 'failed') {
            if (data.status === 'completed' && onSuccess) onSuccess()
          }
        }
      } catch {}
    }, 2000)
    return () => clearInterval(interval)
  }, [provision, onSuccess])

  return (
    <div className="fixed inset-0 z-50 flex items-end sm:items-center justify-center">
      <div className="fixed inset-0 bg-black/40" onClick={onClose} />
      <div className="relative bg-white rounded-t-2xl sm:rounded-xl shadow-xl w-full sm:max-w-lg max-h-[85vh] overflow-y-auto z-10">
        {/* Header */}
        <div className="sticky top-0 bg-white border-b px-5 py-4 flex items-center justify-between rounded-t-2xl sm:rounded-t-xl">
          <div className="flex items-center gap-2">
            <Server className="w-5 h-5 text-blue-600" />
            <h2 className="font-bold text-gray-900">
              {step <= 3 ? '添加节点' : '部署进度'}
            </h2>
          </div>
          <button onClick={onClose} className="p-2 hover:bg-gray-100 rounded-lg">
            <X className="w-5 h-5 text-gray-500" />
          </button>
        </div>

        <div className="p-5">
          {/* Step 1: Connection Info */}
          {step === 1 && (
            <div className="space-y-4">
              <p className="text-sm text-gray-500 mb-4">配置目标节点的 SSH 连接信息</p>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">节点 ID（可选）</label>
                <input
                  type="text"
                  value={nodeId}
                  onChange={e => setNodeId(e.target.value)}
                  placeholder="自动生成"
                  className="w-full px-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>

              <div className="grid grid-cols-3 gap-3">
                <div className="col-span-2">
                  <label className="block text-sm font-medium text-gray-700 mb-1">主机地址 *</label>
                  <div className="relative">
                    <Globe className="absolute left-3 top-2.5 w-4 h-4 text-gray-400" />
                    <input
                      type="text"
                      value={host}
                      onChange={e => setHost(e.target.value)}
                      placeholder="192.168.1.100 或 hostname"
                      className="w-full pl-9 pr-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                    />
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">端口</label>
                  <input
                    type="number"
                    value={port}
                    onChange={e => setPort(Number(e.target.value))}
                    className="w-full px-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                  />
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">SSH 用户名 *</label>
                <input
                  type="text"
                  value={sshUser}
                  onChange={e => setSshUser(e.target.value)}
                  className="w-full px-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">认证方式</label>
                <div className="flex gap-2">
                  <button
                    onClick={() => setAuthMethod('password')}
                    className={`flex-1 flex items-center justify-center gap-2 px-3 py-2.5 rounded-lg border text-sm transition-colors
                      ${authMethod === 'password' ? 'border-blue-500 bg-blue-50 text-blue-700' : 'border-gray-200 text-gray-600 hover:bg-gray-50'}`}
                  >
                    <Lock className="w-4 h-4" /> 密码
                  </button>
                  <button
                    onClick={() => setAuthMethod('pubkey')}
                    className={`flex-1 flex items-center justify-center gap-2 px-3 py-2.5 rounded-lg border text-sm transition-colors
                      ${authMethod === 'pubkey' ? 'border-blue-500 bg-blue-50 text-blue-700' : 'border-gray-200 text-gray-600 hover:bg-gray-50'}`}
                  >
                    <Key className="w-4 h-4" /> SSH 密钥
                  </button>
                </div>
              </div>

              {authMethod === 'password' ? (
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">密码 *</label>
                  <input
                    type="password"
                    value={password}
                    onChange={e => setPassword(e.target.value)}
                    className="w-full px-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                  />
                </div>
              ) : (
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">私钥 *</label>
                  <textarea
                    value={privateKey}
                    onChange={e => setPrivateKey(e.target.value)}
                    rows={4}
                    placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
                    className="w-full px-3 py-2 border rounded-lg text-sm font-mono focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                  />
                </div>
              )}
            </div>
          )}

          {/* Step 2: Version */}
          {step === 2 && (
            <div className="space-y-4">
              <p className="text-sm text-gray-500 mb-4">选择要安装的版本和 API Server 地址</p>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">版本号 *</label>
                <div className="relative">
                  <Download className="absolute left-3 top-2.5 w-4 h-4 text-gray-400" />
                  <input
                    type="text"
                    value={version}
                    onChange={e => setVersion(e.target.value)}
                    placeholder="例如: 0.9.0"
                    className="w-full pl-9 pr-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                  />
                </div>
                <p className="text-xs text-gray-400 mt-1">对应 GitHub Release 的版本号（不含 v 前缀）</p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">GitHub 仓库</label>
                <input
                  type="text"
                  value={githubRepo}
                  onChange={e => setGithubRepo(e.target.value)}
                  className="w-full px-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">API Server 地址 *</label>
                <input
                  type="text"
                  value={apiServerUrl}
                  onChange={e => setApiServerUrl(e.target.value)}
                  placeholder="https://api.example.com:8080"
                  className="w-full px-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
                <p className="text-xs text-gray-400 mt-1">node-manager 连接的 API Server 地址</p>
              </div>
            </div>
          )}

          {/* Step 3: Confirm */}
          {step === 3 && (
            <div className="space-y-4">
              <p className="text-sm text-gray-500 mb-4">确认部署信息</p>
              <div className="bg-gray-50 rounded-lg p-4 space-y-2 text-sm">
                <div className="flex justify-between"><span className="text-gray-500">节点 ID</span><span className="font-medium">{nodeId}</span></div>
                <div className="flex justify-between"><span className="text-gray-500">主机</span><span className="font-medium">{host}:{port}</span></div>
                <div className="flex justify-between"><span className="text-gray-500">用户</span><span className="font-medium">{sshUser}</span></div>
                <div className="flex justify-between"><span className="text-gray-500">认证</span><span className="font-medium">{authMethod === 'password' ? '密码' : 'SSH 密钥'}</span></div>
                <div className="flex justify-between"><span className="text-gray-500">版本</span><span className="font-medium">v{version}</span></div>
                <div className="flex justify-between"><span className="text-gray-500">仓库</span><span className="font-medium">{githubRepo}</span></div>
                <div className="flex justify-between"><span className="text-gray-500">API Server</span><span className="font-medium text-xs">{apiServerUrl}</span></div>
              </div>
              {error && (
                <div className="bg-red-50 border border-red-200 rounded-lg p-3 text-sm text-red-700">{error}</div>
              )}
            </div>
          )}

          {/* Step 4: Progress */}
          {step === 4 && provision && (
            <div className="space-y-4">
              <StepIndicator currentStatus={provision.status} />

              <div className="bg-gray-50 rounded-lg p-4 space-y-2 text-sm">
                <div className="flex justify-between"><span className="text-gray-500">部署 ID</span><span className="font-mono text-xs">{provision.id}</span></div>
                <div className="flex justify-between"><span className="text-gray-500">节点</span><span className="font-medium">{provision.node_id}</span></div>
                <div className="flex justify-between"><span className="text-gray-500">主机</span><span className="font-medium">{provision.host}</span></div>
                <div className="flex justify-between">
                  <span className="text-gray-500">状态</span>
                  <span className={`font-medium ${
                    provision.status === 'completed' ? 'text-green-600' :
                    provision.status === 'failed' ? 'text-red-600' : 'text-blue-600'
                  }`}>
                    {provision.status === 'completed' ? '部署成功' :
                     provision.status === 'failed' ? '部署失败' : '部署中...'}
                  </span>
                </div>
              </div>

              {provision.error_message && (
                <div className="bg-red-50 border border-red-200 rounded-lg p-3 text-sm text-red-700">
                  {provision.error_message}
                </div>
              )}

              {provision.status === 'completed' && (
                <div className="bg-green-50 border border-green-200 rounded-lg p-3 text-sm text-green-700 flex items-center gap-2">
                  <CheckCircle className="w-4 h-4 flex-shrink-0" />
                  <span>节点已成功部署并上线！</span>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="sticky bottom-0 bg-white border-t px-5 py-4 flex justify-between">
          {step <= 3 ? (
            <>
              <button
                onClick={() => step > 1 ? setStep(step - 1) : onClose()}
                className="flex items-center gap-1 px-4 py-2 text-sm text-gray-600 hover:bg-gray-100 rounded-lg"
              >
                <ChevronLeft className="w-4 h-4" />
                {step === 1 ? '取消' : '上一步'}
              </button>
              {step < 3 ? (
                <button
                  onClick={() => setStep(step + 1)}
                  disabled={step === 1 ? !canGoStep2 : !canGoStep3}
                  className="flex items-center gap-1 px-4 py-2 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  下一步 <ChevronRight className="w-4 h-4" />
                </button>
              ) : (
                <button
                  onClick={handleSubmit}
                  disabled={submitting}
                  className="flex items-center gap-1 px-5 py-2 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
                >
                  {submitting ? <Loader2 className="w-4 h-4 animate-spin" /> : <Download className="w-4 h-4" />}
                  开始部署
                </button>
              )}
            </>
          ) : (
            <div className="w-full flex justify-end">
              <button
                onClick={() => { onClose(); if (provision?.status === 'completed' && onSuccess) onSuccess() }}
                className="px-4 py-2 text-sm bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200"
              >
                关闭
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
