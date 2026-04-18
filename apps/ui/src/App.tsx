// Copyright 2023-2026 Ant Investor Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import type { ReactNode } from 'react';
import { startTransition, useDeferredValue, useEffect, useRef, useState } from 'react';
import {
  InstanceRun,
  PageResult,
  SignalMessage,
  SignalWait,
  StateOutput,
  WorkflowDefinition,
  WorkflowExecution,
  WorkflowInstance,
  getExecution,
  getInstanceRun,
  ingestEvent,
  listExecutions,
  listInstances,
  listWorkflows,
  retryExecution,
  retryInstance,
  sendSignal,
} from './lib/api';
import { ConsoleSettings, loadSettings, saveSettings } from './lib/storage';

const navItems = [
  { key: 'overview', label: 'Command Deck' },
  { key: 'runs', label: 'Run Explorer' },
  { key: 'executions', label: 'Execution Queue' },
  { key: 'workflows', label: 'Workflow Catalog' },
] as const;

type NavKey = (typeof navItems)[number]['key'];
type ResourceState<T> = {
  items: T[];
  nextCursor: string;
  loading: boolean;
  loadingMore: boolean;
};

const DEFAULT_PAGE_SIZE = 50;
const CHILD_PAGE_SIZE = 100;
const MAX_CHILD_PAGES = 20;

