# English Anywhere Lab - 学习包类型体系规范（Pack Template + Exercise Type）

## 1. 文档目标
将“学习包”从泛概念落地为可执行规格，统一以下对象定义：
- 学习包模板类型（Pack Template）
- 练习交互类型（Exercise Type）
- 模板与练习的组合规则
- 完成口径与质量门禁

本文作为产品、后端、前端、AI 资源生成与测试的共同基线。

## 2. 核心定义
### 2.1 学习包（Learning Pack）
学习包是“在明确学习目标下，可在 3-7 天完成的一组结构化课程单元”。

学习包必须满足：
- 有明确目标（学完后可完成的真实任务）
- 有目标人群（CEFR 等级 + 领域）
- 有结构化单元（lessons）
- 有可完成与可评估的练习（exercise）
- 有包级完成标准（completion rule）

### 2.2 两层类型模型
- 第一层：`pack_template`（学习包模板）
- 第二层：`exercise_type`（练习交互类型）

约束：
- 任何可学习任务都必须属于某个 `pack_template`。
- 任何任务实例都必须标注明确的 `exercise_type`。
- 禁止无类型任务进入正式学习流程。

## 3. 学习包模板类型（Pack Template）

| 模板编码 | 名称 | 核心目标 | 目标技能 | 推荐时长 |
|---|---|---|---|---|
| `vocab_foundation` | 词汇巩固包 | 建立高频词识别、理解、调用能力 | 读/写/说 | 3-5 天 |
| `scenario_dialog` | 场景会话包 | 在真实场景完成基本交流 | 听/说/读 | 3-7 天 |
| `intensive_listening` | 精听训练包 | 强化辨音、抓关键词、理解细节 | 听/写 | 3-5 天 |
| `reading_comprehension` | 阅读理解包 | 快速获取主旨、细节与推断信息 | 读/写 | 3-5 天 |
| `writing_output` | 写作输出包 | 从句子到短段，提升表达准确与连贯 | 写 | 3-7 天 |
| `review_booster` | 复习强化包 | 对遗忘高风险内容做高强度回收 | 读/写/说 | 3-4 天 |
| `speaking_bootcamp` | 口语表达包 | 提升开口速度、流畅度与场景表达 | 听/说 | 3-7 天 |
| `exam_drill` | 考试专项包 | 针对题型与答题策略做训练 | 读/听/写/说 | 5-7 天 |

阶段建议：
- MVP：`vocab_foundation`、`scenario_dialog`、`intensive_listening`、`reading_comprehension`、`writing_output`、`review_booster`
- V1：新增 `speaking_bootcamp`、`exam_drill`

## 4. 练习交互类型（Exercise Type）字典

### 4.1 输入识别与理解类
| 练习编码 | 名称 | 典型输入 | 典型输出 | 自动判分 |
|---|---|---|---|---|
| `flashcard_recall` | 闪卡回忆 | 词/句正反面 | 自评或选择 | 是 |
| `picture_naming` | 看图说词 | 图片+提示 | 文本/语音 | 部分 |
| `audio_cloze` | 听力填空 | 音频+空位文本 | 文本 | 是 |
| `dictation_short` | 听写短句 | 音频 | 文本 | 是 |
| `reading_mcq` | 阅读选择 | 短文+题干 | 选项 | 是 |
| `keyword_match` | 关键词匹配 | 词组集合 | 匹配结果 | 是 |

### 4.2 结构操练类
| 练习编码 | 名称 | 典型输入 | 典型输出 | 自动判分 |
|---|---|---|---|---|
| `cloze_word` | 单词填空 | 上下文+空位 | 单词 | 是 |
| `cloze_sentence` | 句子填空 | 句子+空位 | 短语/句段 | 是 |
| `sentence_reorder` | 句子重排 | 打乱片段 | 顺序 | 是 |
| `error_correction` | 语法纠错 | 错句 | 修正句 | 部分 |
| `sentence_rewrite` | 句子改写 | 原句+约束 | 改写句 | 部分 |

### 4.3 输出与应用类
| 练习编码 | 名称 | 典型输入 | 典型输出 | 自动判分 |
|---|---|---|---|---|
| `situational_qa` | 情景问答 | 场景+问题 | 文本/语音 | 部分 |
| `picture_description` | 看图描述 | 图片+约束词 | 文本/语音 | 部分 |
| `guided_paragraph` | 引导段落写作 | 主题+提纲 | 段落 | 部分 |
| `free_writing` | 自由写作 | 题目 | 段落/短文 | 否（需AI辅助） |
| `retell_audio` | 听后复述 | 音频+要点 | 文本/语音 | 部分 |

说明：
- “自动判分 = 部分”表示可做规则判分 + AI 结构化反馈混合评估。
- “自动判分 = 否”表示主评估依赖 AI 反馈，需配套校验规则。

