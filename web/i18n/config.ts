import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import LanguageDetector from 'i18next-browser-languagedetector'

// 静态引入所有翻译 JSON（打包进 JS bundle，无额外请求）
import zhCommon from './locales/zh/common.json'
import zhTasks from './locales/zh/tasks.json'
import zhAgents from './locales/zh/agents.json'
import zhAccounts from './locales/zh/accounts.json'
import zhInstances from './locales/zh/instances.json'
import zhMonitor from './locales/zh/monitor.json'
import zhRunners from './locales/zh/runners.json'
import zhNodes from './locales/zh/nodes.json'
import zhProxies from './locales/zh/proxies.json'
import zhSettings from './locales/zh/settings.json'
import zhAuth from './locales/zh/auth.json'

import enCommon from './locales/en/common.json'
import enTasks from './locales/en/tasks.json'
import enAgents from './locales/en/agents.json'
import enAccounts from './locales/en/accounts.json'
import enInstances from './locales/en/instances.json'
import enMonitor from './locales/en/monitor.json'
import enRunners from './locales/en/runners.json'
import enNodes from './locales/en/nodes.json'
import enProxies from './locales/en/proxies.json'
import enSettings from './locales/en/settings.json'
import enAuth from './locales/en/auth.json'

export const supportedLngs = ['zh', 'en'] as const
export type SupportedLng = (typeof supportedLngs)[number]
export const defaultLng: SupportedLng = 'zh'

export const namespaces = [
  'common', 'tasks', 'agents', 'accounts', 'instances',
  'monitor', 'runners', 'nodes', 'proxies', 'settings', 'auth',
] as const

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      zh: {
        common: zhCommon,
        tasks: zhTasks,
        agents: zhAgents,
        accounts: zhAccounts,
        instances: zhInstances,
        monitor: zhMonitor,
        runners: zhRunners,
        nodes: zhNodes,
        proxies: zhProxies,
        settings: zhSettings,
        auth: zhAuth,
      },
      en: {
        common: enCommon,
        tasks: enTasks,
        agents: enAgents,
        accounts: enAccounts,
        instances: enInstances,
        monitor: enMonitor,
        runners: enRunners,
        nodes: enNodes,
        proxies: enProxies,
        settings: enSettings,
        auth: enAuth,
      },
    },
    fallbackLng: defaultLng,
    supportedLngs,
    defaultNS: 'common',
    ns: namespaces as unknown as string[],
    interpolation: {
      escapeValue: false, // React 已自动转义
    },
    detection: {
      order: ['localStorage', 'navigator'],
      lookupLocalStorage: 'i18n-lang',
      caches: ['localStorage'],
    },
  })

export default i18n
