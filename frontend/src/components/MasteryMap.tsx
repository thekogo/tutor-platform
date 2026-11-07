"use client";

import "@xyflow/react/dist/style.css";
import {
  Background,
  Controls,
  Handle,
  MiniMap,
  Position,
  ReactFlow,
  type NodeProps,
  type NodeTypes,
} from "@xyflow/react";
import { useRouter } from "next/navigation";
import { useId, useMemo, useState } from "react";
import type { CourseGraph, Mastery, MasteryLevel } from "@/lib/api/types";
import {
  buildMasteryGraph,
  type ConceptNode,
  type MapNode,
  type TopicGroupNode,
} from "@/lib/graph-layout";
import { T } from "@/lib/terms";
import { cn } from "@/lib/ui";
import { LEVEL_CLASS, MasteryChip, MasteryLegend } from "./MasteryChip";

const LEVEL_FILL: Record<MasteryLevel, string> = {
  unassessed: "#f1f5f9",
  weak: "#fecaca",
  developing: "#fde68a",
  mastered: "#bbf7d0",
};

function ConceptNodeView({ data }: NodeProps<ConceptNode>) {
  return (
    <div
      data-testid="concept-node"
      data-level={data.level}
      data-concept-id={data.conceptId}
      title={`${data.name} — ${data.topicName}`}
      className={cn(
        "flex h-full w-full cursor-pointer flex-col justify-between rounded-lg border-2 border-l-8 bg-white px-2 py-1 text-left shadow-sm",
        LEVEL_CLASS[data.level],
        data.context && "opacity-60",
      )}
      style={{ borderLeftColor: data.topicColor }}
    >
      <Handle type="target" position={Position.Left} className="!size-1.5 !border-0 !bg-slate-400" />
      <span className="line-clamp-2 text-[12px] leading-tight font-medium text-slate-900">{data.name}</span>
      <MasteryChip level={data.level} className="self-start !px-1.5 !py-0 !text-[10px]" />
      <Handle type="source" position={Position.Right} className="!size-1.5 !border-0 !bg-slate-400" />
    </div>
  );
}

function TopicGroupView({ data }: NodeProps<TopicGroupNode>) {
  return (
    <div
      className="h-full w-full rounded-2xl border-2 border-dashed"
      style={{ borderColor: `${data.color}88`, backgroundColor: `${data.color}0d` }}
    >
      <span className="m-2 inline-block rounded px-2 py-0.5 text-xs font-semibold text-white" style={{ backgroundColor: data.color }}>
        {data.name}
      </span>
    </div>
  );
}

const nodeTypes: NodeTypes = { concept: ConceptNodeView, topicGroup: TopicGroupView };

type Props = {
  course: CourseGraph;
  mastery: Mastery[];
  /** Compact embed (e.g. dashboard preview) without controls. */
  preview?: boolean;
  height?: number | string;
};

export function MasteryMap({ course, mastery, preview = false, height = "calc(100dvh - 15rem)" }: Props) {
  const router = useRouter();
  const selectId = useId();
  const [topicId, setTopicId] = useState<string>("");
  const graph = useMemo(
    () => buildMasteryGraph(course, mastery, { topicId: topicId || null }),
    [course, mastery, topicId],
  );

  return (
    <div className="flex flex-col gap-3">
      {!preview && (
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            <label htmlFor={selectId} className="text-sm font-medium text-slate-700">
              {T.topic}
            </label>
            <select
              id={selectId}
              value={topicId}
              onChange={(e) => setTopicId(e.target.value)}
              className="rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-sm"
            >
              <option value="">ทุกหัวข้อ</option>
              {course.topics.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name}
                </option>
              ))}
            </select>
          </div>
          <MasteryLegend />
        </div>
      )}
      <div
        data-testid="mastery-map"
        className="overflow-hidden rounded-xl border border-slate-200 bg-white"
        style={{ height: preview ? 320 : height, minHeight: 360 }}
        role="region"
        aria-label={`${T.masteryMap} (กราฟแบบโต้ตอบ — ดูแบบรายการด้านล่างสำหรับโปรแกรมอ่านหน้าจอ)`}
      >
        <ReactFlow<MapNode>
          nodes={graph.nodes}
          edges={graph.edges}
          nodeTypes={nodeTypes}
          fitView
          fitViewOptions={{ padding: 0.08 }}
          minZoom={0.1}
          maxZoom={2}
          nodesDraggable={false}
          nodesConnectable={false}
          elementsSelectable
          onNodeClick={(_, node) => {
            if (node.type === "concept") router.push(`/concepts/${node.id}`);
          }}
          proOptions={{ hideAttribution: true }}
          defaultEdgeOptions={{ style: { stroke: "#94a3b8", strokeWidth: 1.25 } }}
        >
          <Background gap={24} color="#e2e8f0" />
          {!preview && <Controls showInteractive={false} />}
          {!preview && (
            <MiniMap
              pannable
              zoomable
              nodeColor={(n) => (n.type === "concept" ? LEVEL_FILL[(n as ConceptNode).data.level] : "transparent")}
              nodeStrokeColor={(n) => (n.type === "concept" ? (n as ConceptNode).data.topicColor : "transparent")}
            />
          )}
        </ReactFlow>
      </div>
    </div>
  );
}
