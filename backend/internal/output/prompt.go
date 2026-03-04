package output

import (
	"fmt"

	"github.com/bennyshi/english-anywhere-lab/internal/llm"
)

// BuildWritingFeedbackPrompt constructs the LLM messages for writing feedback.
// The system prompt instructs the LLM to return JSON matching the WritingFeedback schema.
func BuildWritingFeedbackPrompt(task *OutputTask, learnerText string) []llm.Message {
	levelStr := "unknown"
	if task.Level.Valid {
		levelStr = task.Level.String
	}

	system := `You are an English writing tutor. Evaluate the learner's writing and respond with ONLY a JSON object (no markdown, no extra text) matching this schema:
{
  "overall_score": <integer 0-100>,
  "errors": [
    {
      "original": "<the erroneous text>",
      "correction": "<the corrected text>",
      "explanation": "<brief explanation>"
    }
  ],
  "revised_text": "<the full corrected version of the learner's text>",
  "next_actions": ["<suggestion 1>", "<suggestion 2>"]
}

Rules:
- Score 0-100 based on grammar, vocabulary, coherence, and task completion.
- List every grammar, spelling, vocabulary, and style error found.
- The revised_text should be a corrected, polished version of the learner's text.
- Provide 1-3 actionable next_actions for improvement.
- Adjust feedback complexity to the learner's CEFR level.`

	userMsg := fmt.Sprintf("CEFR Level: %s\nTask prompt: %s\n", levelStr, task.PromptText)
	if task.ReferenceAnswer.Valid && task.ReferenceAnswer.String != "" {
		userMsg += fmt.Sprintf("Reference answer: %s\n", task.ReferenceAnswer.String)
	}
	userMsg += fmt.Sprintf("\nLearner's text:\n%s", learnerText)

	return []llm.Message{
		{Role: "system", Content: system},
		{Role: "user", Content: userMsg},
	}
}
