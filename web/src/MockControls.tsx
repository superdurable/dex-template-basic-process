import { useState } from 'react';
import type { FlowView, ProcessState } from './api/generated/types.gen';

type MockAction =
  | 'reset'
  | 'advance'
  | 'emit_reminder'
  | 'fail_next_create'
  | 'fail_next_get'
  | 'fail_next_approve';

type MockControlView = {
  mode: 'mock';
  flow?: FlowView;
  pendingFailures: Array<'create' | 'get' | 'approve'>;
};

type MockControlsProps = {
  flow?: FlowView;
  onFlowChange: (flow: FlowView) => void;
  onPauseRefresh: () => void;
  onRefresh: () => Promise<void>;
  onReset: () => void;
};

const reminderStates = new Set<ProcessState>(['waiting_for_approval', 'reminder_emitted']);

export function MockControls({ flow, onFlowChange, onPauseRefresh, onRefresh, onReset }: MockControlsProps) {
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState('In-memory data survives browser refreshes.');

  async function runControl(action: MockAction) {
    setBusy(true);
    try {
      if (action === 'fail_next_get') onPauseRefresh();
      const response = await fetch('/__mock__/control', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action, flowId: flow?.flowId }),
      });
      const result = await response.json() as MockControlView | { message?: string };
      if (!response.ok) throw new Error('message' in result ? result.message : 'Mock control failed.');
      const control = result as MockControlView;
      if (action === 'reset') onReset();
      else if (control.flow) onFlowChange(control.flow);
      if (action === 'fail_next_get') await onRefresh();
      const pending = control.pendingFailures.length > 0
        ? ` Pending failure: ${control.pendingFailures.join(', ')}.`
        : '';
      setMessage(`Applied ${action.replaceAll('_', ' ')}.${pending}`);
    } catch (error) {
      setMessage(error instanceof Error ? error.message : 'Mock control failed.');
    } finally {
      setBusy(false);
    }
  }

  return (
    <details className="mock-controls" open>
      <summary>Mock controls</summary>
      <p>UI-only simulation. Real Dex durability is verified by <code>make check</code>.</p>
      <div className="mock-control-actions">
        {!flow && <button disabled={busy} type="button" onClick={() => runControl('fail_next_create')}>Fail next Start</button>}
        {flow && <button disabled={busy || flow.state === 'completed'} type="button" onClick={() => runControl('advance')}>Advance</button>}
        {flow && <button disabled={busy || !reminderStates.has(flow.state)} type="button" onClick={() => runControl('emit_reminder')}>Emit reminder</button>}
        {flow && <button disabled={busy} type="button" onClick={() => runControl('fail_next_get')}>Fail next Refresh</button>}
        {flow && <button disabled={busy || !reminderStates.has(flow.state)} type="button" onClick={() => runControl('fail_next_approve')}>Fail next Approval</button>}
        <button className="secondary" disabled={busy} type="button" onClick={() => runControl('reset')}>Reset</button>
      </div>
      <output aria-live="polite">{message}</output>
    </details>
  );
}
