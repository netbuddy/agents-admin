'use client'

import { AuthProvider } from '@/lib/auth'
import I18nProvider from '@/i18n/provider'

export default function Providers({ children }: { children: React.ReactNode }) {
  return (
    <I18nProvider>
      <AuthProvider>{children}</AuthProvider>
    </I18nProvider>
  )
}
