# Agent Context 云服务设计规格

## 1. Acontext 核心能力分析

### 1.1 架构概览

```mermaid
graph TB
    subgraph "Client SDKs"
        PY["Python SDK"]
        TS["TypeScript SDK"]
    end

    subgraph "Acontext Backend"
        API["API Layer<br/>(Go/FastAPI)"]
        CORE["Core Engine<br/>(Python)"]

        subgraph "Infrastructure"
            PG["PostgreSQL<br/>+ pgvector"]
            S3["S3/MinIO"]
            REDIS["Redis"]
            MQ["RabbitMQ"]
        end
    end

    subgraph "Dashboard"
        UI["Web UI"]
    end

    PY & TS -->|REST| API
    UI -->|REST| API
    API -->|MQ| CORE
    CORE --> PG & S3 & REDIS
```

### 1.2 核心功能模块

```mermaid
flowchart TB
    Core((Acontext))

    subgraph Store["📦 Store"]
        direction TB
        S1["Session 会话存储"]
        S1a["多 LLM 格式支持"]
        S1b["消息树结构(分支对话)"]
        S2["Disk 磁盘存储"]
        S2a["文件路径管理"]
        S2b["公开URL下载"]
        S1 --- S1a & S1b
        S2 --- S2a & S2b
    end

    subgraph Observe["👁️ Observe"]
        direction TB
        O1["Task Agent"]
        O1a["任务自动识别"]
        O1b["状态追踪"]
        O1c["进度摘要"]
        O1d["偏好收集"]
        O2["智能缓冲"]
        O2a["批量处理"]
        O2b["减少 LLM 调用"]
        O1 --- O1a & O1b & O1c & O1d
        O2 --- O2a & O2b
    end

    subgraph Learn["🎓 Learn"]
        direction TB
        L1["SOP 提取"]
        L1a["复杂度评估"]
        L1b["工具序列抽象"]
        L2["Space 知识库"]
        L2a["层次化组织"]
        L2b["向量搜索"]
        L2c["Agentic 搜索"]
        L1 --- L1a & L1b
        L2 --- L2a & L2b & L2c
    end

    subgraph Dashboard["📊 Dashboard"]
        direction TB
        D1["会话回放"]
        D2["任务状态"]
        D3["SOP 浏览"]
    end

    Core --> Store & Observe & Learn & Dashboard
```

### 1.3 核心数据流

```
┌─────────┐    ┌───────────┐    ┌────────────┐    ┌─────────────┐
│ Message │───►│ Task Agent│───►│ SOP Agent  │───►│ Space Store │
│ 消息流入 │    │ 任务提取   │    │ 技能提炼   │    │ 知识沉淀    │
└─────────┘    └───────────┘    └────────────┘    └─────────────┘
                    │                                    │
                    ▼                                    ▼
              ┌───────────┐                      ┌─────────────┐
              │ Dashboard │◄─────────────────────│ Experience  │
              │ 可视化     │                      │ Search      │
              └───────────┘                      └─────────────┘
```

### 1.4 关键设计要点

| 模块 | 设计要点 | 价值 |
|------|----------|------|
| **消息存储** | 树形结构(parent_id)、S3 大对象 | 支持分支对话、低成本存储 |
| **Task 观察** | 非侵入式、状态机模式、证据链 | 无需 Agent 特殊格式化 |
| **智能缓冲** | 2 条消息 or 10s 超时触发 | LLM 调用减少 50-80% |
| **SOP 学习** | 复杂度 ≥2 才学习、去语境化 | 只学有价值的、可复用 |
| **Space 搜索** | Fast(向量) + Agentic(LLM 导航) | 简单查询快、复杂查询准 |

---

## 2. Agent Context 云服务功能设计

### 2.1 功能全景图

```mermaid
mindmap
  root((Agent Context<br/>云服务))
    降低 TCO
      开发提效
        P0 统一 Session 存储
        P0 即插即用 SDK
        P0 Context 操作
        P1 主流框架无感接入
        P1 可视化 Dashboard
        P2 Prompt 模板市场
        P2 Built-in Tool
        P1 任务状态/监控告警
      Token 优化
        P1 SOP 复用
        P1 上下文压缩
        P2 智能缓冲
    提升效果
      P1 SOP 提取与应用
      P1 用户偏好积累
      P2 SOP/Skill 市场
```

 ### 2.2 功能详细设计

