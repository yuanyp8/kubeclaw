# kubeclaw
### 1. 项目概述

**项目名称**：OpsBrain
**项目定位**：面向 Kubernetes 环境的 AI 原生运维平台，通过自然语言交互完成日常运维任务，同时提供图形化辅助界面，实现“对话驱动 + UI 辅助”的混合模式，支持多集群管理、技能沉淀、知识库增强及安全审计。

**核心价值**：

- **提升效率**：用自然语言替代 kubectl 命令，快速完成日志查看、资源扩缩容、故障排查。
- **降低门槛**：降低对 Kubernetes 专业知识的依赖，开发者可自助查询服务状态。
- **安全可控**：敏感词过滤、高危操作二次确认、审计日志，保障生产安全。
- **知识沉淀**：通过技能录制将成功操作固化为可复用的 Skill，AI 自迭代进化。
- **生态扩展**：基于 MCP 协议动态接入工具，支持 Jenkins、云厂商等外部系统。

### 2.  核心功能模块设计 (Feature Details)

| 用户角色       | 核心诉求                 | 使用场景                                             |
| :------------- | :----------------------- | :--------------------------------------------------- |
| 运维工程师     | 快速定位问题、批量操作   | 排查 Pod 启动失败、批量重启 Deployment、查看集群事件 |
| 开发工程师     | 查看自己服务的日志、状态 | 查询应用日志、检查 Pod 状态、扩缩容                  |
| SRE/平台负责人 | 多集群管理、审计、备份   | 接入新集群、查看操作审计、配置定时备份               |
| 管理者         | 审计合规、资源使用统计   | 查看操作记录、集群健康度                             |

### 2.1 智能对话中枢 (Agent Hub)

这是用户的核心工作台，采用“左侧 Chat，右侧动态面板”的布局。

- **ReAct Agent 循环**：AI 接收意图后，执行 `思考 (Thought) -> 规划 (Plan) -> 调用工具 (Action) -> 观察结果 (Observation)` 的循环。例如，用户要求“清理脱机的节点”，Agent 会先获取节点列表，观察状态，然后再执行隔离或删除操作。
- **长短记忆机制 (Memory)**：
  - **短期记忆 (Context Window)**：维持当前 Session 的上下文。用户前一句问“查看 prod-mysql 的状态”，后一句说“看看它的日志”，系统自动补齐目标。
  - **长期记忆 (Vector DB)**：通过向量库记录用户的历史偏好与集群特征。
- **高危操作拦截与确认卡片**：当 LLM 决策需要执行写操作（如 `k8s_delete_pod`, `exec_command`）时，中断 Agent Loop，向前端推送一张 **“执行计划确认卡”**，展示即将执行的完整命令与参数，用户点击“授权执行”后，Loop 继续。

### 2.2 多集群与可视化管理 (K8s Visual Engine)

对话是入口，但对于高频结构化数据，UI 呈现效率更高。

- **多集群接入与隔离**：支持通过 Kubeconfig 或 ServiceAccount 导入。每个集群可设置为“公开”或“私有（指定团队）”。
- **智能过滤与定位**：在获取和展示集群资源时（尤其是自动提取业务主 Service），后端内置智能过滤逻辑，利用 jsonpath 排除 metrics 端口和 Headless Service，精准定位业务主体。
- **沉浸式日志与终端**：
  - 通过 WebSocket 提供 xterm.js 终端模拟。
  - **细节约束**：在通过工具下发多行 Shell 脚本（例如利用 Here Document 注入环境配置）时，引擎层严格控制变量转义，防止在网关或 Agent 层发生提前解析（Premature Expansion），确保脚本在目标 Pod 内准确执行。
- **YAML 协同编辑**：右侧面板弹出的 Monaco Editor，支持 YAML 实时修改并回写集群。

### 2.3 MCP 工具与技能沉淀 (Skill & MCP Ecosystem)

