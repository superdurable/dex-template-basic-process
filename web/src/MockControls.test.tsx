import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { MockControls } from './MockControls';

const callbacks = {
  onFlowChange: vi.fn(),
  onPauseRefresh: vi.fn(),
  onRefresh: vi.fn().mockResolvedValue(undefined),
  onReset: vi.fn(),
};

describe('MockControls', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ mode: 'mock', pendingFailures: [] }),
    }));
  });

  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
  });

  it('offers only start failure and reset before a Flow exists', () => {
    render(<MockControls {...callbacks} />);
    expect(screen.getByRole('button', { name: 'Fail next Start' })).toBeEnabled();
    expect(screen.getByRole('button', { name: 'Reset' })).toBeEnabled();
    expect(screen.queryByRole('button', { name: 'Advance' })).not.toBeInTheDocument();
  });

  it('enables controls according to the current Flow state', () => {
    const { rerender } = render(<MockControls {...callbacks} flow={waitingFlow} />);
    expect(screen.getByRole('button', { name: 'Advance' })).toBeEnabled();
    expect(screen.getByRole('button', { name: 'Emit reminder' })).toBeEnabled();
    expect(screen.getByRole('button', { name: 'Fail next Approval' })).toBeEnabled();

    rerender(<MockControls {...callbacks} flow={{ ...waitingFlow, state: 'completed', result: 'done' }} />);
    expect(screen.getByRole('button', { name: 'Advance' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Emit reminder' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Fail next Approval' })).toBeDisabled();
  });

  it('disables controls while applying Reset and clears the UI', async () => {
    let resolveFetch: (value: Response) => void = () => {};
    vi.mocked(fetch).mockReturnValue(new Promise((resolve) => { resolveFetch = resolve; }) as Promise<Response>);
    render(<MockControls {...callbacks} flow={waitingFlow} />);
    fireEvent.click(screen.getByRole('button', { name: 'Reset' }));
    expect(screen.getByRole('button', { name: 'Reset' })).toBeDisabled();
    resolveFetch({ ok: true, json: () => Promise.resolve({ mode: 'mock', pendingFailures: [] }) } as Response);
    await waitFor(() => expect(callbacks.onReset).toHaveBeenCalledOnce());
  });
});

const waitingFlow = {
  flowId: 'process-00000000-0000-4000-8000-000000000000',
  title: 'Review the launch checklist',
  state: 'waiting_for_approval' as const,
  reminderCount: 0,
};
