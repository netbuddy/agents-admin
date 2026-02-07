'use client'

import { useState } from 'react'
import Sidebar from './Sidebar'
import Header from './Header'

interface AdminLayoutProps {
  children: React.ReactNode
  title?: string
  onRefresh?: () => void
  loading?: boolean
}

export default function AdminLayout({
  children,
  title,
  onRefresh,
  loading,
}: AdminLayoutProps) {
  const [mobileOpen, setMobileOpen] = useState(false)
  const [collapsed, setCollapsed] = useState(false)

  return (
    <div className="flex h-screen bg-gray-50 overflow-hidden">
      {/* Sidebar */}
      <Sidebar
        mobileOpen={mobileOpen}
        onMobileClose={() => setMobileOpen(false)}
        collapsed={collapsed}
        onCollapsedChange={setCollapsed}
      />

      {/* Main content area */}
      <div className="flex-1 flex flex-col overflow-hidden min-w-0">
        {/* Header */}
        <Header
          title={title}
          onRefresh={onRefresh}
          loading={loading}
          onMenuToggle={() => setMobileOpen(true)}
        />

        {/* Content */}
        <main className="flex-1 overflow-auto p-3 sm:p-4 md:p-6 touch-scroll">{children}</main>
      </div>
    </div>
  )
}