#### 2.2.1 降低 TCO - 开发提效

##### P0: 统一 Session 存储

**目标**: 开发者无需自建消息存储，一套 API 托管所有对话历史

```mermaid
graph LR
    A[OpenAI Format] --> S[Session API]
    B[Anthropic Format] --> S
    C[Gemini Format] --> S
    S --> D[(Cloud Storage)]
    S --> E[消息树管理]
    S --> F[分支/回滚/复制]
```

**核心能力**:
- 多 LLM 消息格式统一适配
- 消息树结构：支持 retry、edit、branch、clone
- 元数据关联：tool_call_id、图片、文件
- 自动持久化 + 按需加载

---

##### P0: 即插即用 SDK

**目标**: 3 行代码完成接入

```python
# Python 示例
from agent_context import AgentContext
ctx = AgentContext(api_key="xxx")
session = ctx.session.create()
```

```typescript
// TypeScript 示例
import { AgentContext } from '@agent-context/sdk';
const ctx = new AgentContext({ apiKey: 'xxx' });
const session = await ctx.session.create();
```

**核心能力**:
- Python / TypeScript 双语言
- 同步 + 异步 API
- 自动重试、断线重连
- 本地开发模式（Mock）

---

##### P0: Context 操作

**目标**: 一套 API 完成上下文工程的常见操作

###### 即时 Context 操作

对 Session 状态的即时管理，支持快照、分支、回滚等场景。

| 操作 | 说明 | 用途 |
|------|------|------|
| **Checkpoint** | 保存当前状态快照 | 断点恢复、回滚 |
| **Restore** | 恢复到某个快照 | 状态回退 |
| **Clone** | 复制 Session | A/B 测试、并行探索 |
| **Branch** | 从某消息分叉 | 多路径尝试 |
| **Merge** | 合并两个 Session | 多 Agent 协作后合并结果 |

```python
# Checkpoint: 保存快照
cp_id = ctx.session.checkpoint(session_id)

# Restore: 恢复到快照
ctx.session.restore(session_id, checkpoint_id=cp_id)

# Clone: 复制整个 Session
new_session = ctx.session.clone(session_id)

# Branch: 从消息 M 分叉
new_session = ctx.session.branch(session_id, from_message_id="msg_xxx")

# Merge: 合并两个 Session
ctx.session.merge(target_session_id, source_session_id, strategy="interleave")
```

###### Context 编辑

对消息内容的编辑与优化，分为客户端操作和 LLM 辅助操作。

| 类型 | 操作 | 说明 | 用途 |
|------|------|------|------|
| **客户端** | Window | 滑动窗口截取 | 控制上下文长度 |
| **客户端** | Remove Tool Response | 移除 tool_call 响应内容 | 减少冗余 Token |
| **客户端** | Truncate | 截断超长消息 | Token 限制 |
| **客户端** | Mask | 按规则遮蔽敏感信息 | 隐私保护、日志脱敏 |
| **LLM** | Compress | 压缩历史消息为摘要 | 长对话 Token 优化 |
| **LLM** | Deduplicate | 去除重复/相似内容 | 信息去重 |
| **LLM** | Anonymize | 智能识别并脱敏 PII | 合规、隐私保护 |

```python
# 客户端操作（纯本地，无 LLM 调用）
ctx.context.window(session_id, last_n=10)                      # 保留最近 10 条
ctx.context.remove_tool_response(session_id, keep_summary=True) # 移除 tool 响应
ctx.context.truncate(session_id, max_tokens=4000)              # 截断到 4000 tokens
ctx.context.mask(session_id, patterns=["email", "phone"])      # 正则规则脱敏

# LLM 辅助操作（需调用 LLM）
ctx.context.compress(session_id, keep_last=5)    # 压缩前 N 条消息为摘要
ctx.context.deduplicate(session_id)              # 去除重复内容
ctx.context.anonymize(session_id)                # 智能 PII 脱敏
```

###### 高级操作

跨 Session 的复杂操作，支持多 Agent 协作场景。

