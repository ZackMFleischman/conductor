import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { Alert, Box, Button, Chip, CircularProgress, CssBaseline, FormControl, InputAdornment, NativeSelect, Paper, Stack, TextField, ThemeProvider, Typography } from '@mui/material';
import AccountTreeOutlined from '@mui/icons-material/AccountTreeOutlined';
import SearchOutlined from '@mui/icons-material/SearchOutlined';
import VisibilityOutlined from '@mui/icons-material/VisibilityOutlined';
import ScienceOutlined from '@mui/icons-material/ScienceOutlined';
import type { BoardSnapshot, BoardSource, BoardTicket, ConnectionStatus } from './board';
import { KanbanBoard } from './components/KanbanBoard';
import { findSearchMatches } from './search';
import { theme } from './theme';
import { markdownSearchText } from './components/highlightMarkdown';
import { TicketLinksContext } from './components/TicketLinks';
import { TicketDetails } from './components/TicketDetails';
import { ActivityFeed } from './components/ActivityFeed';
import { ticketChanges, type ActivityEntry } from './activity';

type LoadState = { status: 'loading' | 'error' | 'ready'; board?: BoardSnapshot };

export function App({ source, projectControl, projectName }: { source: BoardSource; projectControl?: ReactNode; projectName?: string }) {
  const [state, setState] = useState<LoadState>({ status: 'loading' });
  const [attempt, setAttempt] = useState(0);
  const [query, setQuery] = useState('');
  const [assignee, setAssignee] = useState('all');
  const [group, setGroup] = useState('all');
  const [connection, setConnection] = useState<ConnectionStatus>('connecting');
  const [refreshing, setRefreshing] = useState(false);
  const [stale, setStale] = useState(false);
  const [updatedTickets, setUpdatedTickets] = useState<ReadonlySet<string>>(new Set());
  const [selectedKey, setSelectedKey] = useState<string | null>(null);
  const [activityOpen, setActivityOpen] = useState(false);
  const [activity, setActivity] = useState<ActivityEntry[]>([]);
  const lastSnapshot = useRef<{ source: BoardSource; tickets: BoardTicket[] } | null>(null);
  useEffect(() => { setSelectedKey(null); setActivity([]); }, [source]);
  useEffect(() => {
    const controller = new AbortController();
    setState({ status: 'loading' });
    setConnection('connecting');
    setStale(false);
    setUpdatedTickets(new Set());
    let previousRows = lastSnapshot.current?.source === source ? lastSnapshot.current.tickets : undefined;
    let previousTickets = previousRows && new Map(previousRows.map(ticket => [ticket.id, JSON.stringify(ticket)]));
    const highlightTimers = new Map<string, ReturnType<typeof setTimeout>>();
    let transport: ConnectionStatus = 'connecting';
    let connectionGeneration = 0;
    let running = false;
    let pending = false;
    const refresh = async () => {
      pending = true;
      if (running) return;
      running = true;
      setRefreshing(true);
      while (pending && !controller.signal.aborted) {
        pending = false;
        const generation = connectionGeneration;
        try {
          const board = await source.load(controller.signal);
          if (!controller.signal.aborted && !pending && generation === connectionGeneration) {
            const nextTickets = new Map(board.tickets.map(ticket => [ticket.id, JSON.stringify(ticket)]));
            if (previousRows && source.kind === 'live') {
              const changes = ticketChanges(previousRows, board.tickets, new Date().toISOString());
              if (changes.length) setActivity(previous => [...changes, ...previous].slice(0, 200));
            }
            previousRows = board.tickets;
            lastSnapshot.current = { source, tickets: board.tickets };
            if (previousTickets && source.kind === 'live') {
              const changed = [...nextTickets].filter(([id, value]) => previousTickets!.get(id) !== value).map(([id]) => id);
              if (changed.length) {
                setUpdatedTickets(previous => new Set([...previous, ...changed]));
                for (const id of changed) {
                  clearTimeout(highlightTimers.get(id));
                  highlightTimers.set(id, setTimeout(() => {
                    highlightTimers.delete(id);
                    setUpdatedTickets(previous => {
                      const next = new Set(previous);
                      next.delete(id);
                      return next;
                    });
                  }, 1500));
                }
              }
            }
            previousTickets = nextTickets;
            setState({ status: 'ready', board });
            if (transport === 'connected') setStale(false);
          }
        } catch {
          if (!controller.signal.aborted && !pending && generation === connectionGeneration) setState(previous => ({ status: 'error', board: previous.board }));
        }
      }
      running = false;
      if (!controller.signal.aborted) setRefreshing(false);
    };
    void refresh();
    let unsubscribe: (() => void) | undefined;
    try {
      unsubscribe = source.subscribe?.(() => { void refresh(); }, status => {
        transport = status;
        setConnection(status);
        if (status === 'error' || status === 'disconnected') {
          connectionGeneration++;
          setStale(true);
        }
      });
    } catch { setConnection('error'); setStale(true); }
    return () => {
      controller.abort();
      unsubscribe?.();
      highlightTimers.forEach(timer => clearTimeout(timer));
    };
  }, [source, attempt]);

  const board = state.board;
  const displayedProjectName = board?.project.name ?? projectName;
  useEffect(() => {
    document.title = displayedProjectName ? `${displayedProjectName} · Conductor` : 'Conductor';
    return () => { document.title = 'Conductor'; };
  }, [displayedProjectName]);
  const tickets = board?.tickets ?? [];
  const ticketLinks = useMemo(() => ({ keys: new Set(board?.tickets.map(t => t.key)), open: setSelectedKey }), [board]);
  const descriptionText = useMemo(() => new Map(board?.tickets.map(ticket => [ticket.id, markdownSearchText(ticket.description)])), [board]);
  const term = query.trim();
  const filtered = tickets.filter(ticket => {
    const matchesOwner = assignee === 'all' || (assignee === 'unassigned' ? ticket.assignee === null : ticket.assignee !== null && 'owner:' + ticket.assignee.id === assignee);
    const matchesGroup = group === 'all' || (group === 'ungrouped' ? ticket.ancestors.length === 0 : 'ref:' + ticket.id === group || ticket.ancestors.some(ref => 'ref:' + ref.id === group));
    const searchable = [ticket.key, ticket.title, descriptionText.get(ticket.id) ?? '', ticket.assignee?.name ?? 'Unassigned', ...ticket.blockers.flatMap(b => [b.reason, b.ticketKey ?? ''])];
    return matchesOwner && matchesGroup && (!term || searchable.some(text => findSearchMatches(text, term).length > 0));
  });
  const owners = Array.from(new Map(tickets.flatMap(t => t.assignee ? [[t.assignee.id, t.assignee] as const] : [])).values()).sort((a, b) => a.name.localeCompare(b.name));
  const parents = Array.from(new Map(tickets.flatMap(t => t.ancestors.map(ref => [ref.id, ref] as const))).values()).sort((a, b) => a.title.localeCompare(b.title));
  const hasFilters = Boolean(query || assignee !== 'all' || group !== 'all');
  const clearFilters = () => { setQuery(''); setAssignee('all'); setGroup('all'); };

  return <ThemeProvider theme={theme}><TicketLinksContext.Provider value={ticketLinks}><CssBaseline />
    <Box component="header" sx={{ bgcolor: '#fff', borderBottom: '1px solid', borderColor: 'divider', px: { xs: 2, md: 3 }, py: 1 }}>
      <Stack direction="row" spacing={2} sx={{ alignItems: 'center', justifyContent: 'space-between' }}>
        <Stack direction="row" spacing={1.25} sx={{ alignItems: 'center' }}>
          <Box sx={{ width: 28, height: 28, borderRadius: 1.25, bgcolor: '#244f47', color: '#fff', display: 'grid', placeItems: 'center' }}><AccountTreeOutlined sx={{ fontSize: 19 }} /></Box>
          <Typography sx={{ fontWeight: 700, letterSpacing: '-.035em', fontSize: '1.15rem' }}>conductor</Typography>
          <Typography sx={{ color: '#c6cdd6', pl: 1, display: { xs: 'none', sm: 'block' } }}>/</Typography>
          <Typography variant="body2" color="text.secondary" sx={{ display: { xs: 'none', sm: 'block' } }}>{displayedProjectName ?? 'Conductor'}</Typography>
        </Stack>
        <Chip icon={<VisibilityOutlined />} label="Read only" size="small" variant="outlined" sx={{ borderColor: '#dce2e8', color: 'text.secondary', fontSize: '0.72rem' }} />
      </Stack>
    </Box>
    <Box component="main" sx={{ maxWidth: 1800, mx: 'auto', px: { xs: 2, md: 3 }, pt: 2, pb: 2 }}>
      <Stack direction="row" sx={{ alignItems: 'center', flexWrap: 'wrap', gap: 1.5, mb: 1.5 }}>
        <Typography component="h1" variant="h1" sx={{ overflowWrap: 'anywhere' }}>{displayedProjectName ? `${displayedProjectName} · Work board` : 'Work board'}</Typography>
        <Chip icon={<ScienceOutlined />} label={source.kind === 'fixture' ? 'Fixture data' : 'Live data'} size="small" sx={{ bgcolor: '#e1eee7', color: '#285d4f', fontSize: '0.7rem' }} />
        {projectControl}
        <Button size="small" aria-expanded={activityOpen} onClick={() => setActivityOpen(value => !value)}>Activity{activity.length ? ` (${activity.length})` : ''}</Button>
        {source.subscribe && <Typography role="status" variant="caption" color="text.secondary">{connection === 'disconnected' || connection === 'error' ? 'Disconnected' : refreshing ? 'Refreshing…' : connection === 'connecting' ? 'Connecting…' : 'Connected'}</Typography>}
        {board && <Stack direction="row" sx={{ ml: { sm: 'auto' }, gap: 2, flexWrap: 'wrap' }}>
          {[[tickets.length, 'tickets'], [tickets.filter(t => t.status === 'in_progress').length, 'in progress'], [tickets.filter(t => t.status === 'blocked').length, 'blocked'], [tickets.filter(t => t.status === 'review').length, 'in review']].map(([count, label]) => <Typography key={label} variant="caption" color="text.secondary"><Box component="span" sx={{ fontWeight: 700, color: 'text.primary', mr: 0.5 }}>{count}</Box>{' '}{label}</Typography>)}
        </Stack>}
      </Stack>
      {state.status === 'loading' && <Stack role="status" direction="row" spacing={2} sx={{ py: 8, justifyContent: 'center' }}><CircularProgress size={20} /><Typography>Loading board…</Typography></Stack>}
      {(state.status === 'error' || stale) && <Alert severity={board ? 'warning' : 'error'} sx={{ mb: 2 }} action={<Button color="inherit" onClick={() => setAttempt(a => a + 1)}>Try again</Button>}>{board ? 'Live data is stale. Updates are unavailable; reconnecting or retrying will refresh the board.' : 'Could not load the board. Please try again.'}</Alert>}
      {board && <>
        <Stack direction="row" sx={{ gap: 1, alignItems: 'center', flexWrap: 'wrap', mb: 1.5 }}>
          <TextField hiddenLabel size="small" placeholder="Search tickets…" value={query} onChange={e => setQuery(e.target.value)} sx={{ width: { sm: 280 }, flex: { xs: 1, sm: 'none' }, minWidth: 150, bgcolor: '#fff', '& .MuiInputBase-root': { height: 32, fontSize: '0.8rem' } }} slotProps={{ htmlInput: { 'aria-label': 'Search tickets' }, input: { startAdornment: <InputAdornment position="start"><SearchOutlined sx={{ fontSize: 17 }} /></InputAdornment> } }} />
          <FormControl variant="standard" sx={{ minWidth: 155, height: 32, px: 1, justifyContent: 'center', bgcolor: '#fff', border: '1px solid #c4cbd3', borderRadius: 1 }}>
            <NativeSelect value={assignee} onChange={e => setAssignee(e.target.value)} disableUnderline inputProps={{ id: 'assignee-filter', 'aria-label': 'Assignee' }} sx={{ fontSize: '0.8rem', '& select': { py: 0.5 } }}>
              <option value="all">All assignees</option><option value="unassigned">Unassigned</option>
              {owners.map(owner => <option key={owner.id} value={'owner:' + owner.id}>{owner.name}</option>)}
            </NativeSelect>
          </FormControl>
          <FormControl variant="standard" sx={{ minWidth: 170, maxWidth: '100%', height: 32, px: 1, justifyContent: 'center', bgcolor: '#fff', border: '1px solid #c4cbd3', borderRadius: 1 }}>
            <NativeSelect value={group} onChange={e => setGroup(e.target.value)} disableUnderline inputProps={{ 'aria-label': 'Parent' }} sx={{ fontSize: '0.8rem', '& select': { py: 0.5, maxWidth: 250, textOverflow: 'ellipsis' } }}>
              <option value="all">All parents</option><option value="ungrouped">No parent</option>
              {parents.map(ref => <option key={ref.id} value={'ref:' + ref.id}>{ref.key} · {ref.title}</option>)}
            </NativeSelect>
          </FormControl>
          {hasFilters && <Button onClick={clearFilters} size="small">Clear filters</Button>}
        </Stack>
        {filtered.length === 0 && <Paper variant="outlined" sx={{ p: 3, textAlign: 'center', mb: 2 }}>
          <Typography component="h2" variant="h3">{tickets.length === 0 ? 'No tickets yet' : 'No tickets match these filters'}</Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>{tickets.length === 0 ? 'Tickets will appear here when the source has work to show.' : 'Try another search or clear the filters to see all work.'}</Typography>
        </Paper>}
        <Box sx={{ display: 'flex', gap: 1.5, alignItems: 'flex-start', flexDirection: { xs: 'column-reverse', md: 'row' } }}>
          <Box sx={{ flex: 1, minWidth: 0, width: '100%' }}><KanbanBoard tickets={filtered} query={term} onGroupFilter={setGroup} updatedTickets={updatedTickets} onOpen={setSelectedKey} /></Box>
          {activityOpen && <ActivityFeed entries={activity} onClose={() => setActivityOpen(false)} />}
        </Box>
        <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 1 }}>Assignment shows ownership, not an active claim. This board does not change ticket state.</Typography>
      </>}
    </Box>
    <TicketDetails ticket={tickets.find(t => t.key === selectedKey)} selectedKey={selectedKey} onClose={() => setSelectedKey(null)} />
  </TicketLinksContext.Provider></ThemeProvider>;
}
