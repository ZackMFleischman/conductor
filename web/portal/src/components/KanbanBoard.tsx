import { useState } from 'react';
import { Box, Button, Chip, Stack, Typography } from '@mui/material';
import { statuses, type BoardTicket } from '../board';
import { statusMeta, TicketCard } from './TicketCard';

export function KanbanBoard({ tickets, query, onGroupFilter, updatedTickets, onOpen }: { tickets: BoardTicket[]; query: string; onGroupFilter: (value: string) => void; updatedTickets?: ReadonlySet<string>; onOpen?: (key: string) => void }) {
  const [doneLimit, setDoneLimit] = useState(20);
  return <Box sx={{ flex: '1 1 auto', minHeight: 0, minWidth: 0, overflow: 'auto', scrollbarGutter: 'stable', pb: 1 }} tabIndex={0} role="group" aria-label="Ticket lanes; scroll horizontally on smaller desktop screens">
    <Box sx={{ display: 'grid', gridTemplateColumns: { xs: 'minmax(0, 1fr)', sm: 'repeat(5, minmax(244px, 1fr))' }, gap: 1, alignItems: 'start' }}>
      {statuses.map(status => {
        const meta = statusMeta[status];
        const laneTickets = tickets.filter(ticket => ticket.status === status);
        if (status === 'done') {
          const completed = (ticket: BoardTicket) => Date.parse(ticket.completedAt || ticket.updatedAt || ticket.createdAt || '') || 0;
          laneTickets.sort((a, b) => completed(b) - completed(a));
        }
        return <Box component="section" key={status} aria-labelledby={`lane-${status}`} sx={{ bgcolor: '#eef1f5', p: 0.75, borderRadius: 1.5, minHeight: 100 }}>
          <Stack direction="row" spacing={1} sx={{ alignItems: 'center', px: 0.75, pt: 0.25, pb: 0.75 }}>
            <Box sx={{ width: 8, height: 8, bgcolor: meta.color, borderRadius: '50%' }} />
            <Typography component="h2" variant="h2" id={`lane-${status}`} sx={{ flex: 1 }}>{meta.label}</Typography>
            <Chip size="small" label={laneTickets.length} aria-label={`${laneTickets.length} ${laneTickets.length === 1 ? 'ticket' : 'tickets'}`} sx={{ height: 22, bgcolor: '#fff', fontSize: '0.72rem', color: 'text.secondary' }} />
          </Stack>
          <Stack spacing={0.75}>
            {(status === 'done' ? laneTickets.slice(0, doneLimit) : laneTickets).map(ticket => <TicketCard key={ticket.id} ticket={ticket} query={query} onGroupFilter={onGroupFilter} onOpen={onOpen} updated={updatedTickets?.has(ticket.id)} />)}
            {status === 'done' && laneTickets.length > doneLimit && <Box sx={{ textAlign: 'center', py: 1 }}>
              <Typography variant="caption" color="text.secondary">Showing {doneLimit} of {laneTickets.length}</Typography>
              <Stack direction="row" sx={{ justifyContent: 'center' }}><Button size="small" onClick={() => setDoneLimit(n => n + 20)}>Show more</Button><Button size="small" onClick={() => setDoneLimit(Infinity)}>Show all</Button></Stack>
            </Box>}
            {!laneTickets.length && <Box sx={{ border: '1px dashed #cdd4de', borderRadius: 1.5, p: 3, textAlign: 'center' }}><Typography variant="body2" color="text.secondary">No tickets</Typography></Box>}
          </Stack>
        </Box>;
      })}
    </Box>
  </Box>;
}