- **动态 MCP 注册中心**：平台支持接入遵循 Model Context Protocol 的外部 Server。
- **Skill 录制与复用 (自迭代核心)**：
  - 用户在 Chat 中成功完成一次排障（如：查 Pod -> 看事件 -> 查 Cilium 网络策略 -> 重启），点击“沉淀为 Skill”。
  - 系统提取 Tool 调用序列，生成 YAML 格式的工作流描述。
- **常见内置原生 Tool 示例**：
  - `k8s_get_resources`, `k8s_get_logs`, `k8s_exec_command`, `k8s_apply_yaml`

### 2.4 企业级安全与合规控制 (Security & Audit)

- **双向敏感词管理**：
  - **输入过滤**：阻断带有恶意意图的 prompt。
  - **输出/执行过滤**：基于正则和 AST 树，拦截 `rm -rf`, `drop database`, `kubectl delete ns kube-system` 等高危命令。
- **全局审计日志**：记录 `SessionID, User, Cluster, Action (Tool Name), Parameters, IP, Timestamp`。
- **权限映射**：平台 RBAC 与 K8s RBAC 联动。如果用户在平台只有 Viewer 权限，Agent 拒绝调用任何变更类 MCP 工具。

### 2.5 知识库增强 (RAG)

- 允许用户上传 PDF/Markdown/TXT 格式的架构文档、业务 SOP、历史事故复盘报告。
- Agent 在遇到复杂问题时，优先调用 `search_knowledge_base` 工具，获取上下文再生成排障建议。

### 2.6 自动化与定时任务 (Cron Scheduler)

- 用户可通过自然语言创建任务：“每天凌晨 2 点备份 etcd 数据到 MinIO”。
- Agent 调用 `create_cron_task` 工具。后端通过 Go 的 Cron 引擎维护任务队列，并提供独立的 UI 面板用于查看执行历史、启停任务或触发手动恢复。

### 3. 功能需求

#### 3.1 用户管理与认证

| 功能          | 描述                                                         |
| :------------ | :----------------------------------------------------------- |
| 用户注册/登录 | 支持邮箱/用户名密码注册登录，JWT 令牌认证，支持 OIDC/OAuth2 对接企业 SSO |
| 角色权限      | 内置角色：超级管理员、集群管理员、普通用户、只读用户；支持自定义角色，细粒度权限（集群级、命名空间级） |
| 团队/组织     | 支持创建团队，用户可加入多个团队；资源（集群、知识库、定时任务）可共享给团队或公开 |
| 操作审计      | 记录所有用户操作（登录、集群操作、工具调用），包含时间、用户、操作内容、结果、IP，支持导出和归档 |

#### 3.2 多集群管理

| 功能         | 描述                                                         |
| :----------- | :----------------------------------------------------------- |
| 集群接入     | 支持导入多个 K8s 集群，支持 kubeconfig 文件上传、Token、证书认证；验证连通性后保存加密凭证 |
| 集群隔离     | 用户仅可见有权限的集群；可为每个集群设置用户/团队的访问权限（admin/editor/viewer） |
| 集群状态监控 | 展示集群节点状态、资源使用概览（CPU/内存）、事件列表，支持从 Metrics API 获取数据 |
| 命名空间管理 | 支持按命名空间过滤资源，可为用户授予特定命名空间的访问权限   |

#### 3.3 AI 智能对话助手

**核心设计**：基于 Agent Loop 模式，具备规划、调用工具、观察结果的能力，支持多轮交互。

| 功能           | 描述                                                         |
| :------------- | :----------------------------------------------------------- |
| 对话界面       | 类 ChatGPT 风格，支持 Markdown、代码高亮、流式输出（SSE/WebSocket）；支持新建、历史会话管理 |
| 上下文记忆     | 短期记忆：会话内最近 10-20 轮对话及执行结果；长期记忆：跨会话的用户偏好、常用操作模式（存储于数据库，可编辑） |
| 意图识别与路由 | 系统优先匹配已有 Skill，若无则进入自由工具调用模式；支持多步骤规划，如“排查 order-service 启动失败”自动调用：查 Pod → 看事件 → 查日志 |
| 工具调用       | 支持 MCP 协议的工具注册；AI 解析用户指令，生成结构化参数，调用后端工具执行（如 get_pods, get_logs, restart_deployment） |
| 高风险确认     | 对高危操作（删除资源、重启关键应用）返回执行计划卡片，用户确认后执行 |
| UI 联动        | 用户在 UI 上点击某个 Pod 后，对话上下文自动关联该资源，可直接提问“查看这个 Pod 的日志” |

