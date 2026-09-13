import { Box, Button, Tooltip } from '@mui/material';
import type { BoardTicket } from '../board';

export function GroupLabel({ ticket, onFilter }: { ticket: BoardTicket; onFilter: (value: string) => void }) {
  const group = ticket.ancestors.at(-1);
  if (!group) return null;
  const context = 'Parents (root to nearest):\n' + [...ticket.ancestors].reverse().map(ref => `${ref.key} · ${ref.title}`).join('\n');
  return <Tooltip title={context} describeChild slotProps={{ tooltip: { sx: { whiteSpace: 'pre-line' } } }}>
    <Button size="small" aria-label={`Filter by parent ${group.title}`} onClick={() => onFilter('ref:' + group.id)} sx={{ ml: 'auto', minWidth: 0, maxWidth: '52%', px: 0.25, py: 0, fontSize: '0.65rem', fontWeight: 400, lineHeight: 1.5, color: 'text.secondary', textTransform: 'none', justifyContent: 'flex-start' }}>
      <Box component="span" aria-hidden="true" sx={{ mr: 0.5 }}>↳</Box>
      <Box component="span" sx={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{group.title}</Box>
    </Button>
  </Tooltip>;
}
