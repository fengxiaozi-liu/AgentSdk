# Provider 改造设计

## 结论摘要

本次改造的核心方向是：**Provider 不再绑定单个模型，而是升级为 ProviderService，由它根据 ModelID 和可选厂商选择具体 ProviderClient**。

关键决策：

1. 移除 `client.Options`；连接级 options 迁移到 `provider/option.go`。
2. 新增 `client.Request`，让请求上下文集中表达。
3. `models.Model.DefaultMaxTokens` 改为 `models.Model.MaxTokens`。
4. `ProviderConfig` 下保存 `[]ModelConfig`，启动时解析成 `models.Model` 并放入 `ProviderClient`。
5. ProviderService 持有 `ModelID -> []ProviderClient`，为负载均衡、降级、指定厂商调用预留空间。
6. Agent 只传 `ModelID` 和运行上下文，不显式保存选中的 `models.Model` 或 provider。
7. `TitleProvider`、`SummarizerProvider` 已迁移为附加 Agent 配置入口：`TitleAgent`、`SummarizeAgent`，调用时复用统一 ProviderService。

## 当前问题

当前 `client.Options` 混合了三类配置：

- 连接配置：`APIKey`
- 模型配置：`Model`、`MaxTokens`
- Agent 运行配置：`SystemMessage`、`Debug`

这会导致厂商 client 持有完整 `providerOptions llmclient.Options`，例如 OpenAI、Anthropic、Gemini 等 client 都会长期保存一份 provider options。

这个边界不清晰：

- SDK client 创建后已经持有 `APIKey`、`BaseURL` 等连接信息。
- `Model`、`MaxTokens` 是本次模型调用相关信息。
- `SystemMessage`、`Debug` 是 Agent 或运行时上下文。
- 一个 provider 应该可以支持多个模型，而不是创建时绑定一个固定模型。
- `Options` 本身更接近 provider 创建厂商 client 时的连接配置，不应该放在底层 client 包中。

## 目标架构

改造后的调用链分为四层：

```text
Agent
  -> ProviderService
       -> ProviderClient
            -> client.Client
                 -> vendor SDK client
```

职责划分：

| 对象 | 职责 |
| --- | --- |
| Agent | 组织 system prompt、debug、messages、tools，并指定要调用的 ModelID |
| ProviderService | 根据 ModelID 和可选 provider 选择具体 ProviderClient，记录 ActiveTarget，处理路由、降级、负载均衡 |
| ProviderClient | 绑定一个 provider 下的一个具体 runtime model，以及对应厂商 client |
| client.Client | 适配厂商 SDK，负责协议转换和真实请求 |
| models.Model | 表达解析后的模型能力与调用参数 |

## 核心结构

### ProviderConfig

配置保持 provider 维度。用户天然是按厂商填写 key/baseURL，再在厂商下面配置模型。

```go
type ProviderConfig struct {
	Provider models.ModelProvider `json:"provider"`
	APIKey   string               `json:"apiKey"`
	BaseURL  string               `json:"baseURL"`
	Models   []models.ModelConfig `json:"models"`
	Disabled bool                 `json:"disabled"`
}
```

推荐配置入口为 `providers []ProviderConfig`，ProviderService 会把这些 provider 配置统一构建为 `ModelID -> []ProviderClient`。`provider`、`titleProvider`、`summarizerProvider` 可以作为迁移期配置来源，但主 Agent、标题 Agent、总结 Agent 的模型选择入口分别是 `agent`、`titleAgent`、`summarizeAgent`。

这里保存的是 `[]ModelConfig`，不是 `[]Model`。

原因：

- `ModelConfig` 是用户配置层，表达覆盖项。
- `Model` 是运行时解析结果，由 catalog + 用户配置合成。
- 合成后的 `Model` 应和具体 provider/client 一起放入 `ProviderClient`。

### ModelConfig

`ModelConfig` 表达用户对模型的选择和覆盖。

```go
type ModelConfig struct {
	ModelId         ModelID `json:"model_id"`
	APIModel        string  `json:"api_model"`
	MaxTokens       int64   `json:"maxTokens,omitempty"`
	ReasoningEffort string  `json:"reasoning_effort,omitempty"`
	Weight          int     `json:"weight,omitempty"`
	Priority        int     `json:"priority,omitempty"`
	FallbackModelID ModelID `json:"fallback_model_id,omitempty"`
}
```

字段说明：

