import React from 'react';

interface BufferConfigData {
  ramEvents: number;
  diskMb: number;
  retentionHours: number;
  strategy: 'overwrite_oldest' | 'stop_recording';
}

interface BufferConfigProps {
  config: BufferConfigData;
  onChange: (config: BufferConfigData) => void;
}

export const BufferConfig: React.FC<BufferConfigProps> = ({ config, onChange }) => {
  const updateField = <K extends keyof BufferConfigData>(
    field: K,
    value: BufferConfigData[K]
  ) => {
    onChange({ ...config, [field]: value });
  };

  return (
    <div className="buffer-config">
      <h3>Ring Buffer Settings</h3>

      <div className="buffer-fields">
        <div className="buffer-field">
          <label className="form-label">RAM Events</label>
          <input
            type="number"
            className="form-control"
            value={config.ramEvents}
            min={0}
            onChange={(e) => updateField('ramEvents', parseInt(e.target.value, 10) || 0)}
          />
          <p className="form-hint">Maximum number of events to buffer in memory.</p>
        </div>

        <div className="buffer-field">
          <label className="form-label">Disk MB</label>
          <input
            type="number"
            className="form-control"
            value={config.diskMb}
            min={0}
            onChange={(e) => updateField('diskMb', parseInt(e.target.value, 10) || 0)}
          />
          <p className="form-hint">Maximum disk space for buffered events in megabytes.</p>
        </div>

        <div className="buffer-field">
          <label className="form-label">Retention Hours</label>
          <input
            type="number"
            className="form-control"
            value={config.retentionHours}
            min={0}
            onChange={(e) => updateField('retentionHours', parseInt(e.target.value, 10) || 0)}
          />
          <p className="form-hint">How long buffered events are retained before expiring.</p>
        </div>

        <div className="buffer-field">
          <label className="form-label">Strategy</label>
          <select
            className="form-control"
            value={config.strategy}
            onChange={(e) =>
              updateField(
                'strategy',
                e.target.value as 'overwrite_oldest' | 'stop_recording'
              )
            }
          >
            <option value="overwrite_oldest">Overwrite Oldest</option>
            <option value="stop_recording">Stop Recording</option>
          </select>
          <p className="form-hint">
            Behavior when the buffer is full. "Overwrite Oldest" drops the oldest events
            to make room; "Stop Recording" halts collection until space is freed.
          </p>
        </div>
      </div>
    </div>
  );
};

export type { BufferConfigData };
