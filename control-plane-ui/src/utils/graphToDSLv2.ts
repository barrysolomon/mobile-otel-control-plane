import type {
  WorkflowGraph,
  GraphNode,
  GraphEdge,
  DSLConfigV2,
  DSLBufferConfig,
  DSLWorkflowV2,
  DSLState,
  DSLMatcher,
  DSLMatcherType,
  DSLActionV2,
} from '../types/workflow';

/**
 * Compiles an array of visual workflow graphs into a DSL v2 config
 * that devices consume. Workflows with StateNode nodes compile to
 * multi-state FSMs; simple workflows become single-state FSMs.
 */
export function compileGraphToDSLv2(
  graphs: WorkflowGraph[],
  bufferConfig: DSLBufferConfig
): DSLConfigV2 {
  const workflows = graphs
    .filter((g) => g.nodes.length > 0)
    .map((graph, index) => compileWorkflowV2(graph, index));

  return {
    version: 2,
    buffer_config: bufferConfig,
    workflows,
  };
}

function compileWorkflowV2(graph: WorkflowGraph, index: number): DSLWorkflowV2 {
  const stateNodes = graph.nodes.filter((n) => n.type === 'state');

  // Multi-state FSM when StateNodes are present
  if (stateNodes.length > 0) {
    return compileMultiStateFSM(graph, stateNodes, index);
  }

  // Single-state FSM (backward compatible)
  const entryNode = graph.nodes.find((n) => n.id === graph.entryNodeId);
  if (!entryNode) {
    throw new Error(`Workflow "${graph.name}": entry node ${graph.entryNodeId} not found`);
  }

  const matchers = buildMatchers(entryNode, graph.nodes, graph.edges);
  const actions = buildActionsV2(entryNode, graph.nodes, graph.edges);

  const state: DSLState = {
    id: 'default',
    matchers,
    on_match: { actions },
  };

  return {
    id: graph.id,
    name: graph.name,
    enabled: graph.enabled,
    priority: index + 1,
    initial_state: 'default',
    states: [state],
  };
}

/**
 * Compiles a graph with StateNodes into a multi-state FSM.
 * Each StateNode becomes a DSLState. Edges from trigger/timeout nodes
 * into a state define its matchers. Edges from a state to action nodes
 * define on_match actions. Edges between states define transitions.
 */
function compileMultiStateFSM(
  graph: WorkflowGraph,
  stateNodes: GraphNode[],
  index: number,
): DSLWorkflowV2 {
  const initialState = stateNodes.find(
    (n) => n.type === 'state' && n.data.isInitial
  );
  if (!initialState) {
    throw new Error(`Workflow "${graph.name}": no initial state found`);
  }

  const states: DSLState[] = stateNodes.map((sn) => {
    if (sn.type !== 'state') throw new Error('Expected state node');

    // Find trigger/matcher nodes that connect INTO this state
    const incomingEdges = graph.edges.filter((e) => e.target === sn.id);
    const matcherNodes = incomingEdges
      .map((e) => graph.nodes.find((n) => n.id === e.source))
      .filter((n): n is GraphNode => n !== undefined && !isStateNode(n));

    const matchers: DSLMatcher[] = [];
    let onTimeout: DSLState['on_timeout'] | undefined;

    for (const mn of matcherNodes) {
      if (mn.type === 'timeout_matcher') {
        // Timeout matchers become on_timeout
        const timeoutActions = getActionsAfterState(sn, graph);
        const timeoutTransition = getTransitionFromState(sn, graph, stateNodes);
        onTimeout = {
          after_ms: mn.data.afterMs,
          actions: timeoutActions,
          ...(timeoutTransition && { transition_to: timeoutTransition }),
        };
      } else {
        const m = nodeToMatcher(mn, graph.nodes, graph.edges);
        if (m) matchers.push(m);
      }
    }

    // Find action nodes and transition targets from this state's outgoing edges
    const actions = getActionsAfterState(sn, graph);
    const transitionTo = getTransitionFromState(sn, graph, stateNodes);

    const state: DSLState = {
      id: sn.data.stateName,
      matchers,
      on_match: {
        actions,
        ...(transitionTo && { transition_to: transitionTo }),
      },
    };

    if (onTimeout) {
      state.on_timeout = onTimeout;
    }

    return state;
  });

  return {
    id: graph.id,
    name: graph.name,
    enabled: graph.enabled,
    priority: index + 1,
    initial_state: (initialState as Extract<GraphNode, { type: 'state' }>).data.stateName,
    states,
  };
}

function isStateNode(node: GraphNode): boolean {
  return node.type === 'state';
}

