# English Anywhere Lab - 数据库选型与存储策略决策

## 1. 目标
在“不面向大规模并发、个人使用优先、低成本优先”的前提下，确定可直接开工的数据库方案。

## 2. 选型结论
- 默认主库：`SQLite`
- 可选主库：`PostgreSQL`
- 当前阶段不做迁移工具链要求（按你的要求），通过配置决定运行模式。

## 3. 选型对比
| 维度 | SQLite | PostgreSQL |
| --- | --- | --- |
| 成本 | 极低（单文件） | 中等（需服务实例） |
| 运维复杂度 | 低 | 中 |
| 并发能力 | 中（写并发受限） | 高 |
| 本项目适配 | 高（个人低并发） | 中（可选） |
| 备份难度 | 低（文件备份） | 中 |

结论：MVP 采用 SQLite，可满足当前业务负载和成本目标。

## 4. SQLite 运行要求（必须）
- 打开 WAL：`PRAGMA journal_mode = WAL;`
- 开启外键：`PRAGMA foreign_keys = ON;`
- 写锁等待：`PRAGMA busy_timeout = 5000;`
- 减少长事务：单请求事务尽量控制在 100ms 内。

## 5. 数据访问层约束
- Repository 层禁止直接拼 SQL 字符串，统一放到 `internal/db/queries`。
- SQL 使用 DB-neutral 子集（减少引擎差异）。
- 业务层不得感知 `sqlite/postgres`，仅依赖 Repository 接口。

## 6. 与现有表设计的关系
- [03-database-design.md](/Users/bennyshi/dev/english-anywhere-lab/03-database-design.md) 作为逻辑模型基线。
- MVP 运行时字段类型按 SQLite 适配：
  - `UUID` -> `TEXT`
  - `ENUM` -> `TEXT + CHECK`
  - `JSONB` -> `TEXT(JSON)`
  - `TIMESTAMPTZ` -> `TEXT(ISO-8601)` 或 `INTEGER(unix_ms)`

## 7. 连接与池化建议
- SQLite：
  - `SetMaxOpenConns(1~4)`（避免写争用）
  - `SetConnMaxLifetime` 可不强制
- Postgres：
  - `SetMaxOpenConns(20)`（默认）
  - `SetMaxIdleConns(10)`

## 8. 备份策略（低成本）
- 默认：每天定时复制数据库文件到 `./backup/`。
- 可选：备份到 S3（通过对象存储适配层）。
- 备份命名：`app-YYYYMMDD-HHMM.db`。
- 恢复演练：每周至少一次测试恢复。

## 9. 风险与边界
- 风险：SQLite 在高频写入场景可能出现锁等待。
- 对策：
  - 请求层限制批量写大小
  - 复习日志支持批量异步入库
  - 队列任务限并发

## 10. 配置示例
```env
# 默认 SQLite
DB_DRIVER=sqlite
DB_DSN=file:./data/app.db?_fk=1
SQLITE_WAL=true
SQLITE_BUSY_TIMEOUT_MS=5000

# 可选 Postgres
# DB_DRIVER=postgres
# DB_DSN=postgres://user:pass@127.0.0.1:5432/english_anywhere?sslmode=disable
```

## 11. 启动建议
- 本地开发：SQLite 单文件 + seed 数据。
- 预发布：SQLite 或轻量 Postgres（二选一）。
- 正式生产：若出现明显写锁瓶颈再切换 Postgres。
