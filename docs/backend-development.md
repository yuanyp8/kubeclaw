# Backend Development Guide

## 1. Current scope

The backend is a Go modular monolith.

Current working capabilities:

- auth and JWT
- health and readiness
- users, teams, and tenants
- audit logs
- model registry and connectivity test
- platform log bus
- cluster registry
- Kubernetes validate, overview, list, get, events
- pod log streaming
- Kubernetes write actions used by the approval flow
- MCP server CRUD
- skill CRUD
- security rules
- agent sessions, messages, runs, events, approvals, and SSE

Deferred for later:

- knowledge base persistence
- scheduled task execution
- richer long-term chat memory
- pod exec terminal

## 2. Directory layout

Important directories:

- `backend/cmd/server`
- `backend/configs`
- `backend/internal/app`
- `backend/internal/application`
- `backend/internal/httpapi`
- `backend/internal/infrastructure`
- `backend/internal/logger`

Important application packages:

- `backend/internal/application/agent`
- `backend/internal/application/chat`
- `backend/internal/application/cluster`
- `backend/internal/application/logs`

Important infrastructure packages:

- `backend/internal/infrastructure/llm`
- `backend/internal/infrastructure/agentruntime`
- `backend/internal/infrastructure/kubernetes`
- `backend/internal/infrastructure/mysql`

## 3. Startup and dependency wiring

Entry:

- `backend/cmd/server/main.go`

Wiring:

- `backend/internal/app/app.go`

Startup sequence:

1. Load `configs/default.yaml`
2. Override with environment variables
3. Initialize zap logger and the in-memory log bus
4. Connect to MySQL
5. Auto migrate tables when enabled
6. Bootstrap the system tenant and admin user
7. Build repositories
8. Build services
9. Build middleware and handlers
10. Build the Gin router
11. Start the HTTP server

## 4. Config model

Default config file:

- `backend/configs/default.yaml`

Priority:

1. `CONFIG_FILE` if provided
2. otherwise `configs/default.yaml`
3. environment variables override YAML values

Important runtime variables:

- `HTTP_ADDR`
- `APP_ENV`
- `LOG_LEVEL`
- `LOG_ENCODING`
- `LOG_DEVELOPMENT`
- `JWT_SECRET`
- `DATA_SECRET`
- `MYSQL_HOST`
- `MYSQL_PORT`
- `MYSQL_USER`
- `MYSQL_PASSWORD`
- `MYSQL_DATABASE`

## 5. Logging architecture

Key files:

- `backend/internal/logger/logger.go`
- `backend/internal/logger/bus.go`
- `backend/internal/httpapi/middleware/accesslog.go`
- `backend/internal/infrastructure/mysql/gorm_logger.go`
- `backend/internal/application/logs/service.go`

Current outputs:

- stdout through zap
- in-memory ring buffer for the UI log page

Ring buffer scopes:

- `runtime`
- `access`
- `sql`
- `agent`
- `client`

Audit stays in MySQL and is converted to mutation-oriented rows on read.

## 6. Data model

Core schema definitions:

- `backend/internal/infrastructure/mysql/models.go`

Persistent tables used by agent runtime:

- `agent_runs`
- `agent_events`
- `approval_requests`

Reused tables:

- `chat_sessions`
- `chat_messages`
- `tool_executions`
- `audit_logs`

Relationship-rich API records now expose:

- user detail with tenant summary and team memberships
- team detail with tenant summary and member count
- tenant detail with user and team counts

## 7. HTTP architecture

Router:

- `backend/internal/httpapi/router.go`

Important API groups:

- `/api/logs`
- `/api/agent`
- `/api/clusters`

Model APIs:

- `POST /api/models/:id/test`
- `POST /api/models/:id/set-default`

Agent APIs:

- `POST /api/agent/sessions`
- `GET /api/agent/sessions`
- `GET /api/agent/sessions/:id`
- `GET /api/agent/sessions/:id/messages`
- `POST /api/agent/sessions/:id/messages`
- `DELETE /api/agent/sessions/:id`
- `GET /api/agent/runs/:id/events`
- `GET /api/agent/runs/:id/stream`
- `POST /api/agent/approvals/:id/approve`
- `POST /api/agent/approvals/:id/reject`

Cluster query APIs:

- `POST /api/clusters/:id/validate`
- `GET /api/clusters/:id/overview`
- `GET /api/clusters/:id/namespaces`
- `GET /api/clusters/:id/resources`
- `GET /api/clusters/:id/resources/:type/:name`
- `GET /api/clusters/:id/events`
- `GET /api/clusters/:id/pods/:name/logs`

Cluster action APIs:

- `POST /api/clusters/:id/actions/delete-resource`
- `POST /api/clusters/:id/actions/scale-deployment`
- `POST /api/clusters/:id/actions/restart-deployment`
- `POST /api/clusters/:id/actions/apply-yaml`

Logs APIs:

- `GET /api/logs`
- `GET /api/logs/scopes`
- `POST /api/logs/client`

## 8. Cluster service design

Key files:

- `backend/internal/application/cluster/service.go`
- `backend/internal/infrastructure/kubernetes/gateway.go`
- `backend/internal/httpapi/handlers/cluster.go`

Current cluster runtime flow:

1. Load the cluster connection from MySQL.
2. Build a Kubernetes client from kubeconfig or token-based fields.
3. For kubeconfig uploads, prefer the external API server when provided.
4. Preserve the original kubeconfig server name for TLS verification when needed.
5. Execute list, get, validate, overview, log, or action operations through the gateway.

Current overview data includes:

- namespace count
- node count and ready node count
- pod totals and problem pod summaries
- deployment totals and health summaries
- service count
- recent events

Current pod log streaming behavior:

- uses direct Kubernetes log streaming
- supports namespace, container, follow, tail lines, and since seconds
- returns plain text so the frontend can perform keyword highlighting

## 9. Agent runtime design

Key files:

- `backend/internal/application/agent/service.go`
- `backend/internal/application/chat/service.go`
- `backend/internal/infrastructure/mysql/repository/agent_repository.go`
- `backend/internal/infrastructure/mysql/repository/chat_repository.go`
- `backend/internal/infrastructure/agentruntime/stream_hub.go`
- `backend/internal/infrastructure/llm/openai_compatible.go`

Runtime roles:

- `orchestrator`
- `k8s_expert`
- `skill_mcp_expert`
- `safety_reviewer`

Current runtime flow:

1. User sends a message.
2. Backend stores the user message.
3. Backend creates an `agent_run`.
4. The orchestrator emits `turn_start` and `planning`.
5. The planner first tries a model-based route decision.
6. If planning output is unavailable, heuristics are used as fallback.
7. The run enters one of three paths:
   - read-only tool path
   - risky action requiring approval
   - direct LLM answer
8. Runtime milestones are stored as `agent_events`.
9. Events are pushed to the SSE hub.
10. Final assistant output is stored in `chat_messages`.

Current answer guards:

- the system prompt asks the model to answer in the user language
- hidden `<think>` output is stripped
- model failures are converted into user-friendly assistant messages

## 10. Safety and approval flow

Current deterministic review inputs:

- sensitive word rules
- risky operation keywords

Current risky operations:

- delete resource
- scale deployment
- restart deployment
- apply YAML

These actions can be entered through:

- natural language agent requests
- direct cluster action HTTP APIs

All risky actions are represented as approval requests before execution.

## 11. Access control

Important behaviors already implemented:

- agent sessions can only be read, listed, deleted, or streamed by their owner or an admin
- agent approvals can only be approved or rejected by the requester or an admin
- team resources are tenant-scoped for non-admin users

More detailed team, tenant, and resource authorization can be layered on top later without changing the overall service split.