#### 3.4 工具与技能管理（MCP + Skill）

**目标**：实现能力沉淀与复用。

| 功能          | 描述                                                         |
| :------------ | :----------------------------------------------------------- |
| 工具注册中心  | 支持动态注册 MCP Server，自动发现工具列表并生成 Function Call Schema；内置 K8s 核心工具集（日志、描述、扩缩容、执行命令等） |
| 技能（Skill） | 支持两种类型：**工作流型**（固定步骤编排）和 **Prompt 模板型**（复杂提示词封装）；技能以 YAML 定义，可版本管理 |
| 技能录制      | 用户在对话中完成一系列操作后，可点击“保存为技能”，系统自动提取工具调用序列生成 Skill 草稿，用户编辑后入库 |
| 技能市场      | 管理员可发布公共技能，用户可引用或自定义修改                 |

#### 3.5 运维辅助界面

**目标**：提供直观的可视化操作，补充对话方式的不足。

| 功能         | 描述                                                         |
| :----------- | :----------------------------------------------------------- |
| 资源概览     | 树形结构展示集群资源（按命名空间、资源类型），支持搜索、筛选；展示资源列表、YAML、事件、关联资源 |
| YAML 编辑器  | 集成 Monaco Editor，支持语法高亮、自动补全、校验，支持创建/更新资源 |
| 日志实时流   | WebSocket 实时推送 Pod 日志，支持多 Pod 同时查看，支持容器选择、关键词过滤、时间范围 |
| Web 终端     | 基于 xterm.js 的终端，支持执行 kubectl 命令或进入 Pod Shell，命令受权限和敏感词过滤 |
| dashboard    | 资源看板  项目管理等                                         |
| 资源操作按钮 | 一键重启、扩缩容、删除（二次确认），支持批量操作             |

#### 3.6 定时任务与备份恢复

| 功能       | 描述                                                         |
| :--------- | :----------------------------------------------------------- |
| 定时备份   | 支持备份 etcd 数据或指定资源清单（YAML 集合），配置 Cron 表达式，目标存储支持 S3/MinIO/NFS |
| 备份历史   | 查看备份记录，支持从备份点恢复（恢复时需二次确认）           |
| 自定义任务 | 支持用户编写自定义脚本（需平台执行环境），通过定时任务触发   |
| 任务监控   | 查看任务执行状态、日志，失败告警（可选集成飞书/钉钉）        |

#### 3.7 知识库管理

| 功能       | 描述                                                     |
| :--------- | :------------------------------------------------------- |
| 文档上传   | 支持 Markdown、PDF、文本文件上传，按集群/团队隔离        |
| RAG 检索   | 对话时 AI 自动检索相关文档片段作为上下文，增强回答准确性 |
| 知识库版本 | 支持文档版本管理，允许更新替换                           |
| 内置知识库 | 预置 Kubernetes 官方文档常见问题库，可更新               |

#### 3.8 安全与审计

| 功能         | 描述                                                         |
| :----------- | :----------------------------------------------------------- |
| 敏感词过滤   | 内置敏感词库（可配置），对用户输入和 AI 即将执行的命令进行过滤；高风险词触发拒绝或二次确认 |
| 高危操作拦截 | 系统预置高危操作列表（如 delete namespace, rm -rf），执行前需二次确认并记录审计 |
| 操作审计     | 所有用户操作、工具调用、对话记录均存入审计日志，支持按时间、用户、操作类型检索 |
| 访问控制     | 所有 API 需 JWT 认证，支持 IP 白名单（可选）                 |
| 数据加密     | 集群凭证、数据库密码等敏感数据使用 AES-256 加密存储          |

