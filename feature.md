## 更新以下部分

### 补充
1. 删除了memory，因为当前memory就是通过message来完成的， 不需要额外的message
2. 当前实体的关联关系是这样的session是贯穿会话的上下文， session下有对应的message， history

### 问题

#### LLM部分
1. 现在model与provider耦合验证，应该是provider中引用model， model中不需要有provider的相关信息
2. agent.go 中会显示当前支持的model信息，其实不需要这个，我们不需要这个支持信息, 下面这一部分都是不需要的，让用户配置对应的model即可，如果不对在请求上游的时候会失败
```go
/ Model IDs
const ( // GEMINI
	// Bedrock
	BedrockClaude37Sonnet ModelID = "bedrock.claude-3.7-sonnet"
)

const (
	ProviderBedrock ModelProvider = "bedrock"
	// ForTests
	ProviderMock ModelProvider = "__mock"
)

// Providers in order of popularity
var ProviderPopularity = map[ModelProvider]int{
	ProviderCopilot:    1,
	ProviderAnthropic:  2,
	ProviderOpenAI:     3,
	ProviderGemini:     4,
	ProviderGROQ:       5,
	ProviderOpenRouter: 6,
	ProviderBedrock:    7,
	ProviderAzure:      8,
	ProviderVertexAI:   9,
}

var SupportedModels = map[ModelID]Model{
	//
	// // GEMINI
	// GEMINI25: {
	// 	ID:                 GEMINI25,
	// 	Name:               "Gemini 2.5 Pro",
	// 	Provider:           ProviderGemini,
	// 	APIModel:           "gemini-2.5-pro-exp-03-25",
	// 	CostPer1MIn:        0,
	// 	CostPer1MInCached:  0,
	// 	CostPer1MOutCached: 0,
	// 	CostPer1MOut:       0,
	// },
	//
	// GRMINI20Flash: {
	// 	ID:                 GRMINI20Flash,
	// 	Name:               "Gemini 2.0 Flash",
	// 	Provider:           ProviderGemini,
	// 	APIModel:           "gemini-2.0-flash",
	// 	CostPer1MIn:        0.1,
	// 	CostPer1MInCached:  0,
	// 	CostPer1MOutCached: 0.025,
	// 	CostPer1MOut:       0.4,
	// },
	//
	// // Bedrock
	BedrockClaude37Sonnet: {
		ID:                 BedrockClaude37Sonnet,
		Name:               "Bedrock: Claude 3.7 Sonnet",
		Provider:           ProviderBedrock,
		APIModel:           "anthropic.claude-3-7-sonnet-20250219-v1:0",
		CostPer1MIn:        3.0,
		CostPer1MInCached:  3.75,
		CostPer1MOutCached: 0.30,
		CostPer1MOut:       15.0,
	},
}

func init() {
	maps.Copy(SupportedModels, AnthropicModels)
	maps.Copy(SupportedModels, OpenAIModels)
	maps.Copy(SupportedModels, GeminiModels)
	maps.Copy(SupportedModels, GroqModels)
	maps.Copy(SupportedModels, AzureModels)
	maps.Copy(SupportedModels, OpenRouterModels)
	maps.Copy(SupportedModels, XAIModels)
	maps.Copy(SupportedModels, VertexAIGeminiModels)
	maps.Copy(SupportedModels, CopilotModels)
}
```

3. prompt系统提示词， 这个最好是放在一个文件中， 用户可以传递自己的系统提示词，其实它就是一个文本，这里可以做薄，简单一点，只传递系统提示词就好

#### 架构问题
1. 现在是message/session/history的service是分开的，但是底层都是共用同一个Queries, 现在缺少一个repo层，正确的部分应该是 service -> repo -data数据源，后面可以切不同的数据源
2. data数据源的操作使用gorm来操作，不使用原生的数据来操作
3. data数据源通过