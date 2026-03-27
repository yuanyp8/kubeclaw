# Capability Layer Phase 1

## What Landed

The backend now has a first-class capability application layer:

- `C:\Users\admin\Desktop\kubeclaw\backend\internal\application\capability\service.go`
- `C:\Users\admin\Desktop\kubeclaw\backend\internal\application\capability\service_test.go`

The agent no longer owns capability source assembly by itself. It now consumes the shared capability service and still keeps the existing execution behavior.

There is also a new protected API:

- `GET /api/capabilities`
- `GET /api/capabilities/:ref`
- `GET /api/capabilities?audience=agent`
- `GET /api/capabilities?audience=http`
- `GET /api/capabilities?audience=mcp`
- `POST /api/capabilities/:ref/invoke`
- `POST /api/capabilities/:ref/request`

Handler:

- `C:\Users\admin\Desktop\kubeclaw\backend\internal\httpapi\handlers\capability.go`

## Key Modeling Rule

`MCP` is not treated as a one-to-one copy of every low-level HTTP API.

Instead, the new model separates capabilities by two dimensions:

- `level`
  - `primitive`: low-level building blocks such as list/get/delete/scale/apply
  - `workflow`: higher-level compositions or business abstractions, including external MCP-backed capabilities and skills
  - `catalog`: inventory-style capabilities such as listing models, skills, and MCP servers
- `audiences`
  - `agent`: available to the agent planner/runtime
  - `http`: safe to expose as direct web/backend API surface
  - `mcp`: curated set intended to be exported as MCP tools later

Current defaults:

- built-in read-only K8s capabilities: `primitive`, audiences `agent + http`
- built-in mutating K8s capabilities: `primitive`, audience `agent`
- built-in catalog capabilities: `catalog`, audiences `agent + http`
- skill capabilities: `workflow`, audience `agent`
- external MCP capabilities: `workflow`, audience `agent`

This keeps low-level APIs and high-level MCP workflows from collapsing into the same surface, and it avoids exposing unsafe cluster mutations through the new generic HTTP invoke endpoint before approval semantics are unified there.

The current entry rule is:

- safe read-only capability: direct `invoke`
- mutating capability: `request`, which is routed into the existing agent approval pipeline

## Why This Matters

This is the first step toward a shared execution substrate for:

- Web pages
- Agent chat
- future MCP export
- future workflow/composite execution

The next natural steps are:

1. add invocation records and a `POST /api/capabilities/:id/invoke` flow
2. move more agent routing/matching logic into the capability layer
3. define a curated `audience=mcp` export instead of mirroring every primitive API
