import type { FleetGraphNode } from './fleet';

// Graph format (for React Flow editing)
export interface WorkflowGraph {
  id: string;
  name: string;
  enabled: boolean;
  entryNodeId: string;
  nodes: GraphNode[];
  edges: GraphEdge[];
}

export interface GraphEdge {
  id: string;
  source: string;
  target: string;
}

export type GraphNode =
  // Event Triggers
  | EventMatchNode
  | LogSeverityMatchNode
  | MetricThresholdNode
  // Performance Triggers
  | UIFreezeNode
  | SlowOperationNode
  | FrameDropNode
  // Network Triggers
  | HttpErrorMatchNode
  | NetworkLossNode
  | SlowRequestNode
  // Device Health Triggers
  | LowMemoryNode
  | BatteryDrainNode
  | ThermalThrottlingNode
  | StorageLowNode
  // Crash/Error Triggers
  | CrashMarkerNode
  | ExceptionPatternNode
  // Predictive Triggers
  | PredictiveRiskNode
  // Logic Nodes
  | AnyNode
  | AllNode
  // State Machine Nodes
  | StateNode
  | TimeoutMatcherNode
  // Actions
  | AnnotateTriggerNode
  | FlushWindowNode
  | SetSamplingNode
  | SendAlertNode
  | AdjustConfigNode
  // Insight Actions
  | EmitMetricNode
  | RecordSessionNode
  | CreateFunnelNode
  | CreateSankeyNode
  | TakeScreenshotNode
  // Fleet Intelligence
  | FleetGraphNode;

// Trigger Nodes
export interface EventMatchNode {
  id: string;
  type: 'event_match';
  position: { x: number; y: number };
  data: {
    eventName: string;
    predicates: Predicate[];
  };
}

export interface HttpErrorMatchNode {
  id: string;
  type: 'http_error_match';
  position: { x: number; y: number };
  data: {
    statusMin: number;
    routeContains?: string;
  };
}

export interface CrashMarkerNode {
  id: string;
  type: 'crash_marker';
  position: { x: number; y: number };
  data: Record<string, never>;
}

// Additional Event Triggers
export interface LogSeverityMatchNode {
  id: string;
  type: 'log_severity_match';
  position: { x: number; y: number };
  data: {
    minSeverity: 'DEBUG' | 'INFO' | 'WARN' | 'ERROR' | 'FATAL';
    bodyContains?: string;
  };
}

export interface MetricThresholdNode {
  id: string;
  type: 'metric_threshold';
  position: { x: number; y: number };
  data: {
    metricName: string;
    operator: '>' | '>=' | '<' | '<=';
    threshold: number;
  };
}

// Performance Triggers
export interface UIFreezeNode {
  id: string;
  type: 'ui_freeze';
  position: { x: number; y: number };
  data: {
    durationMs: number;
  };
}

export interface SlowOperationNode {
  id: string;
  type: 'slow_operation';
  position: { x: number; y: number };
  data: {
    operationName: string;
    thresholdMs: number;
  };
}

export interface FrameDropNode {
  id: string;
  type: 'frame_drop';
  position: { x: number; y: number };
  data: {
    droppedFrames: number;
    windowMs: number;
  };
}

// Network Triggers
export interface NetworkLossNode {
  id: string;
  type: 'network_loss';
  position: { x: number; y: number };
  data: {
    consecutiveFailures?: number;
  };
}

export interface SlowRequestNode {
  id: string;
  type: 'slow_request';
  position: { x: number; y: number };
  data: {
    thresholdMs: number;
    route?: string;
  };
}

// Device Health Triggers
export interface LowMemoryNode {
  id: string;
  type: 'low_memory';
  position: { x: number; y: number };
  data: {
    availableMb: number;
  };
}

export interface BatteryDrainNode {
  id: string;
  type: 'battery_drain';
  position: { x: number; y: number };
  data: {
    drainRatePercPerMin: number;
  };
}

export interface ThermalThrottlingNode {
  id: string;
  type: 'thermal_throttling';
  position: { x: number; y: number };
  data: {
    minLevel: 'LIGHT' | 'MODERATE' | 'SEVERE' | 'CRITICAL';
  };
}

export interface StorageLowNode {
  id: string;
  type: 'storage_low';
  position: { x: number; y: number };
  data: {
    availableMb: number;
  };
}

// Crash/Error Triggers
export interface ExceptionPatternNode {
  id: string;
  type: 'exception_pattern';
  position: { x: number; y: number };
  data: {
    exceptionType: string;
    messagePattern?: string;
  };
}

// Predictive Triggers
export interface PredictiveRiskNode {
  id: string;
  type: 'predictive_risk';
  position: { x: number; y: number };
  data: {
    riskType: 'crash' | 'network_loss' | 'performance_degradation' | 'battery_drain';
    minScore: number;
  };
}

