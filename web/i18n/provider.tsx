'use client'

import { useEffect } from 'react'
import { I18nextProvider, useTranslation } from 'react-i18next'
import i18n from './config'
import { LANG_STORAGE_KEY, supportedLngs, defaultLng } from './config'
import type { SupportedLng } from './config'

function LangUpdater() {
  const { i18n: i18nInstance } = useTranslation()

  useEffect(() => {
    // hydration 完成后检测用户语言偏好（避免 SSR 不一致）
    const stored = localStorage.getItem(LANG_STORAGE_KEY)
    if (stored && (supportedLngs as readonly string[]).includes(stored)) {
      if (i18nInstance.language !== stored) {
        i18nInstance.changeLanguage(stored)
      }
    } else {
      // 首次访问：从浏览器语言推断
      const browserLng: SupportedLng = navigator.language?.startsWith('zh') ? 'zh' : 'en'
      if (browserLng !== defaultLng) {
        i18nInstance.changeLanguage(browserLng)
      }
      localStorage.setItem(LANG_STORAGE_KEY, browserLng)
    }

    // 同步 html lang 属性
    document.documentElement.lang = i18nInstance.language?.startsWith('zh') ? 'zh' : 'en'

    const handleLanguageChanged = (lng: string) => {
      document.documentElement.lang = lng.startsWith('zh') ? 'zh' : 'en'
      localStorage.setItem(LANG_STORAGE_KEY, lng)
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
