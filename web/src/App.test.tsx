import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { App } from './App';

describe('App', () => {
  it('introduces the runnable approval automation', () => {
    render(<App />);
    expect(screen.getByRole('heading', { name: /approval automation/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /start process/i })).toBeEnabled();
  });
});
