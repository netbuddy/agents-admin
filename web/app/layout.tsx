import type { Metadata } from 'next'
import './globals.css'

export const metadata: Metadata = {
  title: 'Agent Kanban',
  description: 'AI Agent 任务编排与可观测平台',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="zh">
      <body className="bg-gray-50 text-gray-900">{children}</body>
    </html>
  )
}
