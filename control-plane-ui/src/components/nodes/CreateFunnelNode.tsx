import { Handle, Position } from 'reactflow';
import type { Predicate } from '../../types/workflow';

interface CreateFunnelNodeProps {
  data: {
    funnelName: string;
    steps: { eventName: string; predicates?: Predicate[] }[];
  };
}

export function CreateFunnelNodeComponent({ data }: CreateFunnelNodeProps) {
  return (
    <div className="node node-action">
      <Handle type="target" position={Position.Left} />
      <div className="node-header">
        <div className="node-icon">🔽</div>
        <div className="node-title">Create Funnel</div>
      </div>
      <div className="node-body">
        <div className="node-fields">
          <div className="node-field-display">
            <label>Funnel:</label>
            <span>{data.funnelName || '(not set)'}</span>
          </div>
          <div className="node-field-display">
            <label>Steps:</label>
            <span>{data.steps?.length || 0} steps</span>
          </div>
          {data.steps?.map((step, i) => (
            <div key={i} className="node-field-display" style={{ paddingLeft: 8 }}>
              <label>{i + 1}.</label>
              <span>{step.eventName || '(empty)'}</span>
            </div>
          ))}
        </div>
      </div>
      <Handle type="source" position={Position.Right} />
    </div>
  );
}
