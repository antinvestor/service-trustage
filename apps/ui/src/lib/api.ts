import { ConsoleSettings } from './storage';

type StructLike = Record<string, unknown>;
type RequestOptions = {
  signal?: AbortSignal;
  timeoutMs?: number;
};

export type PageResult<T> = {
  items: T[];
  nextCursor?: string;
};

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

type SearchRequest = {
  query?: string;
  idQuery?: string;
  cursor?: {
    limit?: number;
    page?: string;
  };
  extras?: StructLike;
};

function isAbortError(error: unknown) {
  return error instanceof DOMException && error.name === 'AbortError';
}

function normalizeBase(base: string) {
  if (!base) {
    return '';
  }

  return base.endsWith('/') ? base : `${base}/`;
}

function buildUrl(base: string, path: string) {
  return new URL(path.replace(/^\//, ''), normalizeBase(base)).toString();
}

function combineSignals(input?: AbortSignal, timeoutMs = 15000) {
  const controller = new AbortController();
  const timeout = window.setTimeout(() => controller.abort(), timeoutMs);

  if (input) {
    if (input.aborted) {
      controller.abort(input.reason);
    } else {
      input.addEventListener('abort', () => controller.abort(input.reason), { once: true });
    }
  }

  return {
    signal: controller.signal,
    dispose() {
      window.clearTimeout(timeout);
    },
  };
}

function buildSearch(params: {
  limit?: number;
  cursor?: string;
  query?: string;
  idQuery?: string;
  extras?: StructLike;
}): SearchRequest | undefined {
  const limit = params.limit ?? 50;
  const cursor = params.cursor?.trim();
  const query = params.query?.trim();
  const idQuery = params.idQuery?.trim();
  const extras = params.extras && Object.keys(params.extras).length > 0 ? params.extras : undefined;

  if (!query && !idQuery && !cursor && !extras && limit <= 0) {
    return undefined;
  }

  return {
    ...(query ? { query } : {}),
    ...(idQuery ? { idQuery } : {}),
    cursor: {
      limit,
      ...(cursor ? { page: cursor } : {}),
    },
    ...(extras ? { extras } : {}),
  };
}

async function request<T>(
  settings: ConsoleSettings,
  url: string,
  options?: RequestInit,
  requestOptions?: RequestOptions,
) {
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

  const transport = combineSignals(requestOptions?.signal, requestOptions?.timeoutMs);

  let response: Response;
  try {
    response = await fetch(url, {
      ...options,
      signal: transport.signal,
      headers: {
        ...headers,
        ...(options?.headers || {}),
      },
    });
  } catch (error) {
    transport.dispose();
    if (isAbortError(error)) {
      throw error;
    }
    throw new Error(error instanceof Error ? error.message : 'Network request failed');
  }
  transport.dispose();

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
  requestOptions?: RequestOptions,
) {
  return request<TResponse>(settings, buildUrl(settings.apiBaseUrl, `/${service}/${method}`), {
    method: 'POST',
    body: JSON.stringify(body),
  }, requestOptions);
}

export async function listInstances(
  settings: ConsoleSettings,
  params: {
    workflowName?: string;
    status?: string;
    query?: string;
    idQuery?: string;
    cursor?: string;
    limit?: number;
    parentInstanceId?: string;
    parentExecutionId?: string;
  } = {},
  requestOptions?: RequestOptions,
) {
  const response = await connectUnary<
    {
      workflowName?: string;
      status?: string;
      search?: SearchRequest;
    },
    { items?: WorkflowInstance[]; nextCursor?: { page?: string } }
  >(
    settings,
    'runtime.v1.RuntimeService',
    'ListInstances',
    {
      workflowName: params.workflowName,
      status: params.status,
      search: buildSearch({
        limit: params.limit ?? 50,
        cursor: params.cursor,
        query: params.query,
        idQuery: params.idQuery,
        extras: {
          ...(params.parentInstanceId ? { parent_instance_id: params.parentInstanceId } : {}),
          ...(params.parentExecutionId ? { parent_execution_id: params.parentExecutionId } : {}),
        },
      }),
    },
    requestOptions,
  );

  return {
    items: response.items ?? [],
    nextCursor: response.nextCursor?.page,
  } satisfies PageResult<WorkflowInstance>;
}

export async function listExecutions(
  settings: ConsoleSettings,
  params: {
    instanceId?: string;
    status?: string;
    query?: string;
    idQuery?: string;
    cursor?: string;
    limit?: number;
  } = {},
  requestOptions?: RequestOptions,
) {
  const response = await connectUnary<
    {
      instanceId?: string;
      status?: string;
      search?: SearchRequest;
    },
    { items?: WorkflowExecution[]; nextCursor?: { page?: string } }
  >(
    settings,
    'runtime.v1.RuntimeService',
    'ListExecutions',
    {
      instanceId: params.instanceId,
      status: params.status,
      search: buildSearch({
        limit: params.limit ?? 50,
        cursor: params.cursor,
        query: params.query,
        idQuery: params.idQuery,
      }),
    },
    requestOptions,
  );

  return {
    items: response.items ?? [],
    nextCursor: response.nextCursor?.page,
  } satisfies PageResult<WorkflowExecution>;
}

export async function getExecution(
  settings: ConsoleSettings,
  executionId: string,
  requestOptions?: RequestOptions,
) {
  const response = await connectUnary<
    { executionId: string; includeOutput: boolean },
    { execution?: WorkflowExecution }
  >(settings, 'runtime.v1.RuntimeService', 'GetExecution', {
    executionId,
    includeOutput: true,
  }, requestOptions);

  return response.execution;
}

export async function getInstanceRun(
  settings: ConsoleSettings,
  instanceId: string,
  requestOptions?: RequestOptions,
) {
  return connectUnary<
    { instanceId: string; includePayloads: boolean; executionLimit: number; timelineLimit: number },
    InstanceRun
  >(settings, 'runtime.v1.RuntimeService', 'GetInstanceRun', {
    instanceId,
    includePayloads: true,
    executionLimit: 150,
    timelineLimit: 150,
  }, requestOptions);
}

export async function listWorkflows(
  settings: ConsoleSettings,
  params: {
    name?: string;
    status?: string;
    query?: string;
    idQuery?: string;
    cursor?: string;
    limit?: number;
  } = {},
  requestOptions?: RequestOptions,
) {
  const response = await connectUnary<
    {
      name?: string;
      status?: string;
      search?: SearchRequest;
    },
    { items?: WorkflowDefinition[]; nextCursor?: { page?: string } }
  >(
    settings,
    'workflow.v1.WorkflowService',
    'ListWorkflows',
    {
      name: params.name,
      status: params.status,
      search: buildSearch({
        limit: params.limit ?? 50,
        cursor: params.cursor,
        query: params.query,
        idQuery: params.idQuery,
      }),
    },
    requestOptions,
  );

  return {
    items: response.items ?? [],
    nextCursor: response.nextCursor?.page,
  } satisfies PageResult<WorkflowDefinition>;
}

export async function retryExecution(
  settings: ConsoleSettings,
  executionId: string,
  requestOptions?: RequestOptions,
) {
  return connectUnary<{ executionId: string }, { execution?: WorkflowExecution }>(
    settings,
    'runtime.v1.RuntimeService',
    'RetryExecution',
    { executionId },
    requestOptions,
  );
}

export async function retryInstance(
  settings: ConsoleSettings,
  instanceId: string,
  requestOptions?: RequestOptions,
) {
  return connectUnary<{ instanceId: string }, { execution?: WorkflowExecution }>(
    settings,
    'runtime.v1.RuntimeService',
    'RetryInstance',
    { instanceId },
    requestOptions,
  );
}

export async function resumeExecution(
  settings: ConsoleSettings,
  executionId: string,
  payload: Record<string, unknown>,
  requestOptions?: RequestOptions,
) {
  return connectUnary<
    { executionId: string; payload: Record<string, unknown> },
    { execution?: WorkflowExecution; action?: string }
  >(settings, 'runtime.v1.RuntimeService', 'ResumeExecution', {
    executionId,
    payload,
  }, requestOptions);
}

export async function ingestEvent(
  settings: ConsoleSettings,
  payload: {
    eventType: string;
    source: string;
    idempotencyKey?: string;
    payload: Record<string, unknown>;
  },
  requestOptions?: RequestOptions,
) {
  return connectUnary<
    { eventType: string; source: string; idempotencyKey?: string; payload: Record<string, unknown> },
    EventIngestResponse
  >(settings, 'event.v1.EventService', 'IngestEvent', payload, requestOptions);
}

export async function getInstanceTimeline(
  settings: ConsoleSettings,
  instanceId: string,
  requestOptions?: RequestOptions,
) {
  const response = await connectUnary<{ instanceId: string }, { items?: TimelineEntry[] }>(
    settings,
    'event.v1.EventService',
    'GetInstanceTimeline',
    { instanceId },
    requestOptions,
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
  requestOptions?: RequestOptions,
) {
  return connectUnary<
    { instanceId: string; signalName: string; payload: Record<string, unknown> },
    { delivered?: boolean }
  >(settings, 'signal.v1.SignalService', 'SendSignal', payload, requestOptions);
}
