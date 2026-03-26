import { Handle, Position } from 'reactflow';

interface CreateSankeyNodeProps {
  data: {
    sankeyName: string;
    entryEvent: string;
    exitEvents: string[];
    trackedEvents: string[];
  };
}

export function CreateSankeyNodeComponent({ data }: CreateSankeyNodeProps) {
  return (
    <div className="node node-action">
      <Handle type="target" position={Position.Left} />
      <div className="node-header">
        <div className="node-icon">🌊</div>
        <div className="node-title">Create Sankey</div>
      </div>
      <div className="node-body">
        <div className="node-fields">
          <div className="node-field-display">
            <label>Name:</label>
            <span>{data.sankeyName || '(not set)'}</span>
          </div>
          <div className="node-field-display">
            <label>Entry:</label>
            <span>{data.entryEvent || '(not set)'}</span>
          </div>
          <div className="node-field-display">
            <label>Exits:</label>
            <span>{data.exitEvents?.length || 0}</span>
          </div>
          <div className="node-field-display">
            <label>Tracked:</label>
            <span>{data.trackedEvents?.length || 0} events</span>
          </div>
        </div>
      </div>
      <Handle type="source" position={Position.Right} />
    </div>
  );
}
