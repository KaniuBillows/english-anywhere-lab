# English Anywhere Lab - 开发执行 Checklist（可直接给 Claude 执行）

## 0. 使用说明
- 本清单用于“需求已定，进入实现”的逐项验收。
- 任何条目未完成前，不应宣称对应功能已完成。
- OpenAPI 单一真值：`backend/api/openapi.yaml`。

## 1. 契约与模型
- [ ] 所有新增/变更接口先更新 `backend/api/openapi.yaml`，再改代码。
- [ ] 新增任务类型时，先在 `doc/16-learning-pack-type-system.md` 更新字典。
- [ ] 每个学习包包含 `pack_template`。
- [ ] 每个任务包含 `exercise_type`。
- [ ] 输出任务包含 `evaluation_mode`（`objective|rubric|llm_hybrid`）。
- [ ] OpenAPI 变更后执行契约校验并通过。

## 2. 后端实现（Go）
- [ ] `pack` 生成链路对 `pack_template/exercise_type` 做白名单校验。
- [ ] 生成结果执行 JSON Schema + QC 双重校验。
- [ ] 复习提交使用服务端时间作为调度真值。
- [ ] 关键写入具备幂等（`client_event_id` / `idempotency_key`）。
- [ ] 输出提交失败场景状态一致（无“返回失败但数据看似成功”）。
- [ ] 任务来源必须落库；禁止虚拟不落库任务进入正式计分。

## 3. AI 资源生成
- [ ] 提示词显式传入 `pack_template` 与可用 `exercise_type` 白名单。
- [ ] 生成结果包含完整字段：`title/level/domain/exercise_type/prompt`。
- [ ] 模板必选题型覆盖率达到 `doc/16` 要求。
- [ ] 失败重试链路可观测，错误码可追踪。

## 4. TTS（纯 CPU 离线）
- [ ] 启用离线 CPU provider（默认 `sherpa_onnx`）。
- [ ] TTS 异步任务与主学习流程解耦（不阻塞主请求）。
- [ ] 对象键采用确定性哈希，命中同文本同参数直接复用。
- [ ] 同时验证 `FILES_PROVIDER=local|s3` 两种落盘模式。
- [ ] 回填 `cards.audio_url` 并可被前端播放。
- [ ] 模型与语音许可证记录在案（合规可审计）。

## 5. 前端渲染与交互
- [ ] 前端按 `exercise_type` 分发组件渲染（不是自由文本判断）。
- [ ] 未识别题型有通用降级组件并打埋点。
- [ ] 输出题按 `evaluation_mode` 选择判分反馈 UI。
- [ ] 今日学习主流程支持断点续学。
- [ ] 移动端与桌面端主流程均可用。

## 6. 离线同步
- [ ] IndexedDB 事件队列按 `occurred_at + client_seq` 回放。
- [ ] `/sync/events` 逐条 ack 且可区分 `accepted/duplicate/rejected`。
- [ ] 重复上报不重复统计。
- [ ] 事件上下文带 `pack_template/exercise_type/evaluation_mode`（可空）。

## 7. 埋点与分析
- [ ] 核心事件包含 `pack_template/exercise_type/evaluation_mode`（可空）。
- [ ] 可输出按模板、题型的完成率与中断率报表。
- [ ] 指标口径与 `doc/12` 一致。

## 8. 测试与发布门禁
- [ ] 单测覆盖正常/异常/边界路径。
- [ ] 集成测试覆盖：注册登录、计划、复习、输出、AI 生成。
- [ ] 契约测试基于 `backend/api/openapi.yaml` 全通过。
- [ ] `go test -race ./...` 全通过。
- [ ] TTS 去重与对象存储双 provider 回归通过。
- [ ] 发布前无 P0/P1 缺陷。

## 9. 文档一致性（发布前必须）
- [ ] PRD、类型字典、OpenAPI、测试策略同步更新。
- [ ] 若实现新增能力，必须补文档（先定义后实现）。
- [ ] 执行 `doc/17-documentation-coverage-audit.md` 的一致性检查。
