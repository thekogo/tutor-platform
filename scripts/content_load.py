"""Validate authored Course JSON files and (with --write) push them via MCP.

Usage: COURSE=cu-best-business python3 scripts/content_load.py <topic.json>... [--write]
       LABELS=ABCD for Courses whose Questions have 4 choices (default ABCDE).

Checks each authored file against content/<course>/authoring-brief.md rules; with --write,
upserts the Lessons and creates the Questions as Draft via MCP (scripts/mcp_call.py).
"""
import json
import os
import re
import subprocess
import sys
from collections import Counter

import yaml

REPO = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
COURSE = os.environ.get("COURSE", "cu-best-business")
LABELS = os.environ.get("LABELS", "ABCDE")
graph = yaml.safe_load(open(f"{REPO}/content/{COURSE}/graph.yaml", encoding="utf-8"))
topic_of = {c["id"]: c["topic"] for c in graph["concepts"]}
formats = {f["id"] for f in graph["question_formats"]}


def mcp(tool, args):
    r = subprocess.run(["python3", f"{REPO}/scripts/mcp_call.py", tool, "-"], input=json.dumps(args, ensure_ascii=False),
                       capture_output=True, text=True, cwd=REPO)
    if r.returncode:
        sys.exit(f"{tool} failed: {r.stdout}{r.stderr}")
    return r.stdout


def validate(path):
    d = json.load(open(path, encoding="utf-8"))
    errs, warns = [], []
    qs, lessons = d["questions"], d["lessons"]
    topic = None
    by_concept = {}
    for i, q in enumerate(qs):
        tag = f"q{i} ({q.get('primary_concept')})"
        p = q.get("primary_concept")
        if p not in topic_of:
            errs.append(f"{tag}: unknown primary"); continue
        topic = topic or topic_of[p]
        if topic_of[p] != topic:
            errs.append(f"{tag}: primary outside topic {topic}")
        by_concept.setdefault(p, []).append(q)
        if p not in q.get("concepts", []):
            errs.append(f"{tag}: primary not in concepts")
        for c in q.get("concepts", []):
            if c not in topic_of:
                errs.append(f"{tag}: unknown concept {c}")
        if q.get("format") not in formats:
            errs.append(f"{tag}: bad format {q.get('format')}")
        if q.get("difficulty") not in (1, 2, 3):
            errs.append(f"{tag}: bad difficulty")
        if q.get("source") != "ai_generated":
            errs.append(f"{tag}: source")
        ch = q.get("choices", [])
        if [c["label"] for c in ch] != list(LABELS):
            errs.append(f"{tag}: labels {[c['label'] for c in ch]}")
        correct = [c["label"] for c in ch if c.get("is_correct")]
        if len(correct) != 1:
            errs.append(f"{tag}: {len(correct)} correct"); continue
        if len({c["body_md"].strip() for c in ch}) != len(ch):
            errs.append(f"{tag}: duplicate choice text")
        m = re.findall(rf"ตอบ\s*\(?([{LABELS}])\)?", q.get("explanation_md", ""))
        if not m:
            errs.append(f"{tag}: no 'ตอบ X' in explanation")
        elif m[-1] != correct[0]:
            errs.append(f"{tag}: explanation says {m[-1]} but key is {correct[0]}")
        if not q.get("stem_md", "").strip():
            errs.append(f"{tag}: empty stem")
    concepts_in_topic = [c for c, t in topic_of.items() if t == topic]
    for c in concepts_in_topic:
        cq = by_concept.get(c, [])
        if len(cq) != 12:
            warns.append(f"{c}: {len(cq)} questions (want 12)")
        diff = Counter(q["difficulty"] for q in cq)
        keys = Counter(next(ch["label"] for ch in q["choices"] if ch.get("is_correct")) for q in cq if any(ch.get("is_correct") for ch in q["choices"]))
        if any(keys[l] < 2 for l in LABELS):
            warns.append(f"{c}: key balance {dict(sorted(keys.items()))}")
        if not (diff[1] and diff[2] and diff[3]):
            warns.append(f"{c}: difficulty {dict(diff)}")
    lc = [l["concept"] for l in lessons]
    if sorted(lc) != sorted(concepts_in_topic):
        errs.append(f"lessons {lc} != concepts {concepts_in_topic}")
    for l in lessons:
        for h in ("## แนวคิดหลัก", "## ตัวอย่าง", "## ข้อผิดพลาดที่พบบ่อย"):
            if h not in l["body_md"]:
                warns.append(f"lesson {l['concept']}: missing {h}")
    return d, topic, errs, warns


def main():
    files = [a for a in sys.argv[1:] if not a.startswith("--")]
    write = "--write" in sys.argv
    bad = False
    for f in files:
        d, topic, errs, warns = validate(f)
        print(f"== {f}: topic={topic} lessons={len(d['lessons'])} questions={len(d['questions'])}")
        for e in errs: print("  ERROR", e)
        for w in warns: print("  warn ", w)
        if errs:
            bad = True; continue
        if write:
            for l in d["lessons"]:
                mcp("upsert_lesson", {"course": COURSE, "concept": l["concept"], "body_md": l["body_md"]})
            out = mcp("create_questions", {"course": COURSE, "questions": d["questions"]})
            n = len(json.loads(out).get("question_ids", [])) if out.strip().startswith("{") else out[:200]
            print(f"  wrote {len(d['lessons'])} lessons, created drafts: {n}")
    sys.exit(1 if bad else 0)


main()
