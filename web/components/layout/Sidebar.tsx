'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import {
  LayoutDashboard,
  Users,
  Server,
  Network,
  Settings,
  ChevronLeft,
  ChevronRight,
  Activity,
  Globe,
  Bot,
  X,
  LogOut,
} from 'lucide-react'
import { useEffect } from 'react'
import { useAuth } from '@/lib/auth'
import { useTranslation } from 'react-i18next'

interface NavItem {
  name: string
  href: string
  icon: React.ComponentType<{ className?: string }>
  badge?: number
}

const navigationItems = [
  { key: 'nav.taskBoard', href: '/', icon: LayoutDashboard },
  { key: 'nav.monitor', href: '/monitor', icon: Activity },
  { key: 'nav.agents', href: '/agents', icon: Bot },
  { key: 'nav.accounts', href: '/accounts', icon: Users },
  { key: 'nav.nodes', href: '/nodes', icon: Network },
  { key: 'nav.proxies', href: '/proxies', icon: Globe },
  { key: 'nav.settings', href: '/settings', icon: Settings },
]

interface SidebarProps {
  mobileOpen?: boolean
  onMobileClose?: () => void
  collapsed?: boolean
  onCollapsedChange?: (collapsed: boolean) => void
}

export default function Sidebar({
  mobileOpen = false,
  onMobileClose,
  collapsed = false,
  onCollapsedChange,
}: SidebarProps) {
  const pathname = usePathname()
  const { user, logout } = useAuth()
  const { t } = useTranslation()

  const navigation: NavItem[] = navigationItems.map(item => ({
    name: t(item.key),
    href: item.href,
    icon: item.icon,
  }))

  // 路由变化时关闭移动端侧边栏
  useEffect(() => {
    onMobileClose?.()
  }, [pathname])

  const isActive = (href: string) => {
    if (href === '/') return pathname === '/'
    return pathname.startsWith(href)
  }

  const handleNavClick = () => {
    // 移动端点击导航后关闭侧边栏
    onMobileClose?.()
  }

  const sidebarContent = (
    <>
      {/* Logo */}
      <div className="flex h-14 items-center justify-between px-4 border-b border-gray-800">
        {(!collapsed || mobileOpen) && (
          <span className="font-semibold text-lg">Agent Kanban</span>
        )}
        {/* 移动端显示关闭按钮 */}
        <button
          onClick={() => {
            if (mobileOpen) {
              onMobileClose?.()
            } else {
              onCollapsedChange?.(!collapsed)
            }
          }}
          className="p-1.5 rounded hover:bg-gray-800 min-w-[32px] min-h-[32px] flex items-center justify-center"
        >
          {mobileOpen ? (
            <X className="w-5 h-5" />
          ) : collapsed ? (
            <ChevronRight className="w-5 h-5" />
          ) : (
            <ChevronLeft className="w-5 h-5" />
          )}
        </button>
      </div>

      {/* Navigation */}
      <nav className="flex-1 py-4 space-y-1 overflow-y-auto touch-scroll">
        {navigation.map((item) => {
          const Icon = item.icon
          const active = isActive(item.href)
          const showLabel = !collapsed || mobileOpen
          return (
            <Link
              key={item.name}
              href={item.href}
              onClick={handleNavClick}
              className={`flex items-center gap-3 px-4 py-3 md:py-2.5 mx-2 rounded-lg transition-colors ${
                active
                  ? 'bg-blue-600 text-white'
                  : 'text-gray-300 hover:bg-gray-800 hover:text-white'
              }`}
              title={!showLabel ? item.name : undefined}
            >
              <Icon className="w-5 h-5 flex-shrink-0" />
              {showLabel && <span>{item.name}</span>}
              {showLabel && item.badge !== undefined && (
                <span className="ml-auto bg-red-500 text-white text-xs px-2 py-0.5 rounded-full">
                  {item.badge}
                </span>
              )}
            </Link>
          )
        })}
      </nav>

      {/* Footer: User info + Logout */}
      <div className="border-t border-gray-800 p-3">
        {(!collapsed || mobileOpen) && user && (
          <div className="flex items-center justify-between">
            <div className="min-w-0">
              <p className="text-sm font-medium text-gray-200 truncate">{user.username}</p>
              <p className="text-xs text-gray-500 truncate">{user.email}</p>
            </div>
            <button
              onClick={logout}
              className="p-2 text-gray-400 hover:text-red-400 hover:bg-gray-800 rounded-lg transition-colors flex-shrink-0"
              title={t('header.logout')}
            >
              <LogOut className="w-4 h-4" />
            </button>
          </div>
        )}
        {collapsed && !mobileOpen && (
          <button
            onClick={logout}
            className="w-full flex items-center justify-center p-2 text-gray-400 hover:text-red-400 hover:bg-gray-800 rounded-lg"
            title={t('header.logout')}
          >
            <LogOut className="w-4 h-4" />
          </button>
        )}
      </div>
    </>
  )

  return (
    <>
      {/* 移动端遮罩层 */}
      {mobileOpen && (
        <div
          className="fixed inset-0 bg-black/60 z-40 lg:hidden"
          onClick={onMobileClose}
        />
      )}

      {/* 移动端抽屉式侧边栏 */}
      <aside
        className={`
          fixed inset-y-0 left-0 z-50 flex flex-col bg-gray-900 text-white
          w-64 transition-transform duration-300 ease-in-out
          lg:hidden
          ${mobileOpen ? 'translate-x-0' : '-translate-x-full'}
        `}
      >
        {sidebarContent}
      </aside>

      {/* 桌面端固定侧边栏 */}
      <aside
        className={`hidden lg:flex flex-col bg-gray-900 text-white transition-all duration-300 flex-shrink-0 ${
          collapsed ? 'w-16' : 'w-56'
        }`}
      >
        {sidebarContent}
      </aside>
    </>
  )
}
