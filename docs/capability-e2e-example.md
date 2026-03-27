# Capability E2E Example

## Scenario

Use one concrete example to understand the new shared capability layer from end to end:

**Goal:** view pods in cluster `1`, namespace `default`.

This example shows two entrypoints:

1. direct HTTP capability invocation
2. agent chat invocation

Both now converge on the same capability runtime.

---

## 1. Login

Request:

```http
POST /api/auth/login
Content-Type: application/json

{
  "login": "admin",
  "password": "your-password"
}
```

Read the access token from:

- `data.tokens.accessToken`

Use it in later requests:

```http
Authorization: Bearer <access-token>
```

Code path:

- [auth.go](C:/Users/admin/Desktop/kubeclaw/backend/internal/httpapi/handlers/auth.go)
- [service.go](C:/Users/admin/Desktop/kubeclaw/backend/internal/application/auth/service.go)

---

## 2. List HTTP-safe capabilities

Request:

```http
GET /api/capabilities?audience=http
Authorization: Bearer <access-token>
```

What you should notice:

- `builtin.cluster.resources` is present
- workflow-style MCP capabilities are not present
- mutating K8s capabilities are not present in the `http` audience

Why:

- `http` audience is currently for direct safe/backend-web capability calls
- MCP is treated as a higher-level workflow surface, not a mirror of every low-level API

Code path:

- [capability.go](C:/Users/admin/Desktop/kubeclaw/backend/internal/httpapi/handlers/capability.go)
- [router.go](C:/Users/admin/Desktop/kubeclaw/backend/internal/httpapi/router.go)
- [service.go](C:/Users/admin/Desktop/kubeclaw/backend/internal/application/capability/service.go)

---

## 3. Invoke the shared capability directly over HTTP

Request:

```http
POST /api/capabilities/builtin.cluster.resources/invoke
Authorization: Bearer <access-token>
Content-Type: application/json

{
  "clusterId": 1,
  "namespace": "default",
  "payload": {
    "type": "pods"
  }
}
```

Response shape:

```json
{
  "code": "OK",
  "message": "capability invoked",
  "data": {
    "reference": "builtin.cluster.resources",
    "result": "[ ... pods json ... ]"
  }
}
```

Actual backend chain:

1. HTTP enters [capability.go](C:/Users/admin/Desktop/kubeclaw/backend/internal/httpapi/handlers/capability.go)
2. handler builds `InvokeInput`
3. handler calls `capability.Service.Invoke(...)`
4. runtime resolves `builtin.cluster.resources`
5. runtime picks action `list_resources`
6. runtime calls `cluster.Service.ListResources(...)`
7. cluster service calls K8s gateway

Key files:

- [runtime.go](C:/Users/admin/Desktop/kubeclaw/backend/internal/application/capability/runtime.go)
- [service.go](C:/Users/admin/Desktop/kubeclaw/backend/internal/application/cluster/service.go)
- [gateway.go](C:/Users/admin/Desktop/kubeclaw/backend/internal/infrastructure/kubernetes/gateway.go)

---

## 4. Run the same intent through Agent Chat

First create a session:

```http
POST /api/agent/sessions
Authorization: Bearer <access-token>
Content-Type: application/json

{
  "title": "pods walkthrough",
  "clusterId": 1,
  "namespace": "default"
}
```

Then send a message:

```http
POST /api/agent/sessions/<sessionId>/messages
Authorization: Bearer <access-token>
Content-Type: application/json

{
  "content": "列出 default 命名空间的 pods"
}
```

Then observe:

```http
GET /api/agent/runs/<runId>/events
Authorization: Bearer <access-token>
```

What happens internally:

1. Agent receives user message
2. planner/heuristic determines `list_resources`
3. selected capability is `builtin.cluster.resources`
4. Agent emits planning and execution events
5. Agent no longer executes built-in logic directly here
6. Agent delegates to the shared capability runtime
7. shared capability runtime calls the same `cluster.Service.ListResources(...)`

Critical handoff point:

- [capabilities.go](C:/Users/admin/Desktop/kubeclaw/backend/internal/application/agent/capabilities.go)

The new shared execution line is:

- `Agent -> capability.Service.Invoke -> cluster.Service -> kubernetes.Gateway`

That is the important architectural change.

---

## 5. Request a mutating capability through the unified request entry

Direct HTTP invoke is intentionally blocked for mutating capabilities such as scale/delete/restart/apply.

Instead, use:

```http
POST /api/capabilities/builtin.cluster.scale/request
Authorization: Bearer <access-token>
Content-Type: application/json

{
  "clusterId": 1,
  "namespace": "default",
  "payload": {
    "name": "payments",
    "replicas": 5
  }
}
```

What happens internally:

1. handler resolves capability metadata
2. handler sees this capability is `requestMode=agent_approval`
3. handler converts payload into `ClusterActionRequestInput`
4. handler calls existing agent request pipeline
5. agent creates session + run
6. planner keeps the same capability/action semantics
7. run enters approval flow before mutation executes

This means the platform now has a consistent split:

- read-only capability: direct invoke
- mutating capability: request then approve

Key file:

- [capability.go](C:/Users/admin/Desktop/kubeclaw/backend/internal/httpapi/handlers/capability.go)

---

## 6. Why this example matters

This one example shows the intended future shape of the platform:

- Web/API direct use can call shared capabilities
- Agent chat can call the same shared capabilities
- capability registry decides what is exposed to which audience
- MCP stays at the workflow/composite layer instead of duplicating every primitive API

So from this point on, when we add:

- `POST /api/capabilities/.../invoke` for more safe primitives
- curated MCP export
- master/subagent orchestration

they can all sit on top of the same capability substrate instead of growing separate execution stacks.

---

## 7. Minimal PowerShell Walkthrough

```powershell
$base = "http://127.0.0.1:8080"

$login = Invoke-RestMethod -Method Post -Uri "$base/api/auth/login" -ContentType "application/json" -Body (@{
  login = "admin"
  password = "your-password"
} | ConvertTo-Json)

$token = $login.data.tokens.accessToken
$headers = @{ Authorization = "Bearer $token" }

Invoke-RestMethod -Method Get -Uri "$base/api/capabilities?audience=http" -Headers $headers

Invoke-RestMethod -Method Post -Uri "$base/api/capabilities/builtin.cluster.resources/invoke" -Headers $headers -ContentType "application/json" -Body (@{
  clusterId = 1
  namespace = "default"
  payload = @{
    type = "pods"
  }
} | ConvertTo-Json -Depth 6)

Invoke-RestMethod -Method Post -Uri "$base/api/capabilities/builtin.cluster.scale/request" -Headers $headers -ContentType "application/json" -Body (@{
  clusterId = 1
  namespace = "default"
  payload = @{
    name = "payments"
    replicas = 5
  }
} | ConvertTo-Json -Depth 6)
```

If this direct capability call succeeds, you already know the shared runtime is working.

Then you can test the same target through agent chat and compare the event flow.
