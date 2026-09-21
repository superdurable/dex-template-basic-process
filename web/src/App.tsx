import { FormEvent, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { approveFlow, createFlow, getFlow } from './api/generated/sdk.gen';
import type { FlowView, ProcessState } from './api/generated/types.gen';
import { MockControls } from './MockControls';

const storedFlowID = 'dex-basic-process-flow-id';

const steps: Array<{ state: ProcessState; label: string }> = [
  { state: 'started', label: 'Start process' },
  { state: 'validated', label: 'Validate request' },
  { state: 'waiting_for_approval', label: 'Wait for approval' },
  { state: 'reminder_emitted', label: 'Emit reminder' },
  { state: 'executing', label: 'Execute automation' },
  { state: 'completed', label: 'Complete process' },
];

const stateRank: Record<ProcessState, number> = {
  started: 0,
  validated: 1,
  waiting_for_approval: 2,
  reminder_emitted: 3,
  approved: 4,
  executing: 4,
  completed: 5,
};

const approvableStates = new Set<ProcessState>(['waiting_for_approval', 'reminder_emitted']);

type AppProps = { mockMode?: boolean };

function responseMessage(error: unknown, fallback: string) {
  if (error && typeof error === 'object' && 'message' in error && typeof error.message === 'string') {
    return error.message;
  }
  return fallback;
}

function responseCode(error: unknown) {
  return error && typeof error === 'object' && 'error' in error && typeof error.error === 'string'
    ? error.error
    : '';
}

export function App({ mockMode = false }: AppProps) {
  const [title, setTitle] = useState('Review the launch checklist');
  const [flow, setFlow] = useState<FlowView>();
  const [actionError, setActionError] = useState('');
  const [refreshError, setRefreshError] = useState('');
  const [refreshPaused, setRefreshPaused] = useState(false);
  const [busy, setBusy] = useState<'start' | 'approve' | ''>('');
  const restored = useRef(false);

  const refreshFlow = useCallback(async (flowId: string, resume = false) => {
    if (resume) setRefreshPaused(false);
    const response = await getFlow({ path: { flowId } });
    if (response.data) {
      setFlow(response.data);
      setRefreshError('');
      setRefreshPaused(false);
      return;
    }
    if (responseCode(response.error) === 'unknown_flow') {
      window.localStorage.removeItem(storedFlowID);
      setFlow(undefined);
      setRefreshError('');
      setRefreshPaused(false);
      return;
    }
    setRefreshError(responseMessage(response.error, 'Unable to refresh the automation.'));
    setRefreshPaused(true);
  }, []);

  useEffect(() => {
    if (restored.current) return;
    restored.current = true;
    const flowId = window.localStorage.getItem(storedFlowID);
    if (flowId) void refreshFlow(flowId);
  }, [refreshFlow]);

  useEffect(() => {
    if (flow) window.localStorage.setItem(storedFlowID, flow.flowId);
  }, [flow]);

  useEffect(() => {
    if (!flow || flow.state === 'completed' || refreshPaused) return;
    const timer = window.setInterval(
      () => void refreshFlow(flow.flowId),
      mockMode ? 250 : 750,
    );
    return () => window.clearInterval(timer);
  }, [flow, mockMode, refreshFlow, refreshPaused]);

  const activeRank = useMemo(() => (flow ? stateRank[flow.state] : -1), [flow]);

  async function start(event: FormEvent) {
    event.preventDefault();
    setBusy('start');
    setActionError('');
    setRefreshError('');
    setRefreshPaused(false);
    const response = await createFlow({ body: { title } });
    if (response.data) setFlow(response.data);
    else setActionError(responseMessage(response.error, 'Unable to start the automation.'));
    setBusy('');
  }

  async function approve() {
    if (!flow) return;
    setBusy('approve');
    setActionError('');
    const response = await approveFlow({ path: { flowId: flow.flowId }, body: { approved: true } });
    if (response.data) setFlow(response.data);
    else setActionError(responseMessage(response.error, 'Unable to approve the automation.'));
    setBusy('');
  }

  function resetUI() {
    window.localStorage.removeItem(storedFlowID);
    setFlow(undefined);
    setActionError('');
    setRefreshError('');
    setRefreshPaused(false);
    setBusy('');
  }

  return (
    <main>
      {mockMode && (
        <MockControls
          flow={flow}
          onFlowChange={setFlow}
          onPauseRefresh={() => setRefreshPaused(true)}
          onRefresh={() => flow ? refreshFlow(flow.flowId) : Promise.resolve()}
          onReset={resetUI}
        />
      )}
      <header>
        <p className="eyebrow">SUPERDURABLE DEX</p>
        <h1>Approval automation that survives everything.</h1>
        <p className="lede">A complete six-step durable process with human approval and recurring reminders.</p>
      </header>

      <section className="panel">
        <form onSubmit={start}>
          <label htmlFor="title">Automation request</label>
          <div className="form-row">
            <input id="title" value={title} maxLength={120} onChange={(event) => setTitle(event.target.value)} />
            <button disabled={busy !== '' || title.trim() === ''} type="submit">{busy === 'start' ? 'Starting…' : 'Start process'}</button>
          </div>
        </form>
        {actionError && <p role="alert" className="error">{actionError}</p>}
        {refreshError && (
          <div role="alert" className="refresh-error">
            <span>{refreshError}</span>
            <button className="secondary" type="button" onClick={() => flow && refreshFlow(flow.flowId, true)}>Retry</button>
          </div>
        )}
      </section>

      {flow && (
        <section className="panel process" data-flow-id={flow.flowId}>
          <div className="process-heading">
            <div><p className="eyebrow">LIVE FLOW</p><h2>{flow.title}</h2></div>
            <span className="status">{flow.state.replaceAll('_', ' ')}</span>
          </div>
          <ol className="timeline">
            {steps.map((step, index) => (
              <li key={step.state} className={index <= activeRank ? 'reached' : ''}>
                <span>{index + 1}</span><strong>{step.label}</strong>
              </li>
            ))}
          </ol>
          <div className="actions">
            <p>Reminders emitted: <strong data-testid="reminder-count">{flow.reminderCount}</strong></p>
            {approvableStates.has(flow.state) && <button disabled={busy !== ''} onClick={approve}>{busy === 'approve' ? 'Approving…' : 'Approve'}</button>}
          </div>
          {flow.result && <p className="result" data-testid="result">{flow.result}</p>}
        </section>
      )}
    </main>
  );
}
