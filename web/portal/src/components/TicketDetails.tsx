import { Accordion, AccordionDetails, AccordionSummary, Alert, Box, Chip, Dialog, DialogContent, DialogTitle, Divider, IconButton, Stack, Typography } from '@mui/material';
import Close from '@mui/icons-material/Close';
import ExpandMore from '@mui/icons-material/ExpandMore';
import { useEffect, useRef } from 'react';
import type { BoardTicket } from '../board';
import { statusMeta } from './TicketCard';
import { TicketDescription } from './TicketDescription';
import { TicketLink } from './TicketLinks';
import { TicketAttachments } from './TicketAttachments';

const dateLabel = (value: string) => new Date(value).toLocaleString(undefined, { month: 'short', day: 'numeric', year: 'numeric', hour: 'numeric', minute: '2-digit' });
const sectionTitle = { fontSize: '0.88rem', fontWeight: 700, mb: 1 };

export function TicketDetails({ ticket, tickets = [], selectedKey, onClose }: { ticket?: BoardTicket; tickets?: BoardTicket[]; selectedKey: string | null; onClose: () => void }) {
  const title = useRef<HTMLHeadingElement>(null);
  const content = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!selectedKey) return;
    title.current?.focus();
    if (content.current) content.current.scrollTop = 0;
  }, [selectedKey]);
  const children = ticket ? tickets.filter(t => t.ancestors[0]?.id === ticket.id) : [];
  const meta = ticket && statusMeta[ticket.status];
  return <Dialog open={selectedKey !== null} onClose={onClose} fullWidth maxWidth="lg" aria-labelledby="ticket-details-key ticket-details-title" slotProps={{ paper: { sx: { maxWidth: 1060, maxHeight: '92dvh', m: { xs: 1, sm: 3 }, width: { xs: 'calc(100% - 16px)' }, borderRadius: 2 } } }}>
    <DialogTitle component="div" id="ticket-details-header" sx={{ px: { xs: 2, sm: 3 }, pt: 2.5, pb: 2, borderBottom: '1px solid', borderColor: 'divider' }}>
      <Stack direction="row" sx={{ alignItems: 'center', gap: 1, mb: 1 }}><Typography id="ticket-details-key" variant="caption" sx={{ fontFamily: 'Consolas, monospace', fontWeight: 700, letterSpacing: '.04em' }}>{ticket?.key ?? selectedKey}</Typography>{ticket && <><Chip label={ticket.kind || 'implementation'} size="small" sx={{ height: 22, fontSize: '0.7rem' }} /><Chip label={meta!.label} size="small" sx={{ height: 22, fontSize: '0.7rem', bgcolor: meta!.tint, color: meta!.color }} /></>}<IconButton aria-label="Close ticket details" onClick={onClose} size="small" sx={{ ml: 'auto' }}><Close fontSize="small" /></IconButton></Stack>
      <Typography ref={title} tabIndex={-1} component="h2" id="ticket-details-title" sx={{ fontSize: { xs: '1.15rem', sm: '1.4rem' }, lineHeight: 1.35, fontWeight: 650, letterSpacing: '-.02em', outline: 'none', overflowWrap: 'anywhere' }}>{ticket?.title ?? 'Ticket unavailable'}</Typography>
    </DialogTitle>
    <DialogContent ref={content} sx={{ p: 0 }}>
      {!ticket ? <Alert severity="info">This ticket is no longer available on this board.</Alert> : <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: 'minmax(0, 1fr) 232px' } }}>
        <Stack spacing={3} sx={{ p: { xs: 2, sm: 3 }, minWidth: 0, '& > section > .MuiTypography-body2': { fontSize: '0.86rem', lineHeight: 1.65 } }}>
          {ticket.blockers.length > 0 && <Alert severity="warning" icon={false}><Typography component="h3" sx={sectionTitle}>Blocked by</Typography>{ticket.blockers.map((blocker, i) => <Box key={i}>{blocker.ticketKey && <TicketLink ticketKey={blocker.ticketKey} />}<TicketDescription text={blocker.reason} query="" full /></Box>)}</Alert>}
          <Box component="section"><Typography component="h3" sx={sectionTitle}>Description</Typography><TicketDescription text={ticket.description || 'No description provided.'} query="" full hideImages /></Box>
          <TicketAttachments key={ticket.id} ticket={ticket} />
          {ticket.summary && <Box component="section" sx={{ bgcolor: '#f4f8f6', p: 2, borderRadius: 1.5 }}><Typography component="h3" sx={sectionTitle}>Delivery summary</Typography><TicketDescription text={ticket.summary} query="" full hideImages /></Box>}
          {(ticket.evidence || ticket.qa) && <Box key={`notes-${ticket.id}`}>
            {([['Evidence', ticket.evidence], ['QA notes', ticket.qa]] as const).map(([label, text]) => text && <Accordion key={label} disableGutters elevation={0} sx={{ borderTop: '1px solid', borderColor: 'divider', '&:before': { display: 'none' } }}><AccordionSummary expandIcon={<ExpandMore />} sx={{ px: 0, minHeight: 44 }}><Typography component="h3" sx={{ ...sectionTitle, mb: 0 }}>{label}</Typography></AccordionSummary><AccordionDetails sx={{ px: 0, '& .MuiTypography-body2': { fontSize: '0.86rem', lineHeight: 1.65 } }}><TicketDescription text={text} query="" full hideImages /></AccordionDetails></Accordion>)}
          </Box>}
        </Stack>
        <Stack component="aside" aria-label="Ticket context" spacing={2.5} sx={{ p: 2.5, bgcolor: '#f7f8fa', borderLeft: { sm: '1px solid' }, borderColor: { sm: 'divider' }, minWidth: 0, gridColumn: { sm: 2 } }}>
          <Box><Typography component="h3" sx={sectionTitle}>Owner</Typography><Typography variant="body2" sx={{ overflowWrap: 'anywhere' }}>{ticket.assignee?.name ?? 'Unassigned'}</Typography></Box>
          {ticket.ancestors.length > 0 && <Box><Typography component="h3" sx={sectionTitle}>Parents</Typography><Stack spacing={0.75}>{[...ticket.ancestors].reverse().map(ref => <Typography key={ref.id} variant="body2" sx={{ overflowWrap: 'anywhere' }}><TicketLink ticketKey={ref.key}>{ref.key} · {ref.title}</TicketLink></Typography>)}</Stack></Box>}
          {children.length > 0 && <Box><Typography component="h3" sx={sectionTitle}>Child tickets ({children.length})</Typography><Stack spacing={1}>{children.map(child => <Box key={child.id}><Typography variant="body2"><TicketLink ticketKey={child.key}>{child.key} · {child.title}</TicketLink></Typography><Typography variant="caption" color="text.secondary">{statusMeta[child.status].label}</Typography></Box>)}</Stack></Box>}
          {(ticket.createdAt || ticket.updatedAt) && <><Divider /><Stack spacing={1.5}>{([['Updated', ticket.updatedAt], ['Created', ticket.createdAt]] as const).map(([label, value]) => value && <Box key={label}><Typography variant="caption" color="text.secondary" component="div">{label}</Typography><Typography variant="caption" component="time" dateTime={value}>{dateLabel(value)}</Typography></Box>)}</Stack></>}
        </Stack>
      </Box>}
    </DialogContent>
  </Dialog>;
}