## 5. 模板与练习组合规则（MVP）

| `pack_template` | 必选练习（至少） | 可选练习（任一） | 禁止项 |
|---|---|---|---|
| `vocab_foundation` | `flashcard_recall` + `cloze_word` | `picture_naming` / `sentence_rewrite` | 仅输出类任务构成全包 |
| `scenario_dialog` | `situational_qa` + `sentence_reorder` | `picture_description` / `audio_cloze` | 无场景上下文的纯词表包 |
| `intensive_listening` | `audio_cloze` + `dictation_short` | `retell_audio` / `keyword_match` | 无音频输入任务 |
| `reading_comprehension` | `reading_mcq` + `cloze_sentence` | `keyword_match` / `error_correction` | 只做选择题无输出 |
| `writing_output` | `sentence_rewrite` + `guided_paragraph` | `error_correction` / `free_writing` | 无可提交文本任务 |
| `review_booster` | `flashcard_recall` + `cloze_word` | `audio_cloze` / `situational_qa` | 仅新学不含复习 |

额外约束：
- 每个 `lesson` 至少 2 种不同 `exercise_type`，避免题型单一。
- 每个 `pack` 至少 1 个输出类任务（`situational_qa`/`guided_paragraph`/`free_writing`/`picture_description`/`retell_audio`）。
- 单包总任务数建议 18-36（按 `daily_minutes` 自动缩放）。

## 6. 完成标准与通过规则
### 6.1 任务级完成
- 客观题：提交并通过规则校验即完成。
- 输出题：提交成功且得到结构化反馈即完成。
- 复习题：评分提交成功并更新下次到期即完成。

### 6.2 课程级完成
一个 lesson 满足以下条件视为完成：
- lesson 完成率 >= 80%
- lesson 内必选练习全部至少完成 1 次

### 6.3 包级完成
一个 pack 满足以下条件视为完成：
- lesson 完成率 >= 80%
- 必选练习覆盖率 100%
- 输出类任务完成数 >= 2（MVP）

## 7. 数据模型映射（基于当前结构）
对应表：`resource_packs`、`lessons`、`cards`、`output_tasks`

字段映射建议：
- `resource_packs.metadata.pack_template`: 学习包模板编码
- `resource_packs.metadata.completion_rule`: 包级完成规则（JSON）
- `lessons.lesson_type`: 继续使用现有 `reading/listening/speaking/writing/mixed`
- `lessons.content_json.exercise_mix`: lesson 题型配比
- `cards.metadata.exercise_type`: 卡片类练习编码
- `output_tasks.metadata.exercise_type`: 输出类练习编码

规范约束：
- 所有题目必须落库后再进入计划与学习链路。
- 禁止“随机生成且不落库的虚拟任务”进入正式计分与进度统计。

## 8. API 与前端渲染约定
### 8.1 API 返回最小字段（建议）
任务对象建议至少包含：
- `task_id`
- `task_type`（input/retrieval_quiz/review/output）
- `exercise_type`（本文件字典）
- `prompt`
- `assets`（可选：image/audio）
- `evaluation_mode`（objective/rubric/llm_hybrid）

### 8.2 前端渲染策略
- 由 `exercise_type` 驱动组件渲染，而不是依赖自由文本判断。
- 未识别的 `exercise_type` 必须降级为“通用任务组件”并上报埋点。

## 9. AI 生成约束（用于提示词与 QC）
生成任务必须满足：
- 明确输出 `pack_template`
- 每个任务包含合法 `exercise_type`
- 模板必选练习覆盖率达到要求
- 内容难度与 CEFR 等级一致
- 输出题保证可评估（有 rubric 或结构化反馈 schema）

最小 QC 清单：
- 字段完整性：`title/level/domain/exercise_type/prompt`
- 题型合法性：`exercise_type` 在白名单中
- 重复度阈值：同包相似题占比 < 20%
- 违规内容检测：空题、乱码、提示词泄漏

## 10. 埋点与分析口径
新增或强制字段：
- `pack_template`
- `exercise_type`
- `evaluation_mode`

核心分析问题：
- 哪种模板完成率最高
- 哪种练习类型中断率最高
- 哪种练习类型对复习准确率提升最明显

## 11. 实施优先级（建议）
1. 文档与提示词先统一 `pack_template + exercise_type` 字典。
2. 后端写入链路强制校验类型合法性。
3. 前端用 `exercise_type` 做渲染分发。
4. 统计看板按模板/题型提供切片分析。
5. V1 再扩展高级口语与考试专项题型。

## 12. 版本声明
- 当前版本：v1.0（2026-03-04）
- 变更原则：
  - 新增类型只能“向后兼容”
  - 删除或重命名类型必须提供迁移策略与兼容期
