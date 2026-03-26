import { Handle, Position } from 'reactflow';

interface StateNodeProps {
  data: {
    stateName: string;
    isInitial?: boolean;
    color?: string;
  };
}

const STATE_COLORS: Record<string, string> = {
  idle: '#6b7280',
  recording: '#ef4444',
  uploading: '#3b82f6',
  monitoring: '#10b981',
  alerting: '#f59e0b',
};

export function StateNodeComponent({ data }: StateNodeProps) {
  const color = data.color || STATE_COLORS[data.stateName] || '#8b5cf6';

  return (
    <div
      className="node node-state"
      style={{ borderColor: color, borderWidth: 2 }}
    >
      <Handle type="target" position={Position.Left} />
      <div className="node-header" style={{ backgroundColor: color }}>
        <div className="node-icon">{data.isInitial ? '▶' : '◉'}</div>
        <div className="node-title" style={{ color: '#fff' }}>
          {data.stateName || 'State'}
        </div>
        {data.isInitial && (
          <span className="node-badge" style={{ background: '#fff', color, fontSize: '0.65rem', padding: '1px 5px', borderRadius: 3, marginLeft: 4 }}>
            initial
          </span>
        )}
      </div>
      <div className="node-body">
        <div className="node-description">
          Connect triggers, actions, and transitions
        </div>
      </div>
      <Handle type="source" position={Position.Right} />
    </div>
  );
}