| 操作 | 说明 | 用途 |
|------|------|------|
| **Handoff** | 准备交接上下文 | 多 Agent 接力时精简传递 |

```python
# Handoff: 生成交接包（摘要 + 关键决策 + 待办）
handoff_context = ctx.context.handoff(
    session_id,
    include=["summary", "decisions", "todos"],
    target_agent="support_agent"
)
# 返回结构化的交接上下文，可直接注入目标 Agent
```

---

##### P1: 主流框架无感接入

**目标**: 现有 Agent 代码零改动或极少改动即可接入

| 框架 | 接入方式 | 复杂度 |
|------|----------|--------|
| OpenAI Agent SDK | Middleware 注入 | 低 |
| LangChain | Middleware 注入 | 中 |
| AutoGen | 自定义 Agent 包装 | 中 |

```python
# LangGraph 示例
from agent_context.integrations import AgentContextCheckpointer
graph = StateGraph(...)
graph.compile(checkpointer=AgentContextCheckpointer(session_id="xxx"))
```

---

##### P1: 可视化 Dashboard

**目标**: 免写调试工具，实时观察 Agent 运行状态

```
┌─────────────────────────────────────────────────────────┐
│  Dashboard                                              │
├─────────────┬───────────────────────────────────────────┤
│ Sessions    │  Session Detail                           │
│ ├─ #a1b2c3  │  ┌─────────────────────────────────────┐ │
│ ├─ #d4e5f6  │  │ Message Timeline                    │ │
│ └─ #g7h8i9  │  │ [User] 帮我查询订单                 │ │
│             │  │ [Assistant] 好的，正在查询...       │ │
│ Tasks       │  │ [Tool] query_order({id: 123})       │ │
│ ├─ Running  │  └─────────────────────────────────────┘ │
│ └─ Done     │  ┌─────────────────────────────────────┐ │
│             │  │ Task Status                         │ │
│ Metrics     │  │ ✓ 理解用户意图                      │ │
│ ├─ Tokens   │  │ ● 执行订单查询                      │ │
│ └─ Latency  │  │ ○ 返回结果                          │ │
│             │  └─────────────────────────────────────┘ │
└─────────────┴───────────────────────────────────────────┘
```

**核心能力**:
- Session 列表与消息时间线
- 任务状态实时展示
- Token 消耗统计
- 错误日志与追踪

---

##### P1: 任务状态与监控告警

**目标**: 自动追踪 Agent 任务进度，异常时告警

```mermaid
graph LR
    M[消息流] --> TA[Task Agent]
    TA --> TS[任务状态]
    TS --> |成功率下降| AL[告警]
    TS --> |超时| AL
    AL --> WH[Webhook]
    AL --> EM[Email]
```

**核心能力**:
- Task 自动识别与状态追踪（同 Acontext）
- 成功率/失败率统计
- 自定义告警规则
- Webhook / Email 通知

---

##### P2: Prompt 模板市场

**目标**: 提供高质量系统 Prompt 模板

**分类**:
- 编程助手（Code Agent）
- 数据分析（Data Agent）
- 客服对话（Support Agent）
- 浏览器操作（Browser Agent）

---

##### P2: Built-in Tool

**目标**: 常用工具标准定义，减少重复编写

```python
from agent_context.tools import web_search, code_executor, file_manager

tools = [web_search, code_executor, file_manager]
```

---

#### 2.2.2 降低 TCO - Token 优化

##### P1: SOP 复用

**目标**: 相同/相似任务复用历史成功路径，避免重复试错

```mermaid
sequenceDiagram
    participant U as 用户
    participant A as Agent
    participant S as SOP Store

    U->>A: "帮我 Star GitHub 仓库"
    A->>S: 搜索相关 SOP
    S-->>A: 返回历史成功路径
    A->>A: 注入 SOP 到 Prompt
    A->>U: 直接执行正确路径
```

**预期收益**: 相同任务 Token 减少 30-60%

---

##### P1: 上下文压缩

**目标**: 智能压缩历史消息，保留关键信息

| 策略 | 说明 | 适用场景 |
|------|------|----------|
| **摘要压缩** | LLM 生成历史摘要 | 长对话 |
| **选择性保留** | 只保留关键消息 | 多轮闲聊后 |
| **分层存储** | L1 完整/L2 摘要/L3 SOP | 超长会话 |

