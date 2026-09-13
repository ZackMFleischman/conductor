import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import type { BoardSource } from './board';
import { fixtureBoardSource } from './data/fixtureBoardSource';
import { App } from './App';

describe('read-only work board', () => {
  it('shows all lanes, complete descriptions, ownership and explicit blockers', async () => {
    render(<App source={fixtureBoardSource} />);
    expect(await screen.findByRole('heading', { name: 'Work board' })).toBeInTheDocument();
    for (const lane of ['Ready', 'In progress', 'Blocked', 'Review', 'Done']) {
      expect(await screen.findByRole('region', { name: lane })).toBeInTheDocument();
    }
    expect(screen.getByText('Fixture data')).toBeInTheDocument();
    expect(screen.getByText('Read only')).toBeInTheDocument();
    expect(screen.getAllByRole('article')).toHaveLength(8);
    expect(screen.getAllByText('Unassigned')).not.toHaveLength(0);
    expect(screen.getByText('Waiting for an agreed read API contract.')).toBeInTheDocument();
    expect(screen.getByText((await fixtureBoardSource.load()).tickets[0].description)).toBeInTheDocument();
  });
  it('combines case-insensitive search with assignment and clears to restore the board', async () => {
    const user = userEvent.setup();
    render(<App source={fixtureBoardSource} />);
    await screen.findByText('DEMO-101');
    await user.type(screen.getByRole('textbox', { name: 'Search tickets' }), '  WAITING  ');
    expect(screen.getAllByRole('article')).toHaveLength(1);
    expect(within(screen.getByRole('article')).getByText('DEMO-105')).toBeInTheDocument();
    await user.selectOptions(screen.getByRole('combobox', { name: 'Assignee' }), 'unassigned');
    expect(screen.queryAllByRole('article')).toHaveLength(0);
    expect(screen.getByText('No tickets match these filters')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Clear filters' }));
    expect(screen.getAllByRole('article')).toHaveLength(8);
    await user.selectOptions(screen.getByRole('combobox', { name: 'Assignee' }), screen.getByRole('option', { name: 'docs-agent' }));
    expect(screen.getAllByRole('article')).toHaveLength(2);
  });
  it('distinguishes a valid empty board from filtered empty results', async () => {
    const snapshot = await fixtureBoardSource.load();
    render(<App source={{ kind: 'fixture', load: async () => ({ ...snapshot, tickets: [] }) }} />);
    expect(await screen.findByText('No tickets yet')).toBeInTheDocument();
    expect(screen.queryByText('No tickets match these filters')).not.toBeInTheDocument();
  });
  it('shows a safe error and retries the same source without a fixture fallback', async () => {
    let calls = 0;
    const source: BoardSource = { kind: 'live', load: async () => {
      if (calls++ === 0) throw new Error('private backend details');
      return fixtureBoardSource.load();
    } };
    const user = userEvent.setup();
    render(<App source={source} />);
    expect(await screen.findByRole('alert')).toHaveTextContent('Could not load the board');
    expect(screen.queryByText('private backend details')).not.toBeInTheDocument();
    expect(screen.queryByText('Fixture data')).not.toBeInTheDocument();
    expect(screen.queryAllByRole('article')).toHaveLength(0);
    await user.click(screen.getByRole('button', { name: 'Try again' }));
    expect(await screen.findByText('DEMO-101')).toBeInTheDocument();
    expect(calls).toBe(2);
  });
  it('labels loading and cancels when unmounted', () => {
    let signal: AbortSignal | undefined;
    const source: BoardSource = { kind: 'fixture', load: incoming => {
      signal = incoming;
      return new Promise(() => {});
    } };
    const { unmount } = render(<App source={source} />);
    expect(screen.getByRole('status')).toHaveTextContent('Loading board');
    expect(screen.getByText('Fixture data')).toBeInTheDocument();
    unmount();
    expect(signal?.aborted).toBe(true);
  });
  it('renders untrusted strings as plain text and supplies missing detail fallbacks', async () => {
    const snapshot = await fixtureBoardSource.load();
    snapshot.tickets = [{ ...snapshot.tickets[0], title: '<img src=x onerror=alert(1)>', description: '', status: 'blocked', blockers: [] }];
    render(<App source={{ kind: 'fixture', load: async () => snapshot }} />);
    expect(await screen.findByText('<img src=x onerror=alert(1)>')).toBeInTheDocument();
    expect(screen.getByText('No description provided.')).toBeInTheDocument();
    expect(screen.getByText('Blocker details unavailable.')).toBeInTheDocument();
    expect(screen.queryByRole('img')).not.toBeInTheDocument();
  });
});

it('keeps opaque assignee IDs distinct from special filter options', async () => {
  const snapshot = await fixtureBoardSource.load();
  snapshot.tickets = [
    { ...snapshot.tickets[0], assignee: { id: 'all', name: 'Owner named all' } },
    { ...snapshot.tickets[1], assignee: { id: 'unassigned', name: 'Owner named unassigned' } },
    { ...snapshot.tickets[2], assignee: null },
  ];
  const user = userEvent.setup();
  render(<App source={{ kind: 'fixture', load: async () => snapshot }} />);
  await screen.findByText('DEMO-101');
  await user.selectOptions(screen.getByRole('combobox', { name: 'Assignee' }), screen.getByRole('option', { name: 'Owner named all' }));
  expect(screen.getAllByRole('article')).toHaveLength(1);
  expect(screen.getByText('DEMO-101')).toBeInTheDocument();
  await user.selectOptions(screen.getByRole('combobox', { name: 'Assignee' }), screen.getByRole('option', { name: 'Owner named unassigned' }));
  expect(screen.getAllByRole('article')).toHaveLength(1);
  expect(screen.getByText('DEMO-102')).toBeInTheDocument();
});
