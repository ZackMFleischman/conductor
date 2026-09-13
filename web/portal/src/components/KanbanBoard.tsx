import { Box, Chip, Stack, Typography } from '@mui/material';
import { statuses, type BoardTicket } from '../board';
import { statusMeta, TicketCard } from './TicketCard';

export function KanbanBoard({ tickets }: { tickets: BoardTicket[] }) {
  return <Box sx={{ overflowX: 'auto', pb: 2 }} tabIndex={0} role="group" aria-label="Ticket lanes; scroll horizontally on smaller desktop screens">
    <Box sx={{ display: 'grid', gridTemplateColumns: { xs: 'minmax(0, 1fr)', sm: 'repeat(5, minmax(244px, 1fr))' }, gap: 1.5, alignItems: 'start' }}>
      {statuses.map(status => {
        const meta = statusMeta[status];
        const laneTickets = tickets.filter(ticket => ticket.status === status);
        return <Box component="section" key={status} aria-labelledby={`lane-${status}`} sx={{ bgcolor: '#eef1f5', p: 1, borderRadius: 2, minHeight: 100 }}>
          <Stack direction="row" spacing={1} sx={{ alignItems: 'center', px: 0.75, pt: 0.75, pb: 1.25 }}>
            <Box sx={{ width: 8, height: 8, bgcolor: meta.color, borderRadius: '50%' }} />
            <Typography component="h2" variant="h2" id={`lane-${status}`} sx={{ flex: 1 }}>{meta.label}</Typography>
            <Chip size="small" label={laneTickets.length} aria-label={`${laneTickets.length} ${laneTickets.length === 1 ? 'ticket' : 'tickets'}`} sx={{ height: 22, bgcolor: '#fff', fontSize: '0.72rem', color: 'text.secondary' }} />
          </Stack>
          <Stack spacing={1}>
            {laneTickets.map(ticket => <TicketCard key={ticket.id} ticket={ticket} />)}
            {!laneTickets.length && <Box sx={{ border: '1px dashed #cdd4de', borderRadius: 1.5, p: 3, textAlign: 'center' }}><Typography variant="body2" color="text.secondary">No tickets</Typography></Box>}
          </Stack>
        </Box>;
      })}
    </Box>
  </Box>;
}
