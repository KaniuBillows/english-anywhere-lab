-- Add retry_count column to ai_generation_jobs for TTS retry tracking.
ALTER TABLE ai_generation_jobs ADD COLUMN retry_count INTEGER NOT NULL DEFAULT 0;
