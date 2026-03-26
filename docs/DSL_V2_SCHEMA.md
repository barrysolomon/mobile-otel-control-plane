# DSL v2 Schema Reference

The DSL v2 is the interface contract between the control plane and the Android SDK. Devices fetch this config via `GET /config?dsl_version=2&app_id=...&device_id=...`.

## Top-Level Structure

```json
{
  "version": 2,
  "buffer_config": { ... },
  "targeting": { ... },
  "workflows": [ ... ]
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `version` | `2` | Yes | Always `2` for this schema |
| `buffer_config` | `BufferConfig` | Yes | Ring buffer sizing for on-device storage |
| `targeting` | `Targeting` | No | Device targeting rules (which devices this config applies to) |
| `workflows` | `WorkflowV2[]` | Yes | Finite state machine workflows to execute on device |

## BufferConfig

Controls the on-device ring buffer that stores telemetry locally before upload.

```json
{
  "ram_events": 5000,
  "disk_mb": 50,
  "retention_hours": 24,
  "strategy": "overwrite_oldest"
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `ram_events` | `int` | 5000 | Max events in volatile RAM buffer (MPSC) |
| `disk_mb` | `int` | 50 | Max disk ring buffer size in MB (mmap'd) |
| `retention_hours` | `int` | 24 | Drop events older than this |
| `strategy` | `string` | `overwrite_oldest` | `overwrite_oldest` or `stop_recording` when full |

## Targeting (optional)

Specifies which devices should receive this config. If omitted, all devices receive it.

```json
{
  "platform": "android",
  "app_version_range": ">=2.0.0 <3.0.0",
  "os_version_range": ">=12.0.0",
  "device_models": ["Pixel*", "Samsung*"],
  "device_group": "production",
  "custom_attributes": { "beta": "true" }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `platform` | `string` | `android` or `ios` |
| `app_version_range` | `string` | Semver range expression |
| `os_version_range` | `string` | Semver range expression |
| `device_models` | `string[]` | Glob patterns for device model names |
| `device_group` | `string` | Device group name |
| `custom_attributes` | `Record<string, string>` | Key-value attribute match |

## WorkflowV2

Each workflow is a finite state machine (FSM) that runs on-device.

```json
{
  "id": "crash-handler",
  "name": "Crash Handler",
  "enabled": true,
  "priority": 1,
  "initial_state": "watching",
  "states": [ ... ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `id` | `string` | Unique workflow identifier |
| `name` | `string` | Human-readable name |
| `enabled` | `bool` | Whether workflow is active |
| `priority` | `int` | Execution priority (lower = higher priority) |
| `initial_state` | `string` | ID of the first state to enter |
| `states` | `State[]` | FSM states |

## State

A state in the workflow FSM. The SDK enters the `initial_state` and transitions between states based on matcher results and timeouts.

```json
{
  "id": "watching",
  "matchers": [ ... ],
  "on_match": {
    "actions": [ ... ],
    "transition_to": "recording"
  },
  "on_timeout": {
    "after_ms": 300000,
    "actions": [ ... ],
    "transition_to": "idle"
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `id` | `string` | Unique state ID within this workflow |
| `matchers` | `Matcher[]` | Conditions that trigger this state's `on_match` |
| `on_match` | `MatchResult` | Actions + optional state transition when matchers fire |
| `on_timeout` | `TimeoutResult?` | Actions + transition if no match within `after_ms` milliseconds |

## Matcher

Matchers evaluate every log/event on the device. When a matcher's conditions are met, the state's `on_match` fires.

```json
{
  "type": "crash",
  "config": {},
  "where": [
    { "attr": "exception_type", "op": "contains", "value": "OutOfMemory" }
  ]
}
```

### Matcher Types

| Type | Description | Config Fields |
|------|-------------|---------------|
| `event_match` | Match named events | `event_name: string` |
| `log_severity` | Match log level | `min_severity: "DEBUG"\|"INFO"\|"WARN"\|"ERROR"\|"FATAL"`, `body_contains?: string` |
| `metric_threshold` | Metric exceeds threshold | `metric_name: string`, `operator: ">"\|">="\|"<"\|"<="`, `threshold: number` |
| `http_match` | HTTP request/response | `status_min?: number`, `status_max?: number`, `route_contains?: string`, `method?: string` |
| `crash` | App crash detected | _(none)_ |
| `exception_pattern` | Exception by type/message | `exception_type: string`, `message_pattern?: string` |
| `ui_freeze` | ANR / UI thread blocked | `duration_ms: number` |
| `slow_operation` | Named op exceeds threshold | `operation_name: string`, `threshold_ms: number` |
| `frame_drop` | Dropped rendering frames | `dropped_frames: number`, `window_ms: number` |
| `network_loss` | Network connectivity lost | `consecutive_failures?: number` |
| `low_memory` | Available RAM below threshold | `available_mb: number` |
| `battery_drain` | Battery draining fast | `drain_rate_perc_per_min: number` |
| `thermal_throttle` | Device thermal throttling | `min_level: "LIGHT"\|"MODERATE"\|"SEVERE"\|"CRITICAL"` |
| `storage_low` | Disk space below threshold | `available_mb: number` |
| `predictive_risk` | ML risk prediction | `risk_type: string`, `min_score: number` |
| `anr` | Application Not Responding | _(none)_ |
| `app_lifecycle` | App lifecycle event | `event: "start"\|"foreground"\|"background"\|"terminate"` |
| `resource_snapshot` | Periodic resource check | `metric: string`, `operator: string`, `threshold: number` |
| `field_presence` | Field exists on log | `field_name: string` |
| `field_absence` | Field missing from log | `field_name: string` |
| `timeout` | Time elapsed without match | `duration_ms: number` |

### Compound Matchers (AND/OR)

Use `combine` + `children` for compound conditions:

```json
{
  "type": "event_match",
  "config": {},
  "combine": "any",
  "children": [
    { "type": "crash", "config": {} },
    { "type": "anr", "config": {} }
  ]
}
```

### Predicate Operators (`where` clause)

| Operator | Description |
|----------|-------------|
| `==` | Equals |
| `!=` | Not equals |
| `>`, `>=`, `<`, `<=` | Numeric comparison |
| `contains` | String contains |
| `regex` | Regex match |
| `semver_gt`, `semver_lt`, `semver_gte`, `semver_lte` | Semantic version comparison |
| `exists` | Field is present |
| `not_exists` | Field is absent |

## ActionV2

Actions execute when a state's matchers fire (or on timeout).

```json
{
  "type": "flush_buffer",
  "config": {
    "minutes": 5,
    "scope": "session"
  }
}
```

### Action Types

| Type | Description | Config Fields |
|------|-------------|---------------|
| `flush_buffer` | Upload ring buffer contents | `minutes: int` (lookback), `scope: "session"\|"device"` |
| `record_session` | Record full session until end | `max_duration_minutes: int`, `keep_streaming_until?: string` (event name) |
| `emit_metric` | Generate on-device metric | `metric_name: string`, `metric_type: "counter"\|"histogram"\|"gauge"`, `field_extract?: string`, `group_by?: string[]`, `bucket_boundaries?: number[]` |
| `create_funnel` | Track conversion funnel | `funnel_name: string`, `steps: [{event_name, predicates?}]` |
| `create_sankey` | Track user journey paths | `sankey_name: string`, `entry_event: string`, `exit_events: string[]`, `tracked_events: string[]` |
| `take_screenshot` | Capture app layout | `quality: "low"\|"medium"\|"high"`, `redact_text: boolean` |
| `annotate` | Tag the event | `trigger_id: string`, `reason: string` |
| `set_sampling` | Adjust sampling rate | `rate: number` (0.0-1.0), `duration_minutes: int` |
| `adjust_buffer` | Change buffer config | `parameter: "ram_events"\|"disk_mb"\|"retention_hours"`, `value: number`, `duration_minutes?: int` |
| `send_alert` | Send alert to channels | `severity: "info"\|"warning"\|"critical"`, `message: string`, `channels: ("email"\|"slack"\|"pagerduty")[]` |

## Complete Example

A crash handler workflow that watches for crashes, flushes the ring buffer, records the session, and emits a metric:

```json
{
  "version": 2,
  "buffer_config": {
    "ram_events": 5000,
    "disk_mb": 50,
    "retention_hours": 24,
    "strategy": "overwrite_oldest"
  },
  "workflows": [
    {
      "id": "crash-handler",
      "name": "Crash Handler",
      "enabled": true,
      "priority": 1,
      "initial_state": "watching",
      "states": [
        {
          "id": "watching",
          "matchers": [
            {
              "type": "crash",
              "config": {}
            }
          ],
          "on_match": {
            "actions": [
              {
                "type": "flush_buffer",
                "config": { "minutes": 5, "scope": "session" }
              },
              {
                "type": "record_session",
                "config": { "max_duration_minutes": 10 }
              },
              {
                "type": "emit_metric",
                "config": {
                  "metric_name": "crash_count",
                  "metric_type": "counter"
                }
              },
              {
                "type": "annotate",
                "config": {
                  "trigger_id": "crash-handler",
                  "reason": "crash detected"
                }
              }
            ]
          }
        }
      ]
    },
    {
      "id": "network-monitor",
      "name": "Network Quality Monitor",
      "enabled": true,
      "priority": 2,
      "initial_state": "monitoring",
      "states": [
        {
          "id": "monitoring",
          "matchers": [
            {
              "type": "http_match",
              "config": { "status_min": 500 },
              "where": [
                { "attr": "route", "op": "contains", "value": "/api/" }
              ]
            }
          ],
          "on_match": {
            "actions": [
              {
                "type": "emit_metric",
                "config": {
                  "metric_name": "server_error_count",
                  "metric_type": "counter",
                  "group_by": ["route", "status_code"]
                }
              },
              {
                "type": "flush_buffer",
                "config": { "minutes": 2, "scope": "session" }
              }
            ],
            "transition_to": "cooldown"
          }
        },
        {
          "id": "cooldown",
          "matchers": [],
          "on_match": { "actions": [] },
          "on_timeout": {
            "after_ms": 60000,
            "actions": [],
            "transition_to": "monitoring"
          }
        }
      ]
    }
  ]
}
```

## Version Negotiation

The gateway serves both v1 and v2 configs:

- `GET /config?app_id=X&device_id=Y` — returns v1 DSL (backward compatible)
- `GET /config?app_id=X&device_id=Y&dsl_version=2` — returns v2 DSL (state machines)

Devices running older SDK versions continue to poll v1. New SDK versions request v2.

## Node Type to DSL Matcher Mapping

| Control Plane Node | DSL Matcher Type |
|--------------------|-----------------|
| `event_match` | `event_match` |
| `log_severity_match` | `log_severity` |
| `metric_threshold` | `metric_threshold` |
| `http_error_match` | `http_match` |
| `crash_marker` | `crash` |
| `exception_pattern` | `exception_pattern` |
| `ui_freeze` | `ui_freeze` |
| `slow_operation` | `slow_operation` |
| `frame_drop` | `frame_drop` |
| `network_loss` | `network_loss` |
| `low_memory` | `low_memory` |
| `battery_drain` | `battery_drain` |
| `thermal_throttling` | `thermal_throttle` |
| `storage_low` | `storage_low` |
| `predictive_risk` | `predictive_risk` |
| `any` (logic gate) | `combine: "any"` |
| `all` (logic gate) | `combine: "all"` |

## Node Type to DSL Action Mapping

| Control Plane Node | DSL Action Type |
|--------------------|----------------|
| `flush_window` | `flush_buffer` |
| `set_sampling` | `set_sampling` |
| `annotate_trigger` | `annotate` |
| `send_alert` | `send_alert` |
| `adjust_config` | `adjust_buffer` |
| _(Phase 4)_ `emit_metric` | `emit_metric` |
| _(Phase 4)_ `record_session` | `record_session` |
| _(Phase 4)_ `create_funnel` | `create_funnel` |
| _(Phase 4)_ `create_sankey` | `create_sankey` |
| _(Phase 4)_ `take_screenshot` | `take_screenshot` |
