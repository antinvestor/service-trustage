import { useEffect, useMemo, useState } from 'react';
import {
  ExecutionDetail,
  ExecutionItem,
  InstanceItem,
  TimelineEntry,
  WorkflowItem,
  getExecution,
  getInstanceTimeline,
  ingestEvent,
  listExecutions,
  listInstances,
  listWorkflows,
  retryExecution,
  retryInstance,
} from './lib/api';
import { ConsoleSettings, loadSettings, saveSettings } from './lib/storage';

const navItems = [
  { key: 'overview', label: 'Overview' },
  { key: 'instances', label: 'Instances' },
  { key: 'executions', label: 'Executions' },
  { key: 'workflows', label: 'Workflows' },
] as const;

type NavKey = (typeof navItems)[number]['key'];

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

function statusTone(status?: string) {
  if (!status) {
    return 'bg-slate-100 text-slate-600';
  }
  if (status.includes('failed') || status.includes('fatal') || status.includes('invalid')) {
    return 'bg-rose-100 text-rose-700';
  }
  if (status.includes('running') || status.includes('dispatched')) {
    return 'bg-blue-100 text-blue-700';
  }
  if (status.includes('retry')) {
    return 'bg-amber-100 text-amber-700';
  }
  if (status.includes('completed')) {
    return 'bg-emerald-100 text-emerald-700';
  }
  return 'bg-slate-100 text-slate-600';
}

function canRetry(status?: string) {
  if (!status) return false;
  return (
    status.includes('failed') ||
    status.includes('fatal') ||
    status.includes('invalid') ||
    status.includes('timed_out') ||
    status.includes('retry')
  );
}

