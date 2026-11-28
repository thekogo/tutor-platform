# Adding a new Course to production

How to take a new Course from nothing to live on production (https://tutor.ponlawat.me, the homelab).
Vocabulary is in [CONTEXT.md](../CONTEXT.md). Content is authored by an AI agent through MCP ([ADR 0003](adr/0003-ai-agent-authors-content-via-mcp.md)).
Worked examples: [content/cu-tep/handoff.md](../content/cu-tep/handoff.md) and [content/cka/handoff.md](../content/cka/handoff.md).

A new Course needs **no code change and no image rebuild**. The running backend serves any Course in the database.
Everything below is content, loaded through the MCP server.

## Overview

| Step | Where | Output |
|---|---|---|
| 1. Research the exam | local | sources + exam structure |
| 2. Write the knowledge graph | local, `content/<slug>/graph.yaml` | Topics, Concepts, Prerequisites, Question Formats |
| 3. Write the authoring brief | local, `content/<slug>/authoring-brief.md` | rules for Lessons and Questions |
| 4. Author Lessons + Questions | local JSON (not in git) | one JSON file per Topic |
| 5. Validate | local | `content_load.py` passes |
| 6. Back up prod | homelab | `pg_dump` file |
| 7. Import the graph | homelab → MCP | Course exists on prod |
| 8. Load Lessons + Questions | homelab → MCP | Lessons + Draft Questions |
| 9. Verify Questions | MCP, independent verifier | Draft → Active |
| 10. Create Mock Exams (optional) | MCP | timed Exams |
| 11. Check as a Learner, write the handoff | browser + `content/<slug>/handoff.md` | done |

---

## 1. Research the exam

Find the official structure: sections, item counts, time, scoring, the official syllabus or competencies.
Prefer official pages and use secondary sources only to fill gaps. Write down which facts are **sourced** and which are **inferred**;
they go into the graph header in step 2.

## 2. Write the knowledge graph

Create `content/<slug>/graph.yaml`. Copy the shape of an existing Course (for example `content/cu-tep/graph.yaml`):

```yaml
# <Course> — knowledge graph (draft v0.1)
# Exam structure, sources, sourced vs. inferred, how exam_weight was set ...

course:
  id: <slug>                 # lowercase, hyphens; this is the Course slug everywhere
  name: <display name>
  goal: <one-line goal>

question_formats:
  - id: <format-slug>
    name: <display name>

topics:
  - { id: <topic-slug>, name: "<name>", exam_weight: 1.5 }   # quote names that contain commas or colons

concepts:
  # technique: <the exam technique the Lesson and Questions must teach>
  - id: <concept-slug>
    topic: <topic-slug>
    name: <display name>
    prerequisites: [<concept-slug>, ...]
```

Rules:
- `exam_weight` = the Topic's share of the score ÷ 10 (or `null` if unknown). Explain how you set it in the header.
- `prerequisites` must form a DAG (no cycles) and reference only existing Concepts.
- **Quote** any `name` containing `,` `:` `#` `{` `}` inside a `{ ... }` flow mapping. An unquoted comma silently creates extra keys;
  `graph_check.py` does not catch it, but the MCP import rejects it.

Validate and render the Mermaid map:

```bash
make graph-check COURSE=<slug>        # writes content/<slug>/graph.md
python3 -c "import yaml; g=yaml.safe_load(open('content/<slug>/graph.yaml')); \
  assert all(set(t)=={'id','name','exam_weight'} for t in g['topics']); \
  assert all(set(c)=={'id','topic','name','prerequisites'} for c in g['concepts']); print('keys OK')"
```

## 3. Write the authoring brief

Create `content/<slug>/authoring-brief.md` (copy `content/cu-tep/authoring-brief.md` and adapt it). It must define:
- the output JSON shape (below)
- the language of stems, choices, Explanations and Lessons
- the number of choices (`ABCD` or `ABCDE`)
- each Question Format and how a stem in that format looks
- per Concept: 12 Questions, difficulty mix (for example 4 × 1, 5 × 2, 3 × 3), balanced answer keys (every label at least twice)
- the Lesson headings
- the Explanation ending: `ตอบ X — …` or `Answer: X. …` (the loader checks it matches the key)

## 4. Author Lessons and Questions

Authors (people or agents) write **one JSON file per Topic**, following the brief:

```json
{
  "lessons": [ { "concept": "<concept-slug>", "body_md": "<Lesson Markdown>" } ],
  "questions": [
    {
      "stem_md": "...",
      "explanation_md": "... Answer: B. ...",
      "choices": [
        {"label": "A", "body_md": "...", "is_correct": false},
        {"label": "B", "body_md": "...", "is_correct": true},
        {"label": "C", "body_md": "...", "is_correct": false},
        {"label": "D", "body_md": "...", "is_correct": false}
      ],
      "concepts": ["<primary>", "<optional other>"],
      "primary_concept": "<concept-slug of this Topic>",
      "difficulty": 1,
      "format": "<question_formats id>",
      "source": "ai_generated"
    }
  ]
}
```

- One Lesson per Concept of the Topic. Every Question's `primary_concept` belongs to that Topic.
- Keep these files **out of git**, because they contain the answer keys. Use the scratchpad or another local directory.

## 5. Validate locally

```bash
COURSE=<slug> LABELS=ABCD python3 scripts/content_load.py /path/to/*.json
```

Fix every `ERROR`. `warn` lines (fewer than 12 Questions, key balance, difficulty mix, missing Thai headings in non-Thai Courses)
do not block loading, but read them.

## 6. Back up the production database

```bash
ssh homelab 'mkdir -p ~/tutor-platform/backups && f=~/tutor-platform/backups/tutor-before-<slug>-$(date +%Y%m%d-%H%M%S).sql.gz \
  && docker exec postgres pg_dump -U tutor -d tutor | gzip > $f && ls -lh $f'
```

Restore if needed (stop the backend first):
`zcat <file> | docker exec -i postgres psql -U tutor -d tutor`. Check the target state before you restore over live data.

## 7–8. Import the graph and load the content (on the homelab)

**Why run on the homelab?** The public MCP URL is behind Cloudflare, which rejects `scripts/mcp_call.py` with error 1010
(its Python user-agent is banned). On the homelab, the local proxy on `127.0.0.1:8090` reaches the backend directly.
The MCP token lives only in `~/tutor-platform/.env` on the homelab. Read it at run time and never print it.

Copy the loader, the graph and the JSON files across:

```bash
ssh homelab 'mkdir -p ~/tutor-platform/<slug>-load/{scripts,content/<slug>,data}'
scp scripts/content_load.py scripts/mcp_call.py homelab:tutor-platform/<slug>-load/scripts/
scp content/<slug>/graph.yaml homelab:tutor-platform/<slug>-load/content/<slug>/
scp /path/to/*.json homelab:tutor-platform/<slug>-load/data/
python3 -c "import yaml,json; print(json.dumps({'graph': yaml.safe_load(open('content/<slug>/graph.yaml'))}, ensure_ascii=False))" > graph-args.json
scp graph-args.json homelab:tutor-platform/<slug>-load/data/
```

Then, on the homelab:

```bash
ssh homelab
cd ~/tutor-platform/<slug>-load
export MCP_URL=http://127.0.0.1:8090/mcp
export MCP_TOKEN=$(grep ^MCP_TOKEN= ../.env | cut -d= -f2-)

# 7. graph (idempotent; re-run after graph edits; archive_missing=false keeps removed Concepts)
python3 scripts/mcp_call.py import_course_graph data/graph-args.json

# 8. Lessons (upsert) + Questions (created as Draft)
COURSE=<slug> LABELS=ABCD python3 scripts/content_load.py data/*.json --write

# check
python3 scripts/mcp_call.py get_course_graph --json '{"course":"<slug>"}' | python3 -c \
  "import json,sys; g=json.load(sys.stdin); print(len(g['concepts']), 'concepts,', sum(c['has_lesson'] for c in g['concepts']), 'with lessons')"
python3 scripts/mcp_call.py get_pool_stats --json '{"course":"<slug>"}' > /tmp/pool.json
```

Note: `--write` **creates** Questions every time, so re-running it duplicates them. Load each file once; to fix a Question
later use `update_question` or `retire_question`. Lessons are upserted, so re-running is safe for them.

## 9. Verify Questions (Draft → Active)

Learners only see **Active** Questions. A Question becomes Active when an **independent verifier**, who has never seen the key
or the authored files, answers it and matches the key.

- The verifier calls `list_questions_for_verification` (`course`, `limit` up to 200), which returns no keys or Explanations.
- For each Question it calls `submit_verification` with `question_id`, `chosen_label` and a short `note`.
- Match → Active. Mismatch → stays Draft with the note. Review mismatches: fix the Question (`update_question`) and verify again,
  or `retire_question`.

The author must not verify their own Questions. In past runs a separate agent did this (one per batch or format), with access
only to the MCP tools. Check the result with `get_pool_stats`: `active` should equal what you loaded, and `draft` should be 0.

## 10. Create Mock Exams (optional)

`create_exam` takes `course`, `name`, `time_limit_s` and `question_ids` (Active Questions, in paper order). Exam Questions are
removed from Quizzes, so leave enough Active Questions per Concept for practice (past runs kept 8+ per Concept). Mirror the real
exam's section order and proportions, and record the seed and the builder script in the handoff.

## 11. Check as a Learner and hand off

1. Open https://tutor.ponlawat.me, sign in (Google), pick the new Course from the home page, take the Placement Test, and open
   a Lesson and a Quiz.
2. Write `content/<slug>/handoff.md` (copy an existing one): status table, prod Course id, backup file name, how the content was made,
   known gaps, and a next-run checklist.
3. Commit the graph, `graph.md`, the brief and the handoff on a branch (never the authored JSON), and open a PR.
4. Delete `~/tutor-platform/<slug>-load/data/` on the homelab once the load is verified, because it contains the answer keys.

## Troubleshooting

| Symptom | Cause / fix |
|---|---|
| `HTTP 403 … error 1010` | You called the public URL from a script. Run on the homelab against `127.0.0.1:8090` (steps 7–8). |
| `HTTP 401` | `MCP_TOKEN` not exported, or wrong. |
| `unexpected additional properties ["…"]` on import | An unquoted comma in a `name` inside `{ … }` in graph.yaml. Quote it. |
| `unknown prerequisite` / `cycle` | Fix graph.yaml; `make graph-check COURSE=<slug>` reports both. |
| `explanation says X but key is Y` | The Explanation's `ตอบ X` / `Answer: X` does not match `is_correct`. |
| `labels [...]` error | Wrong number of choices: set `LABELS=ABCD` for 4-choice Courses (default is `ABCDE`). |
| Quizzes or the Placement Test are empty | The Questions are still Draft. Run verification (step 9) and check `get_pool_stats`. |
