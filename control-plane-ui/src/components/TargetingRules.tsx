import React from 'react';

interface TargetingRule {
  platform?: 'android' | 'ios';
  appVersionRange?: string;
  osVersionRange?: string;
  deviceModels?: string;
  deviceGroup?: string;
  customAttributes?: Record<string, string>;
}

interface TargetingRulesProps {
  rules: TargetingRule[];
  onChange: (rules: TargetingRule[]) => void;
}

export const TargetingRules: React.FC<TargetingRulesProps> = ({ rules, onChange }) => {
  const addRule = () => {
    onChange([...rules, { customAttributes: {} }]);
  };

  const removeRule = (index: number) => {
    const updated = rules.filter((_, i) => i !== index);
    onChange(updated);
  };

  const updateRule = (index: number, field: keyof TargetingRule, value: unknown) => {
    const updated = rules.map((rule, i) => {
      if (i !== index) return rule;
      return { ...rule, [field]: value };
    });
    onChange(updated);
  };

  const addCustomAttribute = (ruleIndex: number) => {
    const key = prompt('Attribute name:');
    if (key && key.trim()) {
      const current = rules[ruleIndex].customAttributes || {};
      updateRule(ruleIndex, 'customAttributes', { ...current, [key.trim()]: '' });
    }
  };

  const removeCustomAttribute = (ruleIndex: number, key: string) => {
    const current = { ...(rules[ruleIndex].customAttributes || {}) };
    delete current[key];
    updateRule(ruleIndex, 'customAttributes', current);
  };

  const updateCustomAttribute = (ruleIndex: number, key: string, value: string) => {
    const current = { ...(rules[ruleIndex].customAttributes || {}) };
    current[key] = value;
    updateRule(ruleIndex, 'customAttributes', current);
  };

  return (
    <div className="targeting-rules">
      <div className="targeting-rules-header">
        <h3>Device Targeting Rules</h3>
        <button className="btn-add" onClick={addRule}>
          + Add Rule
        </button>
      </div>

      {rules.length === 0 && (
        <p className="empty-state">
          No targeting rules defined. All devices will receive this workflow.
        </p>
      )}

      {rules.map((rule, index) => (
        <div key={index} className="targeting-rule">
          <div className="targeting-rule-header">
            <span className="targeting-rule-label">Rule {index + 1}</span>
            <button className="btn-remove" onClick={() => removeRule(index)}>
              Remove
            </button>
          </div>

          <div className="rule-fields">
            <div className="rule-field">
              <label className="form-label">Platform</label>
              <select
                className="form-control"
                value={rule.platform || ''}
                onChange={(e) =>
                  updateRule(
                    index,
                    'platform',
                    e.target.value === '' ? undefined : (e.target.value as 'android' | 'ios')
                  )
                }
              >
                <option value="">Any</option>
                <option value="android">Android</option>
                <option value="ios">iOS</option>
              </select>
            </div>

            <div className="rule-field">
              <label className="form-label">App Version Range</label>
              <input
                type="text"
                className="form-control"
                value={rule.appVersionRange || ''}
                onChange={(e) => updateRule(index, 'appVersionRange', e.target.value || undefined)}
                placeholder="e.g. >=2.0.0"
              />
            </div>

            <div className="rule-field">
              <label className="form-label">OS Version Range</label>
              <input
                type="text"
                className="form-control"
                value={rule.osVersionRange || ''}
                onChange={(e) => updateRule(index, 'osVersionRange', e.target.value || undefined)}
                placeholder="e.g. >=12.0"
              />
            </div>

            <div className="rule-field">
              <label className="form-label">Device Models</label>
              <input
                type="text"
                className="form-control"
                value={rule.deviceModels || ''}
                onChange={(e) => updateRule(index, 'deviceModels', e.target.value || undefined)}
                placeholder="Comma-separated, e.g. Pixel 7, SM-S908B"
              />
            </div>

            <div className="rule-field">
              <label className="form-label">Device Group</label>
              <input
                type="text"
                className="form-control"
                value={rule.deviceGroup || ''}
                onChange={(e) => updateRule(index, 'deviceGroup', e.target.value || undefined)}
                placeholder="e.g. beta-testers"
              />
            </div>
          </div>

          <div className="rule-field rule-field-attributes">
            <div className="rule-field-attributes-header">
              <label className="form-label">Custom Attributes</label>
              <button
                className="btn-add btn-add-sm"
                onClick={() => addCustomAttribute(index)}
              >
                + Add Attribute
              </button>
            </div>
            {rule.customAttributes && Object.keys(rule.customAttributes).length > 0 ? (
              <div className="key-value-list">
                {Object.entries(rule.customAttributes).map(([key, value]) => (
                  <div key={key} className="key-value-row">
                    <input
                      type="text"
                      className="form-control-sm"
                      value={key}
                      readOnly
                    />
                    <input
                      type="text"
                      className="form-control-sm"
                      value={value}
                      onChange={(e) => updateCustomAttribute(index, key, e.target.value)}
                      placeholder="value"
                    />
                    <button
                      className="btn-remove"
                      onClick={() => removeCustomAttribute(index, key)}
                    >
                      x
                    </button>
                  </div>
                ))}
              </div>
            ) : (
              <p className="form-hint">No custom attributes defined.</p>
            )}
          </div>
        </div>
      ))}
    </div>
  );
};

export type { TargetingRule };
