import { Handle, Position } from 'reactflow';

interface TakeScreenshotNodeProps {
  data: {
    quality: 'low' | 'medium' | 'high';
    redactText: boolean;
  };
}

export function TakeScreenshotNodeComponent({ data }: TakeScreenshotNodeProps) {
  return (
    <div className="node node-action">
      <Handle type="target" position={Position.Left} />
      <div className="node-header">
        <div className="node-icon">📸</div>
        <div className="node-title">Take Screenshot</div>
      </div>
      <div className="node-body">
        <div className="node-fields">
          <div className="node-field-display">
            <label>Quality:</label>
            <span>{data.quality}</span>
          </div>
          <div className="node-field-display">
            <label>Redact text:</label>
            <span>{data.redactText ? 'Yes' : 'No'}</span>
          </div>
        </div>
      </div>
      <Handle type="source" position={Position.Right} />
    </div>
  );
}
