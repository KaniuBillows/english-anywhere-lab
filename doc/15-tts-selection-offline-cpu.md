# English Anywhere Lab - 离线 TTS 选型与落地方案（纯 CPU）

## 1. 文档目标
定义一个可直接落地的 TTS 方案，用于“AI 生成学习资源”后的音频生产链路，满足以下硬约束：
- 纯 CPU 部署（不依赖 GPU）
- 可离线生成（无外网也能工作）
- 低成本（不依赖按量付费语音 API）
- 生成结果写入既有对象存储抽象（Local FS / S3）

## 2. 场景与边界
### 2.1 主要场景
- 在 `pack/cards` 生成完成后，为卡片示例句或词条批量生成英语音频。
- 音频生成在后端异步执行，不阻塞主 API 请求。

### 2.2 本文不覆盖
- 实时对话 TTS（<300ms 的低延迟流式播报）
- 多说话人克隆、情感语音等高级能力
- 文件跨存储 provider 的自动迁移

## 3. 选型约束（硬性）
- 后端技术栈：Go
- 部署资源：单机 CPU 可运行
- 存储：必须兼容 `doc/10-storage-abstraction-spec.md`（`local|s3`）
- 当前阶段：个人/小规模使用，优先稳定与维护成本

## 4. 候选方案对比
| 方案 | CPU 离线 | Go 接入复杂度 | 音质 | 许可与合规风险 | 结论 |
|---|---|---|---|---|---|
| Sherpa-ONNX（本地库） | 强 | 中（库接入） | 中-上 | 低（Apache-2.0） | 推荐为默认方案 |
| Piper（CLI） | 强 | 低（命令行封装） | 上 | 中（引擎与 voice 许可需逐项核查） | 可作为可选 provider |
| eSpeak NG | 强 | 低 | 低（偏机械） | 中（GPL 系，需评估） | 不建议用于正式学习音频 |

说明：
- 如果优先“音质 + 极简接入”，Piper 很有吸引力。
- 如果优先“Go 本地库 + 许可稳健 + 纯离线可控”，Sherpa-ONNX 更适合当前项目长期维护。

## 5. 决策
### 5.1 M1 默认决策
- 默认 TTS 引擎：`sherpa_onnx`
- 运行模式：离线、本地模型、CPU-only
- 生成格式：`wav`（M1 默认）
- 存储：写入统一对象存储抽象（Local/S3），并将 URL 回填到业务字段（如 `cards.audio_url`）

### 5.2 可选增强（不阻塞 M1）
- 新增 `piper_cli` provider（通过配置切换）
- 增加转码链路（`wav -> mp3/opus`）降低对象存储占用

## 6. 系统设计（与现有架构对齐）
对应文档：
- `doc/05-backend-architecture-go.md`
- `doc/10-storage-abstraction-spec.md`
- `doc/11-llm-integration-spec.md`

### 6.1 异步链路
```mermaid
graph TD
    A[Pack/Cards Generated] --> B[Create TTS Jobs]
    B --> C[(DB-backed Job Queue)]
    C --> D[Worker: TTS Provider]
    D --> E[ObjectStore.Put(local|s3)]
    E --> F[Update cards.audio_url]
```

流程说明：
1. AI 资源生成成功后，提取需要配音的文本（如 `front_text`、`example_text`）。
2. 为每条文本创建 TTS 任务，任务状态沿用现有作业状态机（`queued/running/success/failed`）。
3. Worker 调用本地 TTS provider 生成音频流。
4. 音频写入对象存储，得到对象 key 或可访问 URL。
5. 回写 `cards.audio_url`，并记录任务结果。

### 6.2 去重策略（避免重复生成）
使用“确定性对象键”实现天然去重：
- 归一化文本：trim + 多空格折叠 + 小写（仅用于 hash，不影响实际朗读文本）
- 哈希输入：`normalized_text + voice + speed + format + sample_rate`
- 对象键：`tts/en/{voice}/{format}/{sha256}.wav`

写入前先 `Stat(object_key)`：
- 已存在：直接复用 URL，不重复生成
- 不存在：执行生成并上传

## 7. Go 接口建议
```go
type SynthesizeRequest struct {
    Text       string
    Voice      string
    Speed      float32
    Format     string // wav|mp3
    SampleRate int
}

type SynthesizeResult struct {
    Audio      io.ReadCloser
    DurationMs int
    ContentType string
}

type TTSProvider interface {
    Synthesize(ctx context.Context, req SynthesizeRequest) (SynthesizeResult, error)
}
```

适配层位置建议：
- `internal/tts/provider.go`
- `internal/tts/providers/sherpaonnx`
- `internal/tts/providers/pipercli`（可选）

## 8. 配置项建议
```env
# TTS 主开关
TTS_ENABLED=true

# provider: sherpa_onnx | piper_cli
TTS_PROVIDER=sherpa_onnx

# 模型与语音配置
TTS_MODEL_DIR=./models/tts/en
TTS_VOICE=en_default_female
TTS_SAMPLE_RATE=22050
TTS_SPEED=1.0
TTS_OUTPUT_FORMAT=wav
TTS_MAX_TEXT_CHARS=280

# worker 控制
TTS_WORKER_CONCURRENCY=2
TTS_RETRY_MAX=2

# 可选：若启用转码
TTS_TRANSCODE=false
TTS_TRANSCODE_FORMAT=mp3
TTS_FFMPEG_BIN=ffmpeg
```

## 9. 数据与 API 影响
### 9.1 数据库
M1 不强制新增表：
- 直接复用 `cards.audio_url` 字段承载音频地址。
- 如需更细粒度追踪（版本、voice、hash、耗时），后续可新增 `tts_assets` 元数据表。

### 9.2 API
M1 不新增外部 API：
- 音频 URL 继续通过现有卡片查询接口返回。
- 生成过程由异步任务驱动，状态通过既有 job 查询能力观察。

## 10. 成本估算（个人项目口径）
### 10.1 计算成本
- 云 API 费用：0（离线本地模型）
- 成本主要来自 CPU 时间与对象存储空间

### 10.2 存储估算公式
`size_bytes = count * avg_seconds * bitrate_kbps / 8 * 1000`

示例（MP3 64kbps 便于估算）：
- 20,000 条句子
- 平均 6 秒/条
- 约 `20,000 * 6 * 64 / 8 / 1024 / 1024 ≈ 0.92 GB`

若使用 WAV，体积通常显著增大（可达 MP3 的数倍），建议在规模增长后开启转码。

## 11. 验收标准（M1）
- 纯 CPU 环境可完成端到端生成与落盘。
- 在 `FILES_PROVIDER=local` 与 `FILES_PROVIDER=s3` 两种模式下均可写入并读取音频。
- 对同一文本（同 voice/speed/format）重复请求不会重复生成（命中去重键）。
- 失败任务可重试，最大重试后进入 `failed`，并有可审计错误信息。

## 12. 风险与规避
- 模型音质波动：先做 30~50 条标准句集主观听测。
- CPU 吞吐有限：控制并发（默认 2），防止与主 API 抢资源。
- 许可风险：引擎与 voice 模型分别记录许可证，避免混用不明来源模型。

## 13. 实施顺序建议
1. 落地 `TTSProvider` 抽象与 `sherpa_onnx` provider。
2. 在 `pack generation` 成功后补充 TTS job 创建与 worker 消费。
3. 接入对象存储写入与 `cards.audio_url` 回填。
4. 增加去重、重试、失败审计。
5. （可选）加入转码与 `piper_cli` provider。
