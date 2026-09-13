import { useEffect, useState } from 'react';
import { Alert, Box, Button, Chip, CircularProgress, CssBaseline, FormControl, InputAdornment, NativeSelect, Paper, Stack, TextField, ThemeProvider, Typography } from '@mui/material';
import AccountTreeOutlined from '@mui/icons-material/AccountTreeOutlined';
import SearchOutlined from '@mui/icons-material/SearchOutlined';
import VisibilityOutlined from '@mui/icons-material/VisibilityOutlined';
import ScienceOutlined from '@mui/icons-material/ScienceOutlined';
import type { BoardSnapshot, BoardSource } from './board';
import { KanbanBoard } from './components/KanbanBoard';
import { findSearchMatches } from './search';
import { theme } from './theme';

type LoadState = { status: 'loading' } | { status: 'error' } | { status: 'ready'; board: BoardSnapshot };

export function App({ source }: { source: BoardSource }) {
  const [state, setState] = useState<LoadState>({ status: 'loading' });
  const [attempt, setAttempt] = useState(0);
  const [query, setQuery] = useState('');
  const [assignee, setAssignee] = useState('all');
  useEffect(() => {
    const controller = new AbortController();
    setState({ status: 'loading' });
    const load = async () => {
      try {
        const board = await source.load(controller.signal);
        if (!controller.signal.aborted) setState({ status: 'ready', board });
      } catch {
        if (!controller.signal.aborted) setState({ status: 'error' });
      }
    };
    void load();
    return () => controller.abort();
  }, [source, attempt]);

  const board = state.status === 'ready' ? state.board : undefined;
  const tickets = board?.tickets ?? [];
  const term = query.trim();
  const filtered = tickets.filter(ticket => {
    const matchesOwner = assignee === 'all' || (assignee === 'unassigned' ? ticket.assignee === null : ticket.assignee !== null && 'owner:' + ticket.assignee.id === assignee);
    const searchable = [ticket.key, ticket.title, ticket.description, ticket.assignee?.name ?? 'Unassigned', ...ticket.blockers.flatMap(b => [b.reason, b.ticketKey ?? ''])];
    return matchesOwner && (!term || searchable.some(text => findSearchMatches(text, term).length > 0));
  });
  const owners = Array.from(new Map(tickets.flatMap(t => t.assignee ? [[t.assignee.id, t.assignee] as const] : [])).values()).sort((a, b) => a.name.localeCompare(b.name));
  const hasFilters = Boolean(query || assignee !== 'all');
  const clearFilters = () => { setQuery(''); setAssignee('all'); };

  return <ThemeProvider theme={theme}><CssBaseline />
    <Box component="header" sx={{ bgcolor: '#fff', borderBottom: '1px solid', borderColor: 'divider', px: { xs: 2, md: 3 }, py: 1 }}>
      <Stack direction="row" spacing={2} sx={{ alignItems: 'center', justifyContent: 'space-between' }}>
        <Stack direction="row" spacing={1.25} sx={{ alignItems: 'center' }}>
          <Box sx={{ width: 28, height: 28, borderRadius: 1.25, bgcolor: '#244f47', color: '#fff', display: 'grid', placeItems: 'center' }}><AccountTreeOutlined sx={{ fontSize: 19 }} /></Box>
          <Typography sx={{ fontWeight: 700, letterSpacing: '-.035em', fontSize: '1.15rem' }}>conductor</Typography>
          <Typography sx={{ color: '#c6cdd6', pl: 1, display: { xs: 'none', sm: 'block' } }}>/</Typography>
          <Typography variant="body2" color="text.secondary" sx={{ display: { xs: 'none', sm: 'block' } }}>{board?.project.name ?? 'Conductor'}</Typography>
        </Stack>
        <Chip icon={<VisibilityOutlined />} label="Read only" size="small" variant="outlined" sx={{ borderColor: '#dce2e8', color: 'text.secondary', fontSize: '0.72rem' }} />
      </Stack>
    </Box>
    <Box component="main" sx={{ maxWidth: 1800, mx: 'auto', px: { xs: 2, md: 3 }, pt: 2, pb: 2 }}>
      <Stack direction="row" sx={{ alignItems: 'center', flexWrap: 'wrap', gap: 1.5, mb: 1.5 }}>
        <Typography component="h1" variant="h1">Work board</Typography>
        <Chip icon={<ScienceOutlined />} label={source.kind === 'fixture' ? 'Fixture data' : 'Live data'} size="small" sx={{ bgcolor: '#e1eee7', color: '#285d4f', fontSize: '0.7rem' }} />
        {board && <Stack direction="row" sx={{ ml: { sm: 'auto' }, gap: 2, flexWrap: 'wrap' }}>
          {[[tickets.length, 'tickets'], [tickets.filter(t => t.status === 'in_progress').length, 'in progress'], [tickets.filter(t => t.status === 'blocked').length, 'blocked'], [tickets.filter(t => t.status === 'review').length, 'in review']].map(([count, label]) => <Typography key={label} variant="caption" color="text.secondary"><Box component="span" sx={{ fontWeight: 700, color: 'text.primary', mr: 0.5 }}>{count}</Box>{' '}{label}</Typography>)}
        </Stack>}
      </Stack>
      {state.status === 'loading' && <Stack role="status" direction="row" spacing={2} sx={{ py: 8, justifyContent: 'center' }}><CircularProgress size={20} /><Typography>Loading board…</Typography></Stack>}
      {state.status === 'error' && <Alert severity="error" sx={{ mt: 3 }} action={<Button color="inherit" onClick={() => setAttempt(a => a + 1)}>Try again</Button>}>Could not load the board. Please try again.</Alert>}
      {board && <>
        <Stack direction="row" sx={{ gap: 1, alignItems: 'center', flexWrap: 'wrap', mb: 1.5 }}>
          <TextField hiddenLabel size="small" placeholder="Search tickets…" value={query} onChange={e => setQuery(e.target.value)} sx={{ width: { sm: 280 }, flex: { xs: 1, sm: 'none' }, minWidth: 150, bgcolor: '#fff', '& .MuiInputBase-root': { height: 32, fontSize: '0.8rem' } }} slotProps={{ htmlInput: { 'aria-label': 'Search tickets' }, input: { startAdornment: <InputAdornment position="start"><SearchOutlined sx={{ fontSize: 17 }} /></InputAdornment> } }} />
          <FormControl variant="standard" sx={{ minWidth: 155, height: 32, px: 1, justifyContent: 'center', bgcolor: '#fff', border: '1px solid #c4cbd3', borderRadius: 1 }}>
            <NativeSelect value={assignee} onChange={e => setAssignee(e.target.value)} disableUnderline inputProps={{ id: 'assignee-filter', 'aria-label': 'Assignee' }} sx={{ fontSize: '0.8rem', '& select': { py: 0.5 } }}>
              <option value="all">All assignees</option><option value="unassigned">Unassigned</option>
              {owners.map(owner => <option key={owner.id} value={'owner:' + owner.id}>{owner.name}</option>)}
            </NativeSelect>
          </FormControl>
          {hasFilters && <Button onClick={clearFilters} size="small">Clear filters</Button>}
        </Stack>
        {filtered.length === 0 && <Paper variant="outlined" sx={{ p: 3, textAlign: 'center', mb: 2 }}>
          <Typography component="h2" variant="h3">{tickets.length === 0 ? 'No tickets yet' : 'No tickets match these filters'}</Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>{tickets.length === 0 ? 'Tickets will appear here when the source has work to show.' : 'Try another search or clear the filters to see all work.'}</Typography>
        </Paper>}
        <KanbanBoard tickets={filtered} query={term} />
        <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 1 }}>Assignment shows ownership, not an active claim. This board does not change ticket state.</Typography>
      </>}
    </Box>
  </ThemeProvider>;
}
