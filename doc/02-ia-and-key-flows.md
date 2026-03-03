# English Anywhere Lab - 页面信息架构（IA）与关键流程图

## 1. 信息架构（IA）

### 1.1 顶层导航
- 今日学习
- 学习资源
- 复习中心
- 输出训练
- 进度看板
- 我的（设置/目标/账户）

### 1.2 站点地图（Sitemap）
```mermaid
graph TD
    A[App Shell] --> B[今日学习]
    A --> C[学习资源]
    A --> D[复习中心]
    A --> E[输出训练]
    A --> F[进度看板]
    A --> G[我的]

    B --> B1[任务摘要]
    B --> B2[10分钟快学]
    B --> B3[20分钟完整模式]
    B --> B4[断点续学]

    C --> C1[学习包列表]
    C --> C2[学习包详情]
    C --> C3[AI生成学习包]
    C --> C4[领域筛选]

    D --> D1[到期复习]
    D --> D2[困难词强化]
    D --> D3[复习历史]

    E --> E1[口语任务]
    E --> E2[写作任务]
    E --> E3[AI反馈]

    F --> F1[日视图]
    F --> F2[周报]
    F --> F3[月报]
    F --> F4[能力雷达图]

    G --> G1[学习目标]
    G --> G2[个人资料]
    G --> G3[提醒设置]
    G --> G4[数据导出]
```

## 2. 关键流程 1：首次进入与计划生成
```mermaid
flowchart TD
    A[打开应用] --> B{是否已登录}
    B -- 否 --> C[注册/登录]
    B -- 是 --> D[进入 Onboarding]
    C --> D
    D --> E[设置水平 CEFR]
    E --> F[设置领域目标]
    F --> G[设置每日时长]
    G --> H[生成首周学习计划]
    H --> I[进入今日学习]
```

## 3. 关键流程 2：每日学习闭环
```mermaid
flowchart TD
    A[进入今日学习] --> B[选择快学10分钟/完整20分钟]
    B --> C[输入任务 听/读]
    C --> D[主动检索小测]
    D --> E[卡片复习 SRS]
    E --> F[输出任务 说/写]
    F --> G[记录日志与进度更新]
    G --> H[给出明日建议]
```

## 4. 关键流程 3：AI 学习包生成与上架
```mermaid
sequenceDiagram
    participant U as User
    participant FE as Frontend
    participant BE as Backend
    participant AI as LLM Service
    participant QC as Quality Check
    participant DB as Database

    U->>FE: 输入领域、难度、时长
    FE->>BE: 创建生成任务
    BE->>AI: 调用模板生成学习包
    AI-->>BE: 返回结构化内容(JSON)
    BE->>QC: 难度/词汇/格式校验
    QC-->>BE: 通过或失败原因
    alt 校验通过
        BE->>DB: 写入 Pack/Lesson/Card
        BE-->>FE: 返回可学习包
    else 校验失败
        BE->>AI: 携带失败原因重生成
        AI-->>BE: 返回修复版
        BE->>DB: 写入修复版内容
        BE-->>FE: 返回可学习包
    end
```

## 5. 关键流程 4：复习调度
```mermaid
flowchart TD
    A[用户完成卡片评分] --> B[写入 review_log]
    B --> C[更新 user_card_state]
    C --> D[计算 next_due_at]
    D --> E{今日复习上限是否达到}
    E -- 否 --> F[继续出卡]
    E -- 是 --> G[剩余卡片延期到次日]
    F --> H[完成本次复习]
    G --> H
```

## 6. 页面清单（MVP）
- P01 登录与初始化页
- P02 今日学习页
- P03 学习包列表与详情页
- P04 复习中心页
- P05 输出任务页
- P06 进度看板页
- P07 我的与设置页

## 7. 响应式断点建议
- Mobile：`< 768px`
- Tablet：`768px - 1023px`
- Desktop：`>= 1024px`

## 8. 交互优先级建议
- 优先保证“今日学习”和“复习中心”一跳可达
- 次优先保证“学习包生成”和“进度看板”可发现性
- 首版弱化复杂筛选，优先保留目标导向推荐
