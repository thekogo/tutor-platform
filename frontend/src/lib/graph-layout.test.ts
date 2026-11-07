import { describe, expect, it } from "vitest";
import type { CourseGraph, Mastery } from "@/lib/api/types";
import { courseGraph } from "@/mocks/db";
import { NODE_WIDTH, buildMasteryGraph, countByLevel, topicColor, type ConceptNode } from "./graph-layout";

const small: Pick<CourseGraph, "topics" | "concepts"> = {
  topics: [
    { id: "t1", slug: "t1", name: "ระบบจำนวน" },
    { id: "t2", slug: "t2", name: "ร้อยละ" },
  ],
  concepts: [
    { id: "a", slug: "a", name: "เศษส่วน", topicId: "t1", prerequisiteIds: [] },
    { id: "b", slug: "b", name: "ทศนิยม", topicId: "t1", prerequisiteIds: ["a"] },
    { id: "c", slug: "c", name: "ร้อยละพื้นฐาน", topicId: "t2", prerequisiteIds: ["a", "b"] },
  ],
};
const mastery: Mastery[] = [
  { conceptId: "a", score: 0.9, evidence: 5, level: "mastered" },
  { conceptId: "c", score: 0.2, evidence: 3, level: "weak" },
];

const concepts = (nodes: { type?: string }[]) => nodes.filter((n): n is ConceptNode => n.type === "concept");

describe("buildMasteryGraph", () => {
  it("creates one concept node per Concept with its Mastery Level (Unassessed when missing)", () => {
    const { nodes } = buildMasteryGraph(small, mastery);
    const byId = Object.fromEntries(concepts(nodes).map((n) => [n.id, n.data]));
    expect(Object.keys(byId).sort()).toEqual(["a", "b", "c"]);
    expect(byId.a.level).toBe("mastered");
    expect(byId.b.level).toBe("unassessed");
    expect(byId.c.level).toBe("weak");
    expect(byId.c.topicName).toBe("ร้อยละ");
  });

  it("draws an edge from each Prerequisite to its dependent", () => {
    const { edges } = buildMasteryGraph(small, mastery);
    expect(edges.map((e) => `${e.source}->${e.target}`).sort()).toEqual(["a->b", "a->c", "b->c"]);
  });

  it("lays Prerequisites out to the left of dependents", () => {
    const { nodes } = buildMasteryGraph(small, mastery, { groupByTopic: false });
    const x = Object.fromEntries(concepts(nodes).map((n) => [n.id, n.position.x]));
    expect(x.a).toBeLessThan(x.b);
    expect(x.b).toBeLessThan(x.c);
  });

  it("keeps intra-Topic Prerequisites left of dependents when grouped", () => {
    const { nodes } = buildMasteryGraph(small, mastery);
    const x = Object.fromEntries(concepts(nodes).map((n) => [n.id, n.position.x]));
    expect(x.a).toBeLessThan(x.b);
  });

  it("does not overlap Topic group boxes", () => {
    const { nodes } = buildMasteryGraph(courseGraph, []);
    const groups = nodes.filter((n) => n.type === "topicGroup");
    expect(groups).toHaveLength(courseGraph.topics.length);
    for (let i = 0; i < groups.length; i++) {
      for (let j = i + 1; j < groups.length; j++) {
        const a = groups[i];
        const b = groups[j];
        const overlap =
          a.position.x < b.position.x + (b.width ?? 0) &&
          b.position.x < a.position.x + (a.width ?? 0) &&
          a.position.y < b.position.y + (b.height ?? 0) &&
          b.position.y < a.position.y + (a.height ?? 0);
        expect(overlap, `${a.id} overlaps ${b.id}`).toBe(false);
      }
    }
  });

  it("adds a Topic group node enclosing each Topic's Concepts", () => {
    const { nodes } = buildMasteryGraph(small, mastery);
    const group = nodes.find((n) => n.id === "topic:t1")!;
    expect(group.type).toBe("topicGroup");
    for (const n of concepts(nodes).filter((c) => c.data.topicId === "t1")) {
      expect(n.position.x).toBeGreaterThanOrEqual(group.position.x);
      expect(n.position.x + NODE_WIDTH).toBeLessThanOrEqual(group.position.x + (group.width ?? 0));
    }
  });

  it("filters to one Topic plus its outside Prerequisites as context", () => {
    const { nodes, edges } = buildMasteryGraph(small, mastery, { topicId: "t2" });
    const cs = concepts(nodes);
    expect(cs.map((n) => n.id).sort()).toEqual(["a", "b", "c"]);
    expect(cs.find((n) => n.id === "a")!.data.context).toBe(true);
    expect(cs.find((n) => n.id === "c")!.data.context).toBe(false);
    expect(nodes.some((n) => n.type === "topicGroup")).toBe(false);
    expect(edges).toHaveLength(3);
  });

  it("lays out the full CU-BEST seed graph without overlaps", () => {
    const { nodes } = buildMasteryGraph(courseGraph, []);
    const cs = concepts(nodes);
    expect(cs).toHaveLength(courseGraph.concepts.length);
    const seen = new Set(cs.map((n) => `${Math.round(n.position.x)},${Math.round(n.position.y)}`));
    expect(seen.size).toBe(cs.length);
  });
});

describe("helpers", () => {
  it("counts Concepts per Mastery Level", () => {
    expect(countByLevel(["a", "b", "c"], mastery)).toEqual({ unassessed: 1, weak: 1, developing: 0, mastered: 1 });
  });
  it("cycles Topic colours", () => {
    expect(topicColor(0)).toBe(topicColor(15));
  });
});
