import axios from 'axios';
import type { ConfigVersion, DSLConfig, DSLConfigV2, WorkflowGraph } from '../types/workflow';

const api = axios.create({
  baseURL: '/api',
  headers: {
    'Content-Type': 'application/json',
  },
});

export interface PublishRequest {
  graph_json: string;
  dsl_json: string;
  dsl_v2_json?: string;
  published_by: string;
}

export interface PublishResponse {
  status: string;
  version: number;
}

export interface RollbackRequest {
  version: number;
}

export interface VersionsResponse {
  versions: ConfigVersion[];
}

export const gatewayAPI = {
  // Get current config (for preview)
  async getConfig(appId: string, deviceId: string): Promise<DSLConfig> {
    const response = await api.get<DSLConfig>(
      `/config?app_id=${appId}&device_id=${deviceId}`
    );
    return response.data;
  },

  // Publish new workflow version (v1 DSL required, v2 optional)
  async publish(
    graphs: WorkflowGraph[],
    dslConfig: DSLConfig,
    publishedBy: string,
    dslConfigV2?: DSLConfigV2
  ): Promise<PublishResponse> {
    const request: PublishRequest = {
      graph_json: JSON.stringify(graphs),
      dsl_json: JSON.stringify(dslConfig),
      published_by: publishedBy,
    };
    if (dslConfigV2) {
      request.dsl_v2_json = JSON.stringify(dslConfigV2);
    }

    const response = await api.post<PublishResponse>('/admin/publish', request);
    return response.data;
  },

  // Rollback to previous version
  async rollback(version: number): Promise<void> {
    const request: RollbackRequest = { version };
    await api.post('/admin/rollback', request);
  },

  // List config versions
  async listVersions(limit: number = 50): Promise<ConfigVersion[]> {
    const response = await api.get<VersionsResponse>(
      `/admin/versions?limit=${limit}`
    );
    return response.data.versions;
  },

  // Health check
  async health(): Promise<{ status: string }> {
    const response = await api.get<{ status: string }>('/health');
    return response.data;
  },

  // Device management
  async registerDevice(data: {
    device_id: string;
    os_version: string;
    app_version: string;
    device_group?: string;
  }): Promise<any> {
    const response = await api.post('/v1/devices/register', data);
    return response.data;
  },

  async listDevices(params?: {
    group?: string;
    limit?: number;
    offset?: number;
  }): Promise<any> {
    const response = await api.get('/v1/devices', { params });
    return response.data;
  },

  async getDevice(deviceId: string): Promise<any> {
    const response = await api.get('/v1/devices/detail', {
      params: { device_id: deviceId }
    });
    return response.data;
  },

  async updateDeviceGroup(deviceId: string, deviceGroup: string): Promise<any> {
    const response = await api.patch(`/v1/devices/group?device_id=${deviceId}`, {
      device_group: deviceGroup
    });
    return response.data;
  },

  async listDeviceGroups(): Promise<any> {
    const response = await api.get('/v1/device-groups');
    return response.data;
  },

  async getHeartbeats(limit: number = 100): Promise<any> {
    const response = await api.get('/v1/heartbeats', {
      params: { limit }
    });
    return response.data;
  },

  // OTEL Configuration management
  async createOTELConfig(data: {
    device_group: string;
    protocol: string;
    collector_endpoint: string;
    auth_token?: string;
    dataset?: string;
    ram_buffer_size?: number;
    disk_buffer_mb?: number;
    disk_buffer_ttl_hours?: number;
    export_timeout_seconds?: number;
    max_export_retries?: number;
    environment_vars?: Record<string, string>;
    feature_flags?: Record<string, boolean>;
  }): Promise<any> {
    const response = await api.post('/v1/otel-configs', data);
    return response.data;
  },

  async listOTELConfigs(deviceGroup?: string, limit?: number): Promise<any> {
    const response = await api.get('/v1/otel-configs', {
      params: { device_group: deviceGroup, limit }
    });
    return response.data;
  },

  async getActiveOTELConfig(deviceGroup: string): Promise<any> {
    const response = await api.get('/v1/otel-configs/active', {
      params: { device_group: deviceGroup }
    });
    return response.data;
  },

  async activateOTELConfig(id: number): Promise<any> {
    const response = await api.post('/v1/otel-configs/activate', null, {
      params: { id }
    });
    return response.data;
  },

  async getConfigRolloutStatus(): Promise<any> {
    const response = await api.get('/v1/otel-configs/rollout-status');
    return response.data;
  },

  // Workflow CRUD
  async createWorkflow(workflow: WorkflowGraph): Promise<any> {
    const response = await api.post('/v1/workflows', {
      id: workflow.id,
      name: workflow.name,
      enabled: workflow.enabled,
      graph_json: JSON.stringify(workflow),
    });
    return response.data;
  },

  async getWorkflow(id: string): Promise<WorkflowGraph> {
    const response = await api.get('/v1/workflows/detail', { params: { id } });
    return JSON.parse(response.data.graph_json);
  },

  async listWorkflows(): Promise<WorkflowGraph[]> {
    const response = await api.get('/v1/workflows');
    const workflows = response.data.workflows || [];
    return workflows.map((w: any) => JSON.parse(w.graph_json));
  },

  async updateWorkflow(workflow: WorkflowGraph): Promise<any> {
    const response = await api.put('/v1/workflows/detail', {
      name: workflow.name,
      enabled: workflow.enabled,
      graph_json: JSON.stringify(workflow),
    }, { params: { id: workflow.id } });
    return response.data;
  },

  async deleteWorkflow(id: string): Promise<void> {
    await api.delete('/v1/workflows/detail', { params: { id } });
  },

  // Targeting rules
  async createTargetingRule(data: {
    workflow_id: string;
    device_group: string;
    rules_json: string;
  }): Promise<any> {
    const response = await api.post('/v1/targeting-rules', data);
    return response.data;
  },

  async listTargetingRules(workflowId: string): Promise<any> {
    const response = await api.get('/v1/targeting-rules', {
      params: { workflow_id: workflowId }
    });
    return response.data;
  },

  async deleteTargetingRule(id: number): Promise<void> {
    await api.delete('/v1/targeting-rules', { params: { id } });
  },

  // Buffer configs
  async upsertBufferConfig(data: {
    device_group: string;
    ram_events: number;
    disk_mb: number;
    retention_hours: number;
    strategy: string;
  }): Promise<any> {
    const response = await api.post('/v1/buffer-configs', data);
    return response.data;
  },

  async getBufferConfig(deviceGroup: string): Promise<any> {
    const response = await api.get('/v1/buffer-configs', {
      params: { device_group: deviceGroup }
    });
    return response.data;
  },

  async listBufferConfigs(): Promise<any> {
    const response = await api.get('/v1/buffer-configs/list');
    return response.data;
  },

  // Metrics ingest (from devices)
  async ingestMetrics(data: {
    device_id: string;
    metrics: { metric_name: string; metric_type: string; value: number; labels: Record<string, string> }[];
  }): Promise<any> {
    const response = await api.post('/v1/metrics/ingest', data);
    return response.data;
  },

  // Funnel ingest (from devices)
  async ingestFunnelEvents(data: {
    device_id: string;
    session_id: string;
    events: { funnel_name: string; step_index: number; step_name: string }[];
  }): Promise<any> {
    const response = await api.post('/v1/funnels/ingest', data);
    return response.data;
  },

  // Cohort Management
  async listCohorts(): Promise<any> {
    const response = await api.get('/v1/cohorts');
    return response.data;
  },

  async createCohort(cohort: any): Promise<any> {
    const response = await api.post('/v1/cohorts', cohort);
    return response.data;
  },

  async deleteCohort(id: string): Promise<any> {
    const response = await api.delete('/v1/cohorts', { params: { id } });
    return response.data;
  },

  async getCohortMembers(id: string): Promise<any> {
    const response = await api.get('/v1/cohorts/members', { params: { id } });
    return response.data;
  },

  // Cascade Management
  async listCascades(): Promise<any> {
    const response = await api.get('/v1/cascades');
    return response.data;
  },

  async killSwitch(): Promise<any> {
    const response = await api.post('/admin/cascade/kill');
    return response.data;
  },

  async resumeCascades(): Promise<any> {
    const response = await api.post('/admin/cascade/resume');
    return response.data;
  },

  async getBreakerState(): Promise<any> {
    const response = await api.get('/admin/cascade/breaker-state');
    return response.data;
  },

  // Push Status
  async getPushStatus(): Promise<any> {
    const response = await api.get('/v1/push/status');
    return response.data;
  },

  // Workflow Audit
  async getWorkflowAudit(id: string): Promise<any> {
    const response = await api.get('/v1/workflows/audit', { params: { id } });
    return response.data;
  },
};
