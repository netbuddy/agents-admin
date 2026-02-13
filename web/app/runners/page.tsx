'use client'

import { useEffect, useState, useCallback } from 'react'
import { RefreshCw, Plus, Trash2, Terminal, CheckCircle, XCircle, Clock, Server, X } from 'lucide-react'
import Link from 'next/link'
import { useTranslation } from 'react-i18next'

interface Runner {
  account: string
  container: string
  status: string
  logged_in: boolean
  created_at?: string
}

interface TerminalSession {
  id: string
  account: string
  url: string
  port: number
  status: string
}

export default function RunnersPage() {
  const { t } = useTranslation('runners')
  const [runners, setRunners] = useState<Runner[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [terminalSession, setTerminalSession] = useState<TerminalSession | null>(null)

  const fetchRunners = useCallback(async () => {
    try {
      const res = await fetch('/api/v1/runners')
      if (res.ok) {
        const data = await res.json()
        setRunners(data.runners || [])
      }
    } catch (err) {
      console.error('Failed to fetch runners:', err)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchRunners()
    const interval = setInterval(fetchRunners, 10000)
    return () => clearInterval(interval)
  }, [fetchRunners])

  const createRunner = async (account: string) => {
    try {
      const res = await fetch('/api/v1/runners', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ account }),
      })
      if (res.ok) {
        fetchRunners()
        setShowCreateModal(false)
      }
    } catch (err) {
      console.error('Failed to create runner:', err)
    }
  }

  const openTerminal = async (account: string) => {
    try {
      const res = await fetch(`/api/v1/runners/${encodeURIComponent(account)}/terminal`, {
        method: 'POST',
      })
      if (res.ok) {
        const data = await res.json()
        setTerminalSession(data)
      } else {
        alert(t('terminalError'))
      }
    } catch (err) {
      console.error('Failed to open terminal:', err)
      alert(t('terminalErrorShort'))
    }
  }

  const closeTerminal = async () => {
    if (terminalSession) {
      try {
        await fetch(`/api/v1/runners/${encodeURIComponent(terminalSession.account)}/terminal`, {
          method: 'DELETE',
        })
      } catch (err) {
        console.error('Failed to close terminal:', err)
      }
      setTerminalSession(null)
      fetchRunners()
    }
  }

  const deleteRunner = async (account: string, purge: boolean = false) => {
    if (!confirm(purge ? t('confirmDeletePurge', { account }) : t('confirmDelete', { account }))) return
    try {
      await fetch(`/api/v1/runners?account=${encodeURIComponent(account)}&purge=${purge}`, {
        method: 'DELETE',
      })
      fetchRunners()
    } catch (err) {
      console.error('Failed to delete runner:', err)
    }
  }

  const statusIcon = (runner: Runner) => {
    if (runner.status === 'running') {
      return runner.logged_in 
        ? <CheckCircle className="w-5 h-5 text-green-500" />
        : <Clock className="w-5 h-5 text-yellow-500" />
    }
    return <XCircle className="w-5 h-5 text-gray-400" />
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="bg-white border-b px-3 sm:px-6 py-3 sm:py-4">
        <div className="flex items-center justify-between gap-2">
          <div className="min-w-0">
            <h1 className="text-base sm:text-xl font-semibold flex items-center gap-2">
              <Server className="w-5 h-5 sm:w-6 sm:h-6 flex-shrink-0" />
              <span className="truncate">{t('title')}</span>
            </h1>
            <p className="text-xs sm:text-sm text-gray-500 hidden sm:block">
              {t('subtitle')}
            </p>
          </div>
          <div className="flex gap-1 sm:gap-2 flex-shrink-0">
            <Link href="/" className="hidden sm:flex px-4 py-2 border rounded-lg hover:bg-gray-100 text-sm">
              {t('backToBoard')}
            </Link>
            <button
              onClick={fetchRunners}
              className="p-2 sm:px-4 sm:py-2 border rounded-lg hover:bg-gray-100"
            >
              <RefreshCw className="w-4 h-4" />
            </button>
            <button
              onClick={() => setShowCreateModal(true)}
              className="flex items-center gap-1 sm:gap-2 px-3 sm:px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 text-sm"
            >
              <Plus className="w-4 h-4" />
              <span className="hidden sm:inline">{t('addAccount')}</span>
              <span className="sm:hidden">{t('add')}</span>
            </button>
          </div>
        </div>
      </header>

      <main className="p-3 sm:p-6">
        {loading ? (
          <div className="flex items-center justify-center h-64">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
          </div>
        ) : runners.length === 0 ? (
          <div className="bg-white rounded-lg border p-8 text-center">
            <Server className="w-12 h-12 mx-auto text-gray-400 mb-4" />
            <h3 className="text-lg font-medium mb-2">{t('noRunners')}</h3>
            <p className="text-gray-500 mb-4">{t('noRunnersHint')}</p>
            <button
              onClick={() => setShowCreateModal(true)}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
            >
              {t('addAccount')}
            </button>
          </div>
        ) : (
          <div className="grid gap-3 sm:gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3">
            {runners.map(runner => (
              <div key={runner.container} className="bg-white rounded-lg border p-4">
                <div className="flex items-start justify-between mb-3">
                  <div className="flex items-center gap-2">
                    {statusIcon(runner)}
                    <div>
                      <h3 className="font-medium">{runner.account}</h3>
                      <p className="text-xs text-gray-500">{runner.container}</p>
                    </div>
                  </div>
                  <button
                    onClick={() => deleteRunner(runner.account, true)}
                    className="p-1.5 text-red-500 hover:bg-red-50 rounded"
                    title={t('action.delete', { ns: 'common' })}
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>

                <div className="flex items-center gap-2 text-sm mb-3">
                  <span className={`px-2 py-0.5 rounded text-xs ${
                    runner.status === 'running' ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
                  }`}>
                    {runner.status === 'running' ? t('statusRunning') : t('statusStopped')}
                  </span>
                  <span className={`px-2 py-0.5 rounded text-xs ${
                    runner.logged_in ? 'bg-blue-100 text-blue-800' : 'bg-yellow-100 text-yellow-800'
                  }`}>
                    {runner.logged_in ? t('loggedIn') : t('notLoggedIn')}
                  </span>
                </div>

                <button
                  onClick={() => openTerminal(runner.account)}
                  className="w-full flex items-center justify-center gap-2 px-3 py-2 bg-blue-50 text-blue-700 rounded-lg hover:bg-blue-100"
                >
                  <Terminal className="w-4 h-4" />
                  {runner.logged_in ? t('openTerminal') : t('loginTerminal')}
                </button>

                {runner.created_at && (
                  <p className="text-xs text-gray-400 mt-2">
                    {t('createdAt')} {runner.created_at}
                  </p>
                )}
              </div>
            ))}
          </div>
        )}
      </main>

      {showCreateModal && (
        <CreateRunnerModal
          onClose={() => setShowCreateModal(false)}
          onCreate={createRunner}
        />
      )}

      {terminalSession && (
        <TerminalModal
          session={terminalSession}
          onClose={closeTerminal}
        />
      )}
    </div>
  )
}