| 字段 | 说明 |
| --- | --- |
| `ModelId` | 逻辑模型 ID，用于调用和路由 |
| `APIModel` | 厂商 API 实际模型名，可覆盖 catalog |
| `MaxTokens` | 覆盖模型最大输出 token |
| `ReasoningEffort` | reasoning 模型的推理强度 |
| `Weight` | 同优先级下负载均衡权重 |
| `Priority` | 选择优先级，数值越小优先级越高 |
| `FallbackModelID` | 当前模型不可用时的下一级模型 |

### Model

`DefaultMaxTokens` 改为 `MaxTokens`。catalog 中的值可以是默认值，但用户覆盖后它就是当前模型配置的有效最大输出 token。

```go
type Model struct {
	ID                  ModelID `json:"id"`
	Name                string  `json:"name"`
	APIModel            string  `json:"api_model"`
	CostPer1MIn         float64 `json:"cost_per_1m_in"`
	CostPer1MOut        float64 `json:"cost_per_1m_out"`
	CostPer1MInCached   float64 `json:"cost_per_1m_in_cached"`
	CostPer1MOutCached  float64 `json:"cost_per_1m_out_cached"`
	ContextWindow       int64   `json:"context_window"`
	MaxTokens           int64   `json:"max_tokens"`
	CanReason           bool    `json:"can_reason"`
	SupportsAttachments bool    `json:"supports_attachments"`
	ReasoningEffort     string  `json:"reasoning_effort,omitempty"`
	FallbackModelID      ModelID `json:"fallback_model_id,omitempty"`
}
```

解析规则：

```go
model := models.ResolveModel(providerName, modelCfg.ModelId)

if modelCfg.APIModel != "" {
	model.APIModel = modelCfg.APIModel
}
if modelCfg.MaxTokens > 0 {
	model.MaxTokens = modelCfg.MaxTokens
}
if modelCfg.ReasoningEffort != "" {
	model.ReasoningEffort = modelCfg.ReasoningEffort
}
if modelCfg.FallbackModelID != "" {
	model.FallbackModelID = modelCfg.FallbackModelID
}
```

### ProviderClient

`ProviderClient` 是 ProviderService 的最小可调用目标：**某个 provider 下的某个具体模型 + 对应厂商 client**。

```go
type ProviderClient struct {
	Provider models.ModelProvider
	Model    models.Model
	Client   client.Client
	Weight   int
	Priority int
	Disabled bool
}
```

### ProviderService

`baseProvider` 改名为 `ProviderService`。它不再代表单个 provider，而是模型调用服务。

```go
type ProviderService struct {
	clients       map[models.ModelID][]ProviderClient
	activeTargets map[models.ModelID]ProviderClient
}
```

核心含义：

- `clients`：一个 ModelID 可以对应多个 ProviderClient。
- `activeTargets`：记录某个 ModelID 当前实际选中的 ProviderClient。
- 每一个 active target 都是一个 `ProviderClient`，也就是一个 provider 下的一个具体模型和对应 client。

接口建议：

```go
type ProviderService interface {
	SendMessages(ctx context.Context, request client.Request) (*client.Response, error)
	StreamResponse(ctx context.Context, request client.Request) <-chan client.Event
	ActiveTarget(modelID models.ModelID) (ProviderClient, bool)
	AvailableTargets(modelID models.ModelID) []ProviderClient
}
```

## Provider Options 与 Client Request

连接级 options 迁移到 `provider/option.go` 中维护。底层 `client` 包不再定义 `Options`，只保留 `Request`、`Response`、`Event`、`Client` 等调用协议。

建议在 `internal/data/llm/provider/option.go` 中保留创建厂商 client 所需的连接配置：

```go
type Options struct {
	APIKey  string
	BaseURL string
}
```

也可以直接并入已有的 `providerClientOptions`：

```go
type providerClientOptions struct {
	APIKey  string
	BaseURL string

	AnthropicOptions []anthropicclient.Option
	OpenAIOptions    []openaiclient.Option
	GeminiOptions    []geminiclient.Option
	BedrockOptions   []bedrockclient.Option
	CopilotOptions   []copilotclient.Option
}
```

厂商 client 构造函数接收 provider 侧整理好的连接参数，而不是依赖 `client.Options`：

```go
openaiclient.NewClient(providerOptions, clientOptions.OpenAIOptions...)
```

新增统一请求对象：

```go
type Request struct {
	ModelID       models.ModelID
	Provider      models.ModelProvider
	Model         models.Model
	SystemMessage string
	Debug         bool
	Messages      []message.Message
	Tools         []toolcore.BaseTool
}
```

