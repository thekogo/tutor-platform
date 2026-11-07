import { graphlib, layout } from "@dagrejs/dagre";
import { MarkerType, type Edge, type Node } from "@xyflow/react";
import type { CourseGraph, Mastery, MasteryLevel } from "@/lib/api/types";

/**
 * Pure transform: Course graph + Mastery → positioned React Flow nodes/edges.
 * Prerequisites flow left → right; Concepts are clustered by Topic.
 */

export const NODE_WIDTH = 210;
export const NODE_HEIGHT = 64;
const GROUP_PADDING = 18;
const GROUP_LABEL_HEIGHT = 26;

/** Distinct, colour-blind-friendlier hues for Topics (border/background tint only). */
const TOPIC_PALETTE = [
  "#2563eb", "#9333ea", "#0d9488", "#ea580c", "#db2777",
  "#4f46e5", "#65a30d", "#0891b2", "#c026d3", "#b45309",
  "#475569", "#16a34a", "#dc2626", "#7c3aed", "#0369a1",
];

export function topicColor(index: number): string {
  return TOPIC_PALETTE[((index % TOPIC_PALETTE.length) + TOPIC_PALETTE.length) % TOPIC_PALETTE.length];
}

export type ConceptNodeData = {
  conceptId: string;
  name: string;
  topicId: string;
  topicName: string;
  topicColor: string;
  level: MasteryLevel;
  score: number | null;
  /** True for Prerequisites from other Topics shown for context in a filtered view. */
  context: boolean;
};

export type TopicGroupData = {
  topicId: string;
  name: string;
  color: string;
};

export type ConceptNode = Node<ConceptNodeData, "concept">;
export type TopicGroupNode = Node<TopicGroupData, "topicGroup">;
export type MapNode = ConceptNode | TopicGroupNode;

export type MasteryGraph = {
  nodes: MapNode[];
  edges: Edge[];
};

export type BuildOptions = {
  /** Show only this Topic's Concepts plus their direct Prerequisites (as context). */
  topicId?: string | null;
  /** Cluster Concepts by Topic with background group nodes (default true when unfiltered). */
  groupByTopic?: boolean;
};

export function masteryByConcept(mastery: readonly Mastery[]): Map<string, Mastery> {
  return new Map(mastery.map((m) => [m.conceptId, m]));
}

export function buildMasteryGraph(
  course: Pick<CourseGraph, "topics" | "concepts">,
  mastery: readonly Mastery[],
  options: BuildOptions = {},
): MasteryGraph {
  const topicIndex = new Map(course.topics.map((t, i) => [t.id, i]));
  const topicById = new Map(course.topics.map((t) => [t.id, t]));
  const masteryMap = masteryByConcept(mastery);
  const filterTopic = options.topicId ?? null;
  const group = options.groupByTopic ?? !filterTopic;

  // Which Concepts to show.
  const conceptIds = new Set(course.concepts.map((c) => c.id));
  let visible = course.concepts;
  const contextIds = new Set<string>();
  if (filterTopic) {
    const inTopic = course.concepts.filter((c) => c.topicId === filterTopic);
    const ids = new Set(inTopic.map((c) => c.id));
    for (const c of inTopic) {
      for (const p of c.prerequisiteIds) if (!ids.has(p) && conceptIds.has(p)) contextIds.add(p);
    }
    visible = course.concepts.filter((c) => ids.has(c.id) || contextIds.has(c.id));
  }
  const visibleIds = new Set(visible.map((c) => c.id));

  const edges: Edge[] = [];
  for (const c of visible) {
    for (const p of c.prerequisiteIds) {
      if (!visibleIds.has(p)) continue;
      const cross = conceptTopic(course, p) !== c.topicId;
      edges.push({
        id: `${p}->${c.id}`,
        source: p,
        target: c.id,
        type: "smoothstep",
        markerEnd: { type: MarkerType.ArrowClosed, width: 16, height: 16 },
        ...(cross && group ? { style: { stroke: "#cbd5e1", strokeDasharray: "5 4" } } : {}),
      });
    }
  }

  const positions = group
    ? layoutGrouped(course.topics.map((t) => t.id), visible, edges)
    : layoutFlat(visible.map((c) => c.id), edges);

  const conceptNodes: ConceptNode[] = visible.map((c) => {
    const pos = positions.get(c.id) ?? { x: 0, y: 0 };
    const topic = topicById.get(c.topicId);
    const m = masteryMap.get(c.id);
    return {
      id: c.id,
      type: "concept",
      position: pos,
      width: NODE_WIDTH,
      height: NODE_HEIGHT,
      zIndex: 1,
      data: {
        conceptId: c.id,
        name: c.name,
        topicId: c.topicId,
        topicName: topic?.name ?? "",
        topicColor: topicColor(topicIndex.get(c.topicId) ?? 0),
        level: m?.level ?? "unassessed",
        score: m ? m.score : null,
        context: contextIds.has(c.id),
      },
    };
  });

  const groupNodes: TopicGroupNode[] = [];
  if (group) {
    for (const t of course.topics) {
      const members = conceptNodes.filter((n) => n.data.topicId === t.id);
      if (members.length === 0) continue;
      const box = boundingBox(members);
      groupNodes.push({
        id: groupId(t.id),
        type: "topicGroup",
        position: { x: box.minX - GROUP_PADDING, y: box.minY - GROUP_PADDING - GROUP_LABEL_HEIGHT },
        width: box.maxX - box.minX + GROUP_PADDING * 2,
        height: box.maxY - box.minY + GROUP_PADDING * 2 + GROUP_LABEL_HEIGHT,
        zIndex: 0,
        selectable: false,
        draggable: false,
        focusable: false,
        data: { topicId: t.id, name: t.name, color: topicColor(topicIndex.get(t.id) ?? 0) },
      });
    }
  }

  return { nodes: [...groupNodes, ...conceptNodes], edges };
}

