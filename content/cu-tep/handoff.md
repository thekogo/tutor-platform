# Handoff: CU-TEP content (run 1, 2026-11-24)

What the first authoring run produced for Course `cu-tep`, how it was made, and what the next run should do.
Vocabulary: [CONTEXT.md](../../CONTEXT.md). Workflow rules: [ADR 0003](../../docs/adr/0003-ai-agent-authors-content-via-mcp.md).
Same pipeline as [cu-best-business](../cu-best-business/handoff.md).

## Status

| Item | Target | Run 1 result |
|---|---|---|
| Knowledge graph | from researched exam structure | 7 Topics, 30 Concepts, 47 Prerequisites ([graph.yaml](graph.yaml)); `make graph-check COURSE=cu-tep` passes |
| Lessons | 1 per Concept | 30 / 30 |
| Questions | 12 per Concept, 4 choices A–D | 360 created as Draft |
| Verification | independent pass; match → Active | **360 / 360 Active** (0 mismatches, 0 flagged) |
| Mock Exams | section-proportional | 2 × 40 Questions, 45 min (see Exams) |

All of this was first written to **local dev** (`localhost:8080`).

**Production (2026-11-24):** no image rebuild was needed (commit `5989663` only touched a test and a local script; running
revisions already serve any Course). Seeded the graph on prod, loaded 30 Lessons and 360 Questions through prod MCP, activated all 360
with the dev verifier's answers matched by stem + choices (360/360 matched), and re-created both Mock Exams with the same Questions
and order. Prod Course id `24a3ef83-3e54-4f31-8f56-366f1e4136a1`. Loaders `cutep_prod_verify.py` / `cutep_prod_exams.py` are in the
session scratchpad. Not yet clicked through as a Learner on prod (Google login only).

## How run 1 was made

1. **Outline research** (one agent). The official Chula Academic Testing Center site only names the three skills. The item breakdown
   (120 items: Listening 30, Reading 60 incl. a 15-blank cloze, Writing 30 error identification; 4 choices; no penalty) is from
   consistent secondary sources listed in the [graph.yaml](graph.yaml) header, together with what is sourced vs. inferred.
   Every Concept has a `# technique:` comment that the Lessons and Questions were built around.
2. **Authoring** (eight agents in parallel: one per Topic, Reading Comprehension split in two and merged) following
   [authoring-brief.md](authoring-brief.md). English stems/passages/transcripts, Thai Lessons and Explanations.
3. **Validate + load** with `COURSE=cu-tep LABELS=ABCD python3 scripts/content_load.py … [--write]`. `LABELS` was added to the loader
   for this Course (default stays `ABCDE`).
4. **Verification** (two separate agents, split by format, no access to the authored files): `list_questions_for_verification` →
   `submit_verification`. All 360 matched.

The authored JSON (with keys) is intentionally **not** in git. Run 1's files are in the session scratchpad
(`/tmp/claude-1000/…/scratchpad/cutep/*.json`, `reading-comprehension.json` is the merged reading file) and may be cleared.

## Platform constraints and how they were handled

- **No audio.** Listening is the Topic `listening-transcripts`: written transcripts in the stem, an italic note that the real test plays
  audio once, and Lessons that say it trains question types and technique, not the ear. Learners still need real audio practice.
- **No underline in Markdown** (no raw HTML). Error-identification parts are written `**[A] text**`; choice A's text is part [A].
- **Formats:** `error-identification`, `sentence-completion` (practice only, ≤3 per grammar Concept), `cloze-passage` (one blank per
  Question, since each Question stands alone), `short-passage`, `long-passage`, `short-conversation`, `long-conversation`, `talk`.

## Exams

- **Shape:** each is 40 Questions, 45 min, about one third of the real 120 items / 2 h 10 min, in the real test's section order:
  Listening 8 (2 per Concept) → Cloze 4 (2 per Concept) → Reading 16 (2 per Concept) → Writing 12 error-identification items
  (12 of the 18 grammar/error-ID Concepts per Exam, ≤2 per Concept across both). Within each section, easy to hard.
- **Difficulty:** 12 easy / 16 medium / 12 hard each.
- **Pool left for Quizzes:** 8–11 Active Questions per Concept.
- **Reproducible:** seed 20261124; builder script `cutep_exams.py` (session scratchpad).

## Known gaps / review notes

- **Passage length:** long passages are 320–395 words, shorter than the real test's 600–700. Mock Exams feel easier on stamina.
- **Placement Test skew:** it samples per Concept, and 16 of 30 Concepts are grammar, so a run served 13 of 18 Questions as error ID.
  It does not follow the 25/50/25 score split. Fine for diagnosing, but worth a look if placement should mirror the exam.
- **exam_weight** is set on every Topic (score points ÷ 10); the even split of Writing over the four grammar Topics is inferred.
- **Reused passages:** some reading passages are shared by 2–4 Questions (each stem carries the full passage).
- **Authors' least-sure items:** all passed verification, but they are the first to check if Flags come in:
  - pronoun "neither shares **their** data"
  - "economy growth"
  - "a drop **of** visitor numbers"
  - cloze "heavy rain"
  - attitude "Again? That's the third time this week"
  - reference "former"
  - vocab "benign"
- **End-to-end (dev):** throwaway Learner `cutep-check@dev.test` ran a Placement Test (18 Questions) and Mock Exam 1 (40). It still
  exists in the dev database (harmless).

## Next run: checklist

1. **Production**: done 2026-11-24 (see Status). Open the Course on prod as a Learner once to confirm the UI.
2. **Calibrate** once real past papers are reviewed: the inferred grammar points and reading question types, and `exam_weight`.
3. **Longer passages:** add a few 600–700-word passages for the Mock Exams.
4. **Watch Flags and Content Requests** after Learners start.
