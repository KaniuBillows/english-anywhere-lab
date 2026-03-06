// ─── Auth ───────────────────────────────────────────────────────
export interface RegisterRequest {
  email: string;
  password: string;
  locale?: string;
  timezone?: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface RefreshRequest {
  refresh_token: string;
}

export interface AuthTokens {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface User {
  id: string;
  email: string;
  locale: string;
  timezone: string;
}

export interface AuthResponse {
  user: User;
  tokens: AuthTokens;
}

// ─── Profile ────────────────────────────────────────────────────
export interface LearningProfile {
  current_level: string;
  target_domain: string;
  daily_minutes: number;
  weekly_goal_days: number;
}

export interface MeResponse {
  user: User;
  learning_profile: LearningProfile;
}

export interface UpdateProfileRequest {
  current_level?: string;
  target_domain?: string;
  daily_minutes?: number;
  weekly_goal_days?: number;
}

// ─── Plans ──────────────────────────────────────────────────────
export interface BootstrapPlanRequest {
  level: string;
  target_domain: string;
  daily_minutes: number;
  days?: number;
}

export interface PlanTask {
  task_id: string;
  task_type: string;
  title: string;
  status: string;
  estimated_minutes: number;
  virtual?: boolean;
}

export interface DailyPlan {
  plan_id: string;
  date: string;
  mode: string;
  total_estimated_minutes: number;
  tasks: PlanTask[];
}

export interface DailyPlanResponse {
  daily_plan: DailyPlan;
}

export interface WeeklyPlanResponse {
  week_start: string;
  daily_plans: DailyPlan[];
}

export interface CompleteTaskRequest {
  completed_at: string;
  duration_seconds?: number;
}

export interface TaskCompletionResponse {
  task_id: string;
  status: string;
  next_recommendation?: string;
}

// ─── Reviews ────────────────────────────────────────────────────
export type Rating = 'again' | 'hard' | 'good' | 'easy';

export interface ReviewCard {
  card_id: string;
  user_card_state_id: string;
  front_text: string;
  back_text: string;
  example_text?: string;
  due_at: string;
}

export interface ReviewQueueResponse {
  due_count: number;
  cards: ReviewCard[];
}

export interface ReviewSubmitRequest {
  card_id: string;
  user_card_state_id: string;
  rating: Rating;
  reviewed_at: string;
  response_ms?: number;
  client_event_id: string;
}

export interface ReviewSubmitResponse {
  accepted: boolean;
  card_id: string;
  next_due_at: string;
  scheduled_days: number;
  status: string;
}

// ─── Progress ───────────────────────────────────────────────────
export interface ProgressSummary {
  range: string;
  total_minutes: number;
  active_days: number;
  review_accuracy: number;
  cards_reviewed: number;
  streak_count: number;
}

export interface DailyPoint {
  date: string;
  minutes_learned: number;
  cards_reviewed: number;
  review_accuracy?: number;
}

export interface ProgressDailyResponse {
  points: DailyPoint[];
}

export interface ReviewHealth {
  again: number;
  hard: number;
  good: number;
  easy: number;
  total: number;
  accuracy?: number;
}

export interface WeeklyReportDailyPoint {
  date: string;
  minutes_learned: number;
  lessons_completed: number;
  cards_new: number;
  cards_reviewed: number;
  review_accuracy?: number;
  listening_minutes: number;
  speaking_tasks: number;
  writing_tasks: number;
  streak_count: number;
}

export interface WeeklyComparison {
  minutes_delta: number;
  active_days_delta: number;
  cards_reviewed_delta: number;
  lessons_delta: number;
  accuracy_delta?: number;
}

export interface WeeklyReportResponse {
  week_start: string;
  total_minutes: number;
  active_days: number;
  cards_reviewed: number;
  cards_new: number;
  lessons_completed: number;
  listening_minutes: number;
  speaking_tasks: number;
  writing_tasks: number;
  streak: number;
  weekly_goal_days: number;
  goal_achieved: boolean;
  review_health: ReviewHealth;
  daily_breakdown: WeeklyReportDailyPoint[];
  previous_week_comparison?: WeeklyComparison;
}

export interface SkillMetric {
  skill: string;
  value: number;
  percentage: number;
}

export interface SkillBreakdown {
  listening: SkillMetric;
  speaking: SkillMetric;
  writing: SkillMetric;
  reading: SkillMetric;
}

export interface WeaknessItem {
  skill: string;
  reason: string;
  value: number;
  prev_value?: number;
}

export interface MonthlyComparison {
  minutes_delta: number;
  active_days_delta: number;
  cards_reviewed_delta: number;
  lessons_delta: number;
  accuracy_delta?: number;
}

export interface MonthlyReportResponse {
  month: string;
  days_in_month: number;
  total_minutes: number;
  active_days: number;
  cards_reviewed: number;
  cards_new: number;
  lessons_completed: number;
  listening_minutes: number;
  speaking_tasks: number;
  writing_tasks: number;
  streak: number;
  monthly_goal_days: number;
  goal_achieved: boolean;
  review_health: ReviewHealth;
  daily_breakdown: WeeklyReportDailyPoint[];
  skill_breakdown: SkillBreakdown;
  weaknesses: WeaknessItem[];
  previous_month_comparison?: MonthlyComparison;
}

// ─── Error ──────────────────────────────────────────────────────
export interface ApiError {
  code: string;
  message: string;
  details?: unknown;
}
