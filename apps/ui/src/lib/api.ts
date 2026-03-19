import { ConsoleSettings } from './storage';

type StructLike = Record<string, unknown>;

export type WorkflowInstance = {
  id: string;
  workflowName: string;
  workflowVersion: number;
  currentState: string;
  status: string;
  revision: number;
  triggerEventId?: string;
  metadata?: StructLike;
  startedAt?: string;
  finishedAt?: string;
  createdAt?: string;
  updatedAt?: string;
  parentInstanceId?: string;
  parentExecutionId?: string;
  scopeType?: string;
  scopeParentState?: string;
  scopeEntryState?: string;
  scopeIndex?: number;
};

export type WorkflowExecution = {
  id: string;
  instanceId: string;
  state: string;
  stateVersion: number;
  attempt: number;
  status: string;
  errorClass?: string;
  errorMessage?: string;
  nextRetryAt?: string;
  startedAt?: string;
  finishedAt?: string;
  createdAt?: string;
  updatedAt?: string;
  traceId?: string;
  inputSchemaHash?: string;
  outputSchemaHash?: string;
  inputPayload?: StructLike;
  output?: StructLike;
};

export type WorkflowDefinition = {
  id: string;
  name: string;
  version: number;
  status: string;
  dsl?: StructLike;
  inputSchemaHash?: string;
  timeoutSeconds?: string;
  createdAt?: string;
  updatedAt?: string;
};

export type TimelineEntry = {
  eventType: string;
  state?: string;
  fromState?: string;
  toState?: string;
  executionId?: string;
  traceId?: string;
  payload?: StructLike;
  createdAt?: string;
};

export type StateOutput = {
  executionId: string;
  state: string;
  schemaHash?: string;
  payload?: StructLike;
  createdAt?: string;
};

export type ScopeRun = {
  id: string;
  parentExecutionId: string;
  parentState: string;
  scopeType: string;
  status: string;
  waitAll: boolean;
  totalChildren: number;
  completedChildren: number;
  failedChildren: number;
  nextChildIndex: number;
  maxConcurrency: number;
  itemVar?: string;
  indexVar?: string;
  itemsPayload?: StructLike;
  resultsPayload?: StructLike;
  createdAt?: string;
  updatedAt?: string;
};

export type SignalWait = {
  id: string;
  executionId: string;
  state: string;
  signalName: string;
  outputVar?: string;
  status: string;
  timeoutAt?: string;
  matchedAt?: string;
  timedOutAt?: string;
  messageId?: string;
  attempts: number;
  createdAt?: string;
  updatedAt?: string;
};

export type SignalMessage = {
  id: string;
  signalName: string;
  payload?: StructLike;
  status: string;
  deliveredAt?: string;
  waitId?: string;
  attempts: number;
  createdAt?: string;
  updatedAt?: string;
};

export type InstanceRun = {
  instance?: WorkflowInstance;
  latestExecution?: WorkflowExecution;
  traceId?: string;
  resumeStrategy?: string;
  executions: WorkflowExecution[];
  timeline: TimelineEntry[];
  outputs: StateOutput[];
  scopeRuns: ScopeRun[];
  signalWaits: SignalWait[];
  signalMessages: SignalMessage[];
};

export type EventRecord = {
  eventId: string;
  eventType: string;
  source: string;
  idempotencyKey?: string;
  payload?: StructLike;
};

export type EventIngestResponse = {
  event?: EventRecord;
  idempotent?: boolean;
};

type ConnectErrorShape = {
  code?: string;
  message?: string;
};

function normalizeBase(base: string) {
  if (!base) {
    return '';
  }

  return base.endsWith('/') ? base : `${base}/`;
}