/** Get action nodes reachable from a state node (direct outgoing edges to action nodes) */
function getActionsAfterState(stateNode: GraphNode, graph: WorkflowGraph): DSLActionV2[] {
  const outEdges = graph.edges.filter((e) => e.source === stateNode.id);
  const actions: DSLActionV2[] = [];

  for (const edge of outEdges) {
    const target = graph.nodes.find((n) => n.id === edge.target);
    if (target && ACTION_TYPES.has(target.type)) {
      const action = nodeToActionV2(target);
      if (action) actions.push(action);
    }
  }
  return actions;
}

/** Find the state that this state transitions to (state → state edge) */
function getTransitionFromState(
  stateNode: GraphNode,
  graph: WorkflowGraph,
  stateNodes: GraphNode[],
): string | undefined {
  const stateIds = new Set(stateNodes.map((s) => s.id));
  const outEdges = graph.edges.filter((e) => e.source === stateNode.id);

  for (const edge of outEdges) {
    if (stateIds.has(edge.target)) {
      const targetState = stateNodes.find((s) => s.id === edge.target);
      if (targetState?.type === 'state') {
        return targetState.data.stateName;
      }
    }
  }
  return undefined;
}

// ============================================================================
// Matcher compilation — converts trigger nodes to DSL matchers
// ============================================================================

function buildMatchers(
  node: GraphNode,
  allNodes: GraphNode[],
  edges: GraphEdge[]
): DSLMatcher[] {
  const matcher = nodeToMatcher(node, allNodes, edges);
  if (!matcher) return [];

  // If it's a compound matcher (any/all), return its children directly
  // as the state's matcher array already acts as an implicit AND
  if (matcher.combine && matcher.children) {
    return [matcher];
  }

  return [matcher];
}

function nodeToMatcher(
  node: GraphNode,
  allNodes: GraphNode[],
  edges: GraphEdge[]
): DSLMatcher | null {
  // Logic gates — recurse into children
  if (node.type === 'any' || node.type === 'all') {
    const childEdges = edges.filter((e) => e.source === node.id);
    const childNodes = childEdges
      .map((e) => allNodes.find((n) => n.id === e.target))
      .filter((n): n is GraphNode => n !== undefined);

    const children = childNodes
      .map((child) => nodeToMatcher(child, allNodes, edges))
      .filter((m): m is DSLMatcher => m !== null);

    return {
      type: 'event_match', // placeholder type for compound
      config: {},
      combine: node.type,
      children,
    };
  }

  // Use switch for proper discriminated union narrowing
  switch (node.type) {
    case 'event_match':
      return {
        type: 'event_match',
        config: { event_name: node.data.eventName },
        where: node.data.predicates?.length > 0 ? node.data.predicates : undefined,
      };

    case 'log_severity_match':
      return {
        type: 'log_severity',
        config: {
          min_severity: node.data.minSeverity,
          ...(node.data.bodyContains && { body_contains: node.data.bodyContains }),
        },
      };

    case 'metric_threshold':
      return {
        type: 'metric_threshold',
        config: {
          metric_name: node.data.metricName,
          operator: node.data.operator,
          threshold: node.data.threshold,
        },
      };

    case 'http_error_match':
      return {
        type: 'http_match',
        config: {
          status_min: node.data.statusMin,
          ...(node.data.routeContains && { route_contains: node.data.routeContains }),
        },
      };

    case 'crash_marker':
      return { type: 'crash', config: {} };

    case 'exception_pattern':
      return {
        type: 'exception_pattern',
        config: {
          exception_type: node.data.exceptionType,
          ...(node.data.messagePattern && { message_pattern: node.data.messagePattern }),
        },
      };

    case 'ui_freeze':
      return {
        type: 'ui_freeze',
        config: { duration_ms: node.data.durationMs },
      };

    case 'slow_operation':
      return {
        type: 'slow_operation',
        config: {
          operation_name: node.data.operationName,
          threshold_ms: node.data.thresholdMs,
        },
      };

    case 'frame_drop':
      return {
        type: 'frame_drop',
        config: {
          dropped_frames: node.data.droppedFrames,
          window_ms: node.data.windowMs,
        },
      };

    case 'network_loss':
      return {
        type: 'network_loss',
        config: {
          ...(node.data.consecutiveFailures != null && {
            consecutive_failures: node.data.consecutiveFailures,
          }),
        },
      };

    case 'slow_request':
      return {
        type: 'slow_request' as DSLMatcherType,
        config: {
          threshold_ms: node.data.thresholdMs,
          ...(node.data.route && { route: node.data.route }),
        },
      };

    case 'low_memory':
      return {
        type: 'low_memory',
        config: { available_mb: node.data.availableMb },
      };

    case 'battery_drain':
      return {
        type: 'battery_drain',
        config: { drain_rate_perc_per_min: node.data.drainRatePercPerMin },
      };

    case 'thermal_throttling':
      return {
        type: 'thermal_throttle',
        config: { min_level: node.data.minLevel },
      };

    case 'storage_low':
      return {
        type: 'storage_low',
        config: { available_mb: node.data.availableMb },
      };

    case 'predictive_risk':
      return {
        type: 'predictive_risk',
        config: {
          risk_type: node.data.riskType,
          min_score: node.data.minScore,
        },
      };

    case 'timeout_matcher':
      return {
        type: 'timeout',
        config: {
          after_ms: node.data.afterMs,
          ...(node.data.expectedEvent && { expected_event: node.data.expectedEvent }),
        },
      };

    // State and action nodes are not matchers
    case 'state':
    case 'annotate_trigger':
    case 'flush_window':
    case 'set_sampling':
    case 'send_alert':
    case 'adjust_config':
    case 'emit_metric':
    case 'record_session':
    case 'create_funnel':
    case 'create_sankey':
    case 'take_screenshot':
      return null;
  }
}

