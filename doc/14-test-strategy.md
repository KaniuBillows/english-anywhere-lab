# English Anywhere Lab - 测试策略（Go 后端 + Web 客户端）

## 1. 目标
保证核心学习链路在迭代中可回归、可验证、可发布。

## 2. 测试分层

### 2.1 单元测试
- 目标：领域逻辑（review/scheduler/plan/llm adapter）
- 要求：
  - Go 使用 table-driven tests
  - 覆盖正常路径 + 错误路径 + 边界

### 2.2 集成测试
- 目标：API + DB + queue 的协同
- 场景：注册登录、今日计划、复习提交、AI 任务状态流转

### 2.3 契约测试
- 基于 [openapi.yaml](/Users/bennyshi/dev/english-anywhere-lab/07-openapi.yaml) 校验：
  - 请求参数
  - 响应结构
  - 错误码

### 2.4 端到端测试
- 目标：Web 主流程
- 场景：Onboarding -> 今日学习 -> 复习 -> 看板

## 3. 核心测试清单
- FSRS 调度正确性（含幂等）
- 离线事件补发与去重
- SQLite 锁竞争下的提交稳定性
- 文件存储 Local/S3 双 provider 回归
- LLM 生成失败重试与 QC 拦截

## 4. 质量门禁（建议）
- 单测覆盖率 >= 70%
- 关键域（review/sync/scheduler）>= 85%
- `go test -race ./...` 全通过
- OpenAPI 契约测试全通过
- 关键 E2E 用例全绿

## 5. 环境分层
- `dev`：SQLite + Local FS + Mock LLM
- `staging`：SQLite/Postgres（二选一）+ Local/S3 + Real LLM(低配额)
- `prod`：与 staging 同配置模板

## 6. 数据准备
- 固定 seed 用户和学习包
- 固定 FSRS 参数与测试时间（避免时间漂移导致 flaky）

## 7. 回归节奏
- 每次 PR：单元 + 契约测试
- 每日合并：集成测试
- 发布前：E2E 全量 + 冒烟 + 性能基线

## 8. 性能与稳定性
- 压测目标（MVP）：
  - 50 RPS（不含 LLM）
  - P95 < 800ms
- 稳定性测试：
  - 复习提交连续 1 小时无错误堆积
  - 任务队列在错误注入后可恢复

## 9. 缺陷分级
- P0：数据错乱/进度丢失/认证漏洞
- P1：主流程不可用
- P2：非主流程错误
- P3：体验问题

## 10. 发布检查表
1. 数据库迁移在 staging 演练通过
2. 回归报告无 P0/P1
3. 关键指标告警阈值已配置
4. 回滚方案可执行
