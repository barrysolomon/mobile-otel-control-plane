import { Handle, Position, type NodeProps } from 'reactflow';

const fleetTriggerIcons: Record<string, string> = {
  fleet_threshold: '\u{1F4CA}',
  fleet_rate: '\u{1F4C8}',
  fleet_absence: '\u{1F47B}',
  fleet_correlation: '\u{1F517}',
  fleet_anomaly: '\u{1F52E}',
  fleet_prediction: '\u{1F52D}',
  fleet_root_cause: '\u{1F50D}',
  backend_health: '\u{1F3E5}',
  backend_deploy: '\u{1F680}',
  backend_capacity: '\u{26A1}',
};

const fleetTriggerLabels: Record<string, string> = {
  fleet_threshold: 'Fleet Threshold',
  fleet_rate: 'Fleet Rate',
  fleet_absence: 'Fleet Absence',
  fleet_correlation: 'Fleet Correlation',
  fleet_anomaly: 'Fleet Anomaly',
  fleet_prediction: 'Fleet Prediction',
  fleet_root_cause: 'Fleet Root Cause',
  backend_health: 'Backend Health',
  backend_deploy: 'Backend Deploy',
  backend_capacity: 'Backend Capacity',
};

export function FleetTriggerNode({ data, type }: NodeProps) {
  const icon = fleetTriggerIcons[type!] || '\u{1F310}';
  const label = fleetTriggerLabels[type!] || 'Fleet Trigger';
  const excludeKeys = ['label', 'onChange'];
  const fields = Object.entries(data).filter(([k]) => !excludeKeys.includes(k));

  return (
    <div className="node node-trigger" style={{ borderColor: '#6366f1', borderWidth: 2 }}>
      <div className="node-header">
        <span className="node-icon">{icon}</span>
        <span className="node-title">{label}</span>
      </div>
      {fields.map(([key, value]) => (
        <div key={key} className="node-field">
          <span className="field-label">{key.replace(/([A-Z])/g, ' $1').trim()}:</span>
          <span className="field-value">{String(value)}</span>
        </div>
      ))}
      <Handle type="source" position={Position.Right} />
    </div>
  );
}
