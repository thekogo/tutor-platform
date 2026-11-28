# Handoff: CKA content (run 1, 2026-11-27)

What the first run produced for Course `cka`, how it was made, and what the next run should do.
Vocabulary: [CONTEXT.md](../../CONTEXT.md). Workflow rules: [ADR 0003](../../docs/adr/0003-ai-agent-authors-content-via-mcp.md).

## Status

| Item | Target | Run 1 result |
|---|---|---|
| Knowledge graph | from the official curriculum (v1.37) | 7 Topics, 39 Concepts, 61 Prerequisites ([graph.yaml](graph.yaml)); `make graph-check COURSE=cka` passes |
| Lessons | 1 per Concept | 39 / 39, English, taken from [study-guide/](study-guide/README.md) |
| Questions | 12 per Concept, 4 choices A–D | **119 Draft** (2–4 per Concept) |
| Verification | independent pass; match → Active | **not done**, so all 119 are still Draft |
| Mock Exams | n/a | not created |

Production only (no local dev run). Prod Course id `d3460f78-1245-4ec5-844a-7912831ff849`.
Database backup taken before the load: `~/tutor-platform/backups/tutor-before-cka-20261127-001457.sql.gz` on the homelab.

## How run 1 was made

1. Graph and [outline](outline.md) researched from the LF CKA page and the CNCF curriculum repo (sources in the graph.yaml header).
2. [study-guide/](study-guide/README.md) written by hand: one section per Concept (key ideas, commands, lab, pitfalls, check questions).
3. Converted with `convert.py` (each Concept section → Lesson, minus its check questions; each check question → Question) and
   `shuffle.py` (deterministic choice shuffle, seed 20261127, to balance the answer keys, which were 80% B; letter references in
   explanations remapped). Both scripts live in the session scratchpad, not in git.
   - difficulty = position in the section (Q1 → 1, Q2 → 2, Q3+ → 3), so it is a heuristic only
   - format guessed from the stem and choices (concept-check 63, scenario-diagnosis 21, command-choice 16, output-reading 13,
     manifest-fix 6)
   - explanations start with `Answer: X.`; `scripts/content_load.py` now accepts that as well as `ตอบ X`
4. Loaded **on the homelab** against `http://127.0.0.1:8090/mcp`. The public URL rejects `scripts/mcp_call.py`: Cloudflare error 1010
   blocks the Python-urllib user-agent.

## Platform constraints

The CKA is fully hands-on, but the platform marks multiple choice only. Questions train recognition and diagnosis; every Lesson keeps
its hands-on **Lab** section, and the per-chapter exam-style tasks and the mock exam stay in the study guide (not loaded).

## Next run: checklist

1. **Verify** the 119 Drafts with an independent verifier (`list_questions_for_verification` → `submit_verification`, no access to keys).
2. **Fill the pool to 12 per Concept** (~350 more Questions), with a real 1/2/3 difficulty mix and more command-choice / manifest-fix /
   output-reading items.
3. Optionally build Mock Exams weighted by `exam_weight`.
4. Re-check v1.37-specific facts flagged in the study guide (Calico/Envoy Gateway versions, `kubectl autoscale --cpu` flag, etcdutl on nodes).
