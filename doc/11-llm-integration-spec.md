# English Anywhere Lab - 大模型接入规范（OpenAI Compatible）

## 1. 目标
统一封装 LLM 调用，支持标准 API 协议并可按配置切换模型和厂商。

## 2. 设计原则
- Provider 无关：业务侧不直接依赖具体模型 SDK。
- JSON-first：所有生成任务要求结构化 JSON 输出。
- 可观测：记录请求耗时、token 用量、错误率、重试次数。
- 可控成本：设置并发、重试和预算上限。

## 3. Provider 模式
- `responses`：`POST /v1/responses`
- `chat_completions`：`POST /v1/chat/completions`

通过配置决定使用哪一种：
```env
LLM_PROVIDER=openai_compatible
LLM_BASE_URL=https://api.openai.com/v1
LLM_API_KEY=***
LLM_MODEL=gpt-5.3-codex
LLM_API_MODE=responses
LLM_TIMEOUT_SEC=60
LLM_MAX_RETRIES=2
```

## 4. 统一接口（Go）
```go
type GenerateRequest struct {
    TaskType        string
    Model           string
    Temperature     float64
    SystemPrompt    string
    UserPrompt      string
    JSONSchema      []byte
    MaxOutputTokens int
}

type GenerateResult struct {
    RawText      string
    JSONPayload  []byte
    PromptTokens int
    OutputTokens int
    LatencyMs    int64
    ProviderReqID string
}

type LLMClient interface {
    Generate(ctx context.Context, req GenerateRequest) (GenerateResult, error)
}
```

## 5. curl 示例

### 5.1 Responses API（推荐）
```bash
curl "$LLM_BASE_URL/responses" \
  -H "Authorization: Bearer $LLM_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.3-codex",
    "input": [
      {"role": "system", "content": "Return JSON only."},
      {"role": "user", "content": "Generate a 7-day B1 tech English pack"}
    ],
    "temperature": 0.3
  }'
```

### 5.2 Chat Completions API
```bash
curl "$LLM_BASE_URL/chat/completions" \
  -H "Authorization: Bearer $LLM_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.3-codex",
    "messages": [
      {"role": "system", "content": "Return JSON only."},
      {"role": "user", "content": "Generate 50 A2 travel cards"}
    ],
    "temperature": 0.2
  }'
```

备注：模型名是否可用取决于你配置的服务端；本系统不硬编码模型名。

## 6. Go 侧实现建议

### 6.1 SDK 模式（若 provider 提供官方 SDK）
- 在 `internal/llm/providers/openai` 封装 SDK 细节。
- SDK 仅在 provider adapter 内可见。

### 6.2 标准 HTTP 模式（推荐兜底）
- 使用 `net/http` + `context.WithTimeout`。
- 统一处理 429/5xx 的重试退避。

示例（Go，Responses API）：
```go
func (c *Client) Generate(ctx context.Context, req GenerateRequest) (GenerateResult, error) {
    payload := map[string]any{
        "model": req.Model,
        "input": []map[string]any{
            {"role": "system", "content": req.SystemPrompt},
            {"role": "user", "content": req.UserPrompt},
        },
        "temperature": req.Temperature,
    }
    b, _ := json.Marshal(payload)

    httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+\"/responses\", bytes.NewReader(b))
    if err != nil {
        return GenerateResult{}, fmt.Errorf(\"build request: %w\", err)
    }
    httpReq.Header.Set(\"Authorization\", \"Bearer \"+c.apiKey)
    httpReq.Header.Set(\"Content-Type\", \"application/json\")

    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return GenerateResult{}, fmt.Errorf(\"call provider: %w\", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode >= 400 {
        body, _ := io.ReadAll(resp.Body)
        return GenerateResult{}, fmt.Errorf(\"provider status=%d body=%s\", resp.StatusCode, string(body))
    }

    // parse provider response into GenerateResult (omitted)
    return GenerateResult{}, nil
}
```

## 7. 重试与限流
- 重试触发：网络错误、429、5xx。
- 最大重试：`LLM_MAX_RETRIES`（默认 2）。
- 退避策略：指数退避 + 抖动（200ms, 600ms, 1200ms）。
- 并发上限：同一用户并发生成任务 <= 1，全局 worker 并发 <= 4。

## 8. 结构化输出校验
- 先做 JSON 解析。
- 再做 JSON Schema 校验。
- 再做业务 QC 校验（难度、重复度、字段完整性）。
- 任一步失败则进入“修复重生成”流程。

类型体系强约束（与 `doc/16-learning-pack-type-system.md` 对齐）：
- 学习包输出必须包含合法 `pack_template`。
- 任务级输出必须包含合法 `exercise_type`。
- 输出任务必须标注 `evaluation_mode`（`objective|rubric|llm_hybrid`）。
- 模板必选题型覆盖率不达标时，判定为 `LLM_SCHEMA_VIOLATION`。

## 9. 错误模型
- `LLM_TIMEOUT`
- `LLM_RATE_LIMITED`
- `LLM_INVALID_JSON`
- `LLM_SCHEMA_VIOLATION`
- `LLM_PROVIDER_ERROR`

所有错误写入 `ai_generation_jobs.error_message` 并带错误码。

## 10. 成本控制
- 记录每次请求 token（若 provider 返回）。
- 对每个用户设置日限额：
  - 免费层：2 次生成/天
  - 付费层（后续）：可配置
- 超限直接拒绝，不入队。

## 11. 安全要求
- API key 仅存环境变量或密钥管理器。
- 生成请求写日志时脱敏用户文本。
- 禁止在错误日志打印完整提示词和原文答案。
