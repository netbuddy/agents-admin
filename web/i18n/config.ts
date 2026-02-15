import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'

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
export const LANG_STORAGE_KEY = 'i18n-lang'

export const namespaces = [
  'common', 'tasks', 'agents', 'accounts', 'instances',
  'monitor', 'runners', 'nodes', 'proxies', 'settings', 'auth',
] as const

// 初始化时使用固定语言（defaultLng），避免 SSR / 客户端 hydration 不一致。
// Node.js 22+ 新增全局 navigator，导致 i18next-browser-languagedetector
// 在 SSR 时检测到系统 locale（如 en-US），而浏览器检测到用户 locale（如 zh-CN），
// 引发 React hydration mismatch。
// 语言检测改由 I18nProvider 在 useEffect 中完成（hydration 后执行）。
i18n
  .use(initReactI18next)
  .init({
    lng: defaultLng,
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
  })

export default i18n
