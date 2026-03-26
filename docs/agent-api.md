# Agent API

## 1. Session APIs

Create session:

- `POST /api/agent/sessions`

List sessions:

- `GET /api/agent/sessions`

Get one session:

- `GET /api/agent/sessions/:id`

Delete session:

- `DELETE /api/agent/sessions/:id`

Request body for session creation:

```json
{
  "title": "prod cluster inspection",
  "modelId": 1,
  "clusterId": 1,
  "namespace": "default"
}
```

## 2. Message APIs

List messages:

- `GET /api/agent/sessions/:id/messages`

Send message:

- `POST /api/agent/sessions/:id/messages`

Request body:

```json
{
  "content": "list pods in the current namespace"
}
```

Response body:

```json
{
  "sessionId": 1,
  "userMessageId": 12,
  "runId": 44,
  "status": "queued"
}
```

## 3. Run APIs

List stored events:

- `GET /api/agent/runs/:id/events`

Open SSE stream:

- `GET /api/agent/runs/:id/stream`

SSE payload example:

```json
{
  "id": 91,
  "runId": 44,
  "sessionId": 1,
  "eventType": "planning",
  "role": "orchestrator",
  "status": "running",
  "message": "execution plan created",
  "payload": {
    "intent": "list_resources",
    "tool": "cluster.list_resources",
    "risk": false
  },
  "requestId": "req-demo",
  "createdAt": "2026-03-26T12:00:00Z",
  "updatedAt": "2026-03-26T12:00:00Z"
}
```

## 4. Approval APIs

Approve:

- `POST /api/agent/approvals/:id/approve`

Reject:

- `POST /api/agent/approvals/:id/reject`

The first implementation does not require a request body.

## 5. Typical end-to-end sequence

1. Create session
2. Send message
3. Receive `runId`
4. Open SSE stream
5. Watch `planning` and `tool_*` events
6. If `approval_required` appears, call approve or reject
7. Refresh messages after `message_done` or `turn_end`

## 6. Auth model

Every current agent endpoint requires:

- `Authorization: Bearer <access token>`

The frontend uses a custom `fetch`-based SSE reader because native `EventSource` cannot attach the required Bearer header cleanly.
