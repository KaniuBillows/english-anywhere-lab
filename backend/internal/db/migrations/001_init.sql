-- English Anywhere Lab: Initial Schema (SQLite)
-- Adapted from doc/03-database-design.md
-- UUID -> TEXT, ENUM -> TEXT + CHECK, JSONB -> TEXT (JSON), TIMESTAMPTZ -> TEXT (ISO8601)

-- ===== Users =====
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    locale TEXT NOT NULL DEFAULT 'zh-CN',
    timezone TEXT NOT NULL DEFAULT 'Asia/Shanghai',
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS user_learning_profiles (
    user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    current_level TEXT NOT NULL DEFAULT 'A2' CHECK (current_level IN ('A1','A2','B1','B2','C1','C2')),
    target_domain TEXT NOT NULL DEFAULT 'general',
    daily_minutes INTEGER NOT NULL DEFAULT 20 CHECK (daily_minutes > 0 AND daily_minutes <= 180),
    weekly_goal_days INTEGER NOT NULL DEFAULT 5 CHECK (weekly_goal_days BETWEEN 1 AND 7),
    fsrs_difficulty REAL,
    fsrs_stability REAL,
    placed_at TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- ===== Content =====
CREATE TABLE IF NOT EXISTS resource_packs (
    id TEXT PRIMARY KEY,
    source TEXT NOT NULL CHECK (source IN ('official','ai')),
    title TEXT NOT NULL,
    description TEXT,
    domain TEXT NOT NULL DEFAULT 'general',
    level TEXT NOT NULL CHECK (level IN ('A1','A2','B1','B2','C1','C2')),
    estimated_minutes INTEGER NOT NULL DEFAULT 20,
    created_by_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    metadata TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS lessons (
    id TEXT PRIMARY KEY,
    pack_id TEXT NOT NULL REFERENCES resource_packs(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    lesson_type TEXT NOT NULL DEFAULT 'mixed' CHECK (lesson_type IN ('reading','listening','speaking','writing','mixed')),
    position INTEGER NOT NULL CHECK (position > 0),
    estimated_minutes INTEGER NOT NULL DEFAULT 10,
    content_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE (pack_id, position)
);

CREATE TABLE IF NOT EXISTS cards (
    id TEXT PRIMARY KEY,
    lesson_id TEXT REFERENCES lessons(id) ON DELETE SET NULL,
    front_text TEXT NOT NULL,
    back_text TEXT NOT NULL,
    example_text TEXT,
    ipa TEXT,
    audio_url TEXT,
    image_url TEXT,
    tags TEXT NOT NULL DEFAULT '[]',
    level TEXT CHECK (level IN ('A1','A2','B1','B2','C1','C2')),
    metadata TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- ===== Learning State =====
CREATE TABLE IF NOT EXISTS user_card_states (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    card_id TEXT NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'new' CHECK (status IN ('new','learning','review','relearning','suspended')),
    due_at TEXT NOT NULL DEFAULT (datetime('now')),
    reps INTEGER NOT NULL DEFAULT 0,
    lapses INTEGER NOT NULL DEFAULT 0,
    stability REAL,
    difficulty REAL,
    elapsed_days INTEGER,
    scheduled_days INTEGER,
    last_review_at TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE (user_id, card_id)
);

CREATE INDEX IF NOT EXISTS idx_user_card_states_due_at ON user_card_states (user_id, due_at);
CREATE INDEX IF NOT EXISTS idx_user_card_states_status ON user_card_states (user_id, status);

CREATE TABLE IF NOT EXISTS learning_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_type TEXT NOT NULL CHECK (session_type IN ('daily_fast','daily_full','review_only','output_only')),
    planned_minutes INTEGER NOT NULL DEFAULT 10,
    actual_minutes INTEGER,
    device_type TEXT,
    started_at TEXT NOT NULL DEFAULT (datetime('now')),
    ended_at TEXT,
    completed INTEGER NOT NULL DEFAULT 0,
    metadata TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_learning_sessions_user_started ON learning_sessions (user_id, started_at DESC);

CREATE TABLE IF NOT EXISTS review_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    card_id TEXT NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    user_card_state_id TEXT REFERENCES user_card_states(id) ON DELETE SET NULL,
    session_id TEXT REFERENCES learning_sessions(id) ON DELETE SET NULL,
    rating TEXT NOT NULL CHECK (rating IN ('again','hard','good','easy')),
    response_ms INTEGER,
    state_before TEXT NOT NULL DEFAULT '{}',
    state_after TEXT NOT NULL DEFAULT '{}',
    client_event_id TEXT,
    idempotency_key TEXT,
    reviewed_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_review_logs_user_time ON review_logs (user_id, reviewed_at DESC);
CREATE INDEX IF NOT EXISTS idx_review_logs_card_time ON review_logs (card_id, reviewed_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_review_logs_idempotency ON review_logs (idempotency_key) WHERE idempotency_key IS NOT NULL;

-- ===== Output Tasks & Progress =====
CREATE TABLE IF NOT EXISTS output_tasks (
    id TEXT PRIMARY KEY,
    lesson_id TEXT REFERENCES lessons(id) ON DELETE SET NULL,
    task_type TEXT NOT NULL CHECK (task_type IN ('reading','listening','speaking','writing','mixed')),
    prompt_text TEXT NOT NULL,
    reference_answer TEXT,
    level TEXT CHECK (level IN ('A1','A2','B1','B2','C1','C2')),
    metadata TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS output_submissions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    task_id TEXT NOT NULL REFERENCES output_tasks(id) ON DELETE CASCADE,
    answer_text TEXT,
    audio_url TEXT,
    ai_feedback TEXT,
    score REAL,
    submitted_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_output_submissions_user_time ON output_submissions (user_id, submitted_at DESC);

CREATE TABLE IF NOT EXISTS progress_daily (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    progress_date TEXT NOT NULL,
    minutes_learned INTEGER NOT NULL DEFAULT 0,
    lessons_completed INTEGER NOT NULL DEFAULT 0,
    cards_new INTEGER NOT NULL DEFAULT 0,
    cards_reviewed INTEGER NOT NULL DEFAULT 0,
    review_accuracy REAL,
    listening_minutes INTEGER NOT NULL DEFAULT 0,
    speaking_tasks_completed INTEGER NOT NULL DEFAULT 0,
    writing_tasks_completed INTEGER NOT NULL DEFAULT 0,
    streak_count INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (user_id, progress_date)
);

CREATE INDEX IF NOT EXISTS idx_progress_daily_date ON progress_daily (progress_date);

-- ===== AI Generation Jobs =====
CREATE TABLE IF NOT EXISTS ai_generation_jobs (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    job_type TEXT NOT NULL,
    domain TEXT NOT NULL DEFAULT 'general',
    level TEXT NOT NULL CHECK (level IN ('A1','A2','B1','B2','C1','C2')),
    template_version TEXT NOT NULL,
    request_payload TEXT NOT NULL,
    response_payload TEXT,
    status TEXT NOT NULL DEFAULT 'queued' CHECK (status IN ('queued','running','success','failed')),
    error_message TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    started_at TEXT,
    finished_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_ai_jobs_user_time ON ai_generation_jobs (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_jobs_status ON ai_generation_jobs (status, created_at DESC);

-- ===== Plan Tables (M1 lightweight) =====
CREATE TABLE IF NOT EXISTS plans (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    week_start TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_plans_user ON plans (user_id, week_start DESC);

CREATE TABLE IF NOT EXISTS plan_tasks (
    id TEXT PRIMARY KEY,
    plan_id TEXT NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    task_date TEXT NOT NULL,
    task_type TEXT NOT NULL CHECK (task_type IN ('input','retrieval_quiz','review','output')),
    title TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','in_progress','completed')),
    estimated_minutes INTEGER NOT NULL DEFAULT 10,
    completed_at TEXT,
    duration_seconds INTEGER,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_plan_tasks_plan_date ON plan_tasks (plan_id, task_date);
