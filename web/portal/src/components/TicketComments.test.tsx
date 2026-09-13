import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { TicketComments } from './TicketComments';
import { TicketLinksContext } from './TicketLinks';
import type { TicketNotesPage } from '../data/ticketNotes';

const page: TicketNotesPage = { ticketId: 't', total: 2, notes: [{ id: 'new', author: 'Worker', body: '**Decision:** follow CON-12', createdAt: '2026-09-13T12:00:00Z' }], nextCursor: 'older' };

describe('TicketComments', () => {
 it('loads on expansion, renders linked Markdown and pages older comments', async () => {
  const user=userEvent.setup(); const open=vi.fn();
  const loadPage=vi.fn().mockResolvedValueOnce(page).mockResolvedValueOnce({...page,notes:[{...page.notes[0],id:'old',body:'Earlier decision'}],nextCursor:undefined});
  render(<TicketLinksContext.Provider value={{keys:new Set(['CON-12']),open}}><TicketComments ticketId="t" loadPage={loadPage}/></TicketLinksContext.Provider>);
  expect(loadPage).not.toHaveBeenCalled();
  await user.click(screen.getByRole('button',{name:'Comments'}));
  expect(await screen.findByText('Decision:')).toHaveProperty('tagName','STRONG');
  await user.click(screen.getByRole('link',{name:'CON-12'})); expect(open).toHaveBeenCalledWith('CON-12');
  await user.click(screen.getByRole('button',{name:'Load older'}));
  expect(await screen.findByText('Earlier decision')).toBeInTheDocument();
  expect(loadPage.mock.calls[1][0]).toBe('older');
  expect(screen.queryByRole('button',{name:'Load older'})).not.toBeInTheDocument();
 });
 it('keeps current comments when an older page fails and retries the same page',async()=>{
  const user=userEvent.setup();const loadPage=vi.fn().mockResolvedValueOnce(page).mockRejectedValueOnce(new Error('Offline')).mockResolvedValueOnce({...page,notes:[],nextCursor:undefined});
  render(<TicketComments ticketId="t" loadPage={loadPage}/>);await user.click(screen.getByRole('button',{name:'Comments'}));
  await screen.findByText('Decision:');await user.click(screen.getByRole('button',{name:'Load older'}));
  expect(await screen.findByRole('alert')).toHaveTextContent('Comments could not be loaded');expect(screen.getByText('Decision:')).toBeInTheDocument();
  await user.click(screen.getByRole('button',{name:'Retry'}));await waitFor(()=>expect(loadPage).toHaveBeenCalledTimes(3));expect(loadPage.mock.calls[2][0]).toBe('older');
 });
 it('aborts hidden requests and never shows another ticket’s stale response',async()=>{
  const user=userEvent.setup();let resolve!:(page:TicketNotesPage)=>void;
  const loadPage=vi.fn((_before?:string,_signal?:AbortSignal)=>new Promise<TicketNotesPage>(r=>{resolve=r}));
  const {rerender}=render(<TicketComments ticketId="t" loadPage={loadPage}/>);
  await user.click(screen.getByRole('button',{name:'Comments'}));const signal=loadPage.mock.calls[0][1];
  rerender(<TicketComments ticketId="different" loadPage={loadPage}/>);
  expect(signal?.aborted).toBe(true);resolve(page);await Promise.resolve();expect(screen.queryByText('Decision:')).not.toBeInTheDocument();
 });
 it('refreshes the latest comments after ticket updates only when expanded',async()=>{
  const user=userEvent.setup();const loadPage=vi.fn().mockResolvedValue(page);
  const {rerender}=render(<TicketComments ticketId="t" updatedAt="a" loadPage={loadPage}/>);
  rerender(<TicketComments ticketId="t" updatedAt="b" loadPage={loadPage}/>);expect(loadPage).not.toHaveBeenCalled();
  await user.click(screen.getByRole('button',{name:'Comments'}));await screen.findByText('Decision:');
  rerender(<TicketComments ticketId="t" updatedAt="c" loadPage={loadPage}/>);await waitFor(()=>expect(loadPage).toHaveBeenCalledTimes(2));
 });
});
