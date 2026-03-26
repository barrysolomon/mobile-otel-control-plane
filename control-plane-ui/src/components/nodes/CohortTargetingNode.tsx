import { Handle, Position, type NodeProps } from 'reactflow';

const cohortIcons: Record<string, string> = {
  cohort_static: '\u{1F4CB}',
  cohort_dynamic: '\u{1F504}',
  cohort_discovered: '\u{1F916}',
};

const cohortLabels: Record<string, string> = {
  cohort_static: 'Static Cohort',
  cohort_dynamic: 'Dynamic Cohort',
  cohort_discovered: 'Discovered Cohort',
};

export function CohortTargetingNode({ data, type }: NodeProps) {
  const icon = cohortIcons[type!] || '\u{1F465}';
  const label = cohortLabels[type!] || 'Cohort';
  const excludeKeys = ['label', 'onChange'];
  const fields = Object.entries(data).filter(([k]) => !excludeKeys.includes(k));

  return (
    <div className="node node-cohort" style={{ borderColor: '#8b5cf6', borderWidth: 2 }}>
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
      <Handle type="source" position={Position.Right} />
    </div>
  );
}
