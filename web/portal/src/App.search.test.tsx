import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { expect, it } from 'vitest';
import { App } from './App';
import type { BoardSnapshot, BoardSource } from './board';

const snapshot: BoardSnapshot = {
  project: { id: 'test', name: 'Conductor', description: '' },
  tickets: [{ id: 'one', key: 'DEMO-[a+b]', title: 'Review [a+b]', description: 'Plan [a+b] and [A+B], not ab.', status: 'blocked', assignee: { id: 'owner', name: '[a+b]-agent' }, blockers: [{ reason: 'Wait for [a+b]', ticketKey: 'REF-[a+b]' }] }],
};
const source: BoardSource = { kind: 'fixture', load: async () => structuredClone(snapshot) };

it('highlights every literal case-insensitive match across searchable ticket fields', async () => {
  const user = userEvent.setup();
  const { container } = render(<App source={source} />);
  await screen.findByRole('article');
  await user.type(screen.getByRole('textbox', { name: 'Search tickets' }), '  [[a+b]  ');
  // user-event uses [[ to type a literal opening bracket.
  const marks = [...container.querySelectorAll('article mark')];
  expect(marks.map(m => m.textContent)).toEqual(['[a+b]', '[a+b]', '[a+b]', '[A+B]', '[a+b]', '[a+b]', '[a+b]']);
  expect(screen.getByRole('article')).toHaveTextContent('Plan [a+b] and [A+B], not ab.');
  await user.click(screen.getByRole('button', { name: 'Clear filters' }));
  expect(container.querySelectorAll('mark')).toHaveLength(0);
  expect(screen.getByRole('article')).toBeInTheDocument();
});

it('treats whitespace-only searches as no highlights', async () => {
  const user = userEvent.setup();
  const { container } = render(<App source={source} />);
  await screen.findByRole('article');
  await user.type(screen.getByRole('textbox', { name: 'Search tickets' }), '   ');
  expect(container.querySelectorAll('mark')).toHaveLength(0);
  expect(screen.getByRole('article')).toBeInTheDocument();
});

it('preserves HTML-looking search text as inert text', async () => {
  const user = userEvent.setup();
  const board = structuredClone(snapshot);
  board.tickets[0].title = '<img src=x onerror=alert(1)>';
  const { container } = render(<App source={{ kind: 'fixture', load: async () => board }} />);
  await screen.findByRole('article');
  await user.type(screen.getByRole('textbox', { name: 'Search tickets' }), '<IMG');
  expect(container.querySelector('mark')).toHaveTextContent('<img');
  expect(container.querySelector('article img')).toBeNull();
  expect(screen.getByRole('article')).toHaveTextContent('<img src=x onerror=alert(1)>');
});

it('does not keep a ticket for a phrase that spans unrelated fields', async () => {
  const user = userEvent.setup();
  const board = structuredClone(snapshot);
  board.tickets[0].title = 'Alpha';
  board.tickets[0].description = 'Beta';
  render(<App source={{ kind: 'fixture', load: async () => board }} />);
  await screen.findByRole('article');
  await user.type(screen.getByRole('textbox', { name: 'Search tickets' }), 'Alpha Beta');
  expect(screen.queryAllByRole('article')).toHaveLength(0);
  expect(screen.getByText('No tickets match these filters')).toBeInTheDocument();
});
