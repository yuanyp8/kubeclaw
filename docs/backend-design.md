# OpsBrain 后端设计

## 1. 设计目标

基于 README 的需求，后端需要同时承载四类核心能力：

1. 业务 API 能力：用户、团队、集群、技能、知识库、任务、审计等标准管理接口。
2. 智能运行时能力：对话会话、Agent Loop、工具调用、确认卡片、上下文记忆。
3. Kubernetes 原生能力：多集群接入、资源查询、日志流、终端执行、YAML 回写。
4. 平台治理能力：认证鉴权、敏感词过滤、高危操作拦截、审计、限流、加密存储。

设计上采用“模块化单体优先”的策略：

- 第一阶段先做一个单体 Go 服务，降低部署和调试成本。
- 通过清晰分层和模块边界，为后续按能力拆分为独立服务预留空间。
- 所有对外能力统一从 HTTP/WS/SSE 网关暴露，避免前端直接感知后端内部复杂度。

## 2. MVP 范围

结合 README 的 Phase 1，后端第一阶段建议聚焦以下能力：

1. 认证与基础 RBAC。
2. 单集群接入与连通性校验。
3. 对话会话管理。
4. 最小 Agent Runtime。
5. 内置 `get_pods`、`get_logs` 两个工具。
6. 基础资源列表与日志流接口。
7. 审计记录与高危操作确认框架。

以下内容保留接口和模块边界，但不建议在 MVP 首批就做深：

1. 多集群高级权限隔离。
2. Skill 录制与工作流 DAG 编排。
3. RAG 向量检索。
4. MCP 外部生态接入。
5. 定时备份与恢复编排。

## 3. 分层方案

推荐使用四层结构：

### 3.1 Interface 层

职责：

- 提供 REST、WebSocket、SSE 接口。
- 处理参数绑定、协议转换、响应编码。
- 注入认证上下文、请求追踪 ID、限流和审计切面。

建议目录：

- `cmd/server`
- `internal/httpapi`
- `internal/httpapi/middleware`
- `internal/httpapi/handlers`

### 3.2 Application 层

职责：

- 承载用例编排，不直接依赖 HTTP。
- 负责跨模块调用顺序、事务边界和权限检查。
- 输出明确的 DTO，供接口层组装响应。

建议目录：

- `internal/application/auth`
- `internal/application/cluster`
- `internal/application/chat`
- `internal/application/skill`
- `internal/application/knowledge`
- `internal/application/task`
- `internal/application/audit`

### 3.3 Domain 层

职责：

- 沉淀核心实体、值对象、领域规则。
- 不依赖 Gin、GORM、Redis、client-go 等框架实现。
- 明确哪些操作需要确认、哪些操作属于高危、哪些角色可执行。

建议目录：

- `internal/domain/auth`
- `internal/domain/cluster`
- `internal/domain/chat`
- `internal/domain/agent`
- `internal/domain/skill`
- `internal/domain/security`

### 3.4 Infrastructure 层

职责：

- 封装数据库、缓存、对象存储、client-go、LLM、MCP、向量库。
- 为上层暴露接口实现，不泄漏底层 SDK 细节。

建议目录：

- `internal/infrastructure/mysql`
- `internal/infrastructure/redis`
- `internal/infrastructure/k8s`
- `internal/infrastructure/llm`
- `internal/infrastructure/mcp`
- `internal/infrastructure/objectstore`

## 4. 模块边界

## 4.1 Auth 模块

职责：

- 登录、刷新、登出。
- JWT 签发与校验。
- 组织、团队、角色聚合。
- 平台 RBAC 与 K8s RBAC 映射。

关键对象：

- User
- Role
- Team
- PermissionScope

## 4.2 Cluster 模块

职责：

- 接入集群、保存加密凭证、连通性探测。
- 维护集群元数据和权限分配。
- 提供统一的 `ClusterClientProvider`，为 K8s 查询/操作提供 client-go 客户端。

关键对象：

- Cluster
- ClusterCredential
- ClusterPermission
- NamespaceScope

## 4.3 Chat / Agent 模块

职责：

- 维护会话和消息历史。
- 执行意图识别、工具路由、结果观察、二次确认中断恢复。
- 管理短期上下文和长期记忆访问入口。

关键对象：

- ChatSession
- ChatMessage
- AgentPlan
- ToolCall
- ConfirmationTask

## 4.4 Tool Execution 模块

职责：

- 对工具调用做 schema 校验。
- 判断是否命中敏感词或高危规则。
- 路由到内置工具或 MCP 工具。
- 记录执行日志和耗时。

关键对象：

- ToolDefinition
- ToolExecution
- ToolPolicy
- RiskAssessment

## 4.5 K8s Ops 模块

职责：

