# English Anywhere Lab

本目录包含英语学习产品的第一版产品方案与技术落地文档。

## 文件说明
- `doc/00-concept-plan.md`：产品构思计划（方法论、目标、里程碑）
- `doc/01-prd.md`：可执行 PRD（功能需求、验收标准、指标）
- `doc/02-ia-and-key-flows.md`：信息架构与关键流程图（Mermaid）
- `doc/03-database-design.md`：数据库表设计与 SQL 草案
- `doc/04-ai-resource-prompt-templates.md`：AI 资源生成与质检提示词模板
- `doc/05-backend-architecture-go.md`：Go 后端架构（模块、拓扑、配置、异步链路）
- `doc/06-database-selection.md`：数据库选型决策（SQLite 默认、Postgres 可选）
- `doc/07-openapi-m1.yaml`：M1 单一真值接口契约（仅已实现接口）
- `doc/07-openapi.yaml`：全量路线图接口契约（含后续阶段，不作为 M1 实现验收基线）
- `doc/08-fsrs-scheduling-spec.md`：FSRS 调度规范（状态机、日切、幂等）
- `doc/09-offline-sync-spec.md`：离线同步与冲突处理规范（PWA）
- `doc/10-storage-abstraction-spec.md`：文件存储抽象（Local/S3 可配置）
- `doc/11-llm-integration-spec.md`：LLM 接入规范（OpenAI 兼容 API）
- `doc/12-analytics-metrics-dictionary.md`：指标口径与埋点字典
- `doc/13-content-compliance-policy.md`：版权与合规策略
- `doc/14-test-strategy.md`：测试策略与发布门禁
- `doc/api-openapi-m1.yaml`：`doc/07-openapi-m1.yaml` 的副本，便于 M1 代码生成工具读取
- `doc/api-openapi.yaml`：`doc/07-openapi.yaml` 的副本，便于后续阶段代码生成工具读取

## 建议阅读顺序
1. `doc/00-concept-plan.md`
2. `doc/01-prd.md`
3. `doc/02-ia-and-key-flows.md`
4. `doc/05-backend-architecture-go.md`
5. `doc/06-database-selection.md`
6. `doc/07-openapi-m1.yaml`
7. `doc/08-fsrs-scheduling-spec.md`
8. `doc/09-offline-sync-spec.md`
9. `doc/10-storage-abstraction-spec.md`
10. `doc/11-llm-integration-spec.md`
11. `doc/03-database-design.md`
12. `doc/04-ai-resource-prompt-templates.md`
13. `doc/12-analytics-metrics-dictionary.md`
14. `doc/13-content-compliance-policy.md`
15. `doc/14-test-strategy.md`
16. `doc/07-openapi.yaml`（后续阶段参考）
