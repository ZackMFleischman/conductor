import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { expect, it } from 'vitest';
import { App } from './App';
import { fixtureBoardSource } from './data/fixtureBoardSource';

async function groupSource() {
  const snapshot = await fixtureBoardSource.load();
  const epic = { id: 'all', key: 'DEMO-200', title: 'Portal experience' };
  const parent = { id: 'nested', key: 'DEMO-210', title: 'Search improvements' };
  const child = { id: 'deep', key: 'DEMO-211', title: 'Match rendering' };
  const other = { id: 'ungrouped', key: 'DEMO-300', title: 'Delivery' };
  snapshot.tickets = snapshot.tickets.slice(0, 5).map((ticket, index) => ({ ...ticket,
    ancestors: index === 0 ? [epic] : index === 1 ? [parent, epic] : index === 2 ? [other] : index === 4 ? [child, parent, epic] : [],
  }));
  return { kind: 'fixture' as const, load: async () => snapshot };
}

it('filters the complete parent subtree even when intermediate parent tickets are absent', async () => {
  const user = userEvent.setup();
  render(<App source={await groupSource()} />);
  await screen.findByText('DEMO-101');
  const filter = screen.getByRole('combobox', { name: 'Parent' });
  await user.selectOptions(filter, screen.getByRole('option', { name: 'DEMO-200 · Portal experience' }));
  expect(screen.getAllByRole('article')).toHaveLength(3);
  await user.selectOptions(filter, screen.getByRole('option', { name: 'DEMO-210 · Search improvements' }));
  expect(screen.getAllByRole('article')).toHaveLength(2);
  expect(screen.getByText('DEMO-102')).toBeInTheDocument();
  await user.selectOptions(filter, screen.getByRole('option', { name: 'DEMO-300 · Delivery' }));
  expect(screen.getAllByRole('article')).toHaveLength(1);
  expect(screen.getByText('DEMO-103')).toBeInTheDocument();
  await user.selectOptions(filter, screen.getByRole('option', { name: 'No parent' }));
  expect(screen.getAllByRole('article')).toHaveLength(1);
  expect(screen.getByText('DEMO-104')).toBeInTheDocument();
});

it('combines grouping with search and assignment, then clears every filter', async () => {
  const user = userEvent.setup();
  render(<App source={await groupSource()} />);
  await screen.findByText('DEMO-101');
  await user.selectOptions(screen.getByRole('combobox', { name: 'Parent' }), screen.getByRole('option', { name: 'DEMO-200 · Portal experience' }));
  await user.type(screen.getByRole('textbox', { name: 'Search tickets' }), 'handoffs');
  await user.selectOptions(screen.getByRole('combobox', { name: 'Assignee' }), screen.getByRole('option', { name: 'docs-agent' }));
  expect(screen.getAllByRole('article')).toHaveLength(1);
  expect(screen.getByText('DEMO-102')).toBeInTheDocument();
  await user.click(screen.getByRole('button', { name: 'Clear filters' }));
  expect(screen.getAllByRole('article')).toHaveLength(5);
  expect(screen.getByRole('combobox', { name: 'Parent' })).toHaveValue('all');
  expect(screen.getByRole('combobox', { name: 'Assignee' })).toHaveValue('all');
  expect(screen.getByRole('textbox', { name: 'Search tickets' })).toHaveValue('');
});

it('shows a quiet clickable epic label with direct-parent context available on hover', async () => {
  const user = userEvent.setup();
  render(<App source={await groupSource()} />);
  const card = await screen.findByRole('article', { name: 'Make handoffs easy to follow' });
  const group = within(card).getByRole('button', { name: 'Filter by parent Portal experience' });
  expect(group).toHaveTextContent('Portal experience');
  await user.hover(group);
  expect(await screen.findByRole('tooltip')).toHaveTextContent('DEMO-210 · Search improvements');
  await user.click(group);
  expect(screen.getAllByRole('article')).toHaveLength(3);
  expect(screen.getByRole('combobox', { name: 'Parent' })).toHaveValue('ref:all');
});

it('includes the selected parent ticket itself when it is present on the board', async () => {
  const user = userEvent.setup();
  const source = await groupSource();
  const snapshot = await source.load();
  snapshot.tickets.push({ ...snapshot.tickets[0], id: 'nested', key: 'DEMO-210', title: 'Search improvements', ancestors: [snapshot.tickets[0].ancestors[0]] });
  render(<App source={{ kind: 'fixture', load: async () => snapshot }} />);
  await screen.findByText('DEMO-210');
  await user.selectOptions(screen.getByRole('combobox', { name: 'Parent' }), screen.getByRole('option', { name: 'DEMO-210 · Search improvements' }));
  expect(screen.getAllByRole('article')).toHaveLength(3);
  expect(screen.getByText('DEMO-210')).toBeInTheDocument();
  expect(screen.getByText('DEMO-102')).toBeInTheDocument();
  expect(screen.getByText('DEMO-105')).toBeInTheDocument();
});
