import { Handle, Position, type NodeProps } from 'reactflow';

export function CircuitBreakerNode({ data }: NodeProps) {
  return (
    <div className="node node-safety" style={{ borderColor: '#ef4444', borderWidth: 2 }}>
      <div className="node-header">
        <span className="node-icon">{'\u{1F6E1}\uFE0F'}</span>
        <span className="node-title">Circuit Breakers</span>
      </div>
      <div className="node-field">
        <span className="field-label">Max depth:</span>
        <span className="field-value">{data.maxCascadeDepth || 3}</span>
      </div>
      <div className="node-field">
        <span className="field-label">Cooldown:</span>
        <span className="field-value">{data.cooldownMinutes || 15}m</span>
      </div>
      <div className="node-field">
        <span className="field-label">Budget:</span>
        <span className="field-value">{data.maxPercentAffected || 25}%</span>
      </div>
      <div className="node-field">
        <span className="field-label">Max devices:</span>
        <span className="field-value">{data.maxAbsoluteDevices || 10000}</span>
      </div>
      <Handle type="target" position={Position.Left} />
    </div>
  );
}
