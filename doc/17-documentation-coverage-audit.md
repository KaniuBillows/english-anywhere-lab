# English Anywhere Lab - 文档覆盖审计与补全清单

## 1. 目的与范围
本文件用于对 `doc/` 下的产品与技术文档进行一次完整覆盖审计，识别关键核心功能的描述缺口，并记录补全结果。

审计日期：2026-03-04  
适用版本：PRD v0.4 及以后

## 2. 审计方法
- 以核心能力闭环为主线检查：`计划 -> 输入 -> 复习 -> 输出 -> 进度 -> 同步 -> 资源生成`。
- 以“定义-契约-落地-验证”四层检查一致性：
  - 定义层：PRD / 类型体系
  - 契约层：OpenAPI
  - 落地层：架构、存储、LLM、DB
  - 验证层：测试、指标

## 3. 关键缺口与补全状态

| 缺口编号 | 问题描述 | 影响 | 补全状态 | 对应文档 |
|---|---|---|---|---|
| GAP-01 | 学习包“类型定义”缺少统一规范 | 后端/前端/AI 各自实现不一致 | 已补全 | `doc/16-learning-pack-type-system.md` |
| GAP-02 | 类型体系未完全映射到 OpenAPI 字段 | 契约无法驱动前后端一致开发 | 已补全 | `doc/07-openapi.yaml`, `doc/api-openapi.yaml` |
| GAP-03 | Go 架构未显式纳入 TTS 任务链路 | 音频能力落地路径不清晰 | 已补全 | `doc/05-backend-architecture-go.md` |
| GAP-04 | 对象存储规范缺少 TTS 对象键约定 | 音频资产命名和去重难统一 | 已补全 | `doc/10-storage-abstraction-spec.md` |
| GAP-05 | LLM 接入规范缺少类型字段强校验要求 | 生成数据结构漂移风险 | 已补全 | `doc/11-llm-integration-spec.md` |
| GAP-06 | 指标字典未纳入 `pack_template/exercise_type` | 无法分析题型效果与中断点 | 已补全 | `doc/12-analytics-metrics-dictionary.md` |
| GAP-07 | 测试策略未覆盖类型契约与题型渲染稳定性 | 回归无法兜底核心业务规则 | 已补全 | `doc/14-test-strategy.md` |
| GAP-08 | PRD 与类型规范间缺少单一真值引用约束 | 需求执行存在歧义 | 已补全 | `doc/01-prd.md` |

## 4. 当前文档单一真值（SSOT）划分
- 产品需求基线：`doc/01-prd.md`
- 学习包与题型字典：`doc/16-learning-pack-type-system.md`
- M1 已实现接口契约：`doc/07-openapi-m1.yaml`
- 全量路线图接口契约：`doc/07-openapi.yaml`
- 存储抽象：`doc/10-storage-abstraction-spec.md`
- LLM 接入：`doc/11-llm-integration-spec.md`
- TTS 选型：`doc/15-tts-selection-offline-cpu.md`

## 5. 审计后的执行约束
1. 新增学习任务时，必须同时定义 `pack_template` 与 `exercise_type`。
2. OpenAPI 新增或变更任务字段时，必须与 `doc/16` 字典一致。
3. AI 生成模板上线前，必须通过类型合法性与覆盖率 QC。
4. 统计口径必须能按 `pack_template/exercise_type/evaluation_mode` 切片。
5. 回归测试必须覆盖类型校验、去重与降级路径。

## 6. 后续维护机制
- 每次迭代发布前执行一次“文档一致性检查”：PRD vs OpenAPI vs 测试清单。
- 若出现“实现新增功能但文档无定义”，该功能不得进入正式验收。
- 文档变更遵循“先定义，后实现，最后验收”顺序。
