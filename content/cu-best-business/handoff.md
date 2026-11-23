# Handoff: CU-BEST Business Analysis content (run 1, 2026-11-23)

What the first authoring run produced for Course `cu-best-business`, how it was made, and what the next run should do.
Vocabulary: [CONTEXT.md](../../CONTEXT.md). Workflow rules: [ADR 0003](../../docs/adr/0003-ai-agent-authors-content-via-mcp.md).

## Status

| Item | Target (mirrors cu-best-math) | Run 1 result |
|---|---|---|
| Knowledge graph | from course outline | 6 Topics, 24 Concepts, 29 Prerequisites ([graph.yaml](graph.yaml)) |
| Lessons | 1 per Concept | 24 / 24 written |
| Questions | 12 per Concept (math has 9–12 Active) | 288 created as Draft (12 per Concept) |
| Verification | independent pass; match → Active | **288 / 288 Active** (0 mismatches, 0 flagged) |
| Mock Exams | 2 × 30 Questions, 60 min | 2 created: "Mock Exam 1/2 (ข้อสอบเสมือนจริง ชุดที่ 1/2)", each 8 easy / 13 medium / 9 hard |
| Placement Test / Quizzes | served automatically from Active Questions | working; each Concept keeps 9 Questions for Quizzes (3 at most went to Exams) |

All of this was written to the **local dev** server (`localhost:8080`, `.env` MCP token).

**Production (2026-11-23):** commit `9977770` is deployed to Cloud Run (`tutor-backend-00006`, `tutor-frontend-00003`). The Course content is **not loaded on prod yet**: reading the prod MCP token from Secret Manager needs the Owner. The one-shot loader `prod_load.py` and its `prod_mapping.json` are in the session scratchpad (`…/scratchpad/business/`). It imports the graph, loads the Lessons and Drafts, re-applies the verifier's answers by stem, and creates the same 2 Exams. It refuses to run if the Course already has Questions, and it was rehearsed end to end on the e2e stack (288 Active, 2 Exams).

## How run 1 was made

1. **Outline research** (one agent): no official syllabus exists. Chula publishes only "Business Analysis 50% + Math 50%, GMAT-like".
   The topic list comes from Thai tutoring schools (ChulaTutor, Bright Brain). Sources and which items are inferred are listed in
   the header of [graph.yaml](graph.yaml). The part tests analytical reading of business passages, not business subject knowledge.
2. **Authoring** (six agents, one per Topic, in parallel). All followed [authoring-brief.md](authoring-brief.md) and each wrote one JSON file
   `{lessons:[…], questions:[QuestionInput…]}`. The agents never called MCP.
   - Per Concept: 12 Questions, difficulty 4×1 / 5×2 / 3×3, 5 choices A–E, each label is the key at least twice, fictional companies.
   - Formats: `short-passage`, `long-passage`, `data-passage`.
   - Each Explanation ends with `ตอบ X — …why the tempting distractors are wrong` (same convention as math).
3. **Validate + load** centrally with [`scripts/content_load.py`](../../scripts/content_load.py). It checks labels, exactly one key,
   `ตอบ X` = key, slugs, per-Concept counts, key balance and Lesson headings. Then, with `--write`, it calls `upsert_lesson` and
   `create_questions` (Draft).
4. **Verification** (one separate agent): it calls `list_questions_for_verification`, which returns Drafts without keys, solves each
   Question, and calls `submit_verification`. It had no access to the authors' files.

The authored JSON (with answer keys) is intentionally **not** in git. The database is the source of truth, and keys in the repo would
spoil Questions. `get_pool_stats` reads the database back, but it counts a Question once per tagged Concept, so its totals exceed 288.

## Authors' notes to review

- **business-vocabulary**: a few items go a little beyond plain vocabulary, e.g. stagflation, real interest rate, break-even, NPL,
  share dilution, and vertical vs. horizontal integration. Drop them if they read as subject knowledge.
