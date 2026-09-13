import { Box, Button, Drawer, Stack, Typography } from '@mui/material';
import type { ActivityEntry } from '../activity';
import { TicketLink } from './TicketLinks';

export const activityDrawerWidth = 320;

export function ActivityFeed({ entries, open, desktop, onClose }: { entries: ActivityEntry[]; open: boolean; desktop: boolean; onClose: () => void }) {
  return <Drawer anchor="right" variant={desktop ? 'persistent' : 'temporary'} open={open} onClose={onClose}
    slotProps={{ paper: { component: 'aside', id: 'activity-drawer', 'aria-label': 'Activity feed', sx: { width: activityDrawerWidth, maxWidth: '100vw', boxSizing: 'border-box', p: 1.5 } } }}>
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
  </Drawer>;
}
