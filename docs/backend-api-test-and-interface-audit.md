# Backend API Test And Interface Audit

## Goal

This document records the current backend test focus, how to run the tests, the route permission model, and the current API completeness status.

## How To Run

From [backend](C:/Users/admin/Desktop/kubeclaw/backend):

```powershell
$env:GOCACHE='C:\Users\admin\Desktop\kubeclaw\backend\.gocache'
go test ./internal/application/agent
go test ./internal/httpapi/...
go test ./internal/config
go test ./... -run TestDoesNotExist
```

Notes:

- `GOCACHE` is redirected into the workspace because the default machine cache path may be permission-restricted.
- `go test ./... -run TestDoesNotExist` is used here as a fast compile sweep for all packages.

## Test Coverage Added

### Agent

Files:

- [service_test.go](C:/Users/admin/Desktop/kubeclaw/backend/internal/application/agent/service_test.go)

Coverage:

- Configured K8s MCP can win intent routing for resource queries.
- Skill can delegate to the builtin executor and return a real result.

### HTTP Auth Middleware

Files:

- [auth_test.go](C:/Users/admin/Desktop/kubeclaw/backend/internal/httpapi/middleware/auth_test.go)

Coverage:

- `Bearer` token extraction.
- Missing token returns `401`.
- Role mismatch returns `403`.
- Admin token can pass `RequireRoles(admin)`.

### User Handler

Files:

- [user_test.go](C:/Users/admin/Desktop/kubeclaw/backend/internal/httpapi/handlers/user_test.go)

Coverage:

- `admin` uses global user list.
- `cluster_admin` uses tenant-scoped user list.
- regular `user` is rejected from the user list endpoint.

### Stub Contract

Files:

- [stub_test.go](C:/Users/admin/Desktop/kubeclaw/backend/internal/httpapi/handlers/stub_test.go)

Coverage:

- Unimplemented endpoints return `501 NOT_IMPLEMENTED`.
- Stub payload includes `module`, `action`, `method`, and `path`.

### Router Permission Matrix

Files:

- [router_test.go](C:/Users/admin/Desktop/kubeclaw/backend/internal/httpapi/router_test.go)

Coverage:

- Protected route requires authentication.
- `cluster_admin` can access protected user list.
- `cluster_admin` cannot access admin-only user creation route.
- `cluster_admin` can access ops skill list.
- regular `user` cannot access ops skill list.
- `/api/skills/:id/execute` is still a routed stub and returns `501`.

### Config

Files:

- [config_test.go](C:/Users/admin/Desktop/kubeclaw/backend/internal/config/config_test.go)

Coverage:

- YAML config can be loaded from `CONFIG_FILE`.
- Environment variables override YAML values.
- Invalid env values fall back to the YAML/default value instead of crashing.

## Permission Model

### Public

- `POST /api/auth/login`
- `POST /api/auth/refresh`

### Authenticated

- `POST /api/auth/logout`
- `GET /api/users/me`
- `GET /api/users`
- `PUT /api/users/me`
- `GET /api/logs`
- `GET /api/logs/scopes`
- `POST /api/logs/client`
- `GET|POST|DELETE /api/agent/...`

Important detail:

- `GET /api/users` is not just "authenticated". It is further restricted inside the handler:
- `admin`: full list
- `cluster_admin`: tenant-scoped list
- `user/readonly`: forbidden

### Admin Only

- `/api/users` create/get/update/delete by id
- `/api/models/...`
- `/api/mcp/servers/...`
- `/api/security/...`
- `/api/tenants/...`
- `/api/audit/...`

### Admin Or Cluster Admin

- `/api/clusters/...`
- `/api/skills/...`
- `/api/teams/...`
- `/api/knowledge/...`
- `/api/tasks/...`

## Interface Completeness Audit

### Implemented

- Health: `/healthz`, `/readyz`
- Auth: login, refresh, logout
- Users: me, list, CRUD
- Models: CRUD, test, set default
- MCP servers: CRUD
- Skills: CRUD, list
- Security: IP whitelist, sensitive words, sensitive field rules CRUD
- Clusters: list/detail/update/delete, validate, overview, permissions, namespaces, resources, events
- Agent: session, message, run events, stream, approvals
- Logs: list scopes, list logs, client log ingest

### Routed But Not Implemented

These currently return `501 NOT_IMPLEMENTED` through the common stub handler:

- `POST /api/skills/:id/execute`
- `POST /api/clusters/:id/resources`
- `DELETE /api/clusters/:id/resources/:type/:name`
- `GET /api/clusters/:id/exec`
- `POST /api/knowledge`
- `GET /api/knowledge`
- `GET /api/knowledge/:id`
- `DELETE /api/knowledge/:id`
- `POST /api/knowledge/search`
- `GET /api/tasks`
- `POST /api/tasks`
- `GET /api/tasks/:id`
- `PUT /api/tasks/:id`
- `DELETE /api/tasks/:id`
- `POST /api/tasks/:id/run`
- `GET /api/tasks/:id/history`

## Current Risks And Gaps

- The route-level permission model is now tested, but most business handlers still do not have request/response schema tests.
- Stubbed endpoints are clearly routable, but there is no generated OpenAPI document yet.
- The current agent-side MCP execution is an adapter layer, not a full MCP protocol client.
- Some user-facing Chinese strings in older backend files were historically garbled; the recently touched agent file was normalized enough to keep tests and compilation stable, but a broader encoding cleanup would still be worthwhile.

## Recommended Next Steps

1. Add request validation tests for `cluster`, `model`, `security`, and `tenant` handlers.
2. Add an OpenAPI or Markdown-generated endpoint inventory from [router.go](C:/Users/admin/Desktop/kubeclaw/backend/internal/httpapi/router.go).
3. Replace stubbed skill execution with a real handler once the skill runtime contract is finalized.
4. Add end-to-end database-backed integration tests for auth, user tenancy, and cluster permission sharing.
