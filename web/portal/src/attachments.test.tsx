import { fireEvent, render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { expect, it } from 'vitest';
import { ticketAttachments } from './attachments';
import { App } from './App';
import type { BoardTicket } from './board';

const ticket: BoardTicket = { id: 't', key: 'P-1', title: 'With attachments', description: '![Preview](https://example.com/screen.png)\n[Notes](https://example.com/notes.pdf)\n[Website](https://example.com)\n![again][shot]\n\n[shot]: https://example.com/screen.png\n\n`![code](https://example.com/code.png)`\n![unsafe](javascript:alert(1))', status: 'ready', assignee: null, blockers: [], ancestors: [] };

it('collects and deduplicates safe Markdown images and file links across ticket fields', () => {
  const result = ticketAttachments({ ...ticket, evidence: '[Logs](https://example.com/log.txt)' });
  expect(result.map(a => a.url)).toEqual(['https://example.com/screen.png', 'https://example.com/notes.pdf', 'https://example.com/log.txt']);
  expect(result[0].image).toBe(true);
  expect(result[1].image).toBe(false);
});

it('shows a card thumbnail, compact details and a full-size image viewer with file links', async () => {
  const user = userEvent.setup();
  render(<App source={{ kind: 'fixture', load: async () => ({ project: { id: 'p', name: 'P', description: '' }, tickets: [ticket] }) }} />);
  const card = await screen.findByRole('article', { name: ticket.title });
  expect(within(card).getAllByRole('img')).toHaveLength(1);
  await user.click(within(card).getByRole('button', { name: 'Preview image: Preview' }));
  let viewer = await screen.findByRole('dialog', { name: 'Preview' });
  await user.click(within(viewer).getByRole('button', { name: 'Actual size' }));
  expect(within(viewer).getByRole('button', { name: 'Fit to screen' })).toBeInTheDocument();
  await user.keyboard('{Escape}');
  await user.click(within(card).getByRole('button', { name: '2 attachments' }));
  const details = await screen.findByRole('dialog', { name: /P-1/ });
  expect(within(details).getByRole('heading', { name: 'Attachments (2)' })).toBeInTheDocument();
  expect(within(details).getAllByRole('link', { name: /Notes/ })[0]).toHaveAttribute('href', 'https://example.com/notes.pdf');
  await user.click(within(details).getByRole('button', { name: 'Preview image: Preview' }));
  viewer = await screen.findByRole('dialog', { name: 'Preview' });
  fireEvent.error(within(viewer).getByRole('img'));
  expect(within(viewer).getByText('Image unavailable. The source may have moved or been removed.')).toBeInTheDocument();
  await user.keyboard('{Escape}');
  expect(await screen.findByRole('dialog', { name: /P-1/ })).toBeInTheDocument();
});
