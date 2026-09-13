import { act, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { expect, it } from 'vitest';
import { App } from './App';
import type { BoardSnapshot, BoardSource, ProblemDetail } from './board';
import { decodeProblem } from './data/liveBoardSource';

const report: ProblemDetail = { id: 'report', key: 'CON-P47', summary: 'Observer missed a workflow issue', ticketKey: 'CON-1', createdAt: '2026-09-13T10:00:00Z', updatedAt: '2026-09-13T10:00:00Z', noteCount: 0, expected: '**Expected** observation', actual: 'Missed CON-1', correction: 'Proposed remedy', evidence: 'Recorded evidence', reporter: 'observer', notes: [] };
const initial: BoardSnapshot = { project: { id: 'p', name: 'CON', description: '' }, tickets: [{ id: 't', key: 'CON-1', title: 'Remediate the observer', description: 'Fix CON-P47', status: 'ready', assignee: null, blockers: [], ancestors: [] }], problems: [report] };

it('browses reports separately, opens Markdown details and navigates related tickets and reports', async () => {
  const user = userEvent.setup();
  let loads = 0;
  render(<App source={{ kind: 'fixture', load: async () => initial, loadProblem: async () => { loads++; return report; } }} />);
  await screen.findByRole('article', { name: 'Remediate the observer' });
  expect(loads).toBe(0);
  await user.click(screen.getByRole('tab', { name: 'Problems (1)' }));
  expect(screen.queryByRole('region', { name: 'Ready' })).not.toBeInTheDocument();
  await user.type(screen.getByRole('textbox', { name: 'Search problem reports' }), 'CON-P47');
  await user.click(screen.getByRole('link', { name: 'CON-P47' }));
  let dialog = await screen.findByRole('dialog');
  expect(await within(dialog).findByText('Expected', { selector: 'strong' })).toBeInTheDocument();
  await user.click(within(dialog).getAllByRole('link', { name: 'CON-1' })[0]);
  dialog = await screen.findByRole('dialog', { name: /CON-1/ });
  expect(within(dialog).getByRole('heading', { name: 'Related problems (1)' })).toBeInTheDocument();
  await user.click(within(dialog).getByRole('link', { name: 'CON-P47' }));
  expect(await screen.findByRole('dialog', { name: /CON-P47/ })).toBeInTheDocument();
});

it('refreshes an open report and links live report updates from activity', async () => {
  const user = userEvent.setup();
  let detail = report;
  let board = initial;
  let refresh = () => {};
  const source: BoardSource = { kind: 'live', load: async () => board, loadProblem: async () => detail, subscribe: callback => { refresh = callback; return () => {}; } };
  render(<App source={source} />);
  await screen.findByRole('tab', { name: 'Problems (1)' });
  await user.click(screen.getByRole('tab', { name: 'Problems (1)' }));
  await user.click(screen.getByRole('link', { name: 'CON-P47' }));
  await screen.findByText('No follow-up updates yet.');
  detail = { ...report, noteCount: 1, updatedAt: '2026-09-13T11:00:00Z', notes: [{ id: 'n', body: 'New verification result', createdAt: '2026-09-13T11:00:00Z', reporter: 'reviewer' }] };
  board = { ...initial, problems: [detail] };
  await act(async () => refresh());
  expect(await screen.findByText('New verification result')).toBeInTheDocument();
  await user.keyboard('{Escape}');
  await user.click(screen.getByRole('button', { name: /Activity/ }));
  const feed = screen.getByRole('complementary', { name: 'Activity feed' });
  expect(within(feed).getByText('Problem report update added')).toBeInTheDocument();
  await user.click(within(feed).getByRole('link', { name: /CON-P47/ }));
  await waitFor(() => expect(screen.getByRole('dialog', { name: /CON-P47/ })).toBeInTheDocument());
});

it('rejects a mismatched problem detail identity', () => {
  expect(() => decodeProblem(report, 'different')).toThrow();
  expect(decodeProblem(report, 'report').reporter).toBe('observer');
});

it('keeps keyboard close working after retrying a failed report load', async () => {
  const user = userEvent.setup();
  let failing = true;
  render(<App source={{ kind: 'fixture', load: async () => initial, loadProblem: async () => { if (failing) throw new Error('offline'); return report; } }} />);
  await screen.findByRole('tab', { name: 'Problems (1)' });
  await user.click(screen.getByRole('tab', { name: 'Problems (1)' }));
  await user.click(screen.getByRole('link', { name: 'CON-P47' }));
  await screen.findByText('Could not load this report.');
  failing = false;
  await user.click(screen.getByRole('button', { name: 'Retry report' }));
  await screen.findByText('Proposed remedy');
  await user.keyboard('{Escape}');
  await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
});