约定：

- Agent 调用 ProviderService 时填 `ModelID`。
- 如果要强制指定厂商，可额外填 `Provider`。
- ProviderService 选择 `ProviderClient` 后补全 `Model`。
- 厂商 client 只读取 `Model`，不参与模型或 provider 选择。

Client 接口：

```go
type Client interface {
	Send(ctx context.Context, request Request) (*Response, error)
	Stream(ctx context.Context, request Request) <-chan Event
}
```

## 构建流程

启动时将 provider 维度配置转换成 model 维度索引。

```go
clients := map[models.ModelID][]ProviderClient{}

for _, providerCfg := range providerCfgs {
	if providerCfg.Disabled {
		continue
	}

	vendorClient := createClient(providerCfg)

	for _, modelCfg := range providerCfg.Models {
		model := resolveConfiguredModel(providerCfg.Provider, modelCfg)
		target := ProviderClient{
			Provider: providerCfg.Provider,
			Model:    model,
			Client:   vendorClient,
			Weight:   modelCfg.Weight,
			Priority: modelCfg.Priority,
		}
		clients[model.ID] = append(clients[model.ID], target)
	}
}
```

构建完成后，ProviderService 可以支持：

- `ModelID` 自动选择 provider。
- `ModelID + Provider` 指定厂商。
- 同一 ModelID 多 provider 负载均衡。
- 同一 ModelID 不同 provider 降级。
- 当前模型不可用后降级到下一级模型。

## 调用流程

Agent 只组织请求语义，不保存选中的模型实体。

```go
request := client.Request{
	ModelID:       selectedModelID,
	Provider:      preferredProvider,
	SystemMessage: systemPrompt,
	Debug:         cfg.Debug,
	Messages:      msgHistory,
	Tools:         a.tools,
}

eventChan := a.providerService.StreamResponse(ctx, request)
```

ProviderService 内部选择目标：

```go
func (s *ProviderService) SendMessages(ctx context.Context, request client.Request) (*client.Response, error) {
	request.Messages = s.cleanMessages(request.Messages)

	target, err := s.selectTarget(request.ModelID, request.Provider)
	if err != nil {
		return nil, err
	}

	request.Model = target.Model
	s.activeTargets[request.ModelID] = target

	return target.Client.Send(ctx, request)
}
```

Agent 如需记录实际使用模型，查询 ProviderService：

```go
target, ok := a.providerService.ActiveTarget(selectedModelID)
```

这可用于 assistant message 的 `Model` 字段、usage tracking、前端展示和降级后的实际 target 展示。

## 选择与降级

第一阶段选择策略：

1. 根据 `ModelID` 找到候选 ProviderClient。
2. 如果 request 指定 `Provider`，先筛选指定厂商。
3. 过滤 disabled 或熔断中的 ProviderClient。
4. 按 `Priority` 升序选择。
5. 同优先级下按 `Weight` 加权轮询。
6. 将结果写入 `activeTargets[modelID]`。

降级策略：

1. 优先在同一个 `ModelID` 下切换不同 provider。
2. 如果同 `ModelID` 下所有 provider 都不可用，再选择下一级模型。
3. 下一级模型通过 `FallbackModelID` 或独立路由配置表达。

第二阶段增强：

- 基于错误类型的重试策略。
- 熔断与恢复。
- 延迟、错误率、使用量统计。
- 跨模型降级策略可配置化。

## Reasoning 配置

Reasoning 相关的用户配置归属 `ModelConfig`，解析后进入 `models.Model`。

原因：

- Reasoning 描述的是“某个模型如何推理”，不是 provider 连接能力。
- 同一个 provider 下可能同时配置普通模型和 reasoning 模型。
- 不同 reasoning 模型可能需要不同 `ReasoningEffort`。

厂商 client 只负责把 `request.Model` 转换成 SDK 参数。

```go
if request.Model.CanReason {
	switch request.Model.ReasoningEffort {
	case "low":
		params.ReasoningEffort = shared.ReasoningEffortLow
	case "medium":
		params.ReasoningEffort = shared.ReasoningEffortMedium
	case "high":
		params.ReasoningEffort = shared.ReasoningEffortHigh
	default:
		params.ReasoningEffort = shared.ReasoningEffortMedium
	}
}
```

Anthropic 的 `shouldThinkFn` 属于厂商实现策略，不作为 provider 配置暴露。它可以保留在 Anthropic client 私有 option 中，或内置在 Anthropic client 内部。

## 附加 Agent

