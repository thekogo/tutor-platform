# Authoring brief: CU-TEP (course slug `cu-tep`)

You are the Content Admin author for one Topic of a Thai tutoring platform. Read
`/home/thekogo/project/tutor-platform/content/cu-tep/graph.yaml` first. Its header explains the exam: 120 four-choice items in three
sections (Listening 30, Reading 60 with a 15-blank cloze, Writing 30 error identification). The `# technique:` comment above each
Concept is the exam technique that Concept's Lesson and Questions must teach and exercise. Domain terms:
/home/thekogo/project/tutor-platform/CONTEXT.md.

## Output
Write ONE JSON file (UTF-8, valid JSON, no comments) at the path given in your task:
{
  "lessons":   [ { "concept": "<slug>", "body_md": "<Lesson Markdown>" } ],          // one per Concept of your Topic
  "questions": [ <QuestionInput>, ... ]                                               // exactly 12 per Concept
}
QuestionInput:
{ "stem_md": str, "explanation_md": str,
  "choices": [ {"label":"A","body_md":str,"is_correct":bool}, ... exactly 4: A B C D, exactly one correct ],
  "concepts": [slugs], "primary_concept": slug (one of YOUR Topic's Concepts, must be in concepts),
  "difficulty": 1|2|3, "format": <a question_formats id from graph.yaml>, "source": "ai_generated" }
Do NOT call any MCP tool or modify the repository. Only write your JSON file. When done, run
`python3 -c "import json,sys;d=json.load(open(sys.argv[1]));print(len(d['lessons']),len(d['questions']))" <file>` to check it parses.

## Language
- Stems, passages, transcripts and choices: **English**, as on the real test. Exam-style instructions in English
  (e.g. "Which of the underlined parts is incorrect?", "What is the main idea of the passage?", "What does the woman mean?").
- explanation_md and Lessons: **Thai**, with English examples and grammar terms kept in English where natural.

## Formats (the `format` field)
- `error-identification` (grammar and error-ID Topics; the main format there): one English sentence with exactly four marked parts,
  written as `**[A] word(s)**`, `**[B] ...**`, `**[C] ...**`, `**[D] ...**` in order. Markdown cannot underline, so this bold-label
  form stands in for underlining. Stem = "Which of the underlined parts is incorrect?" line, blank line, the sentence.
  Choices are the four marked parts' text (choice A = the text of [A], etc.). Exactly one marked part is wrong; the other three must be
  clearly correct standard written English. Put the error in a varied position, and make the error fixable by changing only that part.
- `sentence-completion`: a sentence with a blank `_____` and four options. Practice format, not on the real test; use it for at
  most 3 of a grammar Concept's 12 Questions (good for difficulty 1).
- `cloze-passage` (Cloze Topic): a paragraph of 120–250 words with 1 numbered blank, written `(___)`, that the Question asks about.
  Each Question stands alone, so each has its own paragraph and one blank. Ask "Which choice best fills the blank?".
- `short-passage` (100–250 words, including letters, emails, notices) and `long-passage` (350–600 words, multiple paragraphs).
  Academic and general-interest topics: science, health, environment, history, technology, education, psychology, economics, culture.
- `short-conversation`: a transcript of 2–4 turns, `**Man:** ...` / `**Woman:** ...`, then the question
  ("What does the woman mean?", "What will the man probably do next?"). Put a one-line italic note at the top of the stem:
  `*(บทถอดความ — ในข้อสอบจริงจะได้ยินเพียงครั้งเดียว)*`.
- `long-conversation` (8–14 turns) and `talk` (a 150–300-word lecture, announcement or radio talk, `**Speaker:**` or
  `**Professor:**`). Same italic note.
- Every Question stands alone with its own sentence, passage or transcript in the stem. Do not reference other Questions.

## Per Concept (12 Questions)
- Difficulty mix: 4 × 1, 5 × 2, 3 × 3.
- Correct-answer position balanced: across each Concept's 12, use every label A–D at least twice (ideally 3 each); no visible pattern.
- `concepts`: the primary Concept plus 0–2 other Concepts of this course the Question genuinely also tests (prerequisites are typical).
  Only slugs that exist in graph.yaml.
- Original content only: do not copy items from real CU-TEP, TOEFL or other published tests. Fictional people and organisations;
  facts in passages should be general and plausible, never attributed to real living people or recent real events.
- Exactly ONE defensibly correct answer. Distractors must be plausible and reflect typical errors for that Concept (for reading:
  too broad or narrow, copies text words without the meaning, contradicts, not mentioned, extreme qualifier; for listening: repeated
  or sound-alike word, literal reading of an idiom, wrong speaker; for grammar: a correct-looking part that tempts over-correction).
  No "all of the above / none of the above". Choices roughly similar length.
- Error ID: the three non-error parts must be unarguably correct (watch British/American variants, optional commas and
  style preferences; avoid them). Explanation states the correction.
- Hard (3) = long sentence with stacked modifiers, subtle distractor, long passage or two-step reasoning, but still unambiguous.
- explanation_md (Thai): why the key is right (quote the clue in the sentence, passage or transcript; name the technique from the
  Concept's `# technique:` note), for error ID give the corrected form, then a final line `ตอบ X — ` followed by why the 1–2 most
  tempting distractors are wrong. X must equal the correct label.

## Lesson body_md (Thai Markdown, English examples)
## แนวคิดหลัก   (what the skill or rule is and where it appears in CU-TEP, which section and part; mention the prerequisite Concept it builds on, by name)
## หลักการ/วิธีคิดที่ต้องจำ   (bullets: rules, patterns, signal words; tables are welcome, GFM is supported)
## เทคนิคทำข้อสอบ   (step-by-step, built from the Concept's `# technique:` note: what to look at first, how to eliminate, time budget)
## ตัวอย่าง   (**ตัวอย่าง 1 (ง่าย)**, **ตัวอย่าง 2 (กลาง)**, **ตัวอย่าง 3 (ยาก)**, each a CU-TEP-style item + reasoning walk-through)
## ข้อผิดพลาดที่พบบ่อย
About 500–900 Thai words plus examples. Lesson examples must not reuse any Question's sentence or passage.

## Quality bar
Before finishing, re-read every Question as a strict examinee: if two choices could be argued correct, rewrite. For error ID, check the
three non-error parts again. Check label balance, difficulty counts, exactly-one is_correct, and that "ตอบ X" matches.
