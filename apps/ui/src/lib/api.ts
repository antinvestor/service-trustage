import { ConsoleSettings } from './storage';

export type InstanceItem = {
  id: string;
  workflow_name: string;
  workflow_version: number;
  current_state: string;
  status: string;
  trigger_event_id?: string;
  started_at?: string;
  finished_at?: string;
  created_at: string;
};

export type ExecutionItem = {
  execution_id: string;
  instance_id: string;
  state: string;
  attempt: number;
  status: string;
  error_class?: string;
  error_message?: string;
  next_retry_at?: string;
  started_at?: string;
  finished_at?: string;
  created_at: string;
  trace_id?: string;
};

export type ExecutionDetail = ExecutionItem & {
  input_payload?: unknown;
  output?: unknown;
};

export type WorkflowItem = {
  id: string;
  name: string;
  version: number;
  status: string;
};

export type EventIngestResponse = {
  event_id: string;
  idempotent?: boolean;
};

export type TimelineEntry = {
  event_type: string;
  state?: string;
  created_at: string;
};

function buildUrl(base: string, path: string, params?: Record<string, string | number | undefined>) {
  const url = new URL(path, base);
  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== '') {
        url.searchParams.set(key, String(value));
      }
    });
  }
  return url.toString();
}

async function request<T>(settings: ConsoleSettings, url: string, options?: RequestInit) {
  if (!settings.apiBaseUrl) {
    throw new Error('API base URL is not set');
  }

  const headers: HeadersInit = {
    'Content-Type': 'application/json',
  };

  if (settings.authToken) {
    headers.Authorization = `Bearer ${settings.authToken}`;
  }

  const response = await fetch(url, {
    ...options,
    headers: {
      ...headers,
      ...(options?.headers || {}),
    },
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || `Request failed (${response.status})`);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}

export async function listInstances(settings: ConsoleSettings, params: {
  status?: string;
  workflowName?: string;
  limit?: number;
  cursor?: string;
}) {
  return request<{ items: InstanceItem[]; next_cursor?: string }>(
    settings,
    buildUrl(settings.apiBaseUrl, '/api/v1/instances', {
      status: params.status,
      workflow_name: params.workflowName,
      limit: params.limit,
      cursor: params.cursor,
    }),
    { method: 'GET' },
  );
}

export async function listExecutions(settings: ConsoleSettings, params: {
  status?: string;
  instanceId?: string;
  limit?: number;
  cursor?: string;
}) {
  return request<{ items: ExecutionItem[]; next_cursor?: string }>(
    settings,
    buildUrl(settings.apiBaseUrl, '/api/v1/executions', {
      status: params.status,
      instance_id: params.instanceId,
      limit: params.limit,
      cursor: params.cursor,
    }),
    { method: 'GET' },
  );
}

export async function getExecution(settings: ConsoleSettings, executionId: string) {
  return request<ExecutionDetail>(
    settings,
    buildUrl(settings.apiBaseUrl, `/api/v1/executions/${executionId}`, {
      include_output: 'true',
    }),
    { method: 'GET' },
  );
}

export async function listWorkflows(settings: ConsoleSettings) {
  return request<{ items: WorkflowItem[] }>(
    settings,
    buildUrl(settings.apiBaseUrl, '/api/v1/workflows', { status: 'active', limit: 100 }),
    { method: 'GET' },
  );
}

export async function retryExecution(settings: ConsoleSettings, executionId: string) {
  return request<{ execution_id: string; status: string }>(
    settings,
    buildUrl(settings.apiBaseUrl, `/api/v1/executions/${executionId}/retry`),
    { method: 'POST' },
  );
}

export async function retryInstance(settings: ConsoleSettings, instanceId: string) {
  return request<{ execution_id: string; status: string }>(
    settings,
    buildUrl(settings.apiBaseUrl, `/api/v1/instances/${instanceId}/retry`),
    { method: 'POST' },
  );
}

export async function ingestEvent(settings: ConsoleSettings, payload: {
  event_type: string;
  source: string;
  idempotency_key?: string;
  payload: Record<string, unknown>;
}): Promise<EventIngestResponse> {
  return request<EventIngestResponse>(settings, buildUrl(settings.apiBaseUrl, '/api/v1/events'), {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function getInstanceTimeline(settings: ConsoleSettings, instanceId: string) {
  return request<TimelineEntry[]>(
    settings,
    buildUrl(settings.apiBaseUrl, `/api/v1/instances/${instanceId}/timeline`),
    { method: 'GET' },
  );
}
