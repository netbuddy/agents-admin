'use client'

import { Suspense } from 'react'
import { useSearchParams } from 'next/navigation'
import { useTranslation } from 'react-i18next'
import TaskDetailClient from './TaskDetailClient'

function TaskDetailContent() {
  const { t } = useTranslation('tasks')
  const searchParams = useSearchParams()
  const taskId = searchParams.get('id')

  if (!taskId) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <h2 className="text-xl font-semibold mb-2">{t('detail.missingId')}</h2>
          <a href="/" className="text-blue-600 hover:underline">{t('detail.goHome')}</a>
        </div>
      </div>
    )
  }

  return <TaskDetailClient taskId={taskId} />
}

export default function TaskDetailPage() {
  return (
    <Suspense fallback={
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
      </div>
    }>
      <TaskDetailContent />
    </Suspense>
  )
}
