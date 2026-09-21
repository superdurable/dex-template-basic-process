import { FormEvent, useEffect, useMemo, useState } from 'react';
import { approveFlow, createFlow, getFlow } from './api/generated/sdk.gen';
import type { FlowView, ProcessState } from './api/generated/types.gen';

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

export function App() {
  const [title, setTitle] = useState('Review the launch checklist');
  const [flow, setFlow] = useState<FlowView>();
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!flow || flow.state === 'completed') return;
    const timer = window.setInterval(async () => {
      const response = await getFlow({ path: { flowId: flow.flowId } });
      if (response.data) setFlow(response.data);
    }, 750);
    return () => window.clearInterval(timer);
  }, [flow]);

  const activeRank = useMemo(() => (flow ? stateRank[flow.state] : -1), [flow]);

  async function start(event: FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError('');
    const response = await createFlow({ body: { title } });
    if (response.data) setFlow(response.data);
    else setError(response.error?.message ?? 'Unable to start the automation.');
    setBusy(false);
  }

  async function approve() {
    if (!flow) return;
    setBusy(true);
    setError('');
    const response = await approveFlow({ path: { flowId: flow.flowId }, body: { approved: true } });
    if (response.data) setFlow(response.data);
    else setError(response.error?.message ?? 'Unable to approve the automation.');
    setBusy(false);
  }

  return (
    <main>
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
            <button disabled={busy || title.trim() === ''} type="submit">Start process</button>
          </div>
        </form>
        {error && <p role="alert" className="error">{error}</p>}
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
            {approvableStates.has(flow.state) && <button disabled={busy} onClick={approve}>Approve</button>}
          </div>
          {flow.result && <p className="result" data-testid="result">{flow.result}</p>}
        </section>
      )}
    </main>
  );
}