function buildUrl(base: string, path: string) {
  return new URL(path.replace(/^\//, ''), normalizeBase(base)).toString();
}

async function request<T>(settings: ConsoleSettings, url: string, options?: RequestInit) {
  if (!settings.apiBaseUrl) {
    throw new Error('API base URL is not set');
  }

  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    'Connect-Protocol-Version': '1',
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
    let parsedError: ConnectErrorShape | null = null;

    try {
      parsedError = JSON.parse(text) as ConnectErrorShape;
    } catch {
      parsedError = null;
    }

    if (parsedError?.message || parsedError?.code) {
      const label = parsedError.code ? `${parsedError.code}: ` : '';
      throw new Error(`${label}${parsedError.message || `Request failed (${response.status})`}`);
    }

    throw new Error(text || `Request failed (${response.status})`);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}

async function connectUnary<TRequest, TResponse>(
  settings: ConsoleSettings,
  service: string,
  method: string,
  body: TRequest,
) {
  return request<TResponse>(settings, buildUrl(settings.apiBaseUrl, `/${service}/${method}`), {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

export async function listInstances(
  settings: ConsoleSettings,
  params: {
    limit?: number;
  } = {},
) {
  return connectUnary<{ limit?: number }, { items?: WorkflowInstance[] }>(
    settings,
    'runtime.v1.RuntimeService',
    'ListInstances',
    {
      limit: params.limit ?? 200,
    },
  );
}

export async function listExecutions(
  settings: ConsoleSettings,
  params: {
    instanceId?: string;
    limit?: number;
  } = {},
) {
  return connectUnary<{ instanceId?: string; limit?: number }, { items?: WorkflowExecution[] }>(
    settings,
    'runtime.v1.RuntimeService',
    'ListExecutions',
    {
      instanceId: params.instanceId,
      limit: params.limit ?? 200,
    },
  );
}

export async function getExecution(settings: ConsoleSettings, executionId: string) {
  const response = await connectUnary<
    { executionId: string; includeOutput: boolean },
    { execution?: WorkflowExecution }
  >(settings, 'runtime.v1.RuntimeService', 'GetExecution', {
    executionId,
    includeOutput: true,
  });

  return response.execution;
}

export async function getInstanceRun(settings: ConsoleSettings, instanceId: string) {
  return connectUnary<
    { instanceId: string; includePayloads: boolean; executionLimit: number; timelineLimit: number },
    InstanceRun
  >(settings, 'runtime.v1.RuntimeService', 'GetInstanceRun', {
    instanceId,
    includePayloads: true,
    executionLimit: 200,
    timelineLimit: 200,
  });
}

export async function listWorkflows(settings: ConsoleSettings) {
  return connectUnary<{ limit: number }, { items?: WorkflowDefinition[] }>(
    settings,
    'workflow.v1.WorkflowService',
    'ListWorkflows',
    { limit: 200 },
  );
}

export async function retryExecution(settings: ConsoleSettings, executionId: string) {
  return connectUnary<{ executionId: string }, { execution?: WorkflowExecution }>(
    settings,
    'runtime.v1.RuntimeService',
    'RetryExecution',
    { executionId },
  );
}

export async function retryInstance(settings: ConsoleSettings, instanceId: string) {
  return connectUnary<{ instanceId: string }, { execution?: WorkflowExecution }>(
    settings,
    'runtime.v1.RuntimeService',
    'RetryInstance',
    { instanceId },
  );
}

export async function resumeExecution(
  settings: ConsoleSettings,
  executionId: string,
  payload: Record<string, unknown>,
) {
  return connectUnary<
    { executionId: string; payload: Record<string, unknown> },
    { execution?: WorkflowExecution; action?: string }
  >(settings, 'runtime.v1.RuntimeService', 'ResumeExecution', {
    executionId,
    payload,
  });
}

export async function ingestEvent(
  settings: ConsoleSettings,
  payload: {
    eventType: string;
    source: string;
    idempotencyKey?: string;
    payload: Record<string, unknown>;
  },
) {
  return connectUnary<
    { eventType: string; source: string; idempotencyKey?: string; payload: Record<string, unknown> },
    EventIngestResponse
  >(settings, 'event.v1.EventService', 'IngestEvent', payload);
}

export async function getInstanceTimeline(settings: ConsoleSettings, instanceId: string) {
  const response = await connectUnary<{ instanceId: string }, { items?: TimelineEntry[] }>(
    settings,
    'event.v1.EventService',
    'GetInstanceTimeline',
    { instanceId },
  );

  return response.items ?? [];
}

export async function sendSignal(
  settings: ConsoleSettings,
  payload: {
    instanceId: string;
    signalName: string;
    payload: Record<string, unknown>;
  },
) {
  return connectUnary<
    { instanceId: string; signalName: string; payload: Record<string, unknown> },
    { delivered?: boolean }
  >(settings, 'signal.v1.SignalService', 'SendSignal', payload);
}
