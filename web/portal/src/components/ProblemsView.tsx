import { useEffect, useRef, useState } from 'react';
import { Accordion, AccordionDetails, AccordionSummary, Alert, Box, Button, Dialog, DialogContent, DialogTitle, IconButton, Paper, Stack, TextField, Typography } from '@mui/material';
import Close from '@mui/icons-material/Close';
import ExpandMore from '@mui/icons-material/ExpandMore';
import type { BoardSource, ProblemDetail, ProblemSummary } from '../board';
import { TicketDescription } from './TicketDescription';
import { TicketLink } from './TicketLinks';

export function ProblemsView({ problems, onOpen }: { problems: ProblemSummary[]; onOpen: (key: string) => void }) {
  const [query, setQuery] = useState('');
  const [limit, setLimit] = useState(30);
  const needle = query.trim().toLocaleLowerCase();
  const filtered = problems.filter(p => [p.key, p.summary, p.ticketKey ?? ''].some(value => value.toLocaleLowerCase().includes(needle)));
  return <Box role="tabpanel" id="problems-panel" aria-labelledby="problems-tab" sx={{ flex: 1, minHeight: 0, overflow: 'auto', pb: 1 }}>
    <TextField value={query} onChange={event => { setQuery(event.target.value); setLimit(30); }} placeholder="Search problem reports…" size="small" slotProps={{ htmlInput: { 'aria-label': 'Search problem reports' } }} sx={{ my: 1.5, width: { xs: '100%', sm: 360 } }} />
    {!filtered.length && <Typography color="text.secondary" sx={{ py: 3 }}>{problems.length ? 'No problem reports match your search.' : 'No workflow problems reported for this project.'}</Typography>}
    <Stack spacing={1} sx={{ maxWidth: 1100 }}>{filtered.slice(0, limit).map(p => <Paper component="article" key={p.id} variant="outlined" sx={{ px: 2, py: 1.5 }}>
      <Stack direction="row" sx={{ justifyContent: 'space-between', gap: 1 }}><Typography variant="caption" sx={{ fontFamily: 'Consolas, monospace' }}><TicketLink ticketKey={p.key} /></Typography><Typography variant="caption" component="time" dateTime={p.createdAt} color="text.secondary">{new Date(p.createdAt).toLocaleString()}</Typography></Stack>
      <Button onClick={() => onOpen(p.key)} sx={{ display: 'block', textAlign: 'left', p: 0, my: 0.75, color: 'text.primary', fontWeight: 650, overflowWrap: 'anywhere' }}>{p.summary}</Button>
      <Stack direction="row" sx={{ gap: 2 }}><Typography variant="caption" color="text.secondary">{p.noteCount} {p.noteCount === 1 ? 'update' : 'updates'}</Typography>{p.ticketKey && <Typography variant="caption">Related ticket: <TicketLink ticketKey={p.ticketKey} /></Typography>}</Stack>
    </Paper>)}</Stack>
    {filtered.length > limit && <Button onClick={() => setLimit(value => value + 30)} sx={{ mt: 1 }}>Show more reports ({filtered.length - limit} remaining)</Button>}
  </Box>;
}

export function ProblemDetails({ problem, selectedKey, source, onClose }: { problem?: ProblemSummary; selectedKey: string; source: BoardSource; onClose: () => void }) {
  const [detail, setDetail] = useState<ProblemDetail>();
  const [error, setError] = useState(false);
  const [loading, setLoading] = useState(false);
  const [attempt, setAttempt] = useState(0);
  const title = useRef<HTMLHeadingElement>(null);
  useEffect(() => { title.current?.focus(); }, []);
  useEffect(() => {
    if (!problem || !source.loadProblem) return;
    const controller = new AbortController();
    setLoading(true); setError(false);
    source.loadProblem(problem.id, controller.signal).then(value => { if (!controller.signal.aborted) { setDetail(value); setLoading(false); } }).catch(() => { if (!controller.signal.aborted) { setError(true); setLoading(false); } });
    return () => controller.abort();
  }, [problem?.id, problem?.updatedAt, problem?.noteCount, source, attempt]);
  return <Dialog open fullWidth maxWidth="md" onClose={onClose} aria-labelledby="problem-details-title" slotProps={{ paper: { sx: { maxHeight: '92dvh', m: { xs: 1, sm: 3 }, borderRadius: 2 } } }}>
    <DialogTitle id="problem-details-header" component="div" sx={{ borderBottom: '1px solid', borderColor: 'divider' }}><Stack direction="row" sx={{ gap: 1, alignItems: 'flex-start' }}><Box sx={{ flex: 1, minWidth: 0 }}><Typography variant="caption" color="text.secondary">Workflow problem report</Typography><Typography ref={title} tabIndex={-1} component="h2" id="problem-details-title" sx={{ fontSize: '1.2rem', fontWeight: 650, lineHeight: 1.4, outline: 'none', overflowWrap: 'anywhere' }}>{selectedKey} · {problem?.summary ?? 'Report unavailable'}</Typography></Box><IconButton aria-label="Close problem report" onClick={onClose}><Close /></IconButton></Stack></DialogTitle>
    <DialogContent sx={{ pt: '20px !important', '& .MuiTypography-body2': { fontSize: '0.86rem', lineHeight: 1.6 } }}>
      {(!problem || !source.loadProblem) && <Alert severity="info">This problem report is unavailable.</Alert>}
      {loading && <Typography role="status" variant="caption">Loading report…</Typography>}
      {error && <Alert severity="warning" action={<Button onClick={() => { title.current?.focus(); setAttempt(value => value + 1); }}>Retry report</Button>}>{detail ? 'Report updates are unavailable. Showing the last loaded version.' : 'Could not load this report.'}</Alert>}
      {problem && detail && <Stack spacing={2.5}>
        <Typography variant="caption" color="text.secondary">Reported {new Date(detail.createdAt).toLocaleString()} · {detail.reporter}</Typography>
        {detail.ticketKey && <Typography variant="body2">Related ticket: <TicketLink ticketKey={detail.ticketKey} /></Typography>}
        {([['Expected', detail.expected], ['Actual', detail.actual], ['Correction', detail.correction]] as const).map(([label, text]) => text && <Box key={label} sx={{ p: 2, borderRadius: 1, bgcolor: label === 'Actual' ? '#fff6ed' : '#f7f8fa' }}><Typography component="h3" sx={{ fontSize: '0.9rem', fontWeight: 700, mb: 1 }}>{label}</Typography><TicketDescription text={text} query="" full /></Box>)}
        {detail.evidence && <Accordion disableGutters elevation={0}><AccordionSummary expandIcon={<ExpandMore />} sx={{ px: 0 }}><Typography component="h3" sx={{ fontSize: '0.9rem', fontWeight: 700 }}>Evidence</Typography></AccordionSummary><AccordionDetails sx={{ px: 0 }}><TicketDescription text={detail.evidence} query="" full /></AccordionDetails></Accordion>}
        <Box><Typography component="h3" sx={{ fontSize: '0.9rem', fontWeight: 700, mb: 1 }}>Updates ({detail.notes.length})</Typography>{!detail.notes.length && <Typography variant="body2" color="text.secondary">No follow-up updates yet.</Typography>}<Stack spacing={2}>{detail.notes.map(note => <Box key={note.id} sx={{ pl: 2, borderLeft: '2px solid', borderColor: 'divider' }}><Typography variant="caption" color="text.secondary">{new Date(note.createdAt).toLocaleString()} · {note.reporter}</Typography><TicketDescription text={note.body} query="" full /></Box>)}</Stack></Box>
      </Stack>}
    </DialogContent>
  </Dialog>;
}