---

##### P2: 智能缓冲

**目标**: 消息批处理，减少 Task Agent 调用次数

```
消息1 ──┐
消息2 ──┼──► 缓冲区 ──► 批量处理 ──► Task Agent
消息3 ──┘
       (2条 or 10s 触发)
```

**预期收益**: Task Agent 调用减少 50-80%

---

#### 2.2.3 提升 Agent 效果

##### P1: SOP 提取与应用

**目标**: 从成功任务自动提炼可复用技能

```mermaid
graph TB
    subgraph "提取阶段"
        T[Task 完成] --> E[复杂度评估]
        E -->|≥2| S[SOP Agent 提取]
        E -->|<2| X[跳过]
        S --> ST[存储到 Space]
    end

    subgraph "应用阶段"
        Q[新任务] --> SE[搜索 SOP]
        SE --> IN[注入 Prompt]
        IN --> EX[执行]
    end
```

**SOP 结构**:
```json
{
  "use_when": "何时使用",
  "preferences": "用户偏好",
  "tool_sops": [
    {"tool_name": "goto", "action": "导航到目标页面"},
    {"tool_name": "click", "action": "点击目标按钮"}
  ]
}
```

**效果**:
- 减少步骤：直接走正确路径
- 提升成功率：复用验证过的方法
- 效果更优：融合用户偏好

---

##### P1: 用户偏好积累

**目标**: 记住用户约束，个性化 Agent 行为

```
Session 1: 用户说 "我喜欢用 Outlook"
    ↓ 提取偏好
Session 2: Agent 自动用 Outlook 登录
```

**偏好类型**:
- 工具偏好：使用特定工具/服务
- 风格偏好：输出格式、语气
- 约束偏好：禁止某些操作

---

##### P2: SOP/Skill 市场

**目标**: 共享和复用社区沉淀的高质量 SOP

```
┌─────────────────────────────────────┐
│  Skill Market                       │
├─────────────────────────────────────┤
│ Categories:                         │
│  ├─ GitHub Operations      [120]    │
│  ├─ Database Queries       [85]     │
│  ├─ API Integration        [67]     │
│  └─ Browser Automation     [203]    │
├─────────────────────────────────────┤
│ Top Skills:                         │
│  ★★★★★ Star GitHub Repo   (2.3k uses)│
│  ★★★★☆ Query PostgreSQL   (1.8k uses)│
│  ★★★★☆ Send Email         (1.5k uses)│
└─────────────────────────────────────┘
```

---

### 2.3 优先级汇总

| 优先级 | 功能 | 核心价值 |
|--------|------|----------|
| **P0** | 统一 Session 存储 | 基础能力，必须有 |
| **P0** | 即插即用 SDK | 开发者入口 |
| **P1** | 主流框架无感接入 | 降低迁移成本 |
| **P1** | 可视化 Dashboard | 调试/运维必备 |
| **P1** | Context 操作 | 核心差异化 |
| **P1** | 任务状态/监控 | 可观测性 |
| **P1** | SOP 提取与应用 | 效果提升核心 |
| **P1** | 上下文压缩 | Token 优化核心 |
| **P1** | 用户偏好积累 | 个性化 |
| **P2** | Prompt 模板市场 | 生态扩展 |
| **P2** | Built-in Tool | 开发便利 |
| **P2** | 智能缓冲 | Token 优化增强 |
| **P2** | SOP/Skill 市场 | 社区生态 |

---

## 3. 云服务架构

### 3.1 整体架构

