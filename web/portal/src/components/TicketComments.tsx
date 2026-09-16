import { useCallback, useEffect, useId, useRef, useState } from 'react';
import { Alert, Box, Button, Collapse, Divider, Stack, Typography } from '@mui/material';
import ExpandMore from '@mui/icons-material/ExpandMore';
import ChatBubbleOutline from '@mui/icons-material/ChatBubbleOutlined';
import type { TicketNotesLoader, TicketNotesPage } from '../data/ticketNotes';
import { TicketDescription } from './TicketDescription';

type Props = { ticketId: string; updatedAt?: string; commentCount?: number; loadPage: TicketNotesLoader };

// Identity boundaries also reset expansion and cancel a pending previous ticket read.
export function TicketComments(props: Props) {
 return <Comments key={props.ticketId} {...props} />;
}

function Comments({ ticketId, updatedAt, commentCount, loadPage }: Props) {
 const id = useId();
 const [expanded, setExpanded] = useState(false);
 const [page, setPage] = useState<TicketNotesPage | null>(null);
 const [busy, setBusy] = useState(false);
 const [error, setError] = useState<{ before?: string } | null>(null);
 const request = useRef<AbortController | null>(null);
 const toggle = useRef<HTMLButtonElement>(null);

 const load = useCallback(async (before?: string) => {
  request.current?.abort();
  const controller = new AbortController(); request.current = controller;
  setBusy(true); setError(null);
  try {
   const next = await loadPage(before, controller.signal);
   if (controller.signal.aborted) return;
   if (next.ticketId !== ticketId) throw new Error('Wrong ticket');
   setPage(current => before && current ? { ...next, notes: [...current.notes, ...next.notes.filter(note => !current.notes.some(existing => existing.id === note.id))] } : next);
  } catch {
   if (!controller.signal.aborted) setError({ before });
  } finally {
   if (!controller.signal.aborted) setBusy(false);
  }
 }, [loadPage, ticketId]);

 useEffect(() => {
  if (expanded) void load();
  return () => request.current?.abort();
 }, [expanded, load, updatedAt, commentCount]);

 const total = commentCount ?? page?.total;

 return <Box component="section" sx={{ mt: 3 }}>
  <Button ref={toggle} aria-expanded={expanded} aria-controls={id} onClick={() => setExpanded(value => !value)} startIcon={<ChatBubbleOutline />} endIcon={<ExpandMore sx={{ transform: expanded ? 'rotate(180deg)' : undefined }} />} sx={{ px: 0, color: 'text.primary', fontWeight: 700 }}>
   Comments{total !== undefined ? ` (${total})` : ''}
  </Button>
  <Collapse in={expanded}>
   <Box id={id} aria-busy={busy} sx={{ pt: 1 }}>
    {busy && <Typography role="status" variant="body2" color="text.secondary">Loading comments…</Typography>}
    {error && <Alert severity="warning" action={<Button onClick={() => { toggle.current?.focus(); void load(error.before); }}>Retry</Button>} sx={{ mb: 1 }}>Comments could not be loaded.{page ? ' Showing previously loaded comments.' : ''}</Alert>}
    {page && !page.notes.length && <Typography variant="body2" color="text.secondary">No comments yet.</Typography>}
    {page && !!page.notes.length && <Stack component="ol" spacing={2} divider={<Divider />} sx={{ listStyle: 'none', m: 0, p: 0 }}>
     {page.notes.map(note => <Box component="li" key={note.id}>
      <Stack direction="row" sx={{ gap: 1, flexWrap: 'wrap', alignItems: 'baseline', mb: .75 }}>
       <Typography variant="body2" sx={{ fontWeight: 700, overflowWrap: 'anywhere' }}>{note.author}</Typography>
       <Typography component="time" dateTime={note.createdAt} variant="caption" color="text.secondary">{Number.isNaN(Date.parse(note.createdAt)) ? note.createdAt : new Date(note.createdAt).toLocaleString()}</Typography>
      </Stack>
      <TicketDescription text={note.body} query="" full />
     </Box>)}
    </Stack>}
    {page?.nextCursor && !error && <Button disabled={busy} onClick={() => { toggle.current?.focus(); void load(page.nextCursor); }} sx={{ mt: 1.5 }}>Load older</Button>}
   </Box>
  </Collapse>
 </Box>;
}
