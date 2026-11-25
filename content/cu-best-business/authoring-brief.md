# Authoring brief: CU-BEST Business Analysis (course slug `cu-best-business`)

You are the Content Admin author for one Topic of a Thai tutoring platform. Read
`/home/thekogo/project/tutor-platform/content/cu-best-business/graph.yaml` first (header comments explain what the exam part tests:
analytical reading of Thai business-news passages + business vocabulary, GMAT-Critical-Reasoning style; NOT business subject knowledge).
Domain terms: /home/thekogo/project/tutor-platform/CONTEXT.md.

## Output
Write ONE JSON file (UTF-8, valid JSON, no comments) at the path given in your task:
{
  "lessons":   [ { "concept": "<slug>", "body_md": "<Thai Markdown Lesson>" } ],          // one per Concept of your Topic
  "questions": [ <QuestionInput>, ... ]                                                     // exactly 12 per Concept
}
QuestionInput:
{ "stem_md": str, "explanation_md": str,
  "choices": [ {"label":"A","body_md":str,"is_correct":bool}, ... exactly 5: A B C D E, exactly one correct ],
  "concepts": [slugs], "primary_concept": slug (one of YOUR Topic's Concepts, must be in concepts),
  "difficulty": 1|2|3, "format": "short-passage"|"long-passage"|"data-passage", "source": "ai_generated" }
Do NOT call any MCP tool or modify the repository. Only write your JSON file. When done, run
`python3 -c "import json,sys;d=json.load(open(sys.argv[1]));print(len(d['lessons']),len(d['questions']))" <file>` to check it parses.

## Per Concept (12 Questions)
- Difficulty mix: 4 × 1, 5 × 2, 3 × 3.
- Correct-answer position balanced: across each Concept's 12, use every label A–E at least twice; no visible pattern.
- Formats: mostly short-passage (3–6 sentences); ~3 long-passage (2–4 paragraphs) per Concept; use data-passage whenever the passage
  carries figures (required for the data-in-passage Topic; occasional elsewhere).
- `concepts`: the primary Concept, plus 0–2 other Concepts of this course the Question genuinely also tests (prerequisites are typical).
  Only slugs that exist in graph.yaml.
- Every Question stands alone (its own passage in the stem). Stem = passage, blank line, then the question line in **bold** or plain.
- Passages: realistic Thai business/economics news, reports, memos, executive statements, market commentary. Use FICTIONAL companies and
  people (e.g. บริษัท สยามฟู้ดเทค จำกัด); never attribute claims or figures to real companies, people or real recent events. Vary industries
  (retail, banking, tourism, agriculture, energy, tech startups, logistics, F&B, real estate, healthcare, SMEs, public policy).
- Question stems in Thai exam style, e.g. "ข้อใดเป็นข้อสรุปของข้อความข้างต้น", "ข้อใดถูกต้องตามข้อความ", "ข้อใดหากเป็นจริงจะทำให้ข้อโต้แย้งข้างต้นมีน้ำหนักน้อยลงที่สุด",
  "ผู้เขียนมีทัศนคติต่อ ... อย่างไร", "ชื่อเรื่องใดเหมาะสมที่สุด", "ข้อความใดเติมลงในช่องว่างได้เหมาะสมที่สุด".
- Exactly ONE defensibly correct answer. Distractors are plausible and tied to typical errors (too broad, too narrow, out of scope,
  reverses cause/effect, overstated qualifier "ทั้งหมด/เสมอ", restates a premise instead of the conclusion, uses outside knowledge,
  misreads a figure, wrong base for a percentage). No "ถูกทุกข้อ / ไม่มีข้อใดถูก". Choices roughly similar length.
- Numeric Questions: compute carefully and double-check arithmetic; KaTeX allowed with $...$.
- Hard (3) = subtle distractor pairs, long passage, or two-step reasoning — still unambiguous.
- explanation_md (Thai): why the key is right (point to the sentence/logic in the passage), then a final line
  "ตอบ X — " followed by why the 1–3 most tempting distractors are wrong. X must equal the correct label.

## Lesson body_md (Thai Markdown), same structure as the math course:
## แนวคิดหลัก   (what the skill is; mention which prerequisite Concept it builds on, by name)
## หลักการ/วิธีคิดที่ต้องจำ   (bullets: steps, signal words, patterns)
## ตัวอย่าง   (**ตัวอย่าง 1 (ง่าย)**, **ตัวอย่าง 2 (กลาง)**, **ตัวอย่าง 3 (ยาก)** — short passage + reasoning)
## ข้อผิดพลาดที่พบบ่อย
## เทคนิคทำข้อสอบเร็ว
~400–800 Thai words. Lesson examples must not reuse any Question's passage.

## Quality bar
Before finishing, re-read every Question as a strict examinee: if two choices could be argued correct, rewrite. Check label balance,
difficulty counts, exactly-one is_correct, and that "ตอบ X" matches.
