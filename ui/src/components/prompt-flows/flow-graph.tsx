"use client";

import React, { useMemo, useEffect } from "react";
import {
    ReactFlow,
    Background,
    Controls,
    MiniMap,
    Node,
    Edge,
    Handle,
    Position,
    MarkerType,
    useNodesState,
    useEdgesState,
    type NodeTypes,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import type { PromptFlowStage } from "@/lib/types";

// ─── Custom Node Component ────────────────────────────────────────────────────

type StageNodeData = {
    stageIndex: number;
    stage: PromptFlowStage;
    isEntry: boolean;
    onClick: (index: number) => void;
};

function StageNode({ data }: { data: StageNodeData }) {
    const { stage, isEntry, stageIndex, onClick } = data;
    const enabled = stage.enabled !== false;

    return (
        <div className="relative">
            {/* Target handle - for incoming edges */}
            <Handle
                type="target"
                position={Position.Left}
                style={{ background: '#3b82f6', width: 12, height: 12, border: '2px solid white' }}
            />

            <button
                onClick={() => onClick(stageIndex)}
                className={cn(
                    "group relative min-w-48 rounded-xl border-2 bg-card p-3 shadow-sm transition-all hover:shadow-md hover:border-primary",
                    isEntry && "border-primary ring-2 ring-primary/20",
                    !enabled && "opacity-60",
                )}
            >
                {isEntry && (
                    <div className="absolute -top-2 left-1/2 -translate-x-1/2 whitespace-nowrap">
                        <Badge variant="default" className="text-xs">
                            Entry
                        </Badge>
                    </div>
                )}
                <div className="space-y-2">
                    <div className="flex items-center gap-2">
                        <Badge variant="outline" className="uppercase text-xs shrink-0">
                            {stage.type}
                        </Badge>
                        <Badge variant={enabled ? "default" : "secondary"} className="text-xs shrink-0">
                            {enabled ? "On" : "Off"}
                        </Badge>
                    </div>
                    <div className="space-y-1">
                        <p className="text-sm font-medium text-left line-clamp-2">{stage.name}</p>
                        <p className="font-mono text-xs text-muted-foreground text-left truncate">
                            {stage.id}
                        </p>
                    </div>
                    {stage.transitions && stage.transitions.length > 0 && (
                        <div className="pt-1 border-t">
                            <p className="text-xs text-muted-foreground">
                                {stage.transitions.length} transition{stage.transitions.length !== 1 ? "s" : ""}
                            </p>
                        </div>
                    )}
                </div>
            </button>

            {/* Source handle - for outgoing edges */}
            <Handle
                type="source"
                position={Position.Right}
                style={{ background: '#3b82f6', width: 12, height: 12, border: '2px solid white' }}
            />
        </div>
    );
}

const nodeTypes: NodeTypes = {
    stage: StageNode,
};

// ─── Layout Algorithm ──────────────────────────────────────────────────────────

type LayoutNode = {
    id: string;
    level: number;
    children: string[];
};

function computeLayout(stages: PromptFlowStage[], entryStageId: string): Map<string, { x: number; y: number }> {
    const positions = new Map<string, { x: number; y: number }>();

    if (stages.length === 0) return positions;

    // Build adjacency map
    const graph = new Map<string, string[]>();
    stages.forEach((stage) => {
        const targets = (stage.transitions ?? []).map((t) => t.target_stage_id);
        graph.set(stage.id, targets);
    });

    // BFS to assign levels
    const levels = new Map<string, number>();
    const visited = new Set<string>();
    const queue: Array<{ id: string; level: number }> = [];

    const entry = entryStageId || stages[0]?.id;
    if (entry) {
        queue.push({ id: entry, level: 0 });
        visited.add(entry);
    }

    // Assign stages with transitions to levels via BFS
    while (queue.length > 0) {
        const { id, level } = queue.shift()!;
        levels.set(id, level);

        const targets = graph.get(id) ?? [];
        targets.forEach((targetId) => {
            if (!visited.has(targetId)) {
                visited.add(targetId);
                queue.push({ id: targetId, level: level + 1 });
            }
        });
    }

    // Assign remaining unconnected stages
    stages.forEach((stage) => {
        if (!levels.has(stage.id)) {
            levels.set(stage.id, 0);
        }
    });

    // Group by level
    const levelGroups = new Map<number, string[]>();
    levels.forEach((level, id) => {
        if (!levelGroups.has(level)) {
            levelGroups.set(level, []);
        }
        levelGroups.get(level)!.push(id);
    });

    // Position nodes
    const nodeWidth = 250;
    const nodeHeight = 150;
    const horizontalSpacing = 100;
    const verticalSpacing = 80;

    levelGroups.forEach((ids, level) => {
        const x = level * (nodeWidth + horizontalSpacing);
        ids.forEach((id, index) => {
            const y = index * (nodeHeight + verticalSpacing);
            positions.set(id, { x, y });
        });
    });

    return positions;
}

// ─── Main Component ─────────────────────────────────────────────────────────────

type FlowGraphProps = {
    stages: PromptFlowStage[];
    entryStageId?: string;
    onStageClick: (index: number) => void;
    className?: string;
};

export function FlowGraph({ stages, entryStageId, onStageClick, className }: FlowGraphProps) {
    const positions = useMemo(() => computeLayout(stages, entryStageId ?? ""), [stages, entryStageId]);

    // Memoize nodeTypes to prevent React Flow warning
    const memoizedNodeTypes = useMemo(() => nodeTypes, []);

    // Use React Flow's state management for draggable nodes
    const [nodes, setNodes, onNodesChange] = useNodesState<Node<StageNodeData>>([]);
    const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);

    // Update nodes when stages change
    useEffect(() => {
        const newNodes: Node<StageNodeData>[] = stages.map((stage, index) => {
            const pos = positions.get(stage.id) ?? { x: 0, y: index * 200 };
            return {
                id: stage.id,
                type: "stage",
                position: pos,
                data: {
                    stageIndex: index,
                    stage,
                    isEntry: stage.id === entryStageId || (!entryStageId && index === 0),
                    onClick: onStageClick,
                },
            };
        });
        setNodes(newNodes);
    }, [stages, entryStageId, positions, onStageClick, setNodes]);

    // Update edges when stages change
    useEffect(() => {
        const edgeList: Edge[] = [];
        const stageIds = new Set(stages.map(s => s.id));

        stages.forEach((stage) => {
            (stage.transitions ?? []).forEach((transition, ti) => {
                // Only create edge if target stage exists
                if (stageIds.has(transition.target_stage_id)) {
                    edgeList.push({
                        id: `${stage.id}-${transition.target_stage_id}-${ti}`,
                        source: stage.id,
                        target: transition.target_stage_id,
                        label: transition.label || undefined,
                        animated: true,
                        type: 'smoothstep',
                        markerEnd: {
                            type: MarkerType.ArrowClosed,
                            width: 20,
                            height: 20,
                            color: '#3b82f6',
                        },
                        style: {
                            stroke: '#3b82f6',
                            strokeWidth: 2,
                        },
                        labelStyle: {
                            fontSize: 11,
                            fill: '#334155',
                            fontWeight: 500,
                        },
                        labelBgStyle: {
                            fill: '#ffffff',
                            fillOpacity: 0.9,
                        },
                        labelBgPadding: [6, 3] as [number, number],
                        labelBgBorderRadius: 4,
                    });
                }
            });
            if (stage.on_success && stageIds.has(stage.on_success)) {
                // Create edge for on_success if target stage exists
                edgeList.push({
                    id: `${stage.id}-${stage.on_success}-success`,
                    source: stage.id,
                    target: stage.on_success,
                    label: "on_success",
                    animated: true,
                    type: 'smoothstep',
                    markerEnd: {
                        type: MarkerType.ArrowClosed,
                        width: 20,
                        height: 20,
                        color: '#10b981',
                    },
                    style: {
                        stroke: '#10b981',
                        strokeWidth: 2,
                        strokeDasharray: '5,5',
                    },
                    labelStyle: {
                        fontSize: 11,
                        fill: '#065f46',
                        fontWeight: 500,
                    },
                    labelBgStyle: {
                        fill: '#d1fae5',
                        fillOpacity: 0.9,
                    },
                    labelBgPadding: [6, 3] as [number, number],
                    labelBgBorderRadius: 4,
                });
            }
        });
        setEdges(edgeList);
    }, [stages, setEdges]);

    if (stages.length === 0) {
        return (
            <div className={cn("flex items-center justify-center rounded-lg border-2 border-dashed p-12", className)}>
                <p className="text-sm text-muted-foreground">
                    No stages yet. Add stages to visualize the flow.
                </p>
            </div>
        );
    }

    return (
        <div className={cn("rounded-lg border bg-muted/20", className)} style={{ height: "600px" }}>
            <ReactFlow
                nodes={nodes}
                edges={edges}
                onNodesChange={onNodesChange}
                onEdgesChange={onEdgesChange}
                nodeTypes={memoizedNodeTypes}
                nodesDraggable={true}
                nodesConnectable={false}
                elementsSelectable={true}
                fitView
                fitViewOptions={{ padding: 0.2, maxZoom: 1 }}
                minZoom={0.1}
                maxZoom={2}
                defaultEdgeOptions={{
                    animated: true,
                }}
            >
                <Background />
                <Controls />
                <MiniMap
                    nodeColor={(node) => {
                        const data = node.data as StageNodeData;
                        return data.isEntry ? "hsl(var(--primary))" : "hsl(var(--muted))";
                    }}
                    maskColor="rgba(0, 0, 0, 0.1)"
                />
            </ReactFlow>
        </div>
    );
}
