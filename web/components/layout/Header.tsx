'use client'

import { Bell, Menu, RefreshCw, User } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import LanguageSwitcher from '@/components/LanguageSwitcher'

interface HeaderProps {
  title?: string
  onRefresh?: () => void
  loading?: boolean
  onMenuToggle?: () => void
}

export default function Header({ title, onRefresh, loading, onMenuToggle }: HeaderProps) {
  const [showUserMenu, setShowUserMenu] = useState(false)
  const { t } = useTranslation()

  return (
    <header className="h-14 bg-white border-b border-gray-200 flex items-center justify-between px-3 sm:px-6">
      {/* Left: Menu + Page title */}
      <div className="flex items-center gap-2 sm:gap-4 min-w-0">
        {/* 移动端汉堡菜单按钮 */}
        <button
          onClick={onMenuToggle}
          className="p-2 rounded-lg hover:bg-gray-100 lg:hidden flex-shrink-0"
          aria-label={t('header.openMenu')}
        >
          <Menu className="w-5 h-5" />
        </button>
        {title && <h1 className="text-base sm:text-lg font-semibold truncate">{title}</h1>}
      </div>

      {/* Right: Actions */}
      <div className="flex items-center gap-1 sm:gap-2 flex-shrink-0">
        {onRefresh && (
          <button
            onClick={onRefresh}
            disabled={loading}
            className="p-2 rounded-lg hover:bg-gray-100 disabled:opacity-50"
            title={t('header.refresh')}
          >
            <RefreshCw className={`w-5 h-5 ${loading ? 'animate-spin' : ''}`} />
          </button>
        )}

        <button className="p-2 rounded-lg hover:bg-gray-100 relative hidden sm:block" title={t('header.notifications')}>
          <Bell className="w-5 h-5" />
          <span className="absolute top-1 right-1 w-2 h-2 bg-red-500 rounded-full" />
        </button>

        <div className="relative">
          <button
            onClick={() => setShowUserMenu(!showUserMenu)}
            className="flex items-center gap-2 p-2 rounded-lg hover:bg-gray-100"
          >
            <div className="w-8 h-8 bg-blue-600 rounded-full flex items-center justify-center">
              <User className="w-4 h-4 text-white" />
            </div>
          </button>

          {showUserMenu && (
            <>
              <div className="fixed inset-0 z-40" onClick={() => setShowUserMenu(false)} />
              <div className="absolute right-0 mt-2 w-48 bg-white rounded-lg shadow-lg border py-1 z-50">
                <div className="px-4 py-2 border-b">
                  <p className="text-sm font-medium">Admin</p>
                  <p className="text-xs text-gray-500">admin@localhost</p>
                </div>
                <button className="w-full text-left px-4 py-2 text-sm hover:bg-gray-100">
                  {t('header.profileSettings')}
                </button>
                <button className="w-full text-left px-4 py-2 text-sm text-red-600 hover:bg-gray-100">
                  {t('header.logout')}
                </button>
              </div>
            </>
          )}
        </div>

        <LanguageSwitcher />
      </div>
    </header>
  )
}