### 4. 系统架构设计

#### 4.1 总体架构图

text

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          前端层 (Next.js / Vue 3)                       │
│  [Chat 对话流]    [确认执行卡片]    [Web 终端/日志 xterm.js]    [拓扑/YAML] │
└───────┬─────────────────────────────┬───────────────────────────┬───────┘
        │ REST (对话/业务)             │ WS (日志/终端)             │ SSE (流式回复)
┌───────▼─────────────────────────────▼───────────────────────────▼───────┐
│                            API Gateway / 鉴权层 (Gin)                    │
│  - JWT 校验    - 敏感词/高危命令拦截    - 速率限制 (Rate Limit)           │
└───────┬─────────────────────────────────────────────────────────────────┘
        │
┌───────▼─────────────────────────────────────────────────────────────────┐
│                           核心业务逻辑层 (Core Logic)                     │
│ ┌──────────────────────┐  ┌──────────────────────┐  ┌─────────────────┐ │
│ │    Agent Runtime     │  │   Skill / Workflow   │  │ Ops & K8s Native│ │
│ │ - Intent Router      │  │ - DAG 执行器         │  │ - 多集群 Client │ │
│ │ - ReAct Loop 引擎    │  │ - Skill 录制/解析     │  │ - Log/Exec 流化 │ │
│ │ - Memory 管理(会话)   │  │ - Cron 定时任务调度   │  │ - YAML 语法校验 │ │
│ └───────┬──────────────┘  └─────────┬────────────┘  └────────┬────────┘ │
└─────────┼───────────────────────────┼────────────────────────┼──────────┘
          │                           │                        │
┌─────────▼───────────────────────────▼────────────────────────▼──────────┐
│                           MCP 协议与工具层 (MCP Layer)                    │
│  [内置 K8s MCP (Go)]    [知识库 RAG MCP (Python)]   [扩展外部 MCP Server] │
└─────────┬───────────────────────────┬────────────────────────┬──────────┘
          │                           │                        │
