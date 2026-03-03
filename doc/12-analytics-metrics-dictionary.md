# English Anywhere Lab - 数据口径与埋点字典（MVP）

## 1. 目标
定义统一的数据口径，避免前后端、BI、运营对同一指标解释不一致。

## 2. 核心指标定义

### 2.1 有效学习日（North Star）
- 定义：用户在自然学习日内完成以下任一闭环：
  - 新学 + 复习 + 输出中至少两类任务
  - 或完成完整模式（20 分钟）
- 粒度：`user_id + date`

### 2.2 周学习天数
- 定义：近 7 天内有效学习日的天数。

### 2.3 复习正确率
- 定义：`(hard + good + easy) / (again + hard + good + easy)`
- 窗口：默认 7 天滚动。

### 2.4 连续学习天数（streak）
- 定义：以用户本地时区和 `day_cutoff_hour=04` 计算的连续有效学习天数。

### 2.5 AI 学习包使用率
- 定义：`被加入学习计划的 AI 包数量 / 生成成功的 AI 包数量`

## 3. 事件埋点字典

| 事件名 | 触发时机 | 必填属性 |
| --- | --- | --- |
| `onboarding_completed` | 初始化流程完成 | `user_id`, `level`, `domain`, `daily_minutes` |
| `daily_plan_started` | 点击开始今日学习 | `user_id`, `plan_id`, `mode`, `device_type` |
| `lesson_completed` | 单个 lesson 完成 | `user_id`, `lesson_id`, `duration_sec` |
| `card_reviewed` | 提交一次评分 | `user_id`, `card_id`, `rating`, `response_ms` |
| `output_task_submitted` | 提交口语/写作任务 | `user_id`, `task_id`, `task_type`, `score` |
| `ai_pack_generated` | AI 生成任务结束 | `user_id`, `job_id`, `status`, `latency_ms` |
| `dashboard_viewed` | 打开看板 | `user_id`, `range`, `device_type` |

## 4. 公共属性（所有事件）
- `event_id`：UUID
- `event_time`：ISO-8601
- `user_id`
- `session_id`
- `platform`：`web_mobile|web_desktop`
- `app_version`
- `timezone`

## 5. 去重规则
- 主键：`event_id`
- 同一 `event_id` 多次上报仅保留首次。
- 客户端离线补发不改变 `event_time`。

## 6. 指标计算窗口
- 日：按用户本地时区 + 日切 04:00
- 周：近 7 个学习日窗口（含当天）
- 月：自然月（本地时区）

## 7. 质量保障
- 埋点丢失监控：`missing_event_rate`
- 字段空值监控：关键属性空值率 < 0.5%
- 时钟漂移监控：客户端与服务端时间差 > 5 分钟告警

## 8. 与数据表映射
- 行为日志：`learning_sessions`, `review_logs`, `output_submissions`
- 聚合表：`progress_daily`
- AI 任务：`ai_generation_jobs`

## 9. 报表最小集
1. 学习漏斗：开始计划 -> 完成 lesson -> 完成输出
2. 复习健康：到期量、完成量、正确率、积压量
3. AI 资源表现：生成成功率、使用率、复用率
