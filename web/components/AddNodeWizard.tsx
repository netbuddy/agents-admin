'use client'

import { useState, useEffect } from 'react'
import {
  X, Server, Key, Lock, Globe, Download,
  CheckCircle, XCircle, Loader2, ChevronRight, ChevronLeft
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

interface ProvisionRequest {
  node_id: string
  display_name: string
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

const statusStepKeys = ['pending', 'connecting', 'downloading', 'installing', 'configuring', 'completed']

function StepIndicator({ currentStatus }: { currentStatus: string }) {
  const { t } = useTranslation('nodes')
  const failed = currentStatus === 'failed'
  const currentIdx = failed
    ? statusStepKeys.length - 1
    : statusStepKeys.indexOf(currentStatus)

  return (
    <div className="flex items-center gap-1 overflow-x-auto pb-2">
      {statusStepKeys.map((stepKey, i) => {
        const done = !failed && i < currentIdx
        const active = !failed && i === currentIdx
        const isFailed = failed && i === currentIdx

        return (
          <div key={stepKey} className="flex items-center gap-1 flex-shrink-0">
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
              `}>{t(`wizard.steps.${stepKey}`)}</span>
            </div>
          </div>
        )
      })}
    </div>
  )
}

export default function AddNodeWizard({ onClose, onSuccess }: { onClose: () => void; onSuccess?: () => void }) {
  const { t } = useTranslation('nodes')
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
  const [nodeName, setNodeName] = useState('')
  const [nameError, setNameError] = useState('')

  // Step 2: Version
  const [version, setVersion] = useState('')
  const [githubRepo, setGithubRepo] = useState('netbuddy/agents-admin')
  const [apiServerUrl, setApiServerUrl] = useState('')

  useEffect(() => {
    if (typeof window !== 'undefined') {
      setApiServerUrl(`${window.location.protocol}//${window.location.hostname}:8080`)
    }
  }, [])

  const canGoStep2 = host && sshUser && nodeName.trim() && !nameError && (authMethod === 'password' ? password : privateKey)
  const canGoStep3 = version && apiServerUrl

  const handleSubmit = async () => {
    setSubmitting(true)
    setError('')
    try {
      const body: ProvisionRequest = {
        node_id: `node-${host.replace(/\./g, '-')}`,
        display_name: nodeName.trim(),
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
              {step <= 3 ? t('wizard.addNode') : t('wizard.deployProgress')}
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
              <p className="text-sm text-gray-500 mb-4">{t('wizard.sshDesc')}</p>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">{t('wizard.nodeNameLabel')}</label>
                <input
                  type="text"
                  value={nodeName}
                  onChange={e => { setNodeName(e.target.value); setNameError('') }}
                  placeholder={t('wizard.nodeNamePlaceholder')}
                  className={`w-full px-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500 ${nameError ? 'border-red-400' : ''}`}
                />
                {nameError && <p className="text-xs text-red-500 mt-1">{nameError}</p>}
              </div>

              <div className="grid grid-cols-3 gap-3">
                <div className="col-span-2">
                  <label className="block text-sm font-medium text-gray-700 mb-1">{t('wizard.hostLabel')}</label>
                  <div className="relative">
                    <Globe className="absolute left-3 top-2.5 w-4 h-4 text-gray-400" />
                    <input
                      type="text"
                      value={host}
                      onChange={e => setHost(e.target.value)}
                      placeholder={t('wizard.hostPlaceholder')}
                      className="w-full pl-9 pr-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                    />
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">{t('wizard.portLabel')}</label>
                  <input
                    type="number"
                    value={port}
                    onChange={e => setPort(Number(e.target.value))}
                    className="w-full px-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                  />
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">{t('wizard.sshUserLabel')}</label>
                <input
                  type="text"
                  value={sshUser}
                  onChange={e => setSshUser(e.target.value)}
                  className="w-full px-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">{t('wizard.authMethodLabel')}</label>
                <div className="flex gap-2">
                  <button
                    onClick={() => setAuthMethod('password')}
                    className={`flex-1 flex items-center justify-center gap-2 px-3 py-2.5 rounded-lg border text-sm transition-colors
                      ${authMethod === 'password' ? 'border-blue-500 bg-blue-50 text-blue-700' : 'border-gray-200 text-gray-600 hover:bg-gray-50'}`}
                  >
                    <Lock className="w-4 h-4" /> {t('wizard.authPassword')}
                  </button>
                  <button
                    onClick={() => setAuthMethod('pubkey')}
                    className={`flex-1 flex items-center justify-center gap-2 px-3 py-2.5 rounded-lg border text-sm transition-colors
                      ${authMethod === 'pubkey' ? 'border-blue-500 bg-blue-50 text-blue-700' : 'border-gray-200 text-gray-600 hover:bg-gray-50'}`}
                  >
                    <Key className="w-4 h-4" /> {t('wizard.authPubkey')}
                  </button>
                </div>
              </div>

              {authMethod === 'password' ? (
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">{t('wizard.passwordLabel')}</label>
                  <input
                    type="password"
                    value={password}
                    onChange={e => setPassword(e.target.value)}
                    className="w-full px-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                  />
                </div>
              ) : (
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">{t('wizard.privateKeyLabel')}</label>
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
              <p className="text-sm text-gray-500 mb-4">{t('wizard.versionDesc')}</p>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">{t('wizard.versionLabel')}</label>
                <div className="relative">
                  <Download className="absolute left-3 top-2.5 w-4 h-4 text-gray-400" />
                  <input
                    type="text"
                    value={version}
                    onChange={e => setVersion(e.target.value)}
                    placeholder={t('wizard.versionPlaceholder')}
                    className="w-full pl-9 pr-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                  />
                </div>
                <p className="text-xs text-gray-400 mt-1">{t('wizard.versionHint')}</p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">{t('wizard.githubRepoLabel')}</label>
                <input
                  type="text"
                  value={githubRepo}
                  onChange={e => setGithubRepo(e.target.value)}
                  className="w-full px-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">{t('wizard.apiServerLabel')}</label>
                <input
                  type="text"
                  value={apiServerUrl}
                  onChange={e => setApiServerUrl(e.target.value)}
                  placeholder="https://api.example.com:8080"
                  className="w-full px-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
                <p className="text-xs text-gray-400 mt-1">{t('wizard.apiServerHint')}</p>
              </div>
            </div>
          )}

          {/* Step 3: Confirm */}
          {step === 3 && (
            <div className="space-y-4">
              <p className="text-sm text-gray-500 mb-4">{t('wizard.confirmDesc')}</p>
              <div className="bg-gray-50 rounded-lg p-4 space-y-2 text-sm">
                <div className="flex justify-between"><span className="text-gray-500">{t('wizard.summaryNodeName')}</span><span className="font-medium">{nodeName}</span></div>
                <div className="flex justify-between"><span className="text-gray-500">{t('wizard.summaryHost')}</span><span className="font-medium">{host}:{port}</span></div>
                <div className="flex justify-between"><span className="text-gray-500">{t('wizard.summaryUser')}</span><span className="font-medium">{sshUser}</span></div>
                <div className="flex justify-between"><span className="text-gray-500">{t('wizard.summaryAuth')}</span><span className="font-medium">{authMethod === 'password' ? t('wizard.authPassword') : t('wizard.authPubkey')}</span></div>
                <div className="flex justify-between"><span className="text-gray-500">{t('wizard.summaryVersion')}</span><span className="font-medium">v{version}</span></div>
                <div className="flex justify-between"><span className="text-gray-500">{t('wizard.summaryRepo')}</span><span className="font-medium">{githubRepo}</span></div>
                <div className="flex justify-between"><span className="text-gray-500">{t('wizard.summaryApiServer')}</span><span className="font-medium text-xs">{apiServerUrl}</span></div>
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
                <div className="flex justify-between"><span className="text-gray-500">{t('wizard.deployId')}</span><span className="font-mono text-xs">{provision.id}</span></div>
                <div className="flex justify-between"><span className="text-gray-500">{t('wizard.deployNode')}</span><span className="font-medium">{provision.node_id}</span></div>
                <div className="flex justify-between"><span className="text-gray-500">{t('wizard.deployHost')}</span><span className="font-medium">{provision.host}</span></div>
                <div className="flex justify-between">
                  <span className="text-gray-500">{t('wizard.deployStatus')}</span>
                  <span className={`font-medium ${
                    provision.status === 'completed' ? 'text-green-600' :
                    provision.status === 'failed' ? 'text-red-600' : 'text-blue-600'
                  }`}>
                    {provision.status === 'completed' ? t('wizard.deploySuccess') :
                     provision.status === 'failed' ? t('wizard.deployFailed') : t('wizard.deploying')}
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
                  <span>{t('wizard.deployCompleteMsg')}</span>
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
                {step === 1 ? t('action.cancel', { ns: 'common' }) : t('action.prevStep', { ns: 'common' })}
              </button>
              {step < 3 ? (
                <button
                  onClick={() => setStep(step + 1)}
                  disabled={step === 1 ? !canGoStep2 : !canGoStep3}
                  className="flex items-center gap-1 px-4 py-2 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {t('action.nextStep', { ns: 'common' })} <ChevronRight className="w-4 h-4" />
                </button>
              ) : (
                <button
                  onClick={handleSubmit}
                  disabled={submitting}
                  className="flex items-center gap-1 px-5 py-2 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
                >
                  {submitting ? <Loader2 className="w-4 h-4 animate-spin" /> : <Download className="w-4 h-4" />}
                  {t('wizard.startDeploy')}
                </button>
              )}
            </>
          ) : (
            <div className="w-full flex justify-end">
              <button
                onClick={() => { onClose(); if (provision?.status === 'completed' && onSuccess) onSuccess() }}
                className="px-4 py-2 text-sm bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200"
              >
                {t('action.close', { ns: 'common' })}
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
