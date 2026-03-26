# Agent Architecture

## 1. Goal

The first agent implementation is not a placeholder chat box.
It is a working orchestrator with specialist roles, persistent runs, live events, and approval handling.

## 2. Runtime roles

The runtime uses logical roles inside one backend process:

- `orchestrator`
- `k8s_expert`
- `skill_mcp_expert`
- `safety_reviewer`

There are no external sub-services for these roles yet.

## 3. Persistence

Session and message history:

- `chat_sessions`
- `chat_messages`

Execution metadata:

- `agent_runs`
- `agent_events`
- `approval_requests`
- `tool_executions`

## 4. Main execution flow

1. UI creates or selects an agent session.
2. UI posts a user message.
3. Backend stores the user message.
4. Backend creates an `agent_run`.
5. Orchestrator emits `turn_start`.
6. Orchestrator emits `planning`.
7. Runtime chooses one of three paths:
   - read-only tool execution
   - approval-required mutation
   - LLM answer
8. Runtime emits events and stores them.
9. UI receives live SSE updates.
10. Final assistant message is persisted.

## 5. Event types

Current event types:

- `turn_start`
- `planning`
- `agent_spawn`
- `agent_result`
- `tool_start`
- `tool_end`
- `approval_required`
- `message_delta`
- `message_done`
- `error`
- `turn_end`

## 6. Intent routing

Current routing is deterministic and keyword-driven.

Read-only examples:

- namespaces
- events
- pods
- deployments
- model list
- skill list
- MCP list

Risky examples:

- delete resource
- scale deployment
- restart deployment
- apply YAML

Everything else currently falls back to the OpenAI-compatible model adapter.

## 7. Safety reviewer

The first safety implementation is rule-first.

Inputs:

- sensitive words from the security module
- risky verb detection

Result:

- either continue execution
- or create an approval request and pause the run

## 8. Specialist execution

`k8s_expert` currently uses:

- cluster service
- Kubernetes gateway

`skill_mcp_expert` currently uses:

- skill service
- MCP service

`orchestrator` currently uses:

- model service
- OpenAI-compatible chat adapter

## 9. Streaming

Live events use:

- `GET /api/agent/runs/:id/stream`

The backend keeps a lightweight in-memory subscriber hub and also stores every event in MySQL.
This means the UI can both:

- replay history
- receive new events in real time

## 10. Future improvements

- semantic intent parsing instead of keyword routing
- token streaming from providers
- richer approval policy graph
- session memory and context packs
- stronger resource targeting and diff previews
