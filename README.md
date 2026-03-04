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
- `doc/08-fsrs-scheduling-spec.md`：FSRS 调度规范（状态机、日切、幂等）
- `doc/09-offline-sync-spec.md`：离线同步与冲突处理规范（PWA）
- `doc/10-storage-abstraction-spec.md`：文件存储抽象（Local/S3 可配置）
- `doc/11-llm-integration-spec.md`：LLM 接入规范（OpenAI 兼容 API）
- `doc/12-analytics-metrics-dictionary.md`：指标口径与埋点字典
- `doc/13-content-compliance-policy.md`：版权与合规策略
- `doc/14-test-strategy.md`：测试策略与发布门禁
- `doc/15-tts-selection-offline-cpu.md`：离线 TTS 选型（纯 CPU、本地生成、对象存储落盘）
- `doc/16-learning-pack-type-system.md`：学习包类型体系（Pack Template + Exercise Type + 组合规则）
- `doc/17-documentation-coverage-audit.md`：文档覆盖审计与补全清单（定义-契约-落地-验证）
- `doc/18-implementation-checklist.md`：开发执行 checklist（契约、实现、测试、发布）
- `backend/api/openapi.yaml`：唯一 OpenAPI 契约文件（全项目单一真值）

## 建议阅读顺序
1. `doc/00-concept-plan.md`
2. `doc/01-prd.md`
3. `doc/02-ia-and-key-flows.md`
4. `doc/05-backend-architecture-go.md`
5. `doc/06-database-selection.md`
6. `backend/api/openapi.yaml`
7. `doc/08-fsrs-scheduling-spec.md`
8. `doc/09-offline-sync-spec.md`
9. `doc/10-storage-abstraction-spec.md`
10. `doc/11-llm-integration-spec.md`
11. `doc/03-database-design.md`
12. `doc/04-ai-resource-prompt-templates.md`
13. `doc/12-analytics-metrics-dictionary.md`
14. `doc/13-content-compliance-policy.md`
15. `doc/14-test-strategy.md`
16. `doc/15-tts-selection-offline-cpu.md`
17. `doc/16-learning-pack-type-system.md`
18. `doc/17-documentation-coverage-audit.md`
19. `doc/18-implementation-checklist.md`
