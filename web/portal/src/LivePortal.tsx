import { useEffect, useMemo, useState } from 'react';
import { Alert, Box, Button, CssBaseline, FormControl, NativeSelect, ThemeProvider, Typography } from '@mui/material';
import { App } from './App';
import { createLiveBoardSource, loadProjects, type Project } from './data/liveBoardSource';
import { theme } from './theme';

type URLProject = { explicit: boolean; id: string; wellFormed: boolean };

function readURLProject(): URLProject {
  const values = new URL(window.location.href).searchParams.getAll('project');
  if (values.length === 0) return { explicit: false, id: '', wellFormed: true };
  return { explicit: true, id: values[0], wellFormed: values.length === 1 && values[0] !== '' };
}

function writeURLProject(id: string, mode: 'push' | 'replace') {
  const url = new URL(window.location.href);
  url.searchParams.delete('project');
  url.searchParams.append('project', id);
  window.history[`${mode}State`](null, '', `${url.pathname}${url.search}${url.hash}`);
}

export function LivePortal() {
  const [projects, setProjects] = useState<Project[]>();
  const [urlProject, setURLProject] = useState(readURLProject);
  const [error, setError] = useState(false);
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    setError(false);
    void loadProjects(controller.signal).then(items => {
      if (controller.signal.aborted) return;
      setProjects(items);
      const requested = readURLProject();
      if (!requested.explicit && items[0]) {
        writeURLProject(items[0].id, 'replace');
        setURLProject({ explicit: true, id: items[0].id, wellFormed: true });
      } else {
        setURLProject(requested);
      }
    }).catch(() => { if (!controller.signal.aborted) setError(true); });
    return () => controller.abort();
  }, [attempt]);
  useEffect(() => {
    const onPopState = () => {
      const requested = readURLProject();
      if (!requested.explicit && projects?.[0]) {
        writeURLProject(projects[0].id, 'replace');
        setURLProject({ explicit: true, id: projects[0].id, wellFormed: true });
      } else {
        setURLProject(requested);
      }
    };
    window.addEventListener('popstate', onPopState);
    return () => window.removeEventListener('popstate', onPopState);
  }, [projects]);
  const selected = urlProject.wellFormed && projects?.some(project => project.id === urlProject.id) ? urlProject.id : '';
  const invalidSelection = Boolean(projects?.length && urlProject.explicit && !selected);
  const selectProject = (id: string) => {
    writeURLProject(id, 'push');
    setURLProject({ explicit: true, id, wellFormed: true });
  };
  const projectControl = projects?.length ? <FormControl variant="standard" sx={{ minWidth: 130, maxWidth: '100%', px: 1, bgcolor: '#fff', border: '1px solid #c4cbd3', borderRadius: 1 }}>
    <NativeSelect value={invalidSelection ? '__invalid_project__' : selected} onChange={event => selectProject(event.target.value)} disableUnderline inputProps={{ 'aria-label': 'Project' }} sx={{ fontSize: '0.8rem', maxWidth: 260 }}>
      {invalidSelection && <option value="__invalid_project__">Unavailable project: {urlProject.id || '(empty)'}</option>}
      {projects.map(p => <option key={p.id} value={p.id}>{p.name}</option>)}
    </NativeSelect>
  </FormControl> : undefined;
  const source = useMemo(() => selected ? createLiveBoardSource(selected) : undefined, [selected]);
  if (source && projects) return <App key={selected} source={source} projectControl={
    projectControl
  } />;
  return <ThemeProvider theme={theme}><CssBaseline /><Box component="main" sx={{ p: 3 }}>
    <Typography component="h1" variant="h1" sx={{ mb: 2 }}>Work board</Typography>
    <Typography variant="caption">Live data · Read only</Typography>
    {invalidSelection && <Box sx={{ mt: 2 }}>{projectControl}<Alert severity="warning" sx={{ mt: 2 }}>That project is unavailable. Choose a registered project.</Alert></Box>}
    {error ? <Alert severity="error" action={<Button color="inherit" onClick={() => setAttempt(a => a + 1)}>Try again</Button>}>Could not load projects. Please try again.</Alert>
      : <Typography role="status" sx={{ mt: 2 }}>{projects ? 'No projects registered' : 'Loading projects…'}</Typography>}
  </Box></ThemeProvider>;
}
