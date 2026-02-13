'use client'

import { useCallback } from 'react'
import { useTranslation } from 'react-i18next'

export function useFormatDate() {
  const { i18n, t } = useTranslation()
  const locale = i18n.language?.startsWith('zh') ? 'zh-CN' : 'en-US'

  const formatDateTime = useCallback(
    (date: string | Date) => new Date(date).toLocaleString(locale),
    [locale]
  )

  const formatDate = useCallback(
    (date: string | Date) => new Date(date).toLocaleDateString(locale),
    [locale]
  )

  const formatTime = useCallback(
    (date: string | Date) => new Date(date).toLocaleTimeString(locale),
    [locale]
  )

  const formatShortDateTime = useCallback(
    (date: string | Date) =>
      new Date(date).toLocaleString(locale, {
        month: 'numeric',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      }),
    [locale]
  )

  const formatShortTime = useCallback(
    (date: string | Date) =>
      new Date(date).toLocaleString(locale, {
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
      }),
    [locale]
  )

  const formatRelative = useCallback(
    (date: string | Date) => {
      const now = Date.now()
      const then = new Date(date).getTime()
      const diffMs = now - then

      if (diffMs < 60_000) return t('time.justNow')
      if (diffMs < 3_600_000) return t('time.minutesAgo', { count: Math.floor(diffMs / 60_000) })
      if (diffMs < 86_400_000) return t('time.hoursAgo', { count: Math.floor(diffMs / 3_600_000) })
      if (diffMs < 86_400_000 * 30) return t('time.daysAgo', { count: Math.floor(diffMs / 86_400_000) })

      return formatDateTime(date)
    },
    [t, formatDateTime]
  )

  return { formatDateTime, formatDate, formatTime, formatShortDateTime, formatShortTime, formatRelative }
}