// Logic Nodes
export interface AnyNode {
  id: string;
  type: 'any';
  position: { x: number; y: number };
  data: Record<string, never>;
}

export interface AllNode {
  id: string;
  type: 'all';
  position: { x: number; y: number };
  data: Record<string, never>;
}

// State Machine Nodes
export interface StateNode {
  id: string;
  type: 'state';
  position: { x: number; y: number };
  data: {
    stateName: string;
    isInitial?: boolean;
    color?: string;
  };
}

export interface TimeoutMatcherNode {
  id: string;
  type: 'timeout_matcher';
  position: { x: number; y: number };
  data: {
    afterMs: number;
    expectedEvent?: string;
  };
}

// Action Nodes
export interface AnnotateTriggerNode {
  id: string;
  type: 'annotate_trigger';
  position: { x: number; y: number };
  data: {
    triggerId: string;
    reason: string;
  };
}

export interface FlushWindowNode {
  id: string;
  type: 'flush_window';
  position: { x: number; y: number };
  data: {
    minutes: number;
    scope: 'session' | 'device';
  };
}

export interface SetSamplingNode {
  id: string;
  type: 'set_sampling';
  position: { x: number; y: number };
  data: {
    rate: number;
    durationMinutes: number;
  };
}

export interface SendAlertNode {
  id: string;
  type: 'send_alert';
  position: { x: number; y: number };
  data: {
    severity: 'info' | 'warning' | 'critical';
    message: string;
    channels: ('email' | 'slack' | 'pagerduty')[];
  };
}

export interface AdjustConfigNode {
  id: string;
  type: 'adjust_config';
  position: { x: number; y: number };
  data: {
    parameter: string;
    value: string | number | boolean;
    durationMinutes?: number;
  };
}

// Insight Action Nodes
export interface EmitMetricNode {
  id: string;
  type: 'emit_metric';
  position: { x: number; y: number };
  data: {
    metricName: string;
    metricType: 'counter' | 'histogram' | 'gauge';
    fieldExtract?: string;
    groupBy?: string[];
    bucketBoundaries?: number[];
  };
}

export interface RecordSessionNode {
  id: string;
  type: 'record_session';
  position: { x: number; y: number };
  data: {
    keepStreamingUntil?: string;
    maxDurationMinutes: number;
  };
}

export interface CreateFunnelNode {
  id: string;
  type: 'create_funnel';
  position: { x: number; y: number };
  data: {
    funnelName: string;
    steps: { eventName: string; predicates?: Predicate[] }[];
  };
}

export interface CreateSankeyNode {
  id: string;
  type: 'create_sankey';
  position: { x: number; y: number };
  data: {
    sankeyName: string;
    entryEvent: string;
    exitEvents: string[];
    trackedEvents: string[];
  };
}

export interface TakeScreenshotNode {
  id: string;
  type: 'take_screenshot';
  position: { x: number; y: number };
  data: {
    quality: 'low' | 'medium' | 'high';
    redactText: boolean;
  };
}

export interface Predicate {
  attr: string;
  op: '==' | '!=' | '>' | '>=' | '<' | '<=' | 'contains' | 'regex';
  value: string | number | boolean;
}

// DSL format (for device execution)
export interface DSLConfig {
  version: number;
  limits: {
    diskMb: number;
    ramEvents: number;
    retentionHours: number;
  };
  workflows: DSLWorkflow[];
}

export interface DSLWorkflow {
  id: string;
  enabled: boolean;
  trigger: DSLTrigger;
  actions: DSLAction[];
}

export interface DSLTrigger {
  any?: DSLCondition[];
  all?: DSLCondition[];
}

export interface DSLCondition {
  event?: string;
  where?: Predicate[];
}

export type DSLAction =
  | {
      type: 'annotate_trigger';
      trigger_id: string;
      reason: string;
    }
  | {
      type: 'flush_window';
      minutes: number;
      scope: 'session' | 'device';
    }
  | {
      type: 'set_sampling';
      rate: number;
      duration_minutes: number;
    };

// ============================================================================
// DSL v2 types (state-machine-based)
// ============================================================================

export interface DSLConfigV2 {
  version: 2;
  buffer_config: DSLBufferConfig;
  targeting?: DSLTargeting;
  workflows: DSLWorkflowV2[];
}

export interface DSLBufferConfig {
  ram_events: number;
  disk_mb: number;
  retention_hours: number;
  strategy: 'overwrite_oldest' | 'stop_recording';
}

export interface DSLTargeting {
  platform?: 'android' | 'ios';
  app_version_range?: string; // semver range, e.g., ">=2.0.0 <3.0.0"
  os_version_range?: string;  // semver range
  device_models?: string[];   // glob patterns, e.g., ["Pixel*", "Samsung*"]
  device_group?: string;
  custom_attributes?: Record<string, string>;
}

export interface DSLWorkflowV2 {
  id: string;
  name: string;
  enabled: boolean;
  priority: number;
  initial_state: string;
  states: DSLState[];
}