function CreateRunnerModal({ onClose, onCreate }: { onClose: () => void, onCreate: (account: string) => void }) {
  const { t } = useTranslation('runners')
  const [account, setAccount] = useState('')

  return (
    <div className="fixed inset-0 bg-black/50 flex items-end sm:items-center justify-center z-50">
      <div className="bg-white rounded-t-2xl sm:rounded-lg w-full sm:max-w-md p-4 sm:p-6">
        <h2 className="text-lg font-semibold mb-4">{t('create.title')}</h2>
        <div className="mb-4">
          <label className="block text-sm font-medium mb-1">{t('create.accountLabel')}</label>
          <input
            type="text"
            value={account}
            onChange={e => setAccount(e.target.value)}
            className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            placeholder="user@example.com"
          />
          <p className="text-xs text-gray-500 mt-1">
            {t('create.accountHint')}
          </p>
        </div>
        <div className="flex justify-end gap-2">
          <button onClick={onClose} className="px-4 py-2 border rounded-lg hover:bg-gray-100">
            {t('action.cancel', { ns: 'common' })}
          </button>
          <button
            onClick={() => onCreate(account)}
            disabled={!account}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
          >
            {t('action.create', { ns: 'common' })}
          </button>
        </div>
      </div>
    </div>
  )
}

function TerminalModal({ session, onClose }: { session: TerminalSession, onClose: () => void }) {
  const { t } = useTranslation('runners')
  const host = typeof window !== 'undefined' ? window.location.hostname : 'localhost'
  const iframeUrl = `http://${host}:${session.port}/`

  return (
    <div className="fixed inset-0 bg-black/80 flex items-center justify-center z-50 p-2 sm:p-4">
      <div className="bg-gray-900 rounded-lg w-full max-w-4xl h-[85vh] sm:h-[600px] flex flex-col">
        <div className="flex items-center justify-between px-4 py-2 border-b border-gray-700">
          <div className="flex items-center gap-2 text-white">
            <Terminal className="w-5 h-5" />
            <span className="font-medium">{t('terminal.title')} - {session.account}</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-xs text-gray-400">
              {t('terminal.loginHint')}
            </span>
            <button
              onClick={onClose}
              className="p-1 text-gray-400 hover:text-white hover:bg-gray-700 rounded"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>
        <div className="flex-1 bg-black">
          <iframe
            src={iframeUrl}
            className="w-full h-full border-0"
            title="Terminal"
          />
        </div>
      </div>
    </div>
  )
}
