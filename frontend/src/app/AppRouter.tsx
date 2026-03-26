import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'

import { AppErrorBoundary } from './AppErrorBoundary'
import { useSessionStore } from '../store/session-store'
import { AppShell } from '../components/layout/AppShell'
import { LoginPage } from '../features/auth/LoginPage'
import { DashboardPage } from '../features/dashboard/DashboardPage'
import { AgentPage } from '../features/agent/AgentPage'
import { ClustersPage } from '../features/clusters/ClustersPage'
import { ClusterSettingsPage } from '../features/clusters/ClusterSettingsPage'
import { LogsPage } from '../features/logs/LogsPage'
import { ModelsPage } from '../features/models/ModelsPage'
import { SecurityPage } from '../features/security/SecurityPage'
import { UsersPage } from '../features/users/UsersPage'
import { TeamsPage } from '../features/teams/TeamsPage'
import { TenantsPage } from '../features/tenants/TenantsPage'
import { SkillsPage } from '../features/extensions/SkillsPage'
import { MCPPage } from '../features/extensions/MCPPage'
import { AuditPage } from '../features/audit/AuditPage'

function ProtectedLayout() {
  const { session } = useSessionStore()

  if (!session) {
    return <Navigate replace to="/login" />
  }

  return <AppShell />
}

export function AppRouter() {
  const { session, setSession } = useSessionStore()

  return (
    <BrowserRouter>
      <AppErrorBoundary>
        <Routes>
          <Route
            path="/login"
            element={session ? <Navigate replace to="/dashboard" /> : <LoginPage onLogin={setSession} />}
          />

          <Route path="/" element={<ProtectedLayout />}>
            <Route index element={<Navigate replace to="/dashboard" />} />
            <Route path="dashboard" element={<DashboardPage />} />
            <Route path="agent" element={<AgentPage />} />
            <Route path="clusters" element={<ClustersPage />} />
            <Route path="cluster-settings" element={<ClusterSettingsPage />} />
            <Route path="models" element={<ModelsPage />} />
            <Route path="security" element={<SecurityPage />} />
            <Route path="audit" element={<AuditPage />} />
            <Route path="logs" element={<LogsPage />} />
            <Route path="users" element={<UsersPage />} />
            <Route path="teams" element={<TeamsPage />} />
            <Route path="tenants" element={<TenantsPage />} />
            <Route path="skills" element={<SkillsPage />} />
            <Route path="mcp" element={<MCPPage />} />
          </Route>
        </Routes>
      </AppErrorBoundary>
    </BrowserRouter>
  )
}
