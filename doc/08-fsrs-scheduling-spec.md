# English Anywhere Lab - FSRS 调度规范（MVP）

## 1. 目标
定义复习调度的统一规则，确保前后端、离线同步、统计口径一致。

## 2. 版本与范围
- 调度策略版本：`fsrs_mvp_v1`
- 适用范围：卡片状态推进、到期计算、每日限额、幂等处理
- 不包含：复杂个性化参数自动训练（后续版本）

## 3. 术语
- `again/hard/good/easy`：用户评分
- `due_at`：下次应复习时间
- `scheduled_days`：计划间隔（天）
- `stability/difficulty`：FSRS 状态参数
- `day_cutoff_hour`：学习日切点（默认本地时间 04:00）

## 4. 卡片状态机
- `new`：新卡，尚未完成首次记忆
- `learning`：短期学习阶段
- `review`：长期复习阶段
- `relearning`：遗忘后重新学习
- `suspended`：暂停，不进入队列

状态流转：
`new -> learning -> review -> (again) -> relearning -> review`

## 5. 评分语义
- `again`：没记住/答错
- `hard`：勉强记住
- `good`：正常记住
- `easy`：非常熟练

## 6. 调度规则

### 6.1 新卡引入与每日上限
- 每日新卡上限：`new_cards_limit = 20`
- 每日复习上限：`review_cards_limit = 200`
- 队列优先级：
  1. 逾期复习卡（按 `due_at ASC`）
  2. 到期复习卡
  3. 新卡

### 6.2 learning 阶段（短间隔）
- `again` -> `due_at = now + 10m`
- `hard` -> `due_at = now + 30m`
- `good` -> `due_at = now + 1d`，转 `review`
- `easy` -> `due_at = now + 3d`，转 `review`

### 6.3 review/relearning 阶段（FSRS）
实现建议：
- 优先使用 FSRS 库计算间隔（Go 端封装在 `scheduler` 模块）
- 若库不可用，使用降级规则：
  - `again`: 间隔重置为 1d，`lapses + 1`，转 `relearning`
  - `hard`: 新间隔 = `max(1, round(prev_interval * 1.2))`
  - `good`: 新间隔 = `max(1, round(prev_interval * 2.2))`
  - `easy`: 新间隔 = `max(2, round(prev_interval * 3.0))`

### 6.4 日切与时区
- 所有 `due_at` 存储 UTC。
- 计算“今天学习”时，按用户时区和 `day_cutoff_hour=04` 划分。
- 同一用户跨设备必须使用相同 day cutoff。

## 7. 幂等与一致性

### 7.1 幂等键
- 复习提交必须携带：
  - Header: `Idempotency-Key`
  - Body: `client_event_id`
- 任意一个重复命中都视为重复提交，返回上次结果。

### 7.2 事务要求
一次评分提交在单事务内完成：
1. 写 `review_logs`
2. 更新 `user_card_states`
3. 写入 `sync_receipts`（如来自离线事件）

## 8. API 约束
- 输入：`card_id`, `user_card_state_id`, `rating`, `reviewed_at`
- 输出：`next_due_at`, `scheduled_days`, `status`
- 服务端不得信任客户端提交的 `scheduled_days`

## 9. 监控指标
- `review_submit_success_rate`
- `review_duplicate_rate`
- `avg_scheduled_days_by_rating`
- `lapse_rate_7d`
- `overdue_backlog_count`

## 10. 测试样例（必须）

### Case A：new 卡 `good`
- 输入：`status=new`, `rating=good`
- 预期：`status=review`, `due_at=now+1d`

### Case B：review 卡 `again`
- 输入：`status=review`, `prev_interval=7d`, `rating=again`
- 预期：`status=relearning`, `lapses+1`, `due_at <= now+1d`

### Case C：幂等重复提交
- 输入：同一 `Idempotency-Key`
- 预期：返回相同 `next_due_at`，不新增日志行

### Case D：跨日切边界
- 输入：用户时区 `Asia/Shanghai`, 提交时间 `03:30` 与 `04:30`
- 预期：归属不同学习日

## 11. 未来演进
- 引入个体化参数训练（基于历史日志拟合）
- 引入“负担控制器”（自动调节新卡上限）
- 对困难词启用多模态强化（图像/音频）