┌─────────▼────────┐      ┌───────────▼──────────┐      ┌──────▼──────────┐
│  K8s Clusters    │      │    Vector DB (RAG)   │      │ DB & Cache      │
│ (dev, test, prod)│      │  (Milvus / PGVector) │      │ MySQL / Redis   │
└──────────────────┘      └──────────────────────┘      └─────────────────┘
```

#### 4.2 技术选型

| 层次        | 技术栈                                            |
| :---------- | :------------------------------------------------ |
| 前端        | React 18 + TypeScript + TailwindCSS + Vite        |
| UI 组件库   | Ant Design 5（企业级） + Headless UI              |
| 代码编辑器  | Monaco Editor                                     |
| 终端模拟    | xterm.js                                          |
| 实时通信    | WebSocket (gorilla/websocket)                     |
| 后端框架    | Go 1.21+ + Gin                                    |
| K8s 客户端  | client-go                                         |
| 数据库      | MySQL 8.0 + GORM                                  |
| 缓存/消息   | Redis 7.0                                         |
| 对象存储    | MinIO（自托管）或 S3                              |
| 任务调度    | robfig/cron/v3                                    |
| AI 模型适配 | 支持 OpenAI、Azure OpenAI、本地模型（通过适配器） |
| 认证        | golang-jwt                                        |
| 配置管理    | Viper                                             |
| 日志        | Zap + lumberjack                                  |
| 监控        | Prometheus + Grafana（可选）                      |

#### 4.3 核心模块详解

**1. Agent Runtime（智能体运行时）**

- 接收用户消息，维护会话上下文。
- 规划步骤：根据意图匹配 Skill 或进行 Function Calling 循环。
- 调用工具执行器，处理结果，生成最终回复。
- 支持流式输出，边执行边返回。

**2. Tool Executor（工具执行器）**

- 接收工具调用请求，参数校验。
- 根据工具类型路由：内置工具直接调用 Go 函数；MCP 工具通过 JSON-RPC 调用外部 MCP Server。
- 执行结果封装返回，同时记录审计日志。

**3. Skill Engine（技能引擎）**

- 管理 Skill 的 CRUD。
- 执行工作流型 Skill：解析 DAG，按依赖顺序调用工具，支持条件分支。
- 录制功能：将一次对话中的工具调用序列转换为 Skill YAML。

**4. Cluster Manager（集群管理）**

- 维护多个集群的 client-go 客户端池。
- 根据集群 ID 动态获取客户端，支持连接复用。
- 凭证加密存储，支持动态刷新。

**5. Log Stream Service（日志流服务）**

- 使用 WebSocket 建立长连接，后端启动 goroutine 实时读取 Pod 日志（kubectl logs -f），转发给前端。
- 支持多个 Pod 同时查看，每个 Pod 独立 goroutine。

**6. Knowledge Service（知识库服务）**

- 文档存储于 MinIO，元数据存 MySQL。
- 检索时对文档进行分块，使用向量检索（可选对接向量数据库）或简单关键词匹配，返回相关片段。

**7. Scheduler Service（定时任务服务）**

- 基于 cron 表达式触发任务。
- 备份任务：调用备份工具（如 etcd 备份或资源清单导出），存储到 MinIO。
- 恢复任务：从备份点恢复，需二次确认。

### 5. 数据模型设计

#### 5.1 核心表结构

**users**

```sql
id BIGINT PRIMARY KEY AUTO_INCREMENT,
username VARCHAR(50) NOT NULL UNIQUE,
email VARCHAR(100) NOT NULL UNIQUE,
password_hash VARCHAR(255) NOT NULL,
role VARCHAR(20) DEFAULT 'user', -- admin, cluster_admin, user, readonly
status TINYINT DEFAULT 1, -- 1正常 0禁用
created_at DATETIME,
updated_at DATETIME
```

**teams**

```sql
id BIGINT PRIMARY KEY,
name VARCHAR(100) NOT NULL,
owner_id BIGINT,
created_at DATETIME
```

**user_teams**

```sql
user_id BIGINT,
team_id BIGINT,
role VARCHAR(20) -- admin, member
```

**clusters**

```sql
id BIGINT PRIMARY KEY,
name VARCHAR(100) NOT NULL,
api_server VARCHAR(255) NOT NULL,
auth_type ENUM('kubeconfig', 'token', 'cert') NOT NULL,
credentials TEXT NOT NULL, -- 加密存储
owner_id BIGINT,
is_public TINYINT DEFAULT 0,
status ENUM('active', 'inactive') DEFAULT 'active',
created_at DATETIME,
updated_at DATETIME
```

**cluster_permissions**

```sql
id BIGINT PRIMARY KEY,
cluster_id BIGINT,
user_id BIGINT,
team_id BIGINT,
role VARCHAR(20) NOT NULL, -- admin, editor, viewer
UNIQUE KEY (cluster_id, user_id, team_id)
```

**chat_sessions**

```sql
id BIGINT PRIMARY KEY,
user_id BIGINT NOT NULL,
title VARCHAR(200),
context JSON, -- 存储短期记忆的上下文
created_at DATETIME,
updated_at DATETIME
```

**chat_messages**

```sql
id BIGINT PRIMARY KEY,
session_id BIGINT NOT NULL,
role ENUM('user', 'assistant', 'system') NOT NULL,
content TEXT,
tool_calls JSON, -- AI 请求的工具调用
tool_call_id VARCHAR(50), -- 工具执行时的关联ID
created_at DATETIME
```

**tool_executions**

```sql
id BIGINT PRIMARY KEY,
user_id BIGINT,
cluster_id BIGINT,
tool_name VARCHAR(100),
parameters JSON,
result TEXT,
status ENUM('running', 'success', 'failed'),
duration_ms INT,
created_at DATETIME
```

**skills**

```sql
id BIGINT PRIMARY KEY,
name VARCHAR(100) NOT NULL,
description TEXT,
type ENUM('workflow', 'prompt') NOT NULL,
definition JSON NOT NULL, -- 工作流步骤或 prompt 模板
version INT DEFAULT 1,
creator_id BIGINT,
is_public TINYINT DEFAULT 0,
created_at DATETIME,
updated_at DATETIME
```

**knowledge_bases**

```sql
id BIGINT PRIMARY KEY,
name VARCHAR(100) NOT NULL,
owner_id BIGINT,
cluster_id BIGINT, -- 可选，关联集群
is_public TINYINT DEFAULT 0,
file_path VARCHAR(255), -- MinIO 路径
file_type VARCHAR(20), -- pdf, md, txt
status ENUM('processing', 'ready', 'failed'),
created_at DATETIME
```

**scheduled_tasks**

```sql
id BIGINT PRIMARY KEY,
name VARCHAR(100) NOT NULL,
cron VARCHAR(50) NOT NULL,
cluster_id BIGINT,
action ENUM('backup', 'restore', 'custom') NOT NULL,
params JSON,
last_run DATETIME,
next_run DATETIME,
status ENUM('active', 'inactive'),
created_at DATETIME
```

**audit_logs**

```sql
id BIGINT PRIMARY KEY,
user_id BIGINT,
action VARCHAR(50),
target VARCHAR(200), -- 如 cluster:1
details TEXT,
ip VARCHAR(45),
created_at DATETIME
```

**sensitive_words**

```sql
id BIGINT PRIMARY KEY,
word VARCHAR(100) NOT NULL UNIQUE,
level ENUM('low', 'medium', 'high') DEFAULT 'medium',
created_at DATETIME
```

### 6. 接口设计

#### 6.1 认证接口

text

```
POST   /api/auth/login       // 登录
POST   /api/auth/refresh     // 刷新token
POST   /api/auth/logout      // 登出
```



#### 6.2 用户与团队

text

```
GET    /api/users/me         // 获取当前用户信息
PUT    /api/users/me         // 更新个人信息
GET    /api/teams            // 我的团队
POST   /api/teams            // 创建团队
```



#### 6.3 集群管理

text

```
GET    /api/clusters         // 获取有权限的集群列表
POST   /api/clusters         // 添加集群（管理员或集群管理员）
GET    /api/clusters/:id     // 集群详情
PUT    /api/clusters/:id     // 更新集群配置
DELETE /api/clusters/:id     // 删除集群（需权限）
POST   /api/clusters/:id/share // 分享集群给用户/团队
GET    /api/clusters/:id/permissions // 查看权限列表
```



#### 6.4 AI 对话

text

```
POST   /api/chat/sessions                // 创建会话
GET    /api/chat/sessions                // 获取会话列表
GET    /api/chat/sessions/:id/messages   // 获取消息历史
POST   /api/chat/sessions/:id/messages   // 发送消息（支持流式响应，SSE）
WS     /ws/chat/:sessionId               // WebSocket 用于流式输出及实时交互
```



#### 6.5 资源操作（K8s）

text

```
GET    /api/clusters/:id/namespaces      // 获取命名空间
GET    /api/clusters/:id/resources       // 获取资源列表（支持type, namespace, labelSelector）
GET    /api/clusters/:id/resources/:type/:name // 获取单个资源YAML
POST   /api/clusters/:id/resources       // 创建/更新资源（YAML）
DELETE /api/clusters/:id/resources/:type/:name // 删除资源
GET    /api/clusters/:id/events          // 获取事件
```



#### 6.6 日志流与终端

text

```
WS     /api/clusters/:id/logs            // 实时日志（参数: pod, container, tailLines）
WS     /api/clusters/:id/exec            // 终端执行命令（参数: pod, container, command）
```



#### 6.7 技能管理

text

```
GET    /api/skills                       // 获取技能列表
POST   /api/skills                       // 创建技能
GET    /api/skills/:id                   // 技能详情
PUT    /api/skills/:id                   // 更新技能
DELETE /api/skills/:id                   // 删除技能
POST   /api/skills/:id/execute           // 执行技能（返回执行结果）
```



#### 6.8 知识库

text

```
POST   /api/knowledge                    // 上传文档
GET    /api/knowledge                    // 知识库列表
GET    /api/knowledge/:id                // 文档详情
DELETE /api/knowledge/:id                // 删除文档
POST   /api/knowledge/search             // 检索（返回相关片段）
```



#### 6.9 定时任务

text

```
GET    /api/tasks                        // 任务列表
POST   /api/tasks                        // 创建任务
GET    /api/tasks/:id                    // 任务详情
PUT    /api/tasks/:id                    // 更新任务
DELETE /api/tasks/:id                    // 删除任务
POST   /api/tasks/:id/run                // 手动触发一次
GET    /api/tasks/:id/history            // 执行历史
```



#### 6.10 审计

text

```
GET    /api/audit                        // 审计日志列表（管理员）
```



------

### 7. 部署方案

#### 7.1 部署拓扑

- 平台本身作为 Kubernetes 原生应用部署，使用 Deployment 部署后端，使用 Service + Ingress 暴露服务。
- 前端静态文件打包成镜像，使用 Nginx 托管，或与后端合并为单一镜像。
- 数据库 MySQL 和 Redis 可部署在集群内（使用 StatefulSet）或使用云服务。
- 对象存储使用 MinIO 部署在集群内，或使用云厂商 S3。

#### 7.2 权限与安全

- 平台后端使用 ServiceAccount，通过 RBAC 授予对 K8s 资源的操作权限（根据多集群需求，可能需要多个 ServiceAccount 或动态凭证）。
- 对于外部集群，平台存储的凭证需加密，通信使用 HTTPS。
- 设置 NetworkPolicy 限制后端服务仅允许 API Gateway 访问数据库。

#### 7.3 高可用

- 后端无状态，可水平扩展。
- MySQL 采用主从或云托管高可用版本。
- Redis 使用哨兵或集群模式。
- MinIO 使用分布式模式。

#### 7.4 配置管理

- 使用 ConfigMap 存储非敏感配置（如日志级别、集群列表）。
- 敏感配置（数据库密码、加密密钥）使用 Secret 存储。

------

### 8. 开发路线图

**Phase 1：基础能力（MVP） - 2个月**

- 用户认证、基本权限（角色）
- 单集群接入（kubeconfig 上传）
- AI 对话基础（对接 OpenAI，实现 get_pods, get_logs 两个工具）
- 简单前端聊天界面
- 基础资源列表 UI（表格展示 Pod/Deployment）

**Phase 2：核心功能 - 2个月**

- 多集群管理、权限隔离
- Agent Loop 实现（支持多步规划、Function Calling）
- 工具扩展（restart, scale, describe, exec）
- 知识库基础（文件上传、简单检索）
- 日志流（WebSocket 实时日志）
- 技能录制与执行
- 敏感词过滤、高危拦截

**Phase 3：增强与生态 - 2个月**

- MCP 协议集成，支持动态加载外部工具
- 定时任务（备份/恢复）
- Web 终端（xterm.js）
- YAML 编辑器集成
- 长期记忆（存储用户偏好）
- 审计日志查询界面
- 性能优化（缓存、连接池）

**Phase 4：智能化与自迭代 - 1个月**

- RAG 增强（向量检索）
- 用户反馈收集（点赞/点踩），用于 Prompt 优化
- 技能市场（公开技能）
- 多模型适配（支持本地模型）
- 告警集成（钉钉/飞书通知）

------

### 9. 风险与应对

| 风险                    | 应对措施                                                     |
| :---------------------- | :----------------------------------------------------------- |
| AI 模型幻觉导致错误操作 | 高危操作强制二次确认；敏感词过滤；限制工具调用范围；审计日志可追溯 |
| 多集群凭证泄露          | 凭证加密存储（AES-256）；定期轮换；使用最小权限原则          |
| 性能瓶颈（大量日志流）  | Go 协程高效处理；使用 WebSocket 复用；限制单用户并发连接数   |
| 知识库检索不准确        | 支持用户反馈调整；后期引入向量数据库优化召回                 |
| 定时任务执行失败        | 记录失败日志；支持重试机制；告警通知                         |