export default function App() {
  const [activeView, setActiveView] = useState<NavKey>('overview');
  const [settings, setSettings] = useState<ConsoleSettings>(() => loadSettings());
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [statusMessage, setStatusMessage] = useState('');

  const [instances, setInstances] = useState<InstanceItem[]>([]);
  const [executions, setExecutions] = useState<ExecutionItem[]>([]);
  const [workflows, setWorkflows] = useState<WorkflowItem[]>([]);
  const [selectedInstance, setSelectedInstance] = useState<InstanceItem | null>(null);
  const [selectedExecution, setSelectedExecution] = useState<ExecutionDetail | null>(null);
  const [timeline, setTimeline] = useState<TimelineEntry[]>([]);

  const [eventType, setEventType] = useState('order.created');
  const [eventSource, setEventSource] = useState('api');
  const [eventIdempotency, setEventIdempotency] = useState('');
  const [eventPayload, setEventPayload] = useState(
    '{\n  "order_id": "12345",\n  "amount": 199.99,\n  "currency": "USD"\n}',
  );

  const [loadingInstances, setLoadingInstances] = useState(false);
  const [loadingExecutions, setLoadingExecutions] = useState(false);
  const [loadingWorkflows, setLoadingWorkflows] = useState(false);
  const [loadingDetail, setLoadingDetail] = useState(false);

  const hasConfig = useMemo(() => settings.apiBaseUrl.length > 0, [settings.apiBaseUrl]);

  useEffect(() => {
    if (!hasConfig) {
      return;
    }

    refreshAll();
  }, [hasConfig, settings]);

  async function refreshAll() {
    await Promise.all([refreshInstances(), refreshExecutions(), refreshWorkflows()]);
  }

  async function refreshInstances() {
    if (!hasConfig) {
      return;
    }

    setLoadingInstances(true);
    try {
      const response = await listInstances(settings, { limit: 50 });
      setInstances(response.items ?? []);
    } catch (err) {
      setStatusMessage(String(err));
    } finally {
      setLoadingInstances(false);
    }
  }

  async function refreshExecutions() {
    if (!hasConfig) {
      return;
    }

    setLoadingExecutions(true);
    try {
      const response = await listExecutions(settings, { limit: 50 });
      setExecutions(response.items ?? []);
    } catch (err) {
      setStatusMessage(String(err));
    } finally {
      setLoadingExecutions(false);
    }
  }

  async function refreshWorkflows() {
    if (!hasConfig) {
      return;
    }

    setLoadingWorkflows(true);
    try {
      const response = await listWorkflows(settings);
      setWorkflows(response.items ?? []);
    } catch (err) {
      setStatusMessage(String(err));
    } finally {
      setLoadingWorkflows(false);
    }
  }

  async function handleSelectExecution(execution: ExecutionItem) {
    setSelectedExecution(null);
    setLoadingDetail(true);
    try {
      const detail = await getExecution(settings, execution.execution_id);
      setSelectedExecution(detail);
      const timelineData = await getInstanceTimeline(settings, execution.instance_id);
      setTimeline(timelineData ?? []);
    } catch (err) {
      setStatusMessage(String(err));
    } finally {
      setLoadingDetail(false);
    }
  }

  async function handleSelectInstance(instance: InstanceItem) {
    setSelectedInstance(instance);
    setLoadingDetail(true);
    try {
      const execs = await listExecutions(settings, { instanceId: instance.id, limit: 50 });
      setExecutions(execs.items ?? []);
      const timelineData = await getInstanceTimeline(settings, instance.id);
      setTimeline(timelineData ?? []);
    } catch (err) {
      setStatusMessage(String(err));
    } finally {
      setLoadingDetail(false);
    }
  }

  async function handleRetryExecution(executionId: string) {
    setStatusMessage('Retrying execution...');
    try {
      await retryExecution(settings, executionId);
      await refreshExecutions();
      setStatusMessage('Retry scheduled');
    } catch (err) {
      setStatusMessage(String(err));
    }
  }

  async function handleRetryInstance(instanceId: string) {
    setStatusMessage('Retrying instance...');
    try {
      await retryInstance(settings, instanceId);
      await refreshExecutions();
      await refreshInstances();
      setStatusMessage('Retry scheduled');
    } catch (err) {
      setStatusMessage(String(err));
    }
  }

  async function handleTriggerEvent() {
    setStatusMessage('Submitting event...');
    try {
      const payloadObject = JSON.parse(eventPayload);
      const response = await ingestEvent(settings, {
        event_type: eventType,
        source: eventSource,
        idempotency_key: eventIdempotency || undefined,
        payload: payloadObject,
      });
      setStatusMessage(`Event accepted: ${response.event_id}`);
      await refreshInstances();
    } catch (err) {
      setStatusMessage(String(err));
    }
  }

  function saveConsoleSettings(next: ConsoleSettings) {
    setSettings(next);
    saveSettings(next);
  }

  const failedExecutions = executions.filter((exec) => canRetry(exec.status));
  const failedInstances = instances.filter((inst) => canRetry(inst.status));

  return (
    <div className="min-h-screen bg-slate-50 text-slate-900">
      <div className="flex min-h-screen">
        <aside className="w-64 border-r border-slate-200 bg-white/80 px-6 py-6">
          <div className="mb-10 space-y-1">
            <div className="text-xs uppercase tracking-[0.4em] text-slate-400">Trustage</div>
            <h1 className="font-logo text-2xl font-semibold">Operations Console</h1>
            <p className="text-sm text-slate-500">Hatchet-style workflow control</p>
          </div>

          <nav className="space-y-2">
            {navItems.map((item) => (
              <button
                key={item.key}
                className={`w-full rounded-xl px-4 py-3 text-left text-sm font-medium transition ${
                  activeView === item.key
                    ? 'bg-slate-900 text-white shadow-sm'
                    : 'text-slate-600 hover:bg-slate-100'
                }`}
                onClick={() => setActiveView(item.key)}
              >
                {item.label}
              </button>
            ))}
          </nav>

          <div className="mt-10 rounded-2xl border border-slate-200 bg-white px-4 py-4 text-xs text-slate-500">
            <div className="font-semibold text-slate-700">Connection</div>
            <div className="mt-1 break-all">{settings.apiBaseUrl || 'Not configured'}</div>
            <button
              className="mt-4 w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-xs font-semibold text-slate-700 hover:bg-slate-50"
              onClick={() => setSettingsOpen(true)}
            >
              Settings
            </button>
          </div>
        </aside>

        <div className="flex min-h-screen flex-1 flex-col">
          <header className="flex h-16 items-center justify-between border-b border-slate-200 bg-white/70 px-6">
            <div>
              <div className="text-xs uppercase tracking-[0.4em] text-slate-400">Console</div>
              <div className="text-lg font-semibold text-slate-900">{navItems.find((i) => i.key === activeView)?.label}</div>
            </div>
            <div className="flex items-center gap-4 text-xs text-slate-500">
              <div className="rounded-full bg-slate-100 px-3 py-1">Instances: {instances.length}</div>
              <div className="rounded-full bg-slate-100 px-3 py-1">Executions: {executions.length}</div>
            </div>
          </header>

          <main className="flex-1 gap-6 p-6 lg:grid lg:grid-cols-[minmax(0,1fr)_360px]">
            <section className="space-y-6">
              {activeView === 'overview' && (
                <>
                  <div className="grid gap-4 sm:grid-cols-3">
                    <div className="rounded-2xl border border-slate-200 bg-white px-4 py-4">
                      <p className="text-xs uppercase tracking-[0.3em] text-slate-400">Instances</p>
                      <p className="mt-2 text-2xl font-semibold">{instances.length}</p>
                    </div>
                    <div className="rounded-2xl border border-slate-200 bg-white px-4 py-4">
                      <p className="text-xs uppercase tracking-[0.3em] text-slate-400">Executions</p>
                      <p className="mt-2 text-2xl font-semibold">{executions.length}</p>
                    </div>
                    <div className="rounded-2xl border border-slate-200 bg-white px-4 py-4">
                      <p className="text-xs uppercase tracking-[0.3em] text-slate-400">Failures</p>
                      <p className="mt-2 text-2xl font-semibold">{failedExecutions.length}</p>
                    </div>
                  </div>

                  <div className="rounded-2xl border border-slate-200 bg-white p-5">
                    <div className="flex items-start justify-between">
                      <div>
                        <h2 className="text-base font-semibold">Trigger an event</h2>
                        <p className="text-sm text-slate-500">Send a real payload into Trustage.</p>
                      </div>
                      <button
                        className="rounded-lg bg-slate-900 px-4 py-2 text-xs font-semibold uppercase tracking-[0.2em] text-white hover:bg-slate-800"
                        onClick={handleTriggerEvent}
                      >
                        Send
                      </button>
                    </div>
                    <div className="mt-4 grid gap-4 md:grid-cols-3">
                      <label className="space-y-2 text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">
                        Event Type
                        <input
                          className="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm font-normal text-slate-900"
                          value={eventType}
                          onChange={(event) => setEventType(event.target.value)}
                        />
                      </label>
                      <label className="space-y-2 text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">
                        Source
                        <input
                          className="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm font-normal text-slate-900"
                          value={eventSource}
                          onChange={(event) => setEventSource(event.target.value)}
                        />
                      </label>
                      <label className="space-y-2 text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">
                        Idempotency
                        <input
                          className="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm font-normal text-slate-900"
                          value={eventIdempotency}
                          onChange={(event) => setEventIdempotency(event.target.value)}
                          placeholder="optional"
                        />
                      </label>
                    </div>
                    <label className="mt-4 block text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">
                      Payload
                      <textarea
                        className="mt-2 h-40 w-full rounded-xl border border-slate-200 px-3 py-2 font-mono text-xs text-slate-700"
                        value={eventPayload}
                        onChange={(event) => setEventPayload(event.target.value)}
                      />
                    </label>
                  </div>
                </>
              )}

              {activeView === 'instances' && (
                <div className="rounded-2xl border border-slate-200 bg-white">
                  <div className="flex items-center justify-between px-5 py-4">
                    <div>
                      <h2 className="text-base font-semibold">Workflow Instances</h2>
                      <p className="text-sm text-slate-500">Select an instance to inspect timeline.</p>
                    </div>
                    <button
                      className="rounded-lg border border-slate-200 px-3 py-2 text-xs font-semibold text-slate-700 hover:bg-slate-50"
                      onClick={refreshInstances}
                      disabled={loadingInstances}
                    >
                      {loadingInstances ? 'Refreshing...' : 'Refresh'}
                    </button>
                  </div>
                  <div className="border-t border-slate-100">
                    <div className="grid grid-cols-[1.2fr_1fr_1fr_1fr_auto] gap-3 px-5 py-3 text-xs font-semibold uppercase tracking-[0.2em] text-slate-400">
                      <span>Workflow</span>
                      <span>State</span>
                      <span>Status</span>
                      <span>Started</span>
                      <span />
                    </div>
                    <div className="divide-y divide-slate-100">
                      {instances.map((inst) => (
                        <button
                          key={inst.id}
                          onClick={() => handleSelectInstance(inst)}
                          className="grid w-full grid-cols-[1.2fr_1fr_1fr_1fr_auto] items-center gap-3 px-5 py-3 text-left text-sm hover:bg-slate-50"
                        >
                          <div>
                            <div className="font-medium">{inst.workflow_name}</div>
                            <div className="text-xs text-slate-400">v{inst.workflow_version}</div>
                          </div>
                          <div className="text-slate-600">{inst.current_state}</div>
                          <span className={`w-fit rounded-full px-2 py-1 text-xs font-semibold ${statusTone(inst.status)}`}>
                            {inst.status}
                          </span>
                          <div className="text-xs text-slate-500">{formatDate(inst.started_at)}</div>
                          <div>
                            {canRetry(inst.status) && (
                              <button
                                className="rounded-full border border-slate-200 px-3 py-1 text-xs font-semibold text-slate-700 hover:bg-slate-100"
                                onClick={(event) => {
                                  event.stopPropagation();
                                  handleRetryInstance(inst.id);
                                }}
                              >
                                Retry
                              </button>
                            )}
                          </div>
                        </button>
                      ))}
                    </div>
                  </div>
                </div>
              )}

              {activeView === 'executions' && (
                <div className="rounded-2xl border border-slate-200 bg-white">
                  <div className="flex items-center justify-between px-5 py-4">
                    <div>
                      <h2 className="text-base font-semibold">Executions</h2>
                      <p className="text-sm text-slate-500">Click an execution for full output.</p>
                    </div>
                    <button
                      className="rounded-lg border border-slate-200 px-3 py-2 text-xs font-semibold text-slate-700 hover:bg-slate-50"
                      onClick={refreshExecutions}
                      disabled={loadingExecutions}
                    >
                      {loadingExecutions ? 'Refreshing...' : 'Refresh'}
                    </button>
                  </div>
                  <div className="border-t border-slate-100">
                    <div className="grid grid-cols-[1.2fr_1fr_1fr_0.8fr_auto] gap-3 px-5 py-3 text-xs font-semibold uppercase tracking-[0.2em] text-slate-400">
                      <span>Execution</span>
                      <span>State</span>
                      <span>Status</span>
                      <span>Attempt</span>
                      <span />
                    </div>
                    <div className="divide-y divide-slate-100">
                      {executions.map((exec) => (
                        <button
                          key={exec.execution_id}
                          onClick={() => handleSelectExecution(exec)}
                          className="grid w-full grid-cols-[1.2fr_1fr_1fr_0.8fr_auto] items-center gap-3 px-5 py-3 text-left text-sm hover:bg-slate-50"
                        >
                          <div className="font-medium">{exec.execution_id.slice(0, 8)}</div>
                          <div className="text-slate-600">{exec.state}</div>
                          <span className={`w-fit rounded-full px-2 py-1 text-xs font-semibold ${statusTone(exec.status)}`}>
                            {exec.status}
                          </span>
                          <div className="text-xs text-slate-500">#{exec.attempt}</div>
                          <div>
                            {canRetry(exec.status) && (
                              <button
                                className="rounded-full border border-slate-200 px-3 py-1 text-xs font-semibold text-slate-700 hover:bg-slate-100"
                                onClick={(event) => {
                                  event.stopPropagation();
                                  handleRetryExecution(exec.execution_id);
                                }}
                              >
                                Retry
                              </button>
                            )}
                          </div>
                        </button>
                      ))}
                    </div>
                  </div>
                </div>
              )}

              {activeView === 'workflows' && (
                <div className="rounded-2xl border border-slate-200 bg-white">
                  <div className="flex items-center justify-between px-5 py-4">
                    <div>
                      <h2 className="text-base font-semibold">Workflows</h2>
                      <p className="text-sm text-slate-500">Active workflow definitions.</p>
                    </div>
                    <button
                      className="rounded-lg border border-slate-200 px-3 py-2 text-xs font-semibold text-slate-700 hover:bg-slate-50"
                      onClick={refreshWorkflows}
                      disabled={loadingWorkflows}
                    >
                      {loadingWorkflows ? 'Refreshing...' : 'Refresh'}
                    </button>
                  </div>
                  <div className="border-t border-slate-100">
                    <div className="grid grid-cols-[1.4fr_0.6fr_0.6fr] gap-3 px-5 py-3 text-xs font-semibold uppercase tracking-[0.2em] text-slate-400">
                      <span>Name</span>
                      <span>Version</span>
                      <span>Status</span>
                    </div>
                    <div className="divide-y divide-slate-100">
                      {workflows.map((workflow) => (
                        <div
                          key={workflow.id}
                          className="grid grid-cols-[1.4fr_0.6fr_0.6fr] gap-3 px-5 py-3 text-sm"
                        >
                          <div className="font-medium">{workflow.name}</div>
                          <div className="text-slate-600">v{workflow.version}</div>
                          <span className={`w-fit rounded-full px-2 py-1 text-xs font-semibold ${statusTone(workflow.status)}`}>
                            {workflow.status}
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>
                </div>
              )}
            </section>

            <aside className="mt-6 space-y-6 lg:mt-0">
              <div className="rounded-2xl border border-slate-200 bg-white p-5">
                <div className="flex items-center justify-between">
                  <div>
                    <h3 className="text-base font-semibold">Retry center</h3>
                    <p className="text-sm text-slate-500">One click to re-run failed work.</p>
                  </div>
                </div>
                <div className="mt-4 space-y-3">
                  <button
                    className="w-full rounded-lg bg-slate-900 px-4 py-2 text-sm font-semibold text-white hover:bg-slate-800 disabled:opacity-50"
                    onClick={async () => {
                      for (const exec of failedExecutions) {
                        await handleRetryExecution(exec.execution_id);
                      }
                    }}
                    disabled={failedExecutions.length === 0}
                  >
                    Retry {failedExecutions.length} failed executions
                  </button>
                  <button
                    className="w-full rounded-lg border border-slate-200 px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50 disabled:opacity-50"
                    onClick={async () => {
                      for (const inst of failedInstances) {
                        await handleRetryInstance(inst.id);
                      }
                    }}
                    disabled={failedInstances.length === 0}
                  >
                    Retry {failedInstances.length} failed instances
                  </button>
                </div>
              </div>

              <div className="rounded-2xl border border-slate-200 bg-white p-5">
                <h3 className="text-base font-semibold">Selection</h3>
                <p className="text-sm text-slate-500">
                  {selectedExecution
                    ? `Execution ${selectedExecution.execution_id}`
                    : selectedInstance
                      ? `Instance ${selectedInstance.id}`
                      : 'Pick an instance or execution.'}
                </p>

                <div className="mt-4 space-y-3 text-xs text-slate-500">
                  {loadingDetail && <div>Loading details…</div>}
                  {selectedExecution && (
                    <>
                      <div className="rounded-lg bg-slate-50 px-3 py-2">
                        <div className="text-[10px] uppercase tracking-[0.2em] text-slate-400">State</div>
                        <div className="text-sm text-slate-700">{selectedExecution.state}</div>
                      </div>
                      <div className="rounded-lg bg-slate-50 px-3 py-2">
                        <div className="text-[10px] uppercase tracking-[0.2em] text-slate-400">Input</div>
                        <pre className="mt-2 max-h-48 overflow-auto text-xs text-slate-600">
                          {JSON.stringify(selectedExecution.input_payload ?? {}, null, 2)}
                        </pre>
                      </div>
                      <div className="rounded-lg bg-slate-50 px-3 py-2">
                        <div className="text-[10px] uppercase tracking-[0.2em] text-slate-400">Output</div>
                        <pre className="mt-2 max-h-48 overflow-auto text-xs text-slate-600">
                          {JSON.stringify(selectedExecution.output ?? {}, null, 2)}
                        </pre>
                      </div>
                      {canRetry(selectedExecution.status) && (
                        <button
                          className="w-full rounded-lg bg-slate-900 px-4 py-2 text-sm font-semibold text-white hover:bg-slate-800"
                          onClick={() => handleRetryExecution(selectedExecution.execution_id)}
                        >
                          Retry this execution
                        </button>
                      )}
                    </>
                  )}
                  {selectedInstance && !selectedExecution && (
                    <>
                      <div className="rounded-lg bg-slate-50 px-3 py-2">
                        <div className="text-[10px] uppercase tracking-[0.2em] text-slate-400">Current state</div>
                        <div className="text-sm text-slate-700">{selectedInstance.current_state}</div>
                      </div>
                      {canRetry(selectedInstance.status) && (
                        <button
                          className="w-full rounded-lg bg-slate-900 px-4 py-2 text-sm font-semibold text-white hover:bg-slate-800"
                          onClick={() => handleRetryInstance(selectedInstance.id)}
                        >
                          Retry this instance
                        </button>
                      )}
                    </>
                  )}
                </div>
              </div>

              <div className="rounded-2xl border border-slate-200 bg-white p-5">
                <h3 className="text-base font-semibold">Timeline</h3>
                <p className="text-sm text-slate-500">Recent state transitions.</p>
                <div className="mt-4 space-y-3 text-xs text-slate-500">
                  {timeline.length === 0 && <div>No timeline events yet.</div>}
                  {timeline.map((entry, index) => (
                    <div key={`${entry.event_type}-${index}`} className="rounded-lg border border-slate-100 bg-slate-50 px-3 py-2">
                      <div className="text-[10px] uppercase tracking-[0.2em] text-slate-400">{entry.event_type}</div>
                      <div className="mt-1 text-sm text-slate-700">{entry.state ?? '—'}</div>
                      <div className="mt-1 text-[11px] text-slate-500">{formatDate(entry.created_at)}</div>
                    </div>
                  ))}
                </div>
              </div>
            </aside>
          </main>
        </div>
      </div>

      {settingsOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 px-4">
          <div className="w-full max-w-lg rounded-2xl bg-white p-6 shadow-xl">
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold">Console settings</h2>
              <button
                className="rounded-full border border-slate-200 px-3 py-1 text-xs font-semibold text-slate-600"
                onClick={() => setSettingsOpen(false)}
              >
                Close
              </button>
            </div>
            <div className="mt-4 space-y-4">
              <label className="block text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">
                API Base URL
                <input
                  className="mt-2 w-full rounded-lg border border-slate-200 px-3 py-2 text-sm"
                  placeholder="https://trustage.api"
                  value={settings.apiBaseUrl}
                  onChange={(event) => saveConsoleSettings({ ...settings, apiBaseUrl: event.target.value })}
                />
              </label>
              <label className="block text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">
                Auth Token
                <input
                  className="mt-2 w-full rounded-lg border border-slate-200 px-3 py-2 text-sm"
                  placeholder="Bearer token"
                  value={settings.authToken}
                  onChange={(event) => saveConsoleSettings({ ...settings, authToken: event.target.value })}
                />
              </label>
            </div>
            {statusMessage && (
              <div className="mt-4 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-xs text-slate-600">
                {statusMessage}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
