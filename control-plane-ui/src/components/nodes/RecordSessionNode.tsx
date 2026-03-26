import { Handle, Position } from 'reactflow';

interface RecordSessionNodeProps {
  data: {
    keepStreamingUntil?: string;
    maxDurationMinutes: number;
  };
}

export function RecordSessionNodeComponent({ data }: RecordSessionNodeProps) {
  return (
    <div className="node node-action">
      <Handle type="target" position={Position.Left} />
      <div className="node-header">
        <div className="node-icon">🎥</div>
        <div className="node-title">Record Session</div>
      </div>
      <div className="node-body">
        <div className="node-fields">
          <div className="node-field-display">
            <label>Max duration:</label>
            <span>{data.maxDurationMinutes} min</span>
          </div>
          {data.keepStreamingUntil && (
            <div className="node-field-display">
              <label>Until event:</label>
              <span>{data.keepStreamingUntil}</span>
            </div>
          )}
        </div>
      </div>
      <Handle type="source" position={Position.Right} />
    </div>
  );
}
