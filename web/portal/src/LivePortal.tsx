import { useEffect, useMemo, useState } from 'react';
import { Alert, Box, Button, CssBaseline, FormControl, NativeSelect, ThemeProvider, Typography } from '@mui/material';
import { App } from './App';
import { createLiveBoardSource, loadProjects, type Project } from './data/liveBoardSource';
import { theme } from './theme';

export function LivePortal() {
  const [projects, setProjects] = useState<Project[]>();
  const [selected, setSelected] = useState('');
  const [error, setError] = useState(false);
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    setError(false);
    void loadProjects(controller.signal).then(items => {
      if (controller.signal.aborted) return;
      setProjects(items);
      setSelected(items[0]?.id ?? '');
    }).catch(() => { if (!controller.signal.aborted) setError(true); });
    return () => controller.abort();
  }, [attempt]);
  const source = useMemo(() => selected ? createLiveBoardSource(selected) : undefined, [selected]);
  if (source && projects) return <App key={selected} source={source} projectControl={
    <FormControl variant="standard" sx={{ minWidth: 130, maxWidth: '100%', px: 1, bgcolor: '#fff', border: '1px solid #c4cbd3', borderRadius: 1 }}>
      <NativeSelect value={selected} onChange={event => setSelected(event.target.value)} disableUnderline inputProps={{ 'aria-label': 'Project' }} sx={{ fontSize: '0.8rem', maxWidth: 260 }}>
        {projects.map(p => <option key={p.id} value={p.id}>{p.name}</option>)}
      </NativeSelect>
    </FormControl>
  } />;
  return <ThemeProvider theme={theme}><CssBaseline /><Box component="main" sx={{ p: 3 }}>
    <Typography component="h1" variant="h1" sx={{ mb: 2 }}>Work board</Typography>
    <Typography variant="caption">Live data · Read only</Typography>
    {error ? <Alert severity="error" action={<Button color="inherit" onClick={() => setAttempt(a => a + 1)}>Try again</Button>}>Could not load projects. Please try again.</Alert>
      : <Typography role="status" sx={{ mt: 2 }}>{projects ? 'No projects registered' : 'Loading projects…'}</Typography>}
  </Box></ThemeProvider>;
}
