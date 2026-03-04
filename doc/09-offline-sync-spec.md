# English Anywhere Lab - 离线同步与冲突处理规范（PWA）

## 1. 目标
支持移动端和桌面端在离线条件下继续学习，并在恢复联网后安全同步，确保最终一致。

## 2. 数据模型

### 2.1 客户端事件模型（IndexedDB）
```json
{
  "client_event_id": "uuid",
  "event_type": "review_submitted|output_submitted|task_completed|profile_updated",
  "occurred_at": "2026-03-02T09:00:00Z",
  "client_seq": 1024,
  "payload": {},
  "event_context": {
    "pack_template": "string|null",
    "exercise_type": "string|null",
    "evaluation_mode": "objective|rubric|llm_hybrid|null"
  },
  "sync_status": "pending|acked|rejected",
  "retry_count": 0
}
```

### 2.2 服务端接收模型
- 接口：`POST /sync/events`
- 批量上限：500 条
- 服务端返回逐条 ack（accepted/duplicate/rejected）

## 3. 客户端同步策略
- 触发时机：
  - 应用启动并检测到在线
  - 前后台切换到前台
  - 用户手动点击“立即同步”
  - 定时（每 30 秒）
- 发送顺序：按 `occurred_at` + `client_seq` 升序。
- 重试策略：指数退避（1s, 2s, 4s, 最多 5 次）。

## 4. 去重与幂等
- 唯一键：`(user_id, client_event_id)`
- 若重复提交：返回 `duplicate`，客户端标记 ack。
- 复习事件同时校验 `Idempotency-Key`。

## 5. 冲突处理规则

### 5.1 复习事件（review_submitted）
- 冲突策略：按 `occurred_at` 顺序重放。
- 若时间戳相同：按 `client_seq`。
- 若状态版本冲突：服务端以“可重放规则”重新计算后覆盖。

### 5.2 任务完成（task_completed）
- 幂等语义：同一 `task_id` 重复完成不报错。
- 冲突策略：首次完成时间保留，后续重复忽略。

### 5.3 输出提交（output_submitted）
- 同一任务允许多次提交。
- 进度统计按“最后一次高分”或“首次完成”计入口径（二选一，建议首次完成）。
- 若携带 `event_context.exercise_type`，服务端应写入分析事件用于题型效果统计。

### 5.4 配置更新（profile_updated）
- 冲突策略：`last_write_wins`（按服务端接收时间）。

## 6. 服务端变更下发
- 接口：`GET /sync/changes?cursor=...`
- 返回：变更列表 + `next_cursor`
- 客户端在拉取后更新本地镜像（计划、卡片状态、进度摘要）。

## 7. 失败处理
- `rejected` 必须返回原因码：
  - `INVALID_PAYLOAD`
  - `UNKNOWN_EVENT_TYPE`
  - `STATE_CONFLICT_UNRESOLVABLE`
  - `AUTH_REQUIRED`
- 客户端对可修复错误（如认证过期）重试；不可修复错误显示提示并停重试。

## 8. 安全与隐私
- 本地仅缓存必要数据，不缓存明文 token。
- 事件 payload 中避免写入敏感原文（如完整写作文本可用摘要）。

## 9. 最小实现清单
1. IndexedDB 事件队列
2. `/sync/events` 批量上报与逐条 ack
3. 幂等去重表
4. `/sync/changes` 拉取机制
5. 冲突日志可审计

## 10. 验收标准
- 离线 24 小时后恢复在线，事件丢失率 = 0
- 重复提交不会造成重复统计
- 多端同账号复习状态在 1 分钟内收敛一致
