# Frontend Development Guide

## 1. Stack and scope

The frontend is built with:

- Vite
- React 19
- TypeScript
- React Router

The UI is now a fixed workspace application:

- left navigation stays fixed
- top header stays fixed
- only the content area scrolls
- light and dark themes are supported
- user-facing labels are localized in Chinese

Primary routes:

- `/login`
- `/dashboard`
- `/agent`
- `/clusters`
- `/cluster-settings`
- `/models`
- `/security`
- `/audit`
- `/logs`
- `/users`
- `/teams`
- `/tenants`
- `/skills`
- `/mcp`

## 2. Source structure

Top-level folders:

- `frontend/src/app`
- `frontend/src/components`
- `frontend/src/features`
- `frontend/src/lib`
- `frontend/src/store`

Feature folders:

- `frontend/src/features/auth`
- `frontend/src/features/dashboard`
- `frontend/src/features/agent`
- `frontend/src/features/clusters`
- `frontend/src/features/models`
- `frontend/src/features/logs`
- `frontend/src/features/security`
- `frontend/src/features/audit`
- `frontend/src/features/users`
- `frontend/src/features/teams`
- `frontend/src/features/tenants`
- `frontend/src/features/extensions`

## 3. Bootstrap flow

Key files:

- `frontend/src/main.tsx`
- `frontend/src/App.tsx`
- `frontend/src/app/AppProviders.tsx`
- `frontend/src/app/AppRouter.tsx`
- `frontend/src/app/AppErrorBoundary.tsx`

Startup order:

1. `main.tsx` mounts React.
2. `App.tsx` builds the provider tree.
3. `ThemeProvider` restores the saved theme mode.
4. `SessionStoreProvider` restores the login session.
5. `AppRouter` redirects guests to `/login`.
6. Logged-in users enter the workspace shell.

## 4. Theme and layout

Key files:

- `frontend/src/app/theme.tsx`
- `frontend/src/components/layout/AppShell.tsx`
- `frontend/src/index.css`

Theme behavior:

- default mode is `system`
- users can switch to `light` or `dark`
- the selected mode is stored in local storage
- the resolved theme is written to `document.documentElement.dataset.theme`

Layout behavior:

- the app now uses a KubeSphere-like shell with a full-width top header plus a left workspace menu
- the sidebar uses grouped navigation
- the top bar shows quick actions, theme switcher, and identity card
- the content panel is the only scrollable region on desktop
- the compact brand area keeps the app looking like a management console
- navigation items are filtered by the current user role

Navigation groups:

- `工作台`
- `资源中心`
- `治理审计`
- `扩展能力`
- `身份组织`

## 5. API and session model

Key files:

- `frontend/src/store/session-store.tsx`
- `frontend/src/lib/api.ts`

Stored session data:

- current user
- access token
- refresh token

API client responsibilities:

- base URL resolution
- Bearer token injection
- standard response envelope parsing
- request error conversion
- SSE reading for agent runs
- plain text stream reading for pod logs

The frontend still uses page-local React state for most CRUD pages because the product surface is still evolving quickly.

## 6. Dashboard page

Key file:

- `frontend/src/features/dashboard/DashboardPage.tsx`

Purpose:

- provide a KubeSphere-like first screen for cluster operations
- surface the most useful cluster health information before entering the detailed cluster workspace

Current behaviors:

- select a cluster at the top
- load cluster overview metrics
- show namespace, node, pod, deployment, and service counts
- highlight problem pods first
- list deployment health summaries
- show recent cluster events
- link to the cluster workspace

## 7. Agent page

Key file:

- `frontend/src/features/agent/AgentPage.tsx`

Three-column layout:

- left: session list
- center: conversation thread
- right: execution timeline, approvals, and tool events

Current behaviors:

- create sessions with cluster and model context
- send messages
- open SSE streams for runs
- render planning, tool steps, approvals, and final states
- show friendly error states when backend runs fail

## 8. Cluster workspace page

Key file:

- `frontend/src/features/clusters/ClustersPage.tsx`

Purpose:

- focus on cluster observation and runtime operations
- keep resource browsing, log troubleshooting, and approval-driven changes in one workspace

Current behaviors:

- select a cluster from the top action area
- validate cluster connectivity
- load overview summary pills
- switch namespace
- switch resource type
- browse Pod, Deployment, Service, ConfigMap, and Secret resources
- inspect a single resource object
- stream pod logs directly in the page
- highlight log lines by severity-like keywords
- show recent namespace-scoped events
- request delete approval
- request Deployment scale approval
- request Deployment restart approval
- request YAML apply approval
- watch approval and execution progress through the same run event stream used by the agent

## 9. Cluster settings page

Key file:

- `frontend/src/features/clusters/ClusterSettingsPage.tsx`

Purpose:

- manage cluster create, update, delete, and connectivity validation separately from the runtime workspace

Current behaviors:

- list configured clusters
- create cluster configuration
- edit cluster name, endpoint, auth type, environment, status, visibility, and kubeconfig
- delete cluster configuration
- validate a saved cluster connection

## 10. Models page

Key file:

- `frontend/src/features/models/ModelsPage.tsx`

Current behaviors:

- group models by provider
- create and update models
- test connectivity
- set one default model
- show masked key information
- preload example OpenAI-compatible endpoints
- validate before save when creating a new model
- save directly when editing an existing model, so edits do not hang on forced retest
- preserve the existing API key when the API key field is left empty during edit

## 11. Logs page

Key file:

- `frontend/src/features/logs/LogsPage.tsx`

Current behaviors:

- tabs for runtime scopes
- cursor-based polling
- merge and deduplicate new entries
- show structured fields
- show audit rows as mutation-oriented records instead of raw endpoint mirrors

## 12. Client logging

Key file:

- `frontend/src/lib/client-logger.ts`

Behavior:

- development mode writes to console and can upload
- production mode uploads only `warn` and `error`
- API failures are recorded
- agent SSE disconnects are recorded
- route changes are recorded as low-cost events
- React tree crashes are captured by the error boundary

## 13. Identity pages

Relationship-aware pages expose:

- users with tenant summary and team memberships
- teams with tenant summary and member count
- tenants with user and team counts

Current teams page behaviors:

- team member removal refreshes the member list immediately after the API succeeds
- cluster admins can view and remove members
- platform admins can additionally load the user and tenant pickers for member creation
- team member management directly inside the team page

Admin-only pages fail closed in the UI:

- unauthorized users do not see related navigation items
- if a route is opened directly, the page renders a friendly no-permission state instead of a raw request error
- users page also degrades safely when tenant loading fails, so one secondary request does not break the whole page
