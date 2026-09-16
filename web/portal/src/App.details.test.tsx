import { act, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { expect, it, vi } from 'vitest';
import { App } from './App';
import type { BoardSnapshot, BoardSource } from './board';
import { HighlightedText } from './components/HighlightedText';

it('highlights complete phrases across ticket references', () => {
  const { container } = render(<HighlightedText text="Fix P-45 handling" query="Fix P-45" />);
  expect([...container.querySelectorAll('mark')].map(node => node.textContent).join('')).toBe('Fix P-45');
});

const snapshot = (count = 45): BoardSnapshot => ({ project: { id: 'p', name: 'Project', description: '' }, tickets: Array.from({ length: count }, (_, i) => ({ id: String(i), key: `P-${i + 1}`, title: `Ticket ${i + 1}`, description: i === 0 ? 'See P-45 and **important details**.' : 'Description', status: 'done', assignee: null, ancestors: [], blockers: [], kind: 'bug', summary: 'Delivered summary' })) });

it('shows live comment counts on cards and collapsed details without fetching history', async () => {
  const user = userEvent.setup();
  let board = snapshot(2);
  board.tickets = board.tickets.map((ticket, i) => ({ ...ticket, commentCount: i ? 0 : 3 }));
  let refresh = () => {};
  const loadTicketNotes = vi.fn();
  render(<App source={{ kind: 'live', load: async () => board, loadTicketNotes, subscribe: change => { refresh = change; return () => {}; } }} />);
  const card = await screen.findByRole('article', { name: 'Ticket 1' });
  expect(within(card).getByText('3 comments')).toBeInTheDocument();
  expect(within(screen.getByRole('article', { name: 'Ticket 2' })).queryByText(/comments?/)).not.toBeInTheDocument();
  await user.click(within(card).getByRole('heading'));
  expect(within(screen.getByRole('dialog')).getByRole('button', { name: 'Comments (3)' })).toHaveAttribute('aria-expanded', 'false');
  board = { ...board, tickets: board.tickets.map((ticket, i) => ({ ...ticket, commentCount: i ? 0 : 4 })) };
  await act(async () => refresh());
  expect(within(screen.getByRole('dialog')).getByRole('button', { name: 'Comments (4)' })).toBeInTheDocument();
  expect(within(card).getByText('4 comments')).toBeInTheDocument();
  expect(loadTicketNotes).not.toHaveBeenCalled();
});

it('caps Done at20, expands by20, shows all, and keeps the total count', async () => {
  const user = userEvent.setup();
  render(<App source={{ kind: 'fixture', load: async () => snapshot() }} />);
  const done = await screen.findByRole('region', { name: 'Done' });
  expect(within(done).getAllByRole('article')).toHaveLength(20);
  expect(within(done).getByLabelText('45 tickets')).toBeInTheDocument();
  await user.click(within(done).getByRole('button', { name: 'Show more' }));
  expect(within(done).getAllByRole('article')).toHaveLength(40);
  await user.click(within(done).getByRole('button', { name: 'Show all' }));
  expect(within(done).getAllByRole('article')).toHaveLength(45);
}, 15000);

it('orders Done by completion time before paging, unaffected by later comments', async () => {
  const board = snapshot();
  board.tickets = board.tickets.map((ticket, i) => ({ ...ticket, completedAt: new Date(Date.UTC(2026, 0, i + 1)).toISOString(), updatedAt: i ? '2026-01-01T00:00:00Z' : '2027-01-01T00:00:00Z' }));
  render(<App source={{ kind: 'fixture', load: async () => board }} />);
  const done = await screen.findByRole('region', { name: 'Done' });
  expect(within(done).getAllByRole('article')[0]).toHaveAccessibleName('Ticket 45');
  expect(within(done).getAllByRole('article')).toHaveLength(20);
  expect(within(done).queryByRole('article', { name: 'Ticket 1' })).not.toBeInTheDocument();
});

it('opens full Markdown details and follows references to hidden tickets', async () => {
  const user = userEvent.setup();
  render(<App source={{ kind: 'fixture', load: async () => snapshot() }} />);
  const card = await screen.findByRole('article', { name: 'Ticket 1' });
  expect(within(card).getByText('bug')).toBeInTheDocument();
  await user.click(within(card).getByRole('heading'));
  let dialog = await screen.findByRole('dialog');
  expect(within(dialog).getByText('important details').tagName).toBe('STRONG');
  expect(within(dialog).getByText('Delivered summary')).toBeInTheDocument();
  await user.click(within(dialog).getByRole('link', { name: 'P-45' }));
  dialog = screen.getByRole('dialog');
  expect(within(dialog).getByRole('heading', { name: /Ticket 45/ })).toBeInTheDocument();
  await user.keyboard('{Escape}');
  await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
});

it('records live changes after the initial snapshot, skips unchanged refreshes, and links details', async () => {
  const user = userEvent.setup();
  let board = snapshot(2);
  let failing = false;
  let refresh = () => {};
  const source: BoardSource = { kind: 'live', load: async () => { if (failing) throw new Error('offline'); return board; }, subscribe: change => { refresh = change; return () => {}; } };
  render(<App source={source} />);
  await screen.findByRole('article', { name: 'Ticket 1' });
  await user.click(screen.getByRole('button', { name: /Activity/ }));
  const feed = screen.getByRole('complementary', { name: 'Activity feed' });
  expect(within(screen.getByRole('banner')).queryByRole('button', { name: /Activity/ })).not.toBeInTheDocument();
  expect(within(feed).queryByText(/Observed since/)).not.toBeInTheDocument();
  await waitFor(() => expect(within(feed).getByRole('button', { name: 'Collapse' })).toHaveFocus());
  expect(within(feed).getByText(/Waiting for ticket changes/)).toBeInTheDocument();
  board = { ...board, tickets: board.tickets.map((t, i) => i ? t : { ...t, status: 'review' }) };
  await act(async () => refresh());
  expect(within(feed).getByText(/Done → Review/)).toBeInTheDocument();
  await act(async () => refresh());
  expect(within(feed).getAllByRole('listitem')).toHaveLength(1);
  failing = true;
  await act(async () => refresh());
  await screen.findByRole('alert');
  board = { ...board, tickets: board.tickets.map((t, i) => i ? t : { ...t, status: 'ready' }) };
  failing = false;
  await user.click(screen.getByRole('button', { name: 'Try again' }));
  const retriedFeed = await screen.findByRole('complementary', { name: 'Activity feed' });
  await waitFor(() => expect(within(retriedFeed).getAllByRole('listitem')).toHaveLength(2));
  expect(within(retriedFeed).getByText('Review → Ready')).toBeInTheDocument();
  await user.click(within(retriedFeed).getByRole('button', { name: 'Archive all' }));
  expect(within(retriedFeed).queryAllByRole('listitem')).toHaveLength(0);
  expect(within(retriedFeed).getByText(/All caught up/)).toBeInTheDocument();
  board = { ...board, tickets: board.tickets.map((t, i) => i ? t : { ...t, status: 'in_progress' }) };
  await act(async () => refresh());
  await waitFor(() => expect(within(retriedFeed).getAllByRole('listitem')).toHaveLength(1));
  await user.click(within(retriedFeed).getByRole('button', { name: 'Show archived (2)' }));
  expect(within(retriedFeed).getAllByRole('listitem')).toHaveLength(3);
  await user.click(within(retriedFeed).getByRole('button', { name: 'Hide archived' }));
  expect(within(retriedFeed).getAllByRole('listitem')).toHaveLength(1);
  await user.click(within(retriedFeed).getByRole('button', { name: 'Collapse' }));
  expect(screen.getByRole('button', { name: 'Activity (1)' })).toHaveFocus();
  await user.click(screen.getByRole('button', { name: 'Activity (1)' }));
  await user.click(within(retriedFeed).getAllByRole('link', { name: /P-1/ })[0]);
  expect(await screen.findByRole('dialog')).toHaveTextContent('Ticket 1');
});