- 资源列表、详情、事件、日志、终端、YAML apply/delete。
- 统一封装资源查询过滤逻辑，屏蔽 client-go 细节。
- 对流式能力使用 WebSocket 连接管理器。

关键对象：

- ResourceQuery
- LogStreamSession
- ExecSession
- ApplyRequest

## 4.6 Skill 模块

职责：

- Skill CRUD。
- 工作流 Skill 与 Prompt Skill 的统一抽象。
- 将聊天中的工具调用序列录制为 Skill 草稿。

关键对象：

- Skill
- SkillVersion
- WorkflowNode
- PromptTemplate

## 4.7 Knowledge 模块

职责：

- 文档上传、切片、元数据记录、检索。
- 对象存储和索引存储解耦。

关键对象：

- KnowledgeDocument
- KnowledgeChunk
- SearchHit

## 4.8 Scheduler 模块

职责：

- 管理 cron 任务和执行历史。
- 调度备份、恢复或自定义脚本任务。
- 失败重试、告警钩子。

关键对象：

- ScheduledTask
- TaskRun
- BackupArtifact

## 4.9 Audit / Security 模块

职责：

- 记录 API、工具调用、敏感操作、登录行为。
- 维护敏感词、高危命令规则和审计检索。

关键对象：

- AuditLog
- SensitiveWord
- SecurityRule

## 5. 核心流程

## 5.1 对话请求流程

1. 前端调用 `POST /api/chat/sessions/:id/messages`。
2. Interface 层完成鉴权、参数解析、请求审计。
3. Chat Application 加载会话上下文和用户信息。
4. Agent Runtime 判断优先命中 Skill 还是自由工具调用。
5. Tool Executor 执行工具前做权限检查和风险评估。
6. 若命中高危操作，生成 `ConfirmationTask`，中断流程并返回确认卡片数据。
7. 若执行成功，追加消息历史，流式返回结果。

## 5.2 集群资源读取流程

1. 用户通过 API 发起资源查询。
2. Cluster Application 校验用户是否拥有该集群/命名空间访问权限。
3. K8s Provider 按集群 ID 获取复用的 client-go client。
4. Resource Service 执行查询并应用统一过滤规则。
5. 返回结构化资源列表，并写入审计日志。

## 5.3 高危操作确认流程

1. Tool Executor 先用规则库评估是否属于写操作或危险命令。
2. 若需要确认，持久化一条待确认记录。
3. 前端渲染执行计划卡片。
4. 用户确认后调用确认接口继续执行。
5. 全链路写入审计日志，保留原参数、确认人、执行结果。

## 6. API 分组建议

建议按领域拆分路由，而不是按技术能力拆：

- `/api/auth`
- `/api/users`
- `/api/teams`
- `/api/clusters`
- `/api/chat`
- `/api/skills`
- `/api/knowledge`
- `/api/tasks`
- `/api/audit`
- `/ws/chat/:sessionId`
- `/api/clusters/:id/logs`
- `/api/clusters/:id/exec`

建议统一响应包络：

```json
{
  "code": "OK",
  "message": "success",
  "data": {}
}
```

错误响应建议统一为：

```json
{
  "code": "UNAUTHORIZED",
  "message": "missing or invalid token",
  "requestId": "..."
}
```

## 7. 数据存储建议

首版推荐：

- MySQL：主业务数据。
- Redis：会话缓存、限流、短期上下文、任务锁。
- MinIO：知识库原文、备份产物。

后续增强：

- PGVector 或 Milvus：长期记忆、知识库向量检索。
- MQ：异步文档处理、审计异步落库、任务通知。

## 8. 安全设计要点

必须前置设计，不能后补：

1. 集群凭证必须加密存储，密钥不落库。
2. 写操作与读操作在工具层明确分级。
3. 高危命令需要规则引擎和用户确认。
4. 所有工具执行与终端命令都进入审计链路。
5. WebSocket 和 SSE 也必须继承认证上下文。
6. 所有多租户查询默认带上组织或团队过滤条件。

## 9. 推荐目录结构

```text
backend/
  cmd/
    server/
      main.go
  internal/
    app/
      app.go
    config/
      config.go
    httpapi/
      router.go
      server.go
      handlers/
        health.go
        stub.go
      middleware/
        requestid.go
  .env.example
  go.mod
```

这是第一步的最小骨架。后续继续实现时，优先把 `application`、`domain`、`infrastructure` 三层补出来，而不是把所有逻辑塞进 handler。

## 10. 下一步实现顺序

建议我们按下面顺序推进：

1. `Auth + User + Team` 数据模型和登录链路。
2. `Cluster` 接入、凭证加密、连通性测试。
3. `Chat Session` 与消息持久化。
4. `Agent Runtime` 最小闭环。
5. `K8s get_pods / get_logs` 两个工具。
6. `Audit + Risk Confirmation`。
7. 再扩到 Skill、Knowledge、Scheduler。