type Point = { x: number; y: number };

function conceptTopic(course: Pick<CourseGraph, "concepts">, conceptId: string): string | undefined {
  return course.concepts.find((c) => c.id === conceptId)?.topicId;
}

/** Dagre left→right layout; returns top-left positions. */
function layoutFlat(ids: readonly string[], edges: readonly Edge[]): Map<string, Point> {
  const g = new graphlib.Graph();
  g.setGraph({ rankdir: "LR", nodesep: 22, ranksep: 70, marginx: 0, marginy: 0 });
  g.setDefaultEdgeLabel(() => ({}));
  const idSet = new Set(ids);
  for (const id of ids) g.setNode(id, { width: NODE_WIDTH, height: NODE_HEIGHT });
  for (const e of edges) if (idSet.has(e.source) && idSet.has(e.target)) g.setEdge(e.source, e.target);
  layout(g);
  const out = new Map<string, Point>();
  for (const id of ids) {
    const n = g.node(id) as Point;
    out.set(id, { x: n.x - NODE_WIDTH / 2, y: n.y - NODE_HEIGHT / 2 });
  }
  return out;
}

/** Target width:height of the packed map (laptop-ish screens). */
const TARGET_ASPECT = 2.6;
const GROUP_GAP = 56;

/**
 * Each Topic is laid out on its own (intra-Topic Prerequisites only), then the
 * Topic boxes are packed into rows in Course order, with a row width chosen
 * so the whole map is roughly laptop-shaped. Cross-Topic Prerequisites are
 * still drawn as edges.
 */
function layoutGrouped(
  topicOrder: readonly string[],
  visible: readonly { id: string; topicId: string }[],
  edges: readonly Edge[],
): Map<string, Point> {
  const orphanTopics = [...new Set(visible.map((c) => c.topicId))].filter((t) => !topicOrder.includes(t));
  const boxes: { local: Map<string, Point>; w: number; h: number }[] = [];
  for (const topicId of [...topicOrder, ...orphanTopics]) {
    const ids = visible.filter((c) => c.topicId === topicId).map((c) => c.id);
    if (ids.length === 0) continue;
    const local = layoutFlat(ids, edges);
    let w = 0;
    let h = 0;
    for (const p of local.values()) {
      w = Math.max(w, p.x + NODE_WIDTH);
      h = Math.max(h, p.y + NODE_HEIGHT);
    }
    boxes.push({ local, w: w + GROUP_PADDING * 2, h: h + GROUP_PADDING * 2 + GROUP_LABEL_HEIGHT });
  }
  const area = boxes.reduce((sum, b) => sum + (b.w + GROUP_GAP) * (b.h + GROUP_GAP), 0);
  const rowWidth = Math.max(Math.sqrt(area * TARGET_ASPECT), ...boxes.map((b) => b.w));

  const out = new Map<string, Point>();
  let x = 0;
  let y = 0;
  let rowHeight = 0;
  for (const box of boxes) {
    if (x > 0 && x + box.w > rowWidth) {
      x = 0;
      y += rowHeight + GROUP_GAP;
      rowHeight = 0;
    }
    for (const [id, p] of box.local) {
      out.set(id, { x: x + GROUP_PADDING + p.x, y: y + GROUP_PADDING + GROUP_LABEL_HEIGHT + p.y });
    }
    x += box.w + GROUP_GAP;
    rowHeight = Math.max(rowHeight, box.h);
  }
  return out;
}

function groupId(topicId: string): string {
  return `topic:${topicId}`;
}

function boundingBox(nodes: readonly ConceptNode[]) {
  let minX = Infinity;
  let minY = Infinity;
  let maxX = -Infinity;
  let maxY = -Infinity;
  for (const n of nodes) {
    minX = Math.min(minX, n.position.x);
    minY = Math.min(minY, n.position.y);
    maxX = Math.max(maxX, n.position.x + NODE_WIDTH);
    maxY = Math.max(maxY, n.position.y + NODE_HEIGHT);
  }
  return { minX, minY, maxX, maxY };
}

/** Count Concepts per Mastery Level (for the legend / summaries). */
export function countByLevel(
  conceptIds: readonly string[],
  mastery: readonly Mastery[],
): Record<MasteryLevel, number> {
  const m = masteryByConcept(mastery);
  const counts: Record<MasteryLevel, number> = { unassessed: 0, weak: 0, developing: 0, mastered: 0 };
  for (const id of conceptIds) counts[m.get(id)?.level ?? "unassessed"]++;
  return counts;
}
