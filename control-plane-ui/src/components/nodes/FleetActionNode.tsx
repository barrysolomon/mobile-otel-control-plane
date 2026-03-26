import { Handle, Position, type NodeProps } from 'reactflow';

const fleetActionIcons: Record<string, string> = {
  fleet_flush: '\u{1F4BE}',
  fleet_set_sampling: '\u{1F39A}\uFE0F',
  fleet_adjust_config: '\u{2699}\uFE0F',
  fleet_screenshot: '\u{1F4F8}',
  fleet_client_circuit_break: '\u{1F50C}',
};

const fleetActionLabels: Record<string, string> = {
  fleet_flush: 'Fleet Flush',
  fleet_set_sampling: 'Fleet Sampling',
  fleet_adjust_config: 'Fleet Config',
  fleet_screenshot: 'Fleet Screenshot',
  fleet_client_circuit_break: 'Client Circuit Break',
};

export function FleetActionNode({ data, type }: NodeProps) {
  const icon = fleetActionIcons[type!] || '\u{26A1}';
  const label = fleetActionLabels[type!] || 'Fleet Action';
  const excludeKeys = ['label', 'onChange'];
  const fields = Object.entries(data).filter(([k]) => !excludeKeys.includes(k));

  return (
    <div className="node node-action" style={{ borderColor: '#6366f1', borderWidth: 2 }}>
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
      <Handle type="target" position={Position.Left} />
    </div>
  );
}
