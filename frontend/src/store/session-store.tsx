/* eslint-disable react-refresh/only-export-components */
import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'

import { clearSession, loadSession, saveSession } from '../lib/api'
import type { Session } from '../types'

type SessionStoreValue = {
  session: Session | null
  setSession: (nextSession: Session | null) => void
}

const SessionStoreContext = createContext<SessionStoreValue | undefined>(undefined)

export function SessionStoreProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<Session | null>(() => loadSession())

  useEffect(() => {
    if (session) {
      saveSession(session)
      return
    }

    clearSession()
  }, [session])

  const value = useMemo<SessionStoreValue>(() => ({ session, setSession }), [session])

  return <SessionStoreContext.Provider value={value}>{children}</SessionStoreContext.Provider>
}

export function useSessionStore() {
  const value = useContext(SessionStoreContext)
  if (!value) {
    throw new Error('SessionStoreProvider is missing')
  }
  return value
}
