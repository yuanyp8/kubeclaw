import type { ReactNode } from 'react'

import { SessionStoreProvider } from '../store/session-store'
import { ThemeProvider } from './theme'

export function AppProviders({ children }: { children: ReactNode }) {
  return (
    <ThemeProvider>
      <SessionStoreProvider>{children}</SessionStoreProvider>
    </ThemeProvider>
  )
}