`TitleProvider`、`SummarizerProvider` 不再作为 Agent 直接持有的特殊 provider。当前运行时已升级为附加 Agent：

- `TitleAgent`：负责标题生成。
- `SummarizeAgent`：负责历史压缩、会话总结、上下文摘要。

这样它们可以复用统一的 ProviderService、模型选择、降级、debug、usage tracking 机制。配置层新增 `agent`、`titleAgent`、`summarizeAgent` 作为模型选择入口；旧 provider 字段只作为 ProviderService target 来源和默认 Agent 配置推导来源。

建议形态：

```go
type Agent struct {
	providerService ProviderService
	modelID         models.ModelID
	systemPrompt    string
	tools           []toolcore.BaseTool
}

type RuntimeAgents struct {
	Main      *Agent
	Title     *Agent
	Summarize *Agent
}
```

每个 Agent 只持有自己的任务语义和默认 `ModelID`，不直接持有固定 provider/client。

## 包位置评估

### 短期：保留在 `internal/data/llm/provider`

优点：

- 迁移成本低。
- 当前代码已经在这里创建厂商 client。
- 第一阶段可以减少包移动带来的干扰。

缺点：

- ProviderService 会承担运行时路由、负载均衡、降级、active 状态，不再只是 data 层适配器。
- `provider` 名称容易和厂商 provider、Wire `ProviderSet` 混淆。

### 长期：移动到 `internal/provider`

最终命名采用 `internal/provider`。这里的 provider 表示应用级模型供应服务，不再是单个厂商 provider。

```text
internal/
  provider/
    provider_service.go
    router.go
    config.go
  data/
    llm/
      client/
      models/
```

理由：

- `client` 和 `models` 是底层 LLM 能力，可以继续放在 `internal/data/llm`。
- `ProviderService` 是应用运行时服务，负责模型供应、路由、active target、负载均衡与降级。
- `internal/provider` 能突出它是 Agent 调用模型的统一 provider 服务入口。
- 厂商 client 与模型 catalog 仍留在 `internal/data/llm`，避免 ProviderService 和厂商 SDK 适配层混在一起。

## 迁移计划

建议分阶段实施，降低一次性改动风险。

### Phase 1：请求对象与模型字段

1. 将 `models.Model.DefaultMaxTokens` 改为 `MaxTokens`。
2. 同步调整 `models.json` 和所有使用点。
3. 新增 `client.Request`。
4. 将连接级 options 迁移到 `provider/option.go`，移除 `client.Options`。
5. 调整 `Client.Send/Stream` 接口。

### Phase 2：厂商 client 去状态化

1. 移除各厂商 client 中的 `providerOptions llmclient.Options`。
2. 厂商 client 从 `request.Model`、`request.SystemMessage`、`request.Debug` 读取请求级信息。
3. 厂商 client 内部只保留 SDK client 和厂商私有 options。

### Phase 3：ProviderService 路由

1. 将 `ProviderConfig.ModelConfig` 改为 `ProviderConfig.Models`。
2. 引入 `ProviderClient`。
3. 引入 `ProviderService.clients map[ModelID][]ProviderClient`。
4. 实现 `selectTarget(modelID, provider)`。
5. 替换现有 `provider.Model()` 依赖，改为 `ProviderService.ActiveTarget(modelID)`。

### Phase 4：附加 Agent 与位置调整

1. 将 `TitleProvider`、`SummarizerProvider` 迁移为 `TitleAgent`、`SummarizeAgent`。
2. ProviderService 稳定后，从 `internal/data/llm/provider` 移动到 `internal/provider`。
3. Agent 通过 Wire 注入 `internal/provider.Service`，主 Agent、TitleAgent、SummarizeAgent 都构造 `client.Request` 调用 ProviderService。
3. 补充负载均衡、降级、熔断和统计能力。

## 已确认决策

1. `ProviderConfig` 下保存 `[]ModelConfig`，启动时解析为 `models.Model` 并放入 `ProviderClient`。
2. `TitleProvider`、`SummarizerProvider` 已迁移为附加 Agent，即 `TitleAgent` 与 `SummarizeAgent`。
3. 降级先按同 `ModelID` 切换不同 provider；同模型都不可用时，再选择下一级模型。
4. `ActiveTarget` 表示已选择的具体目标：某个 provider 下的某个具体模型。
5. 调用时支持指定厂商，即 `ModelID + Provider`。
6. `client.Options` 不再保留，连接级 options 归 `provider/option.go` 管理。
