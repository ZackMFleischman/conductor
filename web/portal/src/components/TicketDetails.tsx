import { Alert, Box, Button, Chip, Dialog, DialogActions, DialogContent, DialogTitle, Stack, Typography } from '@mui/material';
import { useEffect, useRef } from 'react';
import type { BoardTicket } from '../board';
import { statusMeta } from './TicketCard';
import { TicketDescription } from './TicketDescription';
import { TicketLink } from './TicketLinks';

export function TicketDetails({ ticket, selectedKey, onClose }: { ticket?: BoardTicket; selectedKey: string | null; onClose: () => void }) {
  const title = useRef<HTMLHeadingElement>(null);
  const content = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!selectedKey) return;
    title.current?.focus();
    if (content.current) content.current.scrollTop = 0;
  }, [selectedKey]);
  return <Dialog open={selectedKey !== null} onClose={onClose} fullWidth maxWidth="lg" aria-labelledby="ticket-details-title" slotProps={{ paper: { sx: { height: 'min(900px, 90dvh)', m: { xs: 1, sm: 4 }, width: { xs: 'calc(100% - 16px)' } } } }}>
    <DialogTitle ref={title} tabIndex={-1} id="ticket-details-title">{ticket ? `${ticket.key} · ${ticket.title}` : selectedKey}</DialogTitle>
    <DialogContent ref={content} dividers>
      {!ticket ? <Alert severity="info">This ticket is no longer available on this board.</Alert> : <Stack spacing={2.5}>
        <Stack direction="row" spacing={1} sx={{ flexWrap: 'wrap', gap: 0.5 }}><Chip label={ticket.kind || 'implementation'} size="small" /><Chip label={statusMeta[ticket.status].label} size="small" /><Typography variant="body2">Assignee: {ticket.assignee?.name ?? 'Unassigned'}</Typography></Stack>
        {(ticket.createdAt || ticket.updatedAt) && <Typography variant="caption" color="text.secondary">{ticket.createdAt && `Created: ${new Date(ticket.createdAt).toLocaleString()}`}{ticket.updatedAt && ` · Updated: ${new Date(ticket.updatedAt).toLocaleString()}`}</Typography>}
        {ticket.ancestors.length > 0 && <Box><Typography component="h3" variant="h3">Parents</Typography>{[...ticket.ancestors].reverse().map(ref => <Typography key={ref.id} variant="body2"><TicketLink ticketKey={ref.key}>{ref.key} · {ref.title}</TicketLink></Typography>)}</Box>}
        {ticket.blockers.length > 0 && <Box><Typography component="h3" variant="h3">Blockers</Typography>{ticket.blockers.map((blocker, i) => <Box key={i}>{blocker.ticketKey && <TicketLink ticketKey={blocker.ticketKey} />}<TicketDescription text={blocker.reason} query="" full /></Box>)}</Box>}
        {([['Description', ticket.description || 'No description provided.'], ['Summary', ticket.summary], ['Evidence', ticket.evidence], ['QA notes', ticket.qa]] as const).map(([label, text]) => text && <Box key={label}><Typography component="h3" variant="h3" sx={{ mb: 1 }}>{label}</Typography><TicketDescription text={text} query="" full /></Box>)}
      </Stack>}
    </DialogContent>
    <DialogActions><Button onClick={onClose}>Close</Button></DialogActions>
  </Dialog>;
}
