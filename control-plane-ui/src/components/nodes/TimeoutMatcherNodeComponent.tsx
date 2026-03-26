import { Handle, Position } from 'reactflow';

interface TimeoutMatcherNodeProps {
  data: {
    afterMs: number;
    expectedEvent?: string;
  };
}

export function TimeoutMatcherNodeComponent({ data }: TimeoutMatcherNodeProps) {
  const seconds = data.afterMs / 1000;
  const display = seconds >= 60
    ? `${(seconds / 60).toFixed(0)} min`
    : `${seconds}s`;

  return (
    <div className="node node-timeout">
      <Handle type="target" position={Position.Left} />
      <div className="node-header">
        <div className="node-icon">⏰</div>
        <div className="node-title">Timeout</div>
      </div>
      <div className="node-body">
        <div className="node-fields">
          <div className="node-field-display">
            <label>After:</label>
            <span>{display}</span>
          </div>
          {data.expectedEvent && (
            <div className="node-field-display">
              <label>Expected:</label>
              <span>{data.expectedEvent}</span>
            </div>
          )}
        </div>
      </div>
      <Handle type="source" position={Position.Right} />
    </div>
  );
}