function formatDate(value?: string) {
  if (!value) {
    return '—';
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleString();
}

function shortId(value?: string, size = 8) {
  if (!value) {
    return '—';
  }

  return value.length <= size ? value : value.slice(0, size);
}

function humanizeToken(value?: string) {
  if (!value) {
    return 'unknown';
  }

  return value
    .replace(/^.*STATUS_/, '')
    .replace(/^WORKFLOW_STATUS_/, '')
    .replace(/^INSTANCE_STATUS_/, '')
    .replace(/^EXECUTION_STATUS_/, '')
    .replace(/_/g, ' ')
    .toLowerCase();
}

function statusTone(status?: string) {
  const normalized = humanizeToken(status);

  if (normalized.includes('fatal') || normalized.includes('failed') || normalized.includes('invalid')) {
    return 'bg-rose-100 text-rose-700 border-rose-200';
  }
  if (normalized.includes('waiting') || normalized.includes('suspended')) {
    return 'bg-amber-100 text-amber-800 border-amber-200';
  }
  if (normalized.includes('running') || normalized.includes('dispatched') || normalized.includes('pending')) {
    return 'bg-sky-100 text-sky-700 border-sky-200';
  }
  if (normalized.includes('retry')) {
    return 'bg-orange-100 text-orange-700 border-orange-200';
  }
  if (normalized.includes('completed') || normalized.includes('active')) {
    return 'bg-emerald-100 text-emerald-700 border-emerald-200';
  }

  return 'bg-stone-100 text-stone-600 border-stone-200';
}

function createResourceState<T>(): ResourceState<T> {
  return {
    items: [],
    nextCursor: '',
    loading: false,
    loadingMore: false,
  };
}

function sortByNewest<T extends { createdAt?: string }>(items: T[]) {
  return [...items].sort((left, right) => {
    const a = new Date(left.createdAt || '').getTime() || 0;
    const b = new Date(right.createdAt || '').getTime() || 0;
    return b - a;
  });
}

function outputForExecution(outputs: StateOutput[], executionId?: string) {
  if (!executionId) {
    return undefined;
  }

  return outputs.find((item) => item.executionId === executionId);
}

function canRetry(status?: string) {
  const normalized = humanizeToken(status);
  return (
    normalized.includes('failed') ||
    normalized.includes('fatal') ||
    normalized.includes('timed out') ||
    normalized.includes('invalid') ||
    normalized.includes('retry')
  );
}

function isWaiting(status?: string) {
  return humanizeToken(status).includes('waiting');
}

function isAbortError(error: unknown) {
  return error instanceof DOMException && error.name === 'AbortError';
}

function Panel({
  eyebrow,
  title,
  subtitle,
  actions,
  children,
  className = '',
}: {
  eyebrow?: string;
  title: string;
  subtitle?: string;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  return (
    <section className={`rounded-[28px] border border-stone-200/80 bg-white/90 p-5 shadow-[0_20px_60px_rgba(40,34,24,0.08)] backdrop-blur ${className}`}>
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          {eyebrow && (
            <div className="text-[10px] font-semibold uppercase tracking-[0.35em] text-stone-400">
              {eyebrow}
            </div>
          )}
          <h2 className="mt-1 font-logo text-xl font-semibold text-stone-900">{title}</h2>
          {subtitle && <p className="mt-1 text-sm text-stone-500">{subtitle}</p>}
        </div>
        {actions}
      </div>
      <div className="mt-5">{children}</div>
    </section>
  );
}

function MetricCard({
  label,
  value,
  tone,
  detail,
}: {
  label: string;
  value: string | number;
  tone: string;
  detail?: string;
}) {
  return (
    <div className={`rounded-[24px] border px-4 py-4 ${tone}`}>
      <div className="text-[10px] font-semibold uppercase tracking-[0.3em]">{label}</div>
      <div className="mt-3 text-3xl font-semibold">{value}</div>
      {detail && <div className="mt-2 text-xs opacity-80">{detail}</div>}
    </div>
  );
}

function StatusBadge({ value }: { value?: string }) {
  return (
    <span className={`inline-flex w-fit rounded-full border px-2.5 py-1 text-[11px] font-semibold capitalize ${statusTone(value)}`}>
      {humanizeToken(value)}
    </span>
  );
}

function JsonBlock({
  label,
  value,
  compact = false,
}: {
  label: string;
  value: unknown;
  compact?: boolean;
}) {
  return (
    <div className="rounded-2xl border border-stone-200 bg-stone-950/95 px-4 py-3 text-stone-100">
      <div className="text-[10px] font-semibold uppercase tracking-[0.3em] text-stone-400">{label}</div>
      <pre className={`mt-3 overflow-auto font-mono text-[11px] leading-5 text-stone-200 ${compact ? 'max-h-28' : 'max-h-72'}`}>
        {JSON.stringify(value ?? {}, null, 2)}
      </pre>
    </div>
  );
}

function EmptyState({
  title,
  description,
}: {
  title: string;
  description: string;
}) {
  return (
    <div className="rounded-[24px] border border-dashed border-stone-300 bg-stone-50 px-5 py-8 text-center">
      <div className="font-logo text-lg font-semibold text-stone-800">{title}</div>
      <p className="mt-2 text-sm text-stone-500">{description}</p>
    </div>
  );
}

function ExecutionGraph({
  run,
  childInstances,
  selectedExecutionId,
  onSelectExecution,
  onSelectInstance,
}: {
  run: InstanceRun;
  childInstances: WorkflowInstance[];
  selectedExecutionId?: string;
  onSelectExecution: (execution: WorkflowExecution) => void;
  onSelectInstance: (instance: WorkflowInstance) => void;
}) {
  const executions = sortByNewest(run.executions).reverse();

  return (
    <div className="space-y-5">
      <div className="rounded-[24px] border border-stone-200 bg-stone-50 px-4 py-4">
        <div className="flex flex-wrap items-center gap-3">
          <StatusBadge value={run.instance?.status} />
          <div className="font-semibold text-stone-900">{run.instance?.workflowName}</div>
          <div className="text-xs text-stone-500">instance {run.instance?.id}</div>
          <div className="text-xs text-stone-500">trace {run.traceId || run.latestExecution?.traceId || '—'}</div>
        </div>
        <div className="mt-3 grid gap-3 text-sm text-stone-600 md:grid-cols-4">
          <div>
            <div className="text-[10px] uppercase tracking-[0.25em] text-stone-400">Current state</div>
            <div className="mt-1 font-medium text-stone-900">{run.instance?.currentState || '—'}</div>
          </div>
          <div>
            <div className="text-[10px] uppercase tracking-[0.25em] text-stone-400">Resume strategy</div>
            <div className="mt-1 font-medium text-stone-900">{humanizeToken(run.resumeStrategy)}</div>
          </div>
          <div>
            <div className="text-[10px] uppercase tracking-[0.25em] text-stone-400">Started</div>
            <div className="mt-1 font-medium text-stone-900">{formatDate(run.instance?.startedAt)}</div>
          </div>
          <div>
            <div className="text-[10px] uppercase tracking-[0.25em] text-stone-400">Latest execution</div>
            <div className="mt-1 font-medium text-stone-900">{run.latestExecution?.state || '—'}</div>
          </div>
        </div>
      </div>

      <div className="relative space-y-5 before:absolute before:bottom-0 before:left-[17px] before:top-2 before:w-px before:bg-stone-200">
        {executions.map((execution) => {
          const executionOutput = outputForExecution(run.outputs, execution.id);
          const scopeRuns = run.scopeRuns.filter((item) => item.parentExecutionId === execution.id);
          const waits = run.signalWaits.filter((item) => item.executionId === execution.id);
          const relatedMessages = run.signalMessages.filter((message) =>
            waits.some((wait) => wait.messageId && wait.messageId === message.id),
          );
          const spawnedChildren = childInstances.filter((item) => item.parentExecutionId === execution.id);

          return (
            <div key={execution.id} className="relative pl-10">
              <div className="absolute left-[10px] top-6 h-3.5 w-3.5 rounded-full border-4 border-[var(--canvas)] bg-[var(--ink)]" />
              <button
                className={`w-full rounded-[24px] border px-4 py-4 text-left transition ${
                  selectedExecutionId === execution.id
                    ? 'border-[var(--accent-strong)] bg-[var(--accent-soft)] shadow-[0_12px_30px_rgba(180,82,46,0.18)]'
                    : 'border-stone-200 bg-white hover:border-stone-300 hover:bg-stone-50'
                }`}
                onClick={() => onSelectExecution(execution)}
              >
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <div className="flex flex-wrap items-center gap-2">
                      <StatusBadge value={execution.status} />
                      <div className="font-semibold text-stone-900">{execution.state}</div>
                      <div className="text-xs text-stone-500">attempt #{execution.attempt}</div>
                    </div>
                    <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-stone-500">
                      <span>execution {execution.id}</span>
                      <span>trace {execution.traceId || '—'}</span>
                      <span>started {formatDate(execution.startedAt || execution.createdAt)}</span>
                    </div>
                    {execution.errorMessage && (
                      <div className="mt-3 rounded-2xl border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-700">
                        {execution.errorMessage}
                      </div>
                    )}
                  </div>
                  <div className="min-w-[140px] text-right text-xs text-stone-500">
                    <div>input schema {shortId(execution.inputSchemaHash)}</div>
                    <div className="mt-1">output schema {shortId(execution.outputSchemaHash)}</div>
                  </div>
                </div>
              </button>

              {(executionOutput || scopeRuns.length > 0 || waits.length > 0 || spawnedChildren.length > 0 || relatedMessages.length > 0) && (
                <div className="mt-3 space-y-3 pl-5">
                  {executionOutput && (
                    <div className="rounded-2xl border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-800">
                      <div className="text-[10px] font-semibold uppercase tracking-[0.3em] text-emerald-600">
                        Output snapshot
                      </div>
                      <div className="mt-2 font-mono text-[11px] leading-5">
                        {JSON.stringify(executionOutput.payload ?? {}, null, 2)}
                      </div>
                    </div>
                  )}

                  {scopeRuns.map((scope) => (
                    <div key={scope.id} className="rounded-2xl border border-sky-200 bg-sky-50 px-4 py-3">
                      <div className="flex flex-wrap items-center justify-between gap-2">
                        <div className="flex items-center gap-2">
                          <StatusBadge value={scope.status} />
                          <div className="text-sm font-semibold text-sky-900">
                            {scope.scopeType} scope from {scope.parentState}
                          </div>
                        </div>
                        <div className="text-xs text-sky-700">
                          {scope.completedChildren}/{scope.totalChildren} completed
                        </div>
                      </div>
                      <div className="mt-3 grid gap-2 text-xs text-sky-800 md:grid-cols-4">
                        <div>wait all: {scope.waitAll ? 'yes' : 'no'}</div>
                        <div>failed: {scope.failedChildren}</div>
                        <div>next child: {scope.nextChildIndex}</div>
                        <div>max concurrency: {scope.maxConcurrency || 'auto'}</div>
                      </div>
                    </div>
                  ))}

                  {waits.map((wait) => (
                    <div key={wait.id} className="rounded-2xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900">
                      <div className="flex flex-wrap items-center justify-between gap-2">
                        <div className="flex items-center gap-2">
                          <StatusBadge value={wait.status} />
                          <div className="font-semibold">{wait.signalName}</div>
                        </div>
                        <div className="text-xs text-amber-700">
                          timeout {formatDate(wait.timeoutAt)}
                        </div>
                      </div>
                      <div className="mt-2 text-xs text-amber-800">
                        output var {wait.outputVar || 'direct payload'} • attempts {wait.attempts}
                      </div>
                    </div>
                  ))}

                  {relatedMessages.map((message) => (
                    <div key={message.id} className="rounded-2xl border border-violet-200 bg-violet-50 px-4 py-3 text-sm text-violet-900">
                      <div className="flex flex-wrap items-center justify-between gap-2">
                        <div className="flex items-center gap-2">
                          <StatusBadge value={message.status} />
                          <div className="font-semibold">{message.signalName}</div>
                        </div>
                        <div className="text-xs text-violet-700">delivered {formatDate(message.deliveredAt)}</div>
                      </div>
                    </div>
                  ))}

                  {spawnedChildren.length > 0 && (
                    <div className="rounded-2xl border border-stone-200 bg-stone-50 px-4 py-3">
                      <div className="text-[10px] font-semibold uppercase tracking-[0.3em] text-stone-400">
                        Spawned child runs
                      </div>
                      <div className="mt-3 grid gap-3">
                        {spawnedChildren.map((child) => (
                          <button
                            key={child.id}
                            className="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-stone-200 bg-white px-3 py-3 text-left hover:border-stone-300"
                            onClick={() => onSelectInstance(child)}
                          >
                            <div>
                              <div className="font-semibold text-stone-900">{child.workflowName}</div>
                              <div className="text-xs text-stone-500">
                                {child.id} • {child.currentState}
                              </div>
                            </div>
                            <StatusBadge value={child.status} />
                          </button>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}

export default function App() {
  const [activeView, setActiveView] = useState<NavKey>('overview');
  const [settings, setSettings] = useState<ConsoleSettings>(() => loadSettings());
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [statusMessage, setStatusMessage] = useState('');
  const [lastSyncedAt, setLastSyncedAt] = useState('');

  const [instanceState, setInstanceState] = useState<ResourceState<WorkflowInstance>>(() => createResourceState());
  const [executionState, setExecutionState] = useState<ResourceState<WorkflowExecution>>(() => createResourceState());
  const [workflowState, setWorkflowState] = useState<ResourceState<WorkflowDefinition>>(() => createResourceState());
  const [childInstanceState, setChildInstanceState] = useState<ResourceState<WorkflowInstance>>(() => createResourceState());
  const [selectedRun, setSelectedRun] = useState<InstanceRun | null>(null);
  const [selectedExecution, setSelectedExecution] = useState<WorkflowExecution | null>(null);
  const [selectedExecutionId, setSelectedExecutionId] = useState('');

  const [instanceQuery, setInstanceQuery] = useState('');
  const [executionQuery, setExecutionQuery] = useState('');
  const [workflowQuery, setWorkflowQuery] = useState('');

  const [eventType, setEventType] = useState('order.created');
  const [eventSource, setEventSource] = useState('ops-console');
  const [eventIdempotencyKey, setEventIdempotencyKey] = useState('');
  const [eventPayload, setEventPayload] = useState(
    '{\n  "order_id": "ord_1001",\n  "amount": 199.99,\n  "currency": "USD"\n}',
  );
  const [signalName, setSignalName] = useState('');
  const [signalPayload, setSignalPayload] = useState('{\n  "approved": true\n}');

  const [refreshingData, setRefreshingData] = useState(false);
  const [loadingRun, setLoadingRun] = useState(false);
  const [busyAction, setBusyAction] = useState('');

  const deferredInstanceQuery = useDeferredValue(instanceQuery);
  const deferredExecutionQuery = useDeferredValue(executionQuery);
  const deferredWorkflowQuery = useDeferredValue(workflowQuery);
  const hasConfig = settings.apiBaseUrl.trim().length > 0;

  const instanceAbortRef = useRef<AbortController | null>(null);
  const executionAbortRef = useRef<AbortController | null>(null);
  const workflowAbortRef = useRef<AbortController | null>(null);
  const runAbortRef = useRef<AbortController | null>(null);
  const childAbortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    if (!hasConfig) {
      setInstanceState(createResourceState());
      setExecutionState(createResourceState());
      setWorkflowState(createResourceState());
      setChildInstanceState(createResourceState());
      setSelectedRun(null);
      setSelectedExecution(null);
      return;
    }

    void reloadInstances();
  }, [hasConfig, settings.apiBaseUrl, settings.authToken, deferredInstanceQuery]);

  useEffect(() => {
    if (!hasConfig) {
      return;
    }

    void reloadExecutions();
  }, [hasConfig, settings.apiBaseUrl, settings.authToken, deferredExecutionQuery]);

  useEffect(() => {
    if (!hasConfig) {
      return;
    }

    void reloadWorkflows();
  }, [hasConfig, settings.apiBaseUrl, settings.authToken, deferredWorkflowQuery]);

  useEffect(() => {
    const firstWaitingSignal = selectedRun?.signalWaits.find((item) => humanizeToken(item.status) === 'waiting');
    if (firstWaitingSignal) {
      setSignalName(firstWaitingSignal.signalName);
    }
  }, [selectedRun?.instance?.id]);

  useEffect(() => {
    return () => {
      instanceAbortRef.current?.abort();
      executionAbortRef.current?.abort();
      workflowAbortRef.current?.abort();
      runAbortRef.current?.abort();
      childAbortRef.current?.abort();
    };
  }, []);

  const instances = instanceState.items;
  const executions = executionState.items;
  const workflows = workflowState.items;
  const childInstances = childInstanceState.items;

  function errorText(error: unknown) {
    if (error instanceof Error) {
      return error.message;
    }

    return String(error);
  }

  function markSynced() {
    setLastSyncedAt(new Date().toISOString());
  }

  async function loadInstancesPage(cursor = '', append = false) {
    if (!hasConfig) {
      return;
    }

    instanceAbortRef.current?.abort();
    const controller = new AbortController();
    instanceAbortRef.current = controller;

    setInstanceState((current) => ({
      ...current,
      loading: !append,
      loadingMore: append,
    }));

    try {
      const page = await listInstances(
        settings,
        {
          limit: DEFAULT_PAGE_SIZE,
          query: deferredInstanceQuery,
          cursor,
        },
        { signal: controller.signal },
      );

      if (controller.signal.aborted) {
        return;
      }

      startTransition(() => {
        setInstanceState((current) => ({
          items: append ? [...current.items, ...page.items] : page.items,
          nextCursor: page.nextCursor || '',
          loading: false,
          loadingMore: false,
        }));
      });
      markSynced();
    } catch (error) {
      if (isAbortError(error)) {
        return;
      }

      setInstanceState((current) => ({ ...current, loading: false, loadingMore: false }));
      setStatusMessage(errorText(error));
    }
  }

  async function reloadInstances() {
    await loadInstancesPage('', false);
  }

  async function loadMoreInstances() {
    if (!instanceState.nextCursor || instanceState.loadingMore) {
      return;
    }

    await loadInstancesPage(instanceState.nextCursor, true);
  }

  async function loadExecutionsPage(cursor = '', append = false) {
    if (!hasConfig) {
      return;
    }

    executionAbortRef.current?.abort();
    const controller = new AbortController();
    executionAbortRef.current = controller;

    setExecutionState((current) => ({
      ...current,
      loading: !append,
      loadingMore: append,
    }));

    try {
      const page = await listExecutions(
        settings,
        {
          limit: DEFAULT_PAGE_SIZE,
          query: deferredExecutionQuery,
          cursor,
        },
        { signal: controller.signal },
      );

      if (controller.signal.aborted) {
        return;
      }

      startTransition(() => {
        setExecutionState((current) => ({
          items: append ? [...current.items, ...page.items] : page.items,
          nextCursor: page.nextCursor || '',
          loading: false,
          loadingMore: false,
        }));
      });
      markSynced();
    } catch (error) {
      if (isAbortError(error)) {
        return;
      }

      setExecutionState((current) => ({ ...current, loading: false, loadingMore: false }));
      setStatusMessage(errorText(error));
    }
  }

  async function reloadExecutions() {
    await loadExecutionsPage('', false);
  }

  async function loadMoreExecutions() {
    if (!executionState.nextCursor || executionState.loadingMore) {
      return;
    }

    await loadExecutionsPage(executionState.nextCursor, true);
  }

  async function loadWorkflowsPage(cursor = '', append = false) {
    if (!hasConfig) {
      return;
    }

    workflowAbortRef.current?.abort();
    const controller = new AbortController();
    workflowAbortRef.current = controller;

    setWorkflowState((current) => ({
      ...current,
      loading: !append,
      loadingMore: append,
    }));

    try {
      const page = await listWorkflows(
        settings,
        {
          limit: DEFAULT_PAGE_SIZE,
          query: deferredWorkflowQuery,
          cursor,
          status: 'WORKFLOW_STATUS_ACTIVE',
        },
        { signal: controller.signal },
      );

      if (controller.signal.aborted) {
        return;
      }

      startTransition(() => {
        setWorkflowState((current) => ({
          items: append ? [...current.items, ...page.items] : page.items,
          nextCursor: page.nextCursor || '',
          loading: false,
          loadingMore: false,
        }));
      });
      markSynced();
    } catch (error) {
      if (isAbortError(error)) {
        return;
      }

      setWorkflowState((current) => ({ ...current, loading: false, loadingMore: false }));
      setStatusMessage(errorText(error));
    }
  }

  async function reloadWorkflows() {
    await loadWorkflowsPage('', false);
  }

  async function loadMoreWorkflows() {
    if (!workflowState.nextCursor || workflowState.loadingMore) {
      return;
    }

    await loadWorkflowsPage(workflowState.nextCursor, true);
  }

  async function loadMoreChildInstances() {
    if (!selectedRun?.instance?.id || !childInstanceState.nextCursor || childInstanceState.loadingMore) {
      return;
    }

    await loadChildInstances(selectedRun.instance.id, childInstanceState.nextCursor, true);
  }

  async function loadChildInstances(instanceId: string, cursor = '', append = false) {
    childAbortRef.current?.abort();
    const controller = new AbortController();
    childAbortRef.current = controller;

    setChildInstanceState((current) => ({
      ...current,
      loading: !append,
      loadingMore: append,
    }));

    try {
      const batches: WorkflowInstance[] = [];
      let nextCursor = cursor;
      let pagesLoaded = 0;
      let finalCursor = '';

      do {
        const page: PageResult<WorkflowInstance> = await listInstances(
          settings,
          {
            limit: CHILD_PAGE_SIZE,
            parentInstanceId: instanceId,
            cursor: nextCursor,
          },
          { signal: controller.signal, timeoutMs: 20000 },
        );

        if (controller.signal.aborted) {
          return;
        }

        batches.push(...page.items);
        finalCursor = page.nextCursor || '';
        nextCursor = finalCursor;
        pagesLoaded += 1;
      } while (nextCursor && pagesLoaded < MAX_CHILD_PAGES);

      startTransition(() => {
        setChildInstanceState((current) => ({
          items: append ? [...current.items, ...batches] : batches,
          nextCursor: finalCursor,
          loading: false,
          loadingMore: false,
        }));
      });
    } catch (error) {
      if (isAbortError(error)) {
        return;
      }

      setChildInstanceState((current) => ({ ...current, loading: false, loadingMore: false }));
      setStatusMessage(errorText(error));
    }
  }

  async function refreshVisibleData() {
    if (!hasConfig) {
      return;
    }

    setRefreshingData(true);
    try {
      await Promise.all([reloadInstances(), reloadExecutions(), reloadWorkflows()]);
      if (selectedRun?.instance?.id) {
        await loadInstanceRun(selectedRun.instance.id, selectedExecutionId || undefined, true);
      }
      setStatusMessage('Visible windows refreshed');
    } catch (error) {
      if (!isAbortError(error)) {
        setStatusMessage(errorText(error));
      }
    } finally {
      setRefreshingData(false);
    }
  }

  async function loadInstanceRun(instanceId: string, executionId?: string, quiet = false) {
    if (!quiet) {
      setLoadingRun(true);
    }

    runAbortRef.current?.abort();
    const controller = new AbortController();
    runAbortRef.current = controller;

    try {
      const [run, detail] = await Promise.all([
        getInstanceRun(settings, instanceId, { signal: controller.signal, timeoutMs: 20000 }),
        executionId ? getExecution(settings, executionId, { signal: controller.signal }) : Promise.resolve(null),
      ]);

      if (controller.signal.aborted) {
        return;
      }

      startTransition(() => {
        setSelectedRun(run);
        setSelectedExecutionId(executionId || run.latestExecution?.id || '');
        setSelectedExecution(detail || null);
      });

      await loadChildInstances(instanceId);
    } catch (error) {
      if (!isAbortError(error)) {
        setStatusMessage(errorText(error));
      }
    } finally {
      if (!quiet) {
        setLoadingRun(false);
      }
    }
  }

  async function handleSelectInstance(instance: WorkflowInstance) {
    setActiveView('runs');
    await loadInstanceRun(instance.id);
  }

  async function handleSelectExecution(execution: WorkflowExecution) {
    setActiveView('runs');
    setLoadingRun(true);

    try {
      await loadInstanceRun(execution.instanceId, execution.id, true);
    } catch (error) {
      if (!isAbortError(error)) {
        setStatusMessage(errorText(error));
      }
    } finally {
      setLoadingRun(false);
    }
  }

  async function handleRetryExecution(executionId: string) {
    setBusyAction(`retry-execution-${executionId}`);
    try {
      await retryExecution(settings, executionId);
      await refreshVisibleData();
      if (selectedRun?.instance?.id) {
        await loadInstanceRun(selectedRun.instance.id, executionId, true);
      }
      setStatusMessage(`Retry scheduled for execution ${executionId}`);
    } catch (error) {
      setStatusMessage(errorText(error));
    } finally {
      setBusyAction('');
    }
  }

  async function handleRetryInstance(instanceId: string) {
    setBusyAction(`retry-instance-${instanceId}`);
    try {
      await retryInstance(settings, instanceId);
      await refreshVisibleData();
      await loadInstanceRun(instanceId, undefined, true);
      setStatusMessage(`Retry scheduled for instance ${instanceId}`);
    } catch (error) {
      setStatusMessage(errorText(error));
    } finally {
      setBusyAction('');
    }
  }

  async function handleTriggerEvent() {
    setBusyAction('trigger-event');
    try {
      const payload = JSON.parse(eventPayload) as Record<string, unknown>;
      const response = await ingestEvent(settings, {
        eventType,
        source: eventSource,
        idempotencyKey: eventIdempotencyKey || undefined,
        payload,
      });

      await refreshVisibleData();
      setStatusMessage(
        response.idempotent
          ? `Event matched existing record ${response.event?.eventId || ''}`
          : `Event accepted as ${response.event?.eventId || ''}`,
      );
    } catch (error) {
      setStatusMessage(errorText(error));
    } finally {
      setBusyAction('');
    }
  }

  async function handleSendSignal() {
    if (!selectedRun?.instance?.id) {
      setStatusMessage('Select a workflow run before sending a signal');
      return;
    }
    if (!signalName.trim()) {
      setStatusMessage('Signal name is required');
      return;
    }

    setBusyAction('send-signal');
    try {
      const payload = JSON.parse(signalPayload) as Record<string, unknown>;
      const response = await sendSignal(settings, {
        instanceId: selectedRun.instance.id,
        signalName: signalName.trim(),
        payload,
      });

      await refreshVisibleData();
      await loadInstanceRun(selectedRun.instance.id, selectedExecutionId || undefined, true);
      setStatusMessage(
        response.delivered
          ? `Signal ${signalName} delivered to ${selectedRun.instance.id}`
          : `Signal ${signalName} queued for ${selectedRun.instance.id}`,
      );
    } catch (error) {
      setStatusMessage(errorText(error));
    } finally {
      setBusyAction('');
    }
  }

  function saveConsoleSettings(next: ConsoleSettings) {
    setSettings(next);
    saveSettings(next);
  }

  const filteredInstances = instances;
  const filteredExecutions = executions;
  const filteredWorkflows = workflows;

  const failureCount = executions.filter((execution) => canRetry(execution.status)).length;
  const waitingCount = executions.filter((execution) => isWaiting(execution.status)).length;
  const runningCount = executions.filter((execution) =>
    humanizeToken(execution.status).includes('running') ||
    humanizeToken(execution.status).includes('dispatched') ||
    humanizeToken(execution.status).includes('pending'),
  ).length;
  const selectedExecutionInRun =
    selectedExecution ||
    selectedRun?.executions.find((execution) => execution.id === selectedExecutionId) ||
    null;
  const selectedOutput = outputForExecution(selectedRun?.outputs ?? [], selectedExecutionInRun?.id);
  const activeSignalWaits = selectedRun?.signalWaits.filter((item) => humanizeToken(item.status) === 'waiting') ?? [];
  const pendingSignals = selectedRun?.signalMessages.filter((item) => humanizeToken(item.status) === 'pending') ?? [];
  const openScopes = selectedRun?.scopeRuns.filter((scope) => humanizeToken(scope.status) !== 'completed') ?? [];
  const hotExecutions = sortByNewest(executions)
    .filter((execution) => canRetry(execution.status) || isWaiting(execution.status))
    .slice(0, 6);
  const watchSignals = sortByNewest(selectedRun?.signalMessages ?? []).slice(0, 6);
  const watchTimeline = sortByNewest(selectedRun?.timeline ?? []).slice(0, 12);

  return (
    <div className="min-h-screen bg-[var(--canvas)] text-[var(--ink)]">
      <div className="mx-auto flex min-h-screen max-w-[1800px] flex-col xl:flex-row">
        <aside className="border-b border-stone-200/80 bg-white/70 px-5 py-6 backdrop-blur xl:min-h-screen xl:w-80 xl:border-b-0 xl:border-r">
          <div className="space-y-2">
            <div className="text-[10px] font-semibold uppercase tracking-[0.45em] text-stone-400">Trustage</div>
            <h1 className="font-logo text-3xl font-semibold leading-tight">Operator Console</h1>
            <p className="max-w-sm text-sm text-stone-500">
              Inspect workflow causality, branch scopes, waiting signals, retries, and live event ingress from one control surface.
            </p>
          </div>

          <nav className="mt-8 grid gap-2">
            {navItems.map((item) => (
              <button
                key={item.key}
                className={`rounded-2xl px-4 py-3 text-left text-sm font-semibold transition ${
                  activeView === item.key
                    ? 'bg-[var(--ink)] text-white shadow-[0_16px_40px_rgba(23,32,51,0.18)]'
                    : 'border border-stone-200 bg-white text-stone-600 hover:border-stone-300 hover:bg-stone-50'
                }`}
                onClick={() => setActiveView(item.key)}
              >
                {item.label}
              </button>
            ))}
          </nav>

          <div className="mt-8 grid gap-3">
            <MetricCard
              label="Visible runs"
              value={instances.length}
              tone="border-sky-200 bg-sky-50 text-sky-800"
              detail={`${runningCount} moving in current page`}
            />
            <MetricCard
              label="Waiting work"
              value={waitingCount}
              tone="border-amber-200 bg-amber-50 text-amber-800"
              detail="timers, waits, or manual gates"
            />
            <MetricCard
              label="Needs attention"
              value={failureCount}
              tone="border-rose-200 bg-rose-50 text-rose-800"
              detail="retryable or failed executions"
            />
          </div>

          <div className="mt-8 rounded-[24px] border border-stone-200 bg-white px-4 py-4 text-sm text-stone-600">
            <div className="text-[10px] font-semibold uppercase tracking-[0.35em] text-stone-400">Connection</div>
            <div className="mt-2 break-all font-medium text-stone-900">
              {settings.apiBaseUrl || 'Configure the API base URL'}
            </div>
            <div className="mt-2 text-xs text-stone-500">
              {lastSyncedAt ? `Last sync ${formatDate(lastSyncedAt)}` : 'No live data loaded yet'}
            </div>
            <div className="mt-4 flex gap-2">
              <button
                className="flex-1 rounded-xl bg-[var(--ink)] px-3 py-2 text-xs font-semibold uppercase tracking-[0.18em] text-white hover:bg-[var(--ink-soft)] disabled:opacity-50"
                onClick={() => void refreshVisibleData()}
                disabled={!hasConfig || refreshingData}
              >
                {refreshingData ? 'Syncing' : 'Refresh'}
              </button>
              <button
                className="rounded-xl border border-stone-200 px-3 py-2 text-xs font-semibold uppercase tracking-[0.18em] text-stone-700 hover:bg-stone-50"
                onClick={() => setSettingsOpen(true)}
              >
                Settings
              </button>
            </div>
          </div>
        </aside>

        <div className="flex-1 px-4 py-4 sm:px-6 lg:px-8">
          <div className="grid gap-6 xl:grid-cols-[minmax(0,1.3fr)_420px]">
            <div className="space-y-6">
              <Panel
                eyebrow="Live posture"
                title="Traceable orchestration state"
                subtitle="The console is run-centric: pick an instance or execution and the right side becomes a causal trace, not a raw log dump."
                actions={
                  <div className="flex flex-wrap items-center gap-2">
                    <div className="rounded-full border border-stone-200 bg-white px-3 py-1.5 text-xs text-stone-500">
                      {hasConfig ? 'Connected' : 'Disconnected'}
                    </div>
                    {statusMessage && (
                      <div className="max-w-[360px] rounded-full border border-stone-200 bg-stone-50 px-3 py-1.5 text-xs text-stone-600">
                        {statusMessage}
                      </div>
                    )}
                  </div>
                }
              >
                <div className="grid gap-3 lg:grid-cols-4">
                  <MetricCard
                    label="Executions"
                    value={executions.length}
                    tone="border-stone-200 bg-stone-50 text-stone-800"
                    detail="current result window"
                  />
                  <MetricCard
                    label="Running"
                    value={runningCount}
                    tone="border-sky-200 bg-sky-50 text-sky-800"
                    detail="pending, dispatched, running"
                  />
                  <MetricCard
                    label="Signals waiting"
                    value={activeSignalWaits.length}
                    tone="border-amber-200 bg-amber-50 text-amber-800"
                    detail="selected run"
                  />
                  <MetricCard
                    label="Open scopes"
                    value={openScopes.length}
                    tone="border-violet-200 bg-violet-50 text-violet-800"
                    detail="parallel or foreach branches"
                  />
                </div>
              </Panel>

              {activeView === 'overview' && (
                <>
                  <Panel
                    eyebrow="Ingress"
                    title="Trigger an event"
                    subtitle="Uses the Connect event API, so what you send here is the same typed ingest surface operators and services should rely on."
                    actions={
                      <button
                        className="rounded-2xl bg-[var(--accent-strong)] px-4 py-2 text-xs font-semibold uppercase tracking-[0.24em] text-white hover:bg-[var(--accent)] disabled:opacity-50"
                        onClick={() => void handleTriggerEvent()}
                        disabled={!hasConfig || busyAction === 'trigger-event'}
                      >
                        {busyAction === 'trigger-event' ? 'Sending' : 'Send event'}
                      </button>
                    }
                  >
                    <div className="grid gap-4 lg:grid-cols-3">
                      <label className="text-xs font-semibold uppercase tracking-[0.22em] text-stone-500">
                        Event type
                        <input
                          className="mt-2 w-full rounded-2xl border border-stone-200 bg-stone-50 px-3 py-2.5 text-sm font-normal text-stone-900"
                          value={eventType}
                          onChange={(event) => setEventType(event.target.value)}
                        />
                      </label>
                      <label className="text-xs font-semibold uppercase tracking-[0.22em] text-stone-500">
                        Source
                        <input
                          className="mt-2 w-full rounded-2xl border border-stone-200 bg-stone-50 px-3 py-2.5 text-sm font-normal text-stone-900"
                          value={eventSource}
                          onChange={(event) => setEventSource(event.target.value)}
                        />
                      </label>
                      <label className="text-xs font-semibold uppercase tracking-[0.22em] text-stone-500">
                        Idempotency key
                        <input
                          className="mt-2 w-full rounded-2xl border border-stone-200 bg-stone-50 px-3 py-2.5 text-sm font-normal text-stone-900"
                          value={eventIdempotencyKey}
                          placeholder="optional"
                          onChange={(event) => setEventIdempotencyKey(event.target.value)}
                        />
                      </label>
                    </div>
                    <label className="mt-4 block text-xs font-semibold uppercase tracking-[0.22em] text-stone-500">
                      Payload
                      <textarea
                        className="mt-2 h-48 w-full rounded-[24px] border border-stone-200 bg-stone-950 px-4 py-3 font-mono text-xs text-stone-100"
                        value={eventPayload}
                        onChange={(event) => setEventPayload(event.target.value)}
                      />
                    </label>
                  </Panel>

                  <div className="grid gap-6 lg:grid-cols-2">
                    <Panel
                      eyebrow="Attention"
                      title="Hot executions"
                      subtitle="Retries and waiting work rise here first."
                    >
                      <div className="space-y-3">
                        {hotExecutions.length === 0 && (
                          <EmptyState
                            title="Nothing urgent"
                            description="No waiting or failed executions are visible in the current result window."
                          />
                        )}
                        {hotExecutions.map((execution) => (
                          <button
                            key={execution.id}
                            className="flex w-full flex-wrap items-center justify-between gap-3 rounded-2xl border border-stone-200 bg-stone-50 px-4 py-3 text-left hover:border-stone-300 hover:bg-white"
                            onClick={() => void handleSelectExecution(execution)}
                          >
                            <div>
                              <div className="font-semibold text-stone-900">{execution.state}</div>
                          <div className="text-xs text-stone-500">
                            {execution.id} • instance {shortId(execution.instanceId, 10)}
                          </div>
                            </div>
                            <StatusBadge value={execution.status} />
                          </button>
                        ))}
                      </div>
                    </Panel>

                    <Panel
                      eyebrow="Signals"
                      title="Current signal traffic"
                      subtitle="Pending and delivered signal messages from the selected run."
                    >
                      <div className="space-y-3">
                        {watchSignals.length === 0 && (
                          <EmptyState
                            title="No signal activity"
                            description="Select a run with signal waits or messages to inspect signal flow."
                          />
                        )}
                        {watchSignals.map((message: SignalMessage) => (
                          <div key={message.id} className="rounded-2xl border border-stone-200 bg-stone-50 px-4 py-3">
                            <div className="flex items-center justify-between gap-2">
                              <div className="font-semibold text-stone-900">{message.signalName}</div>
                              <StatusBadge value={message.status} />
                            </div>
                            <div className="mt-2 text-xs text-stone-500">
                              delivered {formatDate(message.deliveredAt)} • attempts {message.attempts}
                            </div>
                          </div>
                        ))}
                      </div>
                    </Panel>
                  </div>
                </>
              )}

              {activeView === 'runs' && (
                <>
                  <Panel
                    eyebrow="Explorer"
                    title="Workflow runs"
                    subtitle="Filter by workflow, state, trace, or parent linkage, then open a run to inspect branch fanout and signals."
                  >
                    <div className="mb-4">
                      <input
                        className="w-full rounded-2xl border border-stone-200 bg-stone-50 px-4 py-3 text-sm text-stone-900"
                        placeholder="Search by workflow, instance id, state, parent execution…"
                        value={instanceQuery}
                        onChange={(event) => setInstanceQuery(event.target.value)}
                      />
                    </div>
                    <div className="grid gap-3">
                      {instanceState.loading && (
                        <div className="rounded-2xl border border-stone-200 bg-stone-50 px-4 py-3 text-sm text-stone-600">
                          Loading runs…
                        </div>
                      )}
                      {filteredInstances.map((instance) => (
                        <button
                          key={instance.id}
                          className={`grid gap-3 rounded-[24px] border px-4 py-4 text-left transition lg:grid-cols-[minmax(0,1.2fr)_180px_170px_auto] ${
                            selectedRun?.instance?.id === instance.id
                              ? 'border-[var(--accent-strong)] bg-[var(--accent-soft)]'
                              : 'border-stone-200 bg-white hover:border-stone-300 hover:bg-stone-50'
                          }`}
                          onClick={() => void handleSelectInstance(instance)}
                        >
                          <div>
                            <div className="font-semibold text-stone-900">{instance.workflowName}</div>
                            <div className="mt-1 text-xs text-stone-500">
                              {instance.id}
                              {instance.parentExecutionId ? ` • child of ${shortId(instance.parentExecutionId, 10)}` : ''}
                            </div>
                          </div>
                          <div>
                            <div className="text-[10px] uppercase tracking-[0.25em] text-stone-400">Current state</div>
                            <div className="mt-1 font-medium text-stone-800">{instance.currentState}</div>
                          </div>
                          <div>
                            <div className="text-[10px] uppercase tracking-[0.25em] text-stone-400">Started</div>
                            <div className="mt-1 font-medium text-stone-800">{formatDate(instance.startedAt)}</div>
                          </div>
                          <div className="flex justify-start lg:justify-end">
                            <StatusBadge value={instance.status} />
                          </div>
                        </button>
                      ))}
                      {filteredInstances.length === 0 && (
                        <EmptyState
                          title="No runs match the current filter"
                          description="The server-side run search returned no matching workflow instance."
                        />
                      )}
                      {instanceState.nextCursor && (
                        <button
                          className="rounded-2xl border border-stone-200 bg-stone-50 px-4 py-3 text-sm font-semibold text-stone-700 hover:bg-white disabled:opacity-50"
                          onClick={() => void loadMoreInstances()}
                          disabled={instanceState.loadingMore}
                        >
                          {instanceState.loadingMore ? 'Loading more runs…' : 'Load more runs'}
                        </button>
                      )}
                    </div>
                  </Panel>

                  <Panel
                    eyebrow="Trace"
                    title="Execution graph"
                    subtitle="This view follows the selected run from root instance to state attempts, scopes, waits, and child runs."
                  >
                    {!selectedRun?.instance && (
                      <EmptyState
                        title="Pick a run"
                        description="Choose a workflow instance from the explorer to render its causal execution path."
                      />
                    )}
                    {loadingRun && (
                      <div className="mb-4 rounded-2xl border border-stone-200 bg-stone-50 px-4 py-3 text-sm text-stone-600">
                        Loading selected run…
                      </div>
                    )}
                    {selectedRun?.instance && (
                      <>
                        <ExecutionGraph
                          run={selectedRun}
                          childInstances={childInstances}
                          selectedExecutionId={selectedExecutionId}
                          onSelectExecution={(execution) => void handleSelectExecution(execution)}
                          onSelectInstance={(instance) => void handleSelectInstance(instance)}
                        />
                        {(childInstanceState.loadingMore || childInstanceState.nextCursor) && (
                          <div className="mt-4 rounded-2xl border border-stone-200 bg-stone-50 px-4 py-3">
                            <div className="flex flex-wrap items-center justify-between gap-3">
                              <div className="text-sm text-stone-600">
                                {childInstanceState.loadingMore
                                  ? 'Loading more child runs for the selected trace…'
                                  : 'More child runs exist for this trace.'}
                              </div>
                              {childInstanceState.nextCursor && (
                                <button
                                  className="rounded-xl border border-stone-200 bg-white px-3 py-2 text-xs font-semibold text-stone-700 hover:bg-stone-100 disabled:opacity-50"
                                  onClick={() => void loadMoreChildInstances()}
                                  disabled={childInstanceState.loadingMore}
                                >
                                  {childInstanceState.loadingMore ? 'Loading…' : 'Load more child runs'}
                                </button>
                              )}
                            </div>
                          </div>
                        )}
                      </>
                    )}
                  </Panel>

                  <Panel
                    eyebrow="Timeline"
                    title="Causal timeline"
                    subtitle="Audit entries are shown in descending order so an operator can reconstruct what happened without joining raw tables by hand."
                  >
                    <div className="space-y-3">
                      {watchTimeline.length === 0 && (
                        <EmptyState
                          title="No timeline available"
                          description="Select a run to inspect audit events, transitions, and emitted state payloads."
                        />
                      )}
                      {watchTimeline.map((entry) => (
                        <div key={`${entry.eventType}-${entry.createdAt}-${entry.executionId}`} className="rounded-2xl border border-stone-200 bg-stone-50 px-4 py-3">
                          <div className="flex flex-wrap items-center justify-between gap-2">
                            <div>
                              <div className="font-semibold text-stone-900">{entry.eventType}</div>
                              <div className="mt-1 text-xs text-stone-500">
                                {entry.state || '—'}
                                {entry.executionId ? ` • ${entry.executionId}` : ''}
                              </div>
                            </div>
                            <div className="text-xs text-stone-500">{formatDate(entry.createdAt)}</div>
                          </div>
                          {(entry.fromState || entry.toState) && (
                            <div className="mt-3 text-xs text-stone-600">
                              {entry.fromState || '—'} → {entry.toState || '—'}
                            </div>
                          )}
                        </div>
                      ))}
                    </div>
                  </Panel>
                </>
              )}

              {activeView === 'executions' && (
                <Panel
                  eyebrow="Queue"
                  title="Execution queue"
                  subtitle="Search across state attempts, trace ids, and error messages, then open the selected execution in the run explorer."
                >
                  <div className="mb-4">
                    <input
                      className="w-full rounded-2xl border border-stone-200 bg-stone-50 px-4 py-3 text-sm text-stone-900"
                      placeholder="Search by execution id, instance id, state, trace, or error…"
                      value={executionQuery}
                      onChange={(event) => setExecutionQuery(event.target.value)}
                    />
                  </div>
                  <div className="grid gap-3">
                    {executionState.loading && (
                      <div className="rounded-2xl border border-stone-200 bg-stone-50 px-4 py-3 text-sm text-stone-600">
                        Loading executions…
                      </div>
                    )}
                    {filteredExecutions.map((execution) => (
                      <button
                        key={execution.id}
                        className="grid gap-3 rounded-[24px] border border-stone-200 bg-white px-4 py-4 text-left transition hover:border-stone-300 hover:bg-stone-50 lg:grid-cols-[minmax(0,1.2fr)_170px_130px_150px_auto]"
                        onClick={() => void handleSelectExecution(execution)}
                      >
                        <div>
                          <div className="font-semibold text-stone-900">{execution.state}</div>
                          <div className="mt-1 text-xs text-stone-500">
                            {execution.id} • trace {execution.traceId || '—'}
                          </div>
                        </div>
                        <div>
                          <div className="text-[10px] uppercase tracking-[0.25em] text-stone-400">Instance</div>
                          <div className="mt-1 font-medium text-stone-800">{shortId(execution.instanceId, 14)}</div>
                        </div>
                        <div>
                          <div className="text-[10px] uppercase tracking-[0.25em] text-stone-400">Attempt</div>
                          <div className="mt-1 font-medium text-stone-800">#{execution.attempt}</div>
                        </div>
                        <div>
                          <div className="text-[10px] uppercase tracking-[0.25em] text-stone-400">Started</div>
                          <div className="mt-1 font-medium text-stone-800">{formatDate(execution.startedAt || execution.createdAt)}</div>
                        </div>
                        <div className="flex items-center justify-between gap-3 lg:justify-end">
                          <StatusBadge value={execution.status} />
                          {canRetry(execution.status) && (
                            <button
                              className="rounded-xl border border-stone-200 px-3 py-2 text-xs font-semibold text-stone-700 hover:bg-stone-100"
                              onClick={(event) => {
                                event.stopPropagation();
                                void handleRetryExecution(execution.id);
                              }}
                              disabled={busyAction === `retry-execution-${execution.id}`}
                            >
                              {busyAction === `retry-execution-${execution.id}` ? 'Retrying' : 'Retry'}
                            </button>
                          )}
                        </div>
                      </button>
                    ))}
                    {filteredExecutions.length === 0 && (
                      <EmptyState
                        title="No executions match"
                        description="The server-side execution search returned no matching state attempts."
                      />
                    )}
                    {executionState.nextCursor && (
                      <button
                        className="rounded-2xl border border-stone-200 bg-stone-50 px-4 py-3 text-sm font-semibold text-stone-700 hover:bg-white disabled:opacity-50"
                        onClick={() => void loadMoreExecutions()}
                        disabled={executionState.loadingMore}
                      >
                        {executionState.loadingMore ? 'Loading more executions…' : 'Load more executions'}
                      </button>
                    )}
                  </div>
                </Panel>
              )}

              {activeView === 'workflows' && (
                <Panel
                  eyebrow="Definitions"
                  title="Workflow catalog"
                  subtitle="Definition status, versioning, and active drafts visible from the workflow Connect API."
                >
                  <div className="mb-4">
                    <input
                      className="w-full rounded-2xl border border-stone-200 bg-stone-50 px-4 py-3 text-sm text-stone-900"
                      placeholder="Search by workflow name or id…"
                      value={workflowQuery}
                      onChange={(event) => setWorkflowQuery(event.target.value)}
                    />
                  </div>
                  <div className="grid gap-3">
                    {workflowState.loading && (
                      <div className="rounded-2xl border border-stone-200 bg-stone-50 px-4 py-3 text-sm text-stone-600">
                        Loading workflows…
                      </div>
                    )}
                    {filteredWorkflows.map((workflow) => (
                      <div
                        key={workflow.id}
                        className="grid gap-3 rounded-[24px] border border-stone-200 bg-white px-4 py-4 lg:grid-cols-[minmax(0,1.4fr)_120px_170px_auto]"
                      >
                        <div>
                          <div className="font-semibold text-stone-900">{workflow.name}</div>
                          <div className="mt-1 text-xs text-stone-500">{workflow.id}</div>
                        </div>
                        <div>
                          <div className="text-[10px] uppercase tracking-[0.25em] text-stone-400">Version</div>
                          <div className="mt-1 font-medium text-stone-800">v{workflow.version}</div>
                        </div>
                        <div>
                          <div className="text-[10px] uppercase tracking-[0.25em] text-stone-400">Updated</div>
                          <div className="mt-1 font-medium text-stone-800">{formatDate(workflow.updatedAt || workflow.createdAt)}</div>
                        </div>
                        <div className="flex lg:justify-end">
                          <StatusBadge value={workflow.status} />
                        </div>
                      </div>
                    ))}
                    {filteredWorkflows.length === 0 && (
                      <EmptyState
                        title="No workflows match"
                        description="The server-side workflow search returned no active definitions."
                      />
                    )}
                    {workflowState.nextCursor && (
                      <button
                        className="rounded-2xl border border-stone-200 bg-stone-50 px-4 py-3 text-sm font-semibold text-stone-700 hover:bg-white disabled:opacity-50"
                        onClick={() => void loadMoreWorkflows()}
                        disabled={workflowState.loadingMore}
                      >
                        {workflowState.loadingMore ? 'Loading more workflows…' : 'Load more workflows'}
                      </button>
                    )}
                  </div>
                </Panel>
              )}
            </div>

            <aside className="space-y-6">
              <Panel
                eyebrow="Selected run"
                title={selectedRun?.instance?.workflowName || 'No run selected'}
                subtitle={
                  selectedRun?.instance
                    ? `Instance ${selectedRun.instance.id}`
                    : 'Pick an instance or execution to see trace facts, payloads, waits, and operator actions.'
                }
              >
                {!selectedRun?.instance && (
                  <EmptyState
                    title="Waiting for selection"
                    description="Use Run Explorer or Execution Queue to open a workflow run."
                  />
                )}

                {selectedRun?.instance && (
                  <div className="space-y-4">
                    <div className="grid gap-3 sm:grid-cols-2">
                      <MetricCard
                        label="Executions"
                        value={selectedRun.executions.length}
                        tone="border-stone-200 bg-stone-50 text-stone-800"
                        detail={`latest ${selectedRun.latestExecution?.state || '—'}`}
                      />
                      <MetricCard
                        label="Child runs"
                        value={childInstances.length}
                        tone="border-sky-200 bg-sky-50 text-sky-800"
                        detail={childInstanceState.nextCursor ? 'partial trace window' : 'branch or foreach children'}
                      />
                      <MetricCard
                        label="Signal waits"
                        value={activeSignalWaits.length}
                        tone="border-amber-200 bg-amber-50 text-amber-800"
                        detail="currently blocked"
                      />
                      <MetricCard
                        label="Pending signals"
                        value={pendingSignals.length}
                        tone="border-violet-200 bg-violet-50 text-violet-800"
                        detail="queued for delivery"
                      />
                    </div>

                    <div className="grid gap-3">
                      <div className="rounded-2xl border border-stone-200 bg-stone-50 px-4 py-3">
                        <div className="text-[10px] font-semibold uppercase tracking-[0.3em] text-stone-400">Trace id</div>
                        <div className="mt-2 font-mono text-sm text-stone-900">
                          {selectedRun.traceId || selectedRun.latestExecution?.traceId || '—'}
                        </div>
                      </div>
                      <div className="rounded-2xl border border-stone-200 bg-stone-50 px-4 py-3">
                        <div className="text-[10px] font-semibold uppercase tracking-[0.3em] text-stone-400">Resume strategy</div>
                        <div className="mt-2 text-sm font-medium text-stone-900">
                          {humanizeToken(selectedRun.resumeStrategy)}
                        </div>
                      </div>
                      <div className="rounded-2xl border border-stone-200 bg-stone-50 px-4 py-3">
                        <div className="text-[10px] font-semibold uppercase tracking-[0.3em] text-stone-400">Trace coverage</div>
                        <div className="mt-2 text-sm font-medium text-stone-900">
                          {childInstanceState.nextCursor
                            ? 'Child runs are paged; load more to continue the trace graph.'
                            : 'Direct child runs loaded for the selected trace.'}
                        </div>
                      </div>
                    </div>
                  </div>
                )}
              </Panel>

              <Panel
                eyebrow="Action center"
                title="Operator controls"
                subtitle="Retries and signals stay attached to the selected execution graph so operators can act with context."
              >
                {!selectedRun?.instance && (
                  <EmptyState
                    title="No active context"
                    description="Select a run first. Actions are bound to the currently inspected instance."
                  />
                )}

                {selectedRun?.instance && (
                  <div className="space-y-4">
                    <div className="grid gap-3">
                      {selectedExecutionInRun && canRetry(selectedExecutionInRun.status) && (
                        <button
                          className="rounded-2xl bg-[var(--ink)] px-4 py-3 text-sm font-semibold text-white hover:bg-[var(--ink-soft)]"
                          onClick={() => void handleRetryExecution(selectedExecutionInRun.id)}
                          disabled={busyAction === `retry-execution-${selectedExecutionInRun.id}`}
                        >
                          {busyAction === `retry-execution-${selectedExecutionInRun.id}`
                            ? 'Retrying execution…'
                            : `Retry execution ${shortId(selectedExecutionInRun.id, 10)}`}
                        </button>
                      )}

                      {canRetry(selectedRun.instance.status) && (
                        <button
                          className="rounded-2xl border border-stone-200 px-4 py-3 text-sm font-semibold text-stone-800 hover:bg-stone-50"
                          onClick={() => void handleRetryInstance(selectedRun.instance!.id)}
                          disabled={busyAction === `retry-instance-${selectedRun.instance.id}`}
                        >
                          {busyAction === `retry-instance-${selectedRun.instance.id}`
                            ? 'Retrying instance…'
                            : 'Retry selected instance'}
                        </button>
                      )}
                    </div>

                    <div className="rounded-[24px] border border-stone-200 bg-stone-50 px-4 py-4">
                      <div className="text-[10px] font-semibold uppercase tracking-[0.3em] text-stone-400">Signal delivery</div>
                      <div className="mt-3 space-y-3">
                        <label className="block text-xs font-semibold uppercase tracking-[0.2em] text-stone-500">
                          Signal name
                          <input
                            className="mt-2 w-full rounded-2xl border border-stone-200 bg-white px-3 py-2.5 text-sm font-normal text-stone-900"
                            value={signalName}
                            onChange={(event) => setSignalName(event.target.value)}
                          />
                        </label>
                        <label className="block text-xs font-semibold uppercase tracking-[0.2em] text-stone-500">
                          Payload
                          <textarea
                            className="mt-2 h-32 w-full rounded-2xl border border-stone-200 bg-stone-950 px-3 py-3 font-mono text-xs text-stone-100"
                            value={signalPayload}
                            onChange={(event) => setSignalPayload(event.target.value)}
                          />
                        </label>
                        <button
                          className="w-full rounded-2xl bg-[var(--accent-strong)] px-4 py-3 text-sm font-semibold text-white hover:bg-[var(--accent)] disabled:opacity-50"
                          onClick={() => void handleSendSignal()}
                          disabled={busyAction === 'send-signal'}
                        >
                          {busyAction === 'send-signal' ? 'Sending signal…' : `Send signal to ${shortId(selectedRun.instance.id, 12)}`}
                        </button>
                      </div>
                    </div>
                  </div>
                )}
              </Panel>

              <Panel
                eyebrow="Execution detail"
                title={selectedExecutionInRun?.state || 'Select an execution node'}
                subtitle={
                  selectedExecutionInRun
                    ? `${selectedExecutionInRun.id} • ${humanizeToken(selectedExecutionInRun.status)}`
                    : 'Click an execution in the queue or graph to inspect its input and output.'
                }
              >
                {!selectedExecutionInRun && (
                  <EmptyState
                    title="No execution focused"
                    description="The graph is active, but no individual execution is currently focused."
                  />
                )}

                {selectedExecutionInRun && (
                  <div className="space-y-4">
                    <div className="grid gap-3 text-sm sm:grid-cols-2">
                      <div className="rounded-2xl border border-stone-200 bg-stone-50 px-4 py-3">
                        <div className="text-[10px] font-semibold uppercase tracking-[0.3em] text-stone-400">Status</div>
                        <div className="mt-2">
                          <StatusBadge value={selectedExecutionInRun.status} />
                        </div>
                      </div>
                      <div className="rounded-2xl border border-stone-200 bg-stone-50 px-4 py-3">
                        <div className="text-[10px] font-semibold uppercase tracking-[0.3em] text-stone-400">Attempt</div>
                        <div className="mt-2 font-medium text-stone-900">#{selectedExecutionInRun.attempt}</div>
                      </div>
                    </div>

                    <JsonBlock
                      label="Input payload"
                      value={selectedExecution?.inputPayload || selectedExecutionInRun.inputPayload || {}}
                    />
                    <JsonBlock
                      label="Output payload"
                      value={selectedExecution?.output || selectedOutput?.payload || {}}
                    />
                  </div>
                )}
              </Panel>

              <Panel
                eyebrow="Blocked state"
                title="Waits and queued signals"
                subtitle="This is where an operator sees whether a run is waiting on time, branch completion, or external signal delivery."
              >
                <div className="space-y-3">
                  {activeSignalWaits.map((wait: SignalWait) => (
                    <div key={wait.id} className="rounded-2xl border border-amber-200 bg-amber-50 px-4 py-3">
                      <div className="flex items-center justify-between gap-3">
                        <div>
                          <div className="font-semibold text-amber-900">{wait.signalName}</div>
                          <div className="text-xs text-amber-700">execution {wait.executionId}</div>
                        </div>
                        <StatusBadge value={wait.status} />
                      </div>
                    </div>
                  ))}

                  {pendingSignals.map((message: SignalMessage) => (
                    <div key={message.id} className="rounded-2xl border border-violet-200 bg-violet-50 px-4 py-3">
                      <div className="flex items-center justify-between gap-3">
                        <div>
                          <div className="font-semibold text-violet-900">{message.signalName}</div>
                          <div className="text-xs text-violet-700">created {formatDate(message.createdAt)}</div>
                        </div>
                        <StatusBadge value={message.status} />
                      </div>
                    </div>
                  ))}

                  {activeSignalWaits.length === 0 && pendingSignals.length === 0 && (
                    <EmptyState
                      title="No blocked signal activity"
                      description="The selected run currently has no waiting signal gates or pending signal messages."
                    />
                  )}
                </div>
              </Panel>
            </aside>
          </div>
        </div>
      </div>

      {settingsOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-stone-950/45 px-4 backdrop-blur-sm">
          <div className="w-full max-w-xl rounded-[32px] border border-stone-200 bg-white px-6 py-6 shadow-[0_30px_100px_rgba(25,24,22,0.25)]">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-[10px] font-semibold uppercase tracking-[0.35em] text-stone-400">Settings</div>
                <h2 className="mt-1 font-logo text-2xl font-semibold text-stone-900">Console connection</h2>
              </div>
              <button
                className="rounded-full border border-stone-200 px-3 py-1 text-xs font-semibold text-stone-600 hover:bg-stone-50"
                onClick={() => setSettingsOpen(false)}
              >
                Close
              </button>
            </div>

            <div className="mt-5 space-y-4">
              <label className="block text-xs font-semibold uppercase tracking-[0.22em] text-stone-500">
                API base URL
                <input
                  className="mt-2 w-full rounded-2xl border border-stone-200 bg-stone-50 px-4 py-3 text-sm font-normal text-stone-900"
                  placeholder="https://trustage.api"
                  value={settings.apiBaseUrl}
                  onChange={(event) => saveConsoleSettings({ ...settings, apiBaseUrl: event.target.value })}
                />
              </label>
              <label className="block text-xs font-semibold uppercase tracking-[0.22em] text-stone-500">
                Auth token
                <input
                  className="mt-2 w-full rounded-2xl border border-stone-200 bg-stone-50 px-4 py-3 text-sm font-normal text-stone-900"
                  placeholder="Bearer token"
                  value={settings.authToken}
                  onChange={(event) => saveConsoleSettings({ ...settings, authToken: event.target.value })}
                />
              </label>
            </div>

            <div className="mt-5 rounded-2xl border border-stone-200 bg-stone-50 px-4 py-3 text-sm text-stone-600">
              Connect-backed endpoints in use: workflow list, event ingest, runtime list/get/run, retries, and signal delivery. The auth token is stored in session storage only.
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
