// Fleet Intelligence node type definitions

// Fleet Triggers
export interface FleetThresholdNode {
  id: string;
  type: 'fleet_threshold';
  position: { x: number; y: number };
  data: {
    threshold: number;
    triggerType: string;
    windowMinutes: number;
    cohortId?: string;
  };
}

export interface FleetRateNode {
  id: string;
  type: 'fleet_rate';
  position: { x: number; y: number };
  data: {
    factor: number;
    triggerType: string;
    baselineWindowMin: number;
  };
}

export interface FleetAbsenceNode {
  id: string;
  type: 'fleet_absence';
  position: { x: number; y: number };
  data: {
    minSilentDevices: number;
    windowMinutes: number;
  };
}

export interface FleetCorrelationNode {
  id: string;
  type: 'fleet_correlation';
  position: { x: number; y: number };
  data: {
    patternType: string;
    windowSeconds: number;
  };
}

export interface FleetAnomalyNode {
  id: string;
  type: 'fleet_anomaly';
  position: { x: number; y: number };
  data: {
    metric: string;
    confidenceThreshold: number;
  };
}

export interface FleetPredictionNode {
  id: string;
  type: 'fleet_prediction';
  position: { x: number; y: number };
  data: {
    metric: string;
    lookaheadMinutes: number;
  };
}

export interface FleetRootCauseNode {
  id: string;
  type: 'fleet_root_cause';
  position: { x: number; y: number };
  data: {
    confidenceThreshold: number;
  };
}

// Backend Triggers
export interface BackendHealthNode {
  id: string;
  type: 'backend_health';
  position: { x: number; y: number };
  data: {
    serviceName: string;
    metricType: string;
    threshold: number;
  };
}

export interface BackendDeployNode {
  id: string;
  type: 'backend_deploy';
  position: { x: number; y: number };
  data: {
    serviceName: string;
  };
}

export interface BackendCapacityNode {
  id: string;
  type: 'backend_capacity';
  position: { x: number; y: number };
  data: {
    serviceName: string;
  };
}

// Fleet Actions
export interface FleetFlushNode {
  id: string;
  type: 'fleet_flush';
  position: { x: number; y: number };
  data: {
    minutes: number;
    scope: string;
  };
}

export interface FleetSetSamplingNode {
  id: string;
  type: 'fleet_set_sampling';
  position: { x: number; y: number };
  data: {
    rate: number;
    durationMinutes: number;
  };
}

export interface FleetAdjustConfigNode {
  id: string;
  type: 'fleet_adjust_config';
  position: { x: number; y: number };
  data: {
    key: string;
    value: string;
    durationMinutes: number;
  };
}

export interface FleetScreenshotNode {
  id: string;
  type: 'fleet_screenshot';
  position: { x: number; y: number };
  data: Record<string, never>;
}

export interface FleetClientCircuitBreakNode {
  id: string;
  type: 'fleet_client_circuit_break';
  position: { x: number; y: number };
  data: {
    action: string;
  };
}

// Cohort Targeting
export interface CohortStaticNode {
  id: string;
  type: 'cohort_static';
  position: { x: number; y: number };
  data: {
    deviceGroup: string;
  };
}

export interface CohortDynamicNode {
  id: string;
  type: 'cohort_dynamic';
  position: { x: number; y: number };
  data: {
    rulesJson: string;
  };
}

export interface CohortDiscoveredNode {
  id: string;
  type: 'cohort_discovered';
  position: { x: number; y: number };
  data: {
    clusterId: string;
  };
}

// Safety
export interface CircuitBreakerConfigNode {
  id: string;
  type: 'circuit_breaker_config';
  position: { x: number; y: number };
  data: {
    maxCascadeDepth: number;
    cooldownMinutes: number;
    maxPercentAffected: number;
    maxAbsoluteDevices: number;
    budgetWindowMinutes: number;
    maxAlertsPerHour: number;
    chainTimeoutMinutes: number;
    hopTimeoutMinutes: number;
  };
}

// Union type for all fleet graph nodes
export type FleetGraphNode =
  | FleetThresholdNode
  | FleetRateNode
  | FleetAbsenceNode
  | FleetCorrelationNode
  | FleetAnomalyNode
  | FleetPredictionNode
  | FleetRootCauseNode
  | BackendHealthNode
  | BackendDeployNode
  | BackendCapacityNode
  | FleetFlushNode
  | FleetSetSamplingNode
  | FleetAdjustConfigNode
  | FleetScreenshotNode
  | FleetClientCircuitBreakNode
  | CohortStaticNode
  | CohortDynamicNode
  | CohortDiscoveredNode
  | CircuitBreakerConfigNode;
