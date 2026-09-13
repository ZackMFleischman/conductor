import { Box, Button, Drawer, Stack, Typography } from '@mui/material';
import { useRef } from 'react';
import type { ActivityEntry } from '../activity';
import { TicketLink } from './TicketLinks';

export const activityDrawerWidth = 320;

export function ActivityFeed({ entries, open, desktop, onClose, archived, showArchived, archivedCount, canArchive, onArchive, onToggleArchived }: { entries: ActivityEntry[]; open: boolean; desktop: boolean; onClose: () => void; archived: ReadonlySet<string>; showArchived: boolean; archivedCount: number; canArchive: boolean; onArchive: () => void; onToggleArchived: () => void }) {
  const collapse = useRef<HTMLButtonElement>(null);
  return <Drawer anchor="right" variant={desktop ? 'persistent' : 'temporary'} open={open} onClose={onClose}
    slotProps={{ transition: { onEntered: () => collapse.current?.focus() }, paper: { component: 'aside', id: 'activity-drawer', 'aria-label': 'Activity feed', sx: { width: activityDrawerWidth, maxWidth: '100vw', boxSizing: 'border-box', p: 1.5 } } }}>
    <Stack direction="row" sx={{ justifyContent: 'space-between', alignItems: 'center' }}><Typography component="h2" variant="h2">Activity</Typography><Button ref={collapse} size="small" onClick={onClose}>Collapse</Button></Stack>
    <Stack direction="row" spacing={1} sx={{ py: 0.5 }}><Button size="small" disabled={!canArchive} onClick={onArchive}>Archive all</Button><Button size="small" disabled={!archivedCount} aria-pressed={showArchived} onClick={onToggleArchived}>{showArchived ? 'Hide archived' : `Show archived (${archivedCount})`}</Button></Stack>
    {!entries.length && <Typography variant="body2" color="text.secondary" sx={{ py: 2 }}>{archivedCount ? 'All caught up. New changes will appear here.' : 'Waiting for ticket changes…'}</Typography>}
    <Box component="ol" sx={{ listStyle: 'none', p: 0, m: 0 }}>
      {entries.map(entry => <Box component="li" key={entry.id} sx={{ borderTop: '1px solid', borderColor: 'divider', mt: 1.25, pt: 1.25, overflowWrap: 'anywhere' }}>
        <Typography variant="caption" color="text.secondary" component="time" dateTime={entry.at}>{new Date(entry.at).toLocaleString()}</Typography>
        {archived.has(entry.id) && <Typography component="span" variant="caption" color="text.secondary"> · Archived</Typography>}
        <Typography variant="body2" sx={{ fontWeight: 600 }}><TicketLink ticketKey={entry.key}>{entry.key} · {entry.title}</TicketLink></Typography>
        {entry.changes.map((change, i) => <Typography key={i} variant="body2" color="text.secondary">{change}</Typography>)}
      </Box>)}
    </Box>
  </Drawer>;
}
