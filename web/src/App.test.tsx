import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { App } from './App';

const api = vi.hoisted(() => ({
  approveFlow: vi.fn(),
  createFlow: vi.fn(),
  getFlow: vi.fn(),
}));

vi.mock('./api/generated/sdk.gen', () => api);

describe('App', () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.clearAllMocks();
  });

  afterEach(cleanup);

  it('introduces the runnable approval automation without production mock controls', () => {
    render(<App />);
    expect(screen.getByRole('heading', { name: /approval automation/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /start process/i })).toBeEnabled();
    expect(screen.queryByText(/mock controls/i)).not.toBeInTheDocument();
  });

  it('shows the development controls only in mock mode', () => {
    render(<App mockMode />);
    expect(screen.getByText(/mock controls/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /fail next start/i })).toBeEnabled();
    expect(screen.getByRole('button', { name: /reset/i })).toBeEnabled();
  });

  it('shows a visible loading state while a process starts', () => {
    api.createFlow.mockReturnValue(new Promise(() => {}));
    render(<App />);
    fireEvent.click(screen.getByRole('button', { name: /start process/i }));
    expect(screen.getByRole('button', { name: /starting/i })).toBeDisabled();
  });

  it('shows approval loading after a restored waiting Flow', async () => {
    window.localStorage.setItem('dex-basic-process-flow-id', 'process-00000000-0000-4000-8000-000000000000');
    api.getFlow.mockResolvedValue({ data: waitingFlow });
    api.approveFlow.mockReturnValue(new Promise(() => {}));
    render(<App />);
    fireEvent.click(await screen.findByRole('button', { name: 'Approve' }));
    expect(screen.getByRole('button', { name: /approving/i })).toBeDisabled();
  });

  it('pauses a failed refresh and resumes it through Retry', async () => {
    window.localStorage.setItem('dex-basic-process-flow-id', 'process-00000000-0000-4000-8000-000000000000');
    api.getFlow
      .mockResolvedValueOnce({ data: waitingFlow })
      .mockResolvedValueOnce({ error: { error: 'mock_get_failed', message: 'mock refresh failure' } })
      .mockResolvedValueOnce({ data: waitingFlow });
    render(<App mockMode />);
    expect(await screen.findByRole('alert')).toHaveTextContent('mock refresh failure');
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
    await waitFor(() => expect(screen.queryByRole('alert')).not.toBeInTheDocument());
    expect(screen.getByRole('button', { name: 'Approve' })).toBeEnabled();
  });

  it('clears a restored Flow when the restarted server no longer has it', async () => {
    window.localStorage.setItem('dex-basic-process-flow-id', 'process-00000000-0000-4000-8000-000000000000');
    api.getFlow.mockResolvedValue({ error: { error: 'unknown_flow', message: 'flow was not found' } });
    render(<App mockMode />);
    await waitFor(() => expect(window.localStorage.getItem('dex-basic-process-flow-id')).toBeNull());
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });
});

const waitingFlow = {
  flowId: 'process-00000000-0000-4000-8000-000000000000',
  title: 'Review the launch checklist',
  state: 'waiting_for_approval' as const,
  reminderCount: 0,
};