export interface DSLState {
  id: string;
  matchers: DSLMatcher[];
  on_match: {
    actions: DSLActionV2[];
    transition_to?: string;
  };
  on_timeout?: {
    after_ms: number;
    actions: DSLActionV2[];
    transition_to?: string;
  };
}

export type DSLMatcherType =
  | 'event_match'
  | 'log_severity'
  | 'metric_threshold'
  | 'http_match'
  | 'crash'
  | 'exception_pattern'
  | 'ui_freeze'
  | 'slow_operation'
  | 'frame_drop'
  | 'network_loss'
  | 'low_memory'
  | 'battery_drain'
  | 'thermal_throttle'
  | 'storage_low'
  | 'field_presence'
  | 'field_absence'
  | 'timeout'
  | 'predictive_risk'
  | 'anr'
  | 'app_lifecycle'
  | 'resource_snapshot'
  | 'fleet_threshold'
  | 'fleet_rate'
  | 'fleet_absence'
  | 'fleet_correlation'
  | 'fleet_anomaly'
  | 'fleet_prediction'
  | 'fleet_root_cause'
  | 'backend_health'
  | 'backend_deploy'
  | 'backend_capacity';

export interface DSLMatcher {
  type: DSLMatcherType;
  config: Record<string, unknown>;
  where?: PredicateV2[];
  combine?: 'any' | 'all';
  children?: DSLMatcher[];
}

export interface PredicateV2 {
  attr: string;
  op: '==' | '!=' | '>' | '>=' | '<' | '<=' | 'contains' | 'regex'
    | 'semver_gt' | 'semver_lt' | 'semver_gte' | 'semver_lte'
    | 'exists' | 'not_exists';
  value?: string | number | boolean;
}

export type DSLActionV2Type =
  | 'flush_buffer'
  | 'record_session'
  | 'emit_metric'
  | 'create_funnel'
  | 'create_sankey'
  | 'take_screenshot'
  | 'annotate'
  | 'set_sampling'
  | 'adjust_buffer'
  | 'send_alert'
  | 'fleet_flush'
  | 'fleet_set_sampling'
  | 'fleet_adjust_config'
  | 'fleet_screenshot'
  | 'fleet_client_circuit_break';

export interface DSLActionV2 {
  type: DSLActionV2Type;
  config: DSLActionConfig;
}

// Action config types (discriminated by action type)
export type DSLActionConfig =
  | FlushBufferConfig
  | RecordSessionConfig
  | EmitMetricConfig
  | CreateFunnelConfig
  | CreateSankeyConfig
  | TakeScreenshotConfig
  | AnnotateConfig
  | FleetActionGenericConfig
  | SetSamplingConfig
  | AdjustBufferConfig
  | SendAlertConfig;

export interface FlushBufferConfig {
  minutes: number;
  scope: 'session' | 'device';
}

export interface RecordSessionConfig {
  keep_streaming_until?: string; // event name that ends recording
  max_duration_minutes: number;
}

export interface EmitMetricConfig {
  metric_name: string;
  metric_type: 'counter' | 'histogram' | 'gauge';
  field_extract?: string;         // field path to extract value from
  group_by?: string[];            // dimensions to group by
  bucket_boundaries?: number[];   // for histograms
}

export interface CreateFunnelConfig {
  funnel_name: string;
  steps: {
    event_name: string;
    predicates?: PredicateV2[];
  }[];
}

export interface CreateSankeyConfig {
  sankey_name: string;
  entry_event: string;
  exit_events: string[];
  tracked_events: string[];
}

export interface TakeScreenshotConfig {
  quality: 'low' | 'medium' | 'high';
  redact_text: boolean;
}

export interface AnnotateConfig {
  trigger_id: string;
  reason: string;
}

export interface SetSamplingConfig {
  rate: number;
  duration_minutes: number;
}

export interface AdjustBufferConfig {
  parameter: 'ram_events' | 'disk_mb' | 'retention_hours';
  value: number;
  duration_minutes?: number; // temporary adjustment, reverts after
}

export interface SendAlertConfig {
  severity: 'info' | 'warning' | 'critical';
  message: string;
  channels: ('email' | 'slack' | 'pagerduty')[];
}

// Generic config for fleet-level actions (flexible key/value pairs)
export interface FleetActionGenericConfig {
  [key: string]: unknown;
}

// ============================================================================
// Config version (from Gateway)
// ============================================================================

export interface ConfigVersion {
  version: number;
  graph_json: string;
  dsl_json: string;
  dsl_v2_json?: string;
  published_at: string;
  published_by: string;
  is_active: boolean;
}

// Device heartbeat
export interface DeviceHeartbeat {
  device_id: string;
  app_id: string;
  session_id: string;
  buffer_usage_mb: number;
  last_triggers: string[];
  config_version: number;
  timestamp: string;
}
