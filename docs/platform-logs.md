# Platform Logs

## 1. Goal

The platform log system gives the frontend a unified way to inspect runtime behavior without tailing local terminals.

## 2. Current sources

In-memory ring buffer:

- `runtime`
- `access`
- `sql`
- `agent`
- `client`

Persistent source:

- `audit`

## 3. Backend components

Important files:

- `backend/internal/logger/bus.go`
- `backend/internal/logger/logger.go`
- `backend/internal/application/logs/service.go`
- `backend/internal/httpapi/handlers/log.go`
- `backend/internal/httpapi/middleware/accesslog.go`
- `backend/internal/infrastructure/mysql/gorm_logger.go`

## 4. Entry shape

Every log entry exposed to the UI has:

- `id`
- `timestamp`
- `level`
- `scope`
- `message`
- `fields`
- `requestId`
- `runId`

## 5. Query contract

List scopes:

- `GET /api/logs/scopes`

Query entries:

- `GET /api/logs?scope=agent&cursor=0&limit=50`

Response shape:

```json
{
  "scope": "agent",
  "cursor": 0,
  "nextCursor": 120,
  "entries": []
}
```

## 6. Client-side logging

Client upload endpoint:

- `POST /api/logs/client`

Current frontend sources:

- API request failures
- route changes
- agent SSE disconnects
- React render crashes

## 7. Polling model

The current logs page uses:

- scope tabs
- cursor-based polling
- merge and dedupe on the client

This keeps the implementation simple while still making the platform observable.

## 8. Future improvements

- retention config for the ring buffer
- filter by request ID and run ID
- direct links from logs to agent runs
- optional WebSocket or SSE for live log tails
