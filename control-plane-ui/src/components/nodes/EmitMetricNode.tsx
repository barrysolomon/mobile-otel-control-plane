import { Handle, Position } from 'reactflow';

interface EmitMetricNodeProps {
  data: {
    metricName: string;
    metricType: 'counter' | 'histogram' | 'gauge';
    fieldExtract?: string;
    groupBy?: string[];
    bucketBoundaries?: number[];
  };
}

export function EmitMetricNodeComponent({ data }: EmitMetricNodeProps) {
  return (
    <div className="node node-action">
      <Handle type="target" position={Position.Left} />
      <div className="node-header">
        <div className="node-icon">📈</div>
        <div className="node-title">Emit Metric</div>
      </div>
      <div className="node-body">
        <div className="node-fields">
          <div className="node-field-display">
            <label>Name:</label>
            <span>{data.metricName || '(not set)'}</span>
          </div>
          <div className="node-field-display">
            <label>Type:</label>
            <span>{data.metricType}</span>
          </div>
          {data.fieldExtract && (
            <div className="node-field-display">
              <label>Extract:</label>
              <span>{data.fieldExtract}</span>
            </div>
          )}
          {data.groupBy && data.groupBy.length > 0 && (
            <div className="node-field-display">
              <label>Group by:</label>
              <span>{data.groupBy.join(', ')}</span>
            </div>
          )}
        </div>
      </div>
      <Handle type="source" position={Position.Right} />
    </div>
  );
}
