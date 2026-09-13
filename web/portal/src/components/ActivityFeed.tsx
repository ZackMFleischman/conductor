import { Box, Button, Stack, Typography } from '@mui/material';
import type { ActivityEntry } from '../activity';
import { TicketLink } from './TicketLinks';

export function ActivityFeed({ entries, onClose }: { entries: ActivityEntry[]; onClose: () => void }) {
  return <Box component="aside" aria-label="Activity feed" sx={{ width: { xs: '100%', md: 300 }, flexShrink: 0, bgcolor: '#fff', border: '1px solid', borderColor: 'divider', borderRadius: 1.5, p: 1.5, position: { md: 'sticky' }, top: 12, maxHeight: 'calc(100vh - 32px)', overflowY: 'auto' }}>
    <Stack direction="row" sx={{ justifyContent: 'space-between', alignItems: 'center' }}><Typography component="h2" variant="h2">Activity</Typography><Button size="small" onClick={onClose}>Collapse</Button></Stack>
    <Typography variant="caption" color="text.secondary">Observed since opening this board · newest first · latest 200 changes. Reconnecting may combine changes.</Typography>
    {!entries.length && <Typography variant="body2" color="text.secondary" sx={{ py: 2 }}>Waiting for ticket changes…</Typography>}
    <Box component="ol" sx={{ listStyle: 'none', p: 0, m: 0 }}>
      {entries.map(entry => <Box component="li" key={entry.id} sx={{ borderTop: '1px solid', borderColor: 'divider', mt: 1.25, pt: 1.25, overflowWrap: 'anywhere' }}>
        <Typography variant="caption" color="text.secondary" component="time" dateTime={entry.at}>{new Date(entry.at).toLocaleString()}</Typography>
        <Typography variant="body2" sx={{ fontWeight: 600 }}><TicketLink ticketKey={entry.key}>{entry.key} · {entry.title}</TicketLink></Typography>
        {entry.changes.map((change, i) => <Typography key={i} variant="body2" color="text.secondary">{change}</Typography>)}
      </Box>)}
    </Box>
  </Box>;
}
