# English Anywhere Lab - 数据库表设计（PostgreSQL）

## 1. 设计目标
- 支持学习闭环数据：学习包、卡片、复习、输出、进度
- 支持 SRS/FSRS 的卡片状态演进
- 支持 AI 资源生成任务可追踪、可回放

## 2. 技术选择
- 数据库：PostgreSQL 15+
- 主键：`uuid`（日志表可用 `bigserial`）
- 时间字段：`timestamptz`
- JSON 扩展字段：`jsonb`

## 3. 核心实体关系（简述）
- `users` 1:1 `user_learning_profiles`
- `resource_packs` 1:N `lessons`
- `lessons` 1:N `cards`
- `users` N:N `cards`（通过 `user_card_states`）
- `user_card_states` 1:N `review_logs`
- `users` 1:N `learning_sessions`
- `users` 1:N `progress_daily`
- `users` 1:N `ai_generation_jobs`

## 4. DDL（可执行草案）

```sql
-- Required extension
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ===== Enums =====
CREATE TYPE cefr_level AS ENUM ('A1', 'A2', 'B1', 'B2', 'C1', 'C2');
CREATE TYPE pack_source AS ENUM ('official', 'ai');
CREATE TYPE lesson_type AS ENUM ('reading', 'listening', 'speaking', 'writing', 'mixed');
CREATE TYPE card_status AS ENUM ('new', 'learning', 'review', 'relearning', 'suspended');
CREATE TYPE review_rating AS ENUM ('again', 'hard', 'good', 'easy');
CREATE TYPE session_type AS ENUM ('daily_fast', 'daily_full', 'review_only', 'output_only');
CREATE TYPE generation_status AS ENUM ('queued', 'running', 'success', 'failed');

-- ===== Users =====
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    locale VARCHAR(16) NOT NULL DEFAULT 'zh-CN',
    timezone VARCHAR(64) NOT NULL DEFAULT 'Asia/Shanghai',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_learning_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    current_level cefr_level NOT NULL DEFAULT 'A2',
    target_domain VARCHAR(64) NOT NULL DEFAULT 'general',
    daily_minutes INT NOT NULL DEFAULT 20 CHECK (daily_minutes > 0 AND daily_minutes <= 180),
    weekly_goal_days INT NOT NULL DEFAULT 5 CHECK (weekly_goal_days BETWEEN 1 AND 7),
    fsrs_difficulty NUMERIC(6,3),
    fsrs_stability NUMERIC(8,3),
    placed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ===== Content =====
CREATE TABLE resource_packs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source pack_source NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    domain VARCHAR(64) NOT NULL DEFAULT 'general',
    level cefr_level NOT NULL,
    estimated_minutes INT NOT NULL DEFAULT 20,
    created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE lessons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pack_id UUID NOT NULL REFERENCES resource_packs(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    lesson_type lesson_type NOT NULL DEFAULT 'mixed',
    position INT NOT NULL CHECK (position > 0),
    estimated_minutes INT NOT NULL DEFAULT 10,
    content_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (pack_id, position)
);

CREATE TABLE cards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lesson_id UUID REFERENCES lessons(id) ON DELETE SET NULL,
    front_text TEXT NOT NULL,
    back_text TEXT NOT NULL,
    example_text TEXT,
    ipa VARCHAR(120),
    audio_url TEXT,
    image_url TEXT,
    tags TEXT[] NOT NULL DEFAULT '{}',
    level cefr_level,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ===== Learning State =====
CREATE TABLE user_card_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    card_id UUID NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    status card_status NOT NULL DEFAULT 'new',
    due_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reps INT NOT NULL DEFAULT 0,
    lapses INT NOT NULL DEFAULT 0,
    stability NUMERIC(8,3),
    difficulty NUMERIC(6,3),
    elapsed_days INT,
    scheduled_days INT,
    last_review_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, card_id)
);

CREATE INDEX idx_user_card_states_due_at ON user_card_states (user_id, due_at);
CREATE INDEX idx_user_card_states_status ON user_card_states (user_id, status);

CREATE TABLE learning_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_type session_type NOT NULL,
    planned_minutes INT NOT NULL DEFAULT 10,
    actual_minutes INT,
    device_type VARCHAR(16),
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    completed BOOLEAN NOT NULL DEFAULT FALSE,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX idx_learning_sessions_user_started ON learning_sessions (user_id, started_at DESC);

CREATE TABLE review_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    card_id UUID NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    user_card_state_id UUID REFERENCES user_card_states(id) ON DELETE SET NULL,
    session_id UUID REFERENCES learning_sessions(id) ON DELETE SET NULL,
    rating review_rating NOT NULL,
    response_ms INT,
    state_before JSONB NOT NULL DEFAULT '{}'::jsonb,
    state_after JSONB NOT NULL DEFAULT '{}'::jsonb,
    reviewed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_review_logs_user_time ON review_logs (user_id, reviewed_at DESC);
CREATE INDEX idx_review_logs_card_time ON review_logs (card_id, reviewed_at DESC);

-- ===== Output Tasks & Progress =====
CREATE TABLE output_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lesson_id UUID REFERENCES lessons(id) ON DELETE SET NULL,
    task_type lesson_type NOT NULL,
    prompt_text TEXT NOT NULL,
    reference_answer TEXT,
    level cefr_level,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE output_submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    task_id UUID NOT NULL REFERENCES output_tasks(id) ON DELETE CASCADE,
    answer_text TEXT,
    audio_url TEXT,
    ai_feedback JSONB,
    score NUMERIC(5,2),
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_output_submissions_user_time ON output_submissions (user_id, submitted_at DESC);

CREATE TABLE progress_daily (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    progress_date DATE NOT NULL,
    minutes_learned INT NOT NULL DEFAULT 0,
    lessons_completed INT NOT NULL DEFAULT 0,
    cards_new INT NOT NULL DEFAULT 0,
    cards_reviewed INT NOT NULL DEFAULT 0,
    review_accuracy NUMERIC(5,2),
    listening_minutes INT NOT NULL DEFAULT 0,
    speaking_tasks_completed INT NOT NULL DEFAULT 0,
    writing_tasks_completed INT NOT NULL DEFAULT 0,
    streak_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, progress_date)
);

CREATE INDEX idx_progress_daily_date ON progress_daily (progress_date);

-- ===== AI Generation Jobs =====
CREATE TABLE ai_generation_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    job_type VARCHAR(32) NOT NULL, -- pack / cards / listening / speaking
    domain VARCHAR(64) NOT NULL DEFAULT 'general',
    level cefr_level NOT NULL,
    template_version VARCHAR(32) NOT NULL,
    request_payload JSONB NOT NULL,
    response_payload JSONB,
    status generation_status NOT NULL DEFAULT 'queued',
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ
);

CREATE INDEX idx_ai_jobs_user_time ON ai_generation_jobs (user_id, created_at DESC);
CREATE INDEX idx_ai_jobs_status ON ai_generation_jobs (status, created_at DESC);
```

## 5. 字段与实现注意事项
- `review_logs.state_before/state_after` 用于回放和调试 FSRS 行为。
- `progress_daily` 通过离线任务或流式聚合更新，避免在线重算。
- `metadata/jsonb` 仅作为扩展字段，核心检索字段必须结构化落列。

## 6. 典型查询（示例）

### 6.1 查询用户今日到期复习卡
```sql
SELECT ucs.card_id, c.front_text, c.back_text
FROM user_card_states ucs
JOIN cards c ON c.id = ucs.card_id
WHERE ucs.user_id = $1
  AND ucs.due_at <= NOW()
  AND ucs.status <> 'suspended'
ORDER BY ucs.due_at ASC
LIMIT 100;
```

### 6.2 查询最近 7 天学习表现
```sql
SELECT progress_date, minutes_learned, cards_reviewed, review_accuracy, streak_count
FROM progress_daily
WHERE user_id = $1
  AND progress_date >= CURRENT_DATE - INTERVAL '6 days'
ORDER BY progress_date ASC;
```

## 7. 迁移顺序建议
1. 创建 enum
2. 创建用户与内容主表
3. 创建学习状态表与日志表
4. 创建进度聚合与 AI 任务表
5. 创建索引
6. 初始化基础数据（demo pack）
