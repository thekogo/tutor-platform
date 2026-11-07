"""Validate a course knowledge graph YAML and render it as Mermaid markdown.

Usage: python3 scripts/graph_check.py content/<course>/graph.yaml [out.md]
"""
import sys
from collections import Counter, defaultdict

import yaml


def main(path, out=None):
    g = yaml.safe_load(open(path, encoding="utf-8"))
    topics = {t["id"]: t for t in g["topics"]}
    concepts = {c["id"]: c for c in g["concepts"]}
    errors = []

    dupes = [k for k, n in Counter(c["id"] for c in g["concepts"]).items() if n > 1]
    errors += [f"duplicate concept id: {d}" for d in dupes]
    for c in concepts.values():
        if c["topic"] not in topics:
            errors.append(f"{c['id']}: unknown topic {c['topic']}")
        for p in c["prerequisites"]:
            if p not in concepts:
                errors.append(f"{c['id']}: unknown prerequisite {p}")
            if p == c["id"]:
                errors.append(f"{c['id']}: self prerequisite")

    # cycle check (DFS, white/grey/black)
    state = {}

    def visit(n, stack):
        state[n] = 1
        for p in concepts[n]["prerequisites"]:
            if p not in concepts:
                continue
            if state.get(p) == 1:
                errors.append("cycle: " + " -> ".join(stack[stack.index(p):] + [p]))
            elif p not in state:
                visit(p, stack + [p])
        state[n] = 2

    for n in concepts:
        if n not in state:
            visit(n, [n])

    for t in topics:
        if not any(c["topic"] == t for c in concepts.values()):
            errors.append(f"topic {t} has no concepts")

    # depth = longest prerequisite chain below a concept
    depth = {}

    def d(n):
        if n not in depth:
            depth[n] = 1 + max((d(p) for p in concepts[n]["prerequisites"] if p in concepts), default=0)
        return depth[n]

    if errors:
        print("\n".join(errors))
        sys.exit(1)

    for n in concepts:
        d(n)
    dependents = defaultdict(int)
    for c in concepts.values():
        for p in c["prerequisites"]:
            dependents[p] += 1
    cross = sum(1 for c in concepts.values() for p in c["prerequisites"]
                if concepts[p]["topic"] != c["topic"])
    edges = sum(len(c["prerequisites"]) for c in concepts.values())
    print(f"OK: {len(topics)} topics, {len(concepts)} concepts, {edges} prerequisites ({cross} cross-topic)")
    print(f"max depth: {max(depth.values())} ({max(depth, key=depth.get)})")
    print("most depended-on:", ", ".join(f"{k}({v})" for k, v in
                                          sorted(dependents.items(), key=lambda x: -x[1])[:8]))

    if out:
        lines = ["```mermaid", "flowchart LR"]
        for t in topics.values():
            lines.append(f'  subgraph {t["id"]}["{t["name"]}"]')
            for c in concepts.values():
                if c["topic"] == t["id"]:
                    lines.append(f'    {c["id"]}["{c["name"]}"]')
            lines.append("  end")
        for c in concepts.values():
            for p in c["prerequisites"]:
                lines.append(f'  {p} --> {c["id"]}')
        lines.append("```")
        with open(out, "w", encoding="utf-8") as f:
            f.write(f"# {g['course']['name']} — knowledge graph\n\n"
                    f"Generated from `{path}` by `scripts/graph_check.py`. Do not edit by hand.\n\n")
            f.write("\n".join(lines) + "\n")


if __name__ == "__main__":
    main(*sys.argv[1:])