- **argument-evaluation**: paradox-resolution P5 and P10 are "which does NOT explain" (EXCEPT) items. P10 is deliberately subtle.
- **analytical-reading-basics**: the hardest items to double-check are fact-vs-opinion Q4 and Q7, conclusion-evidence Q7, and
  argument-structure Q12.
- **reading-techniques**: two inference items also carry the `causal-reasoning` tag. Contradiction items always include a
  "not mentioned" distractor on purpose.
- **keywords**: one passage names a real port (แหลมฉบัง), with no figures attributed to it.
- **Lesson length**: all Lessons are about 2,300–3,800 Thai characters, roughly 450–800 words.

## Verification

- **Result:** a separate agent submitted all 288 Drafts and every chosen label matched the key, so all 288 are Active. It used only
  `list_questions_for_verification` and `submit_verification`, never read the authors' files, and flagged nothing AMBIGUOUS or ERROR.
- **Weakest key** (fix wording on the next pass): Question `508f6aee…`, an author-viewpoint item. Choice A ("dismisses the objection
  as baseless") is close to the key E, because the author actually answers the objection with a remedy. Make A clearly wrong.
- **What 288/288 means:** it only says the keys are consistent for one independent solver. Learner Flags remain the safety net
  (one Flag pulls a Question).

## Exams

- **How they were built:** each Exam takes one Question from every Concept, plus 6 extras from reasoning-heavy Concepts:
  - Exam 1: strengthen, weaken, inference, assumption, main-idea, numbers-in-passage
  - Exam 2: paradox-resolution, causal-reasoning, detail-check, conclusion-evidence, text-completion, growth-and-share
- **Order:** by Topic, easy to hard within each Topic.
- **Seed:** selection was seeded (20261123) so it can be reproduced. Exam Questions are excluded from Quizzes.

## End-to-end check (dev, 2026-11-23)

- **What ran:** a REST script ran as a throwaway Learner `business-check@dev.test`, which still exists in the dev database (harmless;
  delete it if it bothers you). It ran a Placement Test (11 Questions across 5 Topics), a Quiz on `strengthen` (9 Questions, with
  feedback and Tutor Chat offered) and Mock Exam 1 (30 Questions, feedback at the end).
- **Per-Course status:** after the Placement Test, `placementDone` was true for business and still false for math.

## Next run: checklist

1. **Production (not done yet)**:
   - seed the graph: `DATABASE_URL=<prod> make seed COURSE=cu-best-business`
   - re-load the content with `MCP_URL=https://<prod-backend>/mcp MCP_TOKEN=<prod token> python3 scripts/content_load.py … --write`
     using the authored JSON. Run 1's JSON is only in the session scratchpad
     (`/tmp/claude-1000/…/scratchpad/business/*.json`), which may be cleared. If it is gone, export the Questions from the dev
     database or regenerate them from the brief.
   - re-verify on prod with a fresh independent agent, then re-create the 2 Exams
     (`exams.py` logic: 1 per Concept + 6 extras, 8/13/9 difficulty)
   - deploy (see the Cloud Run notes)
2. **Polish**: reword choice A of `508f6aee…` (see Verification). Review the authors' notes above.
3. **Calibrate** once real past papers are found:
   - set Topic `exam_weight`
   - revisit the three inferred Concepts (`assumption`, `causal-reasoning`, `paradox-resolution`)
   - add `imported` Questions with `source_ref`
4. **Watch Flags and Content Requests**: run `list_flags` and `list_content_requests` after Learners start.

## Re-running authoring for one Topic

```sh
# 1. Give an agent content/cu-best-business/authoring-brief.md + the Topic's Concept slugs; it writes <topic>.json
# 2. Validate, then load
COURSE=cu-best-business python3 scripts/content_load.py <topic>.json
COURSE=cu-best-business python3 scripts/content_load.py <topic>.json --write
# 3. Independent verification agent: list_questions_for_verification → submit_verification (no access to <topic>.json)
```
