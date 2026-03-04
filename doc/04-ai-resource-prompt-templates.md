# English Anywhere Lab - AI 学习资源提示词模板（可重复生成）

## 1. 使用原则
- 模板必须输出结构化 JSON，便于程序入库。
- 所有模板显式声明难度（CEFR）和目标领域（domain）。
- 每次生成都记录 `template_version`、`seed`、`temperature`。
- 先生成，再质检，失败则按错误原因重生成。

## 2. 变量约定
- `{{level}}`：A1/A2/B1/B2/C1/C2
- `{{domain}}`：如 `trade`, `tech`, `travel`, `interview`, `general`
- `{{pack_template}}`：学习包模板类型（见 `doc/16-learning-pack-type-system.md`）
- `{{allowed_exercise_types}}`：允许的题型白名单（来自 `doc/16-learning-pack-type-system.md`）
- `{{daily_minutes}}`：每天学习分钟数
- `{{days}}`：计划周期天数（默认 7）
- `{{focus_skills}}`：`listening, speaking, reading, writing`
- `{{known_vocab_limit}}`：已掌握词汇量估计
- `{{native_language}}`：用户母语，如 `zh-CN`

## 3. 基础系统提示词（System Prompt）

```text
You are an English curriculum generator for adult learners.
You must generate practical, domain-specific, CEFR-aligned learning content.
Always return valid JSON only. No markdown, no comments, no extra text.
Constraints:
1) Respect target level strictly (grammar, sentence length, vocabulary difficulty).
2) Prioritize usefulness in real-life scenarios.
3) Include spaced-repetition ready card data.
4) Include at least one output task (speaking or writing) per lesson.
5) If input is insufficient, make conservative assumptions and continue.
```

## 4. 模板 A：生成 7 天学习包

### 4.1 User Prompt 模板
```text
Generate a {{days}}-day English study pack.

Inputs:
- CEFR level: {{level}}
- Domain: {{domain}}
- Pack template: {{pack_template}}
- Daily study minutes: {{daily_minutes}}
- Focus skills: {{focus_skills}}
- Native language: {{native_language}}
- Known vocabulary estimate: {{known_vocab_limit}}

Output JSON schema:
{
  "pack": {
    "title": "string",
    "pack_template": "vocab_foundation|scenario_dialog|intensive_listening|reading_comprehension|writing_output|review_booster|speaking_bootcamp|exam_drill",
    "domain": "string",
    "level": "A1|A2|B1|B2|C1|C2",
    "days": number,
    "estimated_daily_minutes": number,
    "learning_goals": ["string"]
  },
  "lessons": [
    {
      "day": number,
      "title": "string",
      "input": {
        "reading_text": "string",
        "listening_script": "string",
        "key_vocab": [
          {"word": "string", "meaning_native": "string", "example": "string"}
        ]
      },
      "retrieval_quiz": [
        {
          "type": "mcq|cloze|short_answer",
          "exercise_type": "flashcard_recall|cloze_word|cloze_sentence|reading_mcq|audio_cloze|dictation_short|keyword_match|sentence_reorder|error_correction|sentence_rewrite|situational_qa|picture_naming|picture_description|guided_paragraph|free_writing|retell_audio",
          "question": "string",
          "answer": "string",
          "explanation": "string"
        }
      ],
      "srs_cards": [
        {
          "front": "string",
          "back": "string",
          "example": "string",
          "tags": ["string"]
        }
      ],
      "output_task": {
        "type": "speaking|writing",
        "exercise_type": "situational_qa|picture_description|guided_paragraph|free_writing|retell_audio",
        "prompt": "string",
        "rubric": ["string"]
      }
    }
  ]
}

Hard constraints:
- For {{level}}, keep reading_text within reasonable complexity.
- `exercise_type` must be within {{allowed_exercise_types}}.
- Each lesson must include 8-12 key vocab items.
- Each lesson must include 12-20 srs_cards.
- Output task must be scenario-based and domain-specific.
```

## 5. 模板 B：只生成卡片集（用于补充复习）

### 5.1 User Prompt 模板
```text
Generate an SRS card set for this topic:
- Topic: {{domain}}
- CEFR: {{level}}
- Card count: 50
- Native language: {{native_language}}

Return JSON:
{
  "card_set_title": "string",
  "cards": [
    {
      "front": "string",
      "back": "string",
      "example": "string",
      "phonetic_ipa": "string",
      "mnemonic_visual_hint": "string",
      "difficulty": 1-5,
      "tags": ["string"]
    }
  ]
}

Rules:
- Avoid duplicate meanings.
- Ensure 70% high-frequency practical expressions, 30% domain-specific.
- Examples must be natural and short.
```

## 6. 模板 C：口语任务生成

### 6.1 User Prompt 模板
```text
Create 10 speaking tasks for:
- CEFR: {{level}}
- Domain: {{domain}}
- User native language: {{native_language}}

Return JSON:
{
  "tasks": [
    {
      "scenario": "string",
      "role": "string",
      "prompt": "string",
      "target_expressions": ["string"],
      "sample_answer": "string",
      "grading_rubric": {
        "accuracy": "string",
        "fluency": "string",
        "vocabulary": "string"
      }
    }
  ]
}

Rules:
- Each task should be answerable in 30-90 seconds.
- Use realistic workplace/life scenes.
- Include at least 3 target expressions per task.
```

## 7. 模板 D：写作纠错反馈

### 7.1 User Prompt 模板
```text
You are a writing coach.
Evaluate and revise the learner text.

Inputs:
- CEFR: {{level}}
- Domain: {{domain}}
- Learner text: {{learner_text}}

Return JSON:
{
  "overall_score": 0-100,
  "errors": [
    {"type": "grammar|word_choice|coherence|tone", "original": "string", "suggestion": "string", "reason": "string"}
  ],
  "revised_text": "string",
  "next_actions": ["string"]
}

Rules:
- Preserve learner intent.
- Keep revised text at target CEFR level.
- Give no more than 8 correction points.
```

## 8. 模板 E：生成质量检查（QC）

### 8.1 User Prompt 模板
```text
You are a strict curriculum QA validator.
Validate this generated pack against constraints.

Inputs:
- Target CEFR: {{level}}
- Target domain: {{domain}}
- Generated JSON: {{generated_json}}

Return JSON:
{
  "pass": true|false,
  "score": 0-100,
  "issues": [
    {"code": "LEVEL_DRIFT|FORMAT_ERROR|LOW_UTILITY|DUPLICATION", "message": "string", "severity": "high|medium|low"}
  ],
  "fix_instructions": ["string"]
}
```

## 9. 模板 F：按 QC 结果重生成

### 9.1 User Prompt 模板
```text
Regenerate the content by fixing these issues only:
- Issues: {{fix_instructions}}
- Keep unchanged parts if valid.
- Keep CEFR at {{level}} and domain at {{domain}}.

Return JSON with the exact same schema as original output.
```

## 10. 后端接入建议（最小实现）
1. 读取模板（按 `template_version`）
2. 填充变量并提交模型
3. 用 JSON Schema 校验格式
4. 调用 QC 模板检查质量
5. 失败时调用重生成模板
6. 成功后写入 `resource_packs/lessons/cards/output_tasks`
7. 记录 `ai_generation_jobs` 请求和结果

## 11. 版本管理建议
- `template_version` 使用语义化版本，如 `pack.v1.0.0`
- 每次改模板都记录变更目的与影响指标
- 保留最近 3 个稳定模板用于灰度对比