// ============================================================================
// Action compilation — converts action nodes to DSL v2 actions
// ============================================================================

const ACTION_TYPES = new Set([
  'annotate_trigger',
  'flush_window',
  'set_sampling',
  'send_alert',
  'adjust_config',
  'emit_metric',
  'record_session',
  'create_funnel',
  'create_sankey',
  'take_screenshot',
]);

function isActionNode(node: GraphNode): boolean {
  return ACTION_TYPES.has(node.type);
}

function buildActionsV2(
  entryNode: GraphNode,
  allNodes: GraphNode[],
  edges: GraphEdge[]
): DSLActionV2[] {
  const actions: DSLActionV2[] = [];
  const visited = new Set<string>();

  const walk = (nodeId: string) => {
    if (visited.has(nodeId)) return;
    visited.add(nodeId);

    const node = allNodes.find((n) => n.id === nodeId);
    if (!node) return;

    if (isActionNode(node)) {
      const action = nodeToActionV2(node);
      if (action) actions.push(action);
    }

    const outgoing = edges.filter((e) => e.source === nodeId);
    for (const edge of outgoing) {
      walk(edge.target);
    }
  };

  walk(entryNode.id);
  return actions;
}

function nodeToActionV2(node: GraphNode): DSLActionV2 | null {
  switch (node.type) {
    case 'annotate_trigger':
      return {
        type: 'annotate',
        config: {
          trigger_id: node.data.triggerId,
          reason: node.data.reason,
        },
      };

    case 'flush_window':
      return {
        type: 'flush_buffer',
        config: {
          minutes: node.data.minutes,
          scope: node.data.scope,
        },
      };

    case 'set_sampling':
      return {
        type: 'set_sampling',
        config: {
          rate: node.data.rate,
          duration_minutes: node.data.durationMinutes,
        },
      };

    case 'send_alert':
      return {
        type: 'send_alert',
        config: {
          severity: node.data.severity,
          message: node.data.message,
          channels: node.data.channels,
        },
      };

    case 'adjust_config':
      return {
        type: 'adjust_buffer',
        config: {
          parameter: node.data.parameter as 'ram_events' | 'disk_mb' | 'retention_hours',
          value: Number(node.data.value),
          ...(node.data.durationMinutes != null && {
            duration_minutes: node.data.durationMinutes,
          }),
        },
      };

    case 'emit_metric':
      return {
        type: 'emit_metric',
        config: {
          metric_name: node.data.metricName,
          metric_type: node.data.metricType,
          ...(node.data.fieldExtract && { field_extract: node.data.fieldExtract }),
          ...(node.data.groupBy?.length && { group_by: node.data.groupBy }),
          ...(node.data.bucketBoundaries?.length && { bucket_boundaries: node.data.bucketBoundaries }),
        },
      };

    case 'record_session':
      return {
        type: 'record_session',
        config: {
          max_duration_minutes: node.data.maxDurationMinutes,
          ...(node.data.keepStreamingUntil && { keep_streaming_until: node.data.keepStreamingUntil }),
        },
      };

    case 'create_funnel':
      return {
        type: 'create_funnel',
        config: {
          funnel_name: node.data.funnelName,
          steps: node.data.steps.map((s) => ({
            event_name: s.eventName,
            ...(s.predicates?.length && { predicates: s.predicates }),
          })),
        },
      };

    case 'create_sankey':
      return {
        type: 'create_sankey',
        config: {
          sankey_name: node.data.sankeyName,
          entry_event: node.data.entryEvent,
          exit_events: node.data.exitEvents,
          tracked_events: node.data.trackedEvents,
        },
      };

    case 'take_screenshot':
      return {
        type: 'take_screenshot',
        config: {
          quality: node.data.quality,
          redact_text: node.data.redactText,
        },
      };

    default:
      return null;
  }
}
