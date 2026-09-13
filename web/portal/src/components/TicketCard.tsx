import { Avatar, Box, Chip, Paper, Stack, Typography } from '@mui/material';
import LockOutlined from '@mui/icons-material/LockOutlined';
import PersonOutlined from '@mui/icons-material/PersonOutlined';
import type { BoardTicket, TicketStatus } from '../board';

export const statusMeta: Record<TicketStatus, { label: string; color: string; tint: string }> = {
  ready: { label: 'Ready', color: '#667085', tint: '#eef0f4' },
  in_progress: { label: 'In progress', color: '#3968ae', tint: '#eaf1fd' },
  blocked: { label: 'Blocked', color: '#a65126', tint: '#fff0e6' },
  review: { label: 'Review', color: '#7957a2', tint: '#f2ecfa' },
  done: { label: 'Done', color: '#287661', tint: '#e8f3ed' },
};

export function TicketCard({ ticket }: { ticket: BoardTicket }) {
  const meta = statusMeta[ticket.status];
  const blockers = ticket.blockers.length ? ticket.blockers : ticket.status === 'blocked' ? [{ reason: 'Blocker details unavailable.' }] : [];
  return <Paper component="article" variant="outlined" aria-labelledby={`ticket-${ticket.id}`} sx={{ p: 1.5, boxShadow: '0 2px 3px #25385804', overflowWrap: 'anywhere' }}>
    <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', gap: 1, mb: 1 }}>
      <Typography variant="caption" sx={{ fontFamily: 'Consolas, monospace', color: 'text.secondary', letterSpacing: '.02em' }}>{ticket.key}</Typography>
      <Chip size="small" label={meta.label} sx={{ height: 22, fontSize: '0.66rem', borderRadius: 1, bgcolor: meta.tint, color: meta.color }} />
    </Stack>
    <Typography component="h3" variant="h3" id={`ticket-${ticket.id}`} sx={{ mb: 1 }}>{ticket.title}</Typography>
    <Typography variant="body2" color="text.secondary" sx={{ whiteSpace: 'pre-wrap' }}>{ticket.description || 'No description provided.'}</Typography>
    {blockers.length > 0 && <Box sx={{ mt: 1.5, p: 1, bgcolor: '#fff5ed', border: '1px solid #f3decd', borderRadius: 1.5 }}>
      <Stack direction="row" spacing={0.75} sx={{ alignItems: 'center', mb: 0.5, color: '#8d4522' }}>
        <LockOutlined sx={{ fontSize: 13 }} /><Typography variant="caption" sx={{ fontWeight: 700 }}>Blocked by</Typography>
      </Stack>
      {blockers.map((blocker, index) => <Box key={index} sx={{ mt: index ? 1 : 0 }}>
        {blocker.ticketKey && <Typography variant="caption" sx={{ color: '#8d4522', fontFamily: 'Consolas, monospace' }}>{blocker.ticketKey}</Typography>}
        <Typography variant="body2" sx={{ fontSize: '0.76rem', color: '#784a32' }}>{blocker.reason}</Typography>
      </Box>)}
    </Box>}
    <Stack direction="row" spacing={1} sx={{ alignItems: 'center', mt: 1.5, pt: 1, borderTop: '1px solid', borderColor: 'divider' }}>
      <Avatar sx={{ width: 24, height: 24, bgcolor: ticket.assignee ? '#e8eeed' : '#f0f2f5', color: '#4e6761', fontSize: '0.65rem', fontWeight: 700 }}>
        {ticket.assignee ? ticket.assignee.name.split('-').map(p => p[0]).slice(0, 2).join('').toUpperCase() : <PersonOutlined sx={{ fontSize: 15 }} />}
      </Avatar>
      <Typography variant="caption" color="text.secondary">{ticket.assignee?.name ?? 'Unassigned'}</Typography>
    </Stack>
  </Paper>;
}
