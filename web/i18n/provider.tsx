'use client'

import { useEffect } from 'react'
import { I18nextProvider, useTranslation } from 'react-i18next'
import i18n from './config'

function LangUpdater() {
  const { i18n: i18nInstance } = useTranslation()

  useEffect(() => {
    // 初始化时设置 html lang
    document.documentElement.lang = i18nInstance.language?.startsWith('zh') ? 'zh' : 'en'

    const handleLanguageChanged = (lng: string) => {
      document.documentElement.lang = lng.startsWith('zh') ? 'zh' : 'en'
    }

    i18nInstance.on('languageChanged', handleLanguageChanged)
    return () => {
      i18nInstance.off('languageChanged', handleLanguageChanged)
    }
  }, [i18nInstance])

  return null
}

export default function I18nProvider({ children }: { children: React.ReactNode }) {
  return (
    <I18nextProvider i18n={i18n}>
      <LangUpdater />
      {children}
    </I18nextProvider>
  )
}
