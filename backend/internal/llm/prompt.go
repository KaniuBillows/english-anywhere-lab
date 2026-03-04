package llm

import (
	"fmt"
	"strings"
)

const systemPrompt = `You are an expert English language curriculum designer. Generate a structured learning pack in JSON format.

The JSON MUST conform to the following schema exactly:

{
  "title": "string (required)",
  "description": "string",
  "estimated_minutes": integer (required, >= 1),
  "lessons": [
    {
      "title": "string (required)",
      "lesson_type": "string (required, one of: reading, listening, speaking, writing, mixed)",
      "position": integer (required, >= 1, sequential starting from 1),
      "estimated_minutes": integer (required, >= 1),
      "cards": [
        {
          "front_text": "string (required, the English word/phrase/sentence)",
          "back_text": "string (required, the Chinese translation or explanation)",
          "example_text": "string (optional, example sentence using the word/phrase)"
        }
      ],
      "output_tasks": [
        {
          "task_type": "string (required, one of: reading, listening, speaking, writing, mixed)",
          "prompt_text": "string (required, the task prompt for the student)",
          "reference_answer": "string (optional, a model answer)"
        }
      ]
    }
  ]
}

Rules:
- Return ONLY valid JSON. No markdown, no explanation, no code fences.
- Each lesson should have at least 3-5 vocabulary cards.
- Each lesson should have at least 1 output task.
- Lesson positions must be sequential starting from 1.
- Content should be appropriate for the specified CEFR level.
- Cards should have English on front_text and Chinese explanation/translation on back_text.
- The total estimated_minutes of all lessons should roughly match daily_minutes * days.`

// BuildPrompt constructs the system and user messages for pack generation.
func BuildPrompt(level, domain string, dailyMinutes, days int, focusSkills []string) []Message {
	skills := "all skills"
	if len(focusSkills) > 0 {
		skills = strings.Join(focusSkills, ", ")
	}

	userMsg := fmt.Sprintf(`Generate a learning pack with the following parameters:
- CEFR Level: %s
- Domain/Topic: %s
- Daily study time: %d minutes
- Number of days: %d
- Focus skills: %s

Create lessons that cover the specified number of days, with each lesson fitting within the daily study time. Include vocabulary cards and output tasks for each lesson.`, level, domain, dailyMinutes, days, skills)

	return []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMsg},
	}
}