```mermaid
graph TB
    subgraph "Client Layer"
        SDK_PY["Python SDK"]
        SDK_TS["TypeScript SDK"]
        FW["框架集成<br/>LangGraph/OpenAI"]
    end

    subgraph "API Gateway"
        GW["API Gateway<br/>认证/限流/路由"]
    end

    subgraph "Service Layer"
        SS["Session Service<br/>会话管理"]
        TS["Task Service<br/>任务追踪"]
        CS["Context Service<br/>上下文操作"]
        LS["Learn Service<br/>SOP学习"]
        ES["Experience Service<br/>经验搜索"]
    end

    subgraph "Core Engine"
        TA["Task Agent<br/>任务提取"]
        SA["SOP Agent<br/>技能提炼"]
        CA["Compress Agent<br/>上下文压缩"]
    end

    subgraph "Storage Layer"
        PG[(PostgreSQL<br/>元数据+向量)]
        S3[(对象存储<br/>消息/文件)]
        REDIS[(Redis<br/>缓存/锁)]
        MQ[消息队列]
    end

    subgraph "Dashboard"
        UI["Web Dashboard"]
    end

    SDK_PY & SDK_TS & FW --> GW
    GW --> SS & TS & CS & LS & ES
    SS & TS --> MQ --> TA & SA & CA
    TA & SA & CA --> PG & S3 & REDIS
    SS & CS & LS & ES --> PG & S3
    UI --> GW
```

### 3.2 数据模型

```mermaid
erDiagram
    Project ||--o{ Session : contains
    Project ||--o{ Space : contains
    Session ||--o{ Message : contains
    Session ||--o{ Task : contains
    Session }o--o| Space : connects
    Space ||--o{ Block : contains
    Block ||--o{ Block : parent

    Project {
        uuid id PK
        string name
        json config
    }

    Session {
        uuid id PK
        uuid project_id FK
        uuid space_id FK
        timestamp created_at
    }

    Message {
        uuid id PK
        uuid session_id FK
        uuid parent_id FK
        string role
        string content_pointer
        json metadata
    }

    Task {
        uuid id PK
        uuid session_id FK
        int order
        string status
        json data
    }

    Space {
        uuid id PK
        uuid project_id FK
        string name
    }

    Block {
        uuid id PK
        uuid space_id FK
        uuid parent_id FK
        string type
        string title
        json content
        vector embedding
    }
```

### 3.3 关键流程

#### 消息存储与任务提取

```mermaid
sequenceDiagram
    participant C as Client SDK
    participant A as API
    participant Q as MQ
    participant T as Task Agent
    participant DB as Database

    C->>A: store_message(session_id, msg)
    A->>DB: 写入消息
    A->>Q: 发布消息事件
    A-->>C: 200 OK

    Q->>T: 触发(缓冲满/超时)
    T->>DB: 读取新消息
    T->>T: LLM 分析任务
    T->>DB: 更新 Task 状态
```

#### SOP 学习与应用

```mermaid
sequenceDiagram
    participant T as Task Agent
    participant S as SOP Agent
    participant SP as Space
    participant C as Client

    T->>T: Task 标记为 success
    T->>S: 触发 SOP 提取
    S->>S: 评估复杂度
    S->>S: 提取工具序列
    S->>SP: 存储到 Space

    C->>SP: experience_search(query)
    SP-->>C: 返回匹配的 SOP
    C->>C: 注入到 Agent Prompt
```

### 3.4 部署架构

```mermaid
graph TB
    subgraph "华为云"
        subgraph "接入层"
            LB["负载均衡"]
            APIG["API 网关"]
        end

        subgraph "计算层"
            API["API 服务<br/>多副本"]
            CORE["Core 引擎<br/>多副本"]
        end

        subgraph "存储层"
            RDS["RDS PostgreSQL"]
            OBS["对象存储 OBS"]
            DCS["分布式缓存 DCS"]
            DMS["消息服务 DMS"]
        end

        subgraph "AI 服务"
            LLM["ModelArts LLM"]
            EMB["Embedding 服务"]
        end
    end

    LB --> APIG --> API
    API --> CORE
    API & CORE --> RDS & OBS & DCS & DMS
    CORE --> LLM & EMB
```

### 3.5 扩展性设计

| 维度 | 策略 |
|------|------|
| **水平扩展** | API/Core 无状态，按需扩容 |
| **存储扩展** | PostgreSQL 分库分表 + OBS 无限扩展 |
| **消息队列** | 按 session_id 分区，并行消费 |
| **LLM 调用** | 异步队列 + 限流 + 重试 |
| **多租户** | Project 隔离 + 配额管理 |

---

## 4. 下一步

1. **MVP 范围确定**: P0 + 核心 P1 功能
2. **API 设计**: RESTful API 规格定义
3. **SDK 设计**: Python/TS SDK 接口设计
4. **原型开发**: 核心流程验证
