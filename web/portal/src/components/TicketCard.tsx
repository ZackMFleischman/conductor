import { Avatar, Box, Chip, Paper, Stack, Typography } from '@mui/material';
import LockOutlined from '@mui/icons-material/LockOutlined';
import PersonOutlined from '@mui/icons-material/PersonOutlined';
import { HighlightedText } from './HighlightedText';
import type { BoardTicket, TicketStatus } from '../board';

export const statusMeta: Record<TicketStatus, { label: string; color: string; tint: string }> = {
  ready: { label: 'Ready', color: '#667085', tint: '#eef0f4' },
  in_progress: { label: 'In progress', color: '#3968ae', tint: '#eaf1fd' },
  blocked: { label: 'Blocked', color: '#a65126', tint: '#fff0e6' },
  review: { label: 'Review', color: '#7957a2', tint: '#f2ecfa' },
  done: { label: 'Done', color: '#287661', tint: '#e8f3ed' },
};

export function TicketCard({ ticket, query }: { ticket: BoardTicket; query: string }) {
  const meta = statusMeta[ticket.status];
  const blockers = ticket.blockers.length ? ticket.blockers : ticket.status === 'blocked' ? [{ reason: 'Blocker details unavailable.' }] : [];
  return <Paper component="article" variant="outlined" aria-labelledby={`ticket-${ticket.id}`} sx={{ p: 1, boxShadow: '0 2px 3px #25385804', overflowWrap: 'anywhere' }}>
    <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', gap: 0.75, mb: 0.5 }}>
      <Typography variant="caption" sx={{ fontFamily: 'Consolas, monospace', color: 'text.secondary', letterSpacing: '.02em' }}><HighlightedText text={ticket.key} query={query} /></Typography>
      <Chip size="small" label={meta.label} sx={{ height: 18, fontSize: '0.66rem', borderRadius: 1, bgcolor: meta.tint, color: meta.color }} />
    </Stack>
    <Typography component="h3" variant="h3" id={`ticket-${ticket.id}`} sx={{ mb: 0.5 }}><HighlightedText text={ticket.title} query={query} /></Typography>
    <Typography variant="body2" color="text.secondary" sx={{ whiteSpace: 'pre-wrap' }}><HighlightedText text={ticket.description || 'No description provided.'} query={query} /></Typography>
    {blockers.length > 0 && <Box sx={{ mt: 0.75, p: 0.75, bgcolor: '#fff5ed', border: '1px solid #f3decd', borderRadius: 1.5 }}>
      <Stack direction="row" spacing={0.75} sx={{ alignItems: 'center', mb: 0.5, color: '#8d4522' }}>
        <LockOutlined sx={{ fontSize: 13 }} /><Typography variant="caption" sx={{ fontWeight: 700 }}>Blocked by</Typography>
      </Stack>
      {blockers.map((blocker, index) => <Box key={index} sx={{ mt: index ? 1 : 0 }}>
        {blocker.ticketKey && <Typography variant="caption" sx={{ color: '#8d4522', fontFamily: 'Consolas, monospace' }}><HighlightedText text={blocker.ticketKey} query={query} /></Typography>}
        <Typography variant="body2" sx={{ fontSize: '0.76rem', color: '#784a32' }}><HighlightedText text={blocker.reason} query={query} /></Typography>
      </Box>)}
    </Box>}
    <Stack direction="row" spacing={1} sx={{ alignItems: 'center', mt: 0.75 }}>
      <Avatar sx={{ width: 18, height: 18, bgcolor: ticket.assignee ? '#e8eeed' : '#f0f2f5', color: '#4e6761', fontSize: '0.55rem', fontWeight: 700 }}>
        {ticket.assignee ? ticket.assignee.name.split('-').map(p => p[0]).slice(0, 2).join('').toUpperCase() : <PersonOutlined sx={{ fontSize: 13 }} />}
      </Avatar>
      <Typography variant="caption" color="text.secondary"><HighlightedText text={ticket.assignee?.name ?? 'Unassigned'} query={query} /></Typography>
    </Stack>
  </Paper>;
}
