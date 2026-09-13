import { Avatar, Box, Chip, Paper, Stack, Typography } from '@mui/material';
import LockOutlined from '@mui/icons-material/LockOutlined';
import PersonOutlined from '@mui/icons-material/PersonOutlined';
import { GroupLabel } from './GroupLabel';
import { HighlightedText } from './HighlightedText';
import { TicketDescription } from './TicketDescription';
import type { BoardTicket, TicketStatus } from '../board';

export const statusMeta: Record<TicketStatus, { label: string; color: string; tint: string }> = {
  ready: { label: 'Ready', color: '#667085', tint: '#eef0f4' },
  in_progress: { label: 'In progress', color: '#3968ae', tint: '#eaf1fd' },
  blocked: { label: 'Blocked', color: '#a65126', tint: '#fff0e6' },
  review: { label: 'Review', color: '#7957a2', tint: '#f2ecfa' },
  done: { label: 'Done', color: '#287661', tint: '#e8f3ed' },
};

export function TicketCard({ ticket, query, onGroupFilter, onOpen, updated = false }: { ticket: BoardTicket; query: string; onGroupFilter: (value: string) => void; onOpen?: (key: string) => void; updated?: boolean }) {
  const meta = statusMeta[ticket.status];
  const blockers = ticket.blockers.length ? ticket.blockers : ticket.status === 'blocked' ? [{ reason: 'Blocker details unavailable.' }] : [];
  return <Paper component="article" variant="outlined" tabIndex={onOpen ? 0 : undefined} onKeyDown={event => {
    if (event.target === event.currentTarget && (event.key === 'Enter' || event.key === ' ')) { event.preventDefault(); onOpen?.(ticket.key); }
  }} onClick={event => { if (!(event.target as HTMLElement).closest('a,button,input')) onOpen?.(ticket.key); }} aria-labelledby={`ticket-${ticket.id}`} data-live-updated={updated || undefined} sx={{
    cursor: onOpen ? 'pointer' : undefined, '&:focus-visible': { outline: '2px solid #285d4f', outlineOffset: 2 },
    p: 1, boxShadow: '0 2px 3px #25385804', overflowWrap: 'anywhere',
    bgcolor: updated ? '#fff0b3' : 'background.paper',
    borderColor: updated ? '#d69e24' : 'divider',
    transition: updated ? 'none' : 'background-color 800ms ease-out, border-color 800ms ease-out',
    '@media (prefers-reduced-motion: reduce)': { transition: 'none' },
  }}>
    <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', gap: 0.75, mb: 0.5 }}>
      <Typography variant="caption" sx={{ fontFamily: 'Consolas, monospace', color: 'text.secondary', letterSpacing: '.02em' }}><HighlightedText text={ticket.key} query={query} /></Typography>
      <Stack direction="row" spacing={0.5}><Chip size="small" label={ticket.kind || 'implementation'} sx={{ height: 18, fontSize: '0.66rem', borderRadius: 1 }} /><Chip size="small" label={meta.label} sx={{ height: 18, fontSize: '0.66rem', borderRadius: 1, bgcolor: meta.tint, color: meta.color }} /></Stack>
    </Stack>
    <Typography component="h3" variant="h3" id={`ticket-${ticket.id}`} sx={{ mb: 0.5 }}><HighlightedText text={ticket.title} query={query} /></Typography>
    <TicketDescription text={ticket.description || 'No description provided.'} query={query} />
    {blockers.length > 0 && <Box sx={{ mt: 0.75, px: 0.75, py: 0.5, bgcolor: '#fff5ed', border: '1px solid #f3decd', borderRadius: 1 }}>
      {blockers.map((blocker, index) => <Box key={index} sx={{ mt: index ? 0.75 : 0 }}>
        <Stack direction="row" sx={{ alignItems: 'baseline', flexWrap: 'wrap', columnGap: 0.5, color: '#8d4522' }}>
          <LockOutlined sx={{ fontSize: 11, alignSelf: 'center' }} />
          <Typography variant="caption" sx={{ fontWeight: 700 }}>Blocked by</Typography>
          {blocker.ticketKey && <Typography variant="caption" sx={{ fontFamily: 'Consolas, monospace' }}><HighlightedText text={blocker.ticketKey} query={query} /></Typography>}
        </Stack>
        <Typography variant="body2" sx={{ color: '#784a32', mt: 0.25 }}><HighlightedText text={blocker.reason} query={query} /></Typography>
      </Box>)}
    </Box>}
    <Stack direction="row" sx={{ gap: 0.5, alignItems: 'center', flexWrap: 'wrap', mt: 0.75 }}>
      <Avatar sx={{ width: 18, height: 18, bgcolor: ticket.assignee ? '#e8eeed' : '#f0f2f5', color: '#4e6761', fontSize: '0.55rem', fontWeight: 700 }}>
        {ticket.assignee ? ticket.assignee.name.split('-').map(p => p[0]).slice(0, 2).join('').toUpperCase() : <PersonOutlined sx={{ fontSize: 13 }} />}
      </Avatar>
      <Typography variant="caption" color="text.secondary"><HighlightedText text={ticket.assignee?.name ?? 'Unassigned'} query={query} /></Typography>
      <GroupLabel ticket={ticket} onFilter={onGroupFilter} />
    </Stack>
  </Paper>;
}
