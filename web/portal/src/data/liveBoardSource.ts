import { statuses, type BoardSnapshot, type BoardSource, type BoardTicket } from '../board';

export type Project = BoardSnapshot['project'];
export type LiveSnapshot = BoardSnapshot & { revision: string };
const invalid = () => new Error('Invalid live response');
function object(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw invalid();
  return value as Record<string, unknown>;
}
function text(value: unknown, nonempty = false): string {
  if (typeof value !== 'string' || (nonempty && !value.trim())) throw invalid();
  return value;
}
function array(value: unknown): unknown[] {
  if (!Array.isArray(value)) throw invalid();
  return value;
}
function project(value: unknown): Project {
  const p = object(value);
  return { id: text(p.id, true), name: text(p.name, true), description: text(p.description) };
}
function reference(value: unknown) {
  const r = object(value);
  return { id: text(r.id, true), key: text(r.key, true), title: text(r.title) };
}
export function decodeProjects(value: unknown): Project[] {
  const projects = array(object(value).projects).map(project);
  if (new Set(projects.map(p => p.id)).size !== projects.length) throw invalid();
  return projects;
}
export function decodeBoard(value: unknown, projectId: string): LiveSnapshot {
  const data = object(value);
  const boardProject = project(data.project);
  if (boardProject.id !== projectId) throw invalid();
  const tickets: BoardTicket[] = array(data.tickets).map(value => {
    const t = object(value);
    if (!statuses.some(s => s === t.status)) throw invalid();
    const owner = t.assignee === null ? null : object(t.assignee);
    return {
      ...reference(t), description: text(t.description), status: t.status as BoardTicket['status'],
      ...Object.fromEntries(['kind', 'summary', 'evidence', 'qa', 'createdAt', 'updatedAt'].filter(key => t[key] !== undefined).map(key => [key, text(t[key])])),
      assignee: owner === null ? null : { id: text(owner.id, true), name: text(owner.name, true) },
      blockers: array(t.blockers).map(value => {
        const b = object(value);
        return { reason: text(b.reason), ...(b.ticketKey === undefined ? {} : { ticketKey: text(b.ticketKey, true) }) };
      }),
      ancestors: array(t.ancestors).map(reference),
    };
  });
  const byId = new Map(tickets.map(t => [t.id, t]));
  const keys = new Set(tickets.map(t => t.key));
  if (byId.size !== tickets.length || keys.size !== tickets.length) throw invalid();
  const owners = new Map<string, string>();
  for (const t of tickets) {
    const seen = new Set([t.id]);
    for (const ref of t.ancestors) {
      const row = byId.get(ref.id);
      if (seen.has(ref.id) || !row || row.key !== ref.key || row.title !== ref.title) throw invalid();
      seen.add(ref.id);
    }
    // Every parent is present in v1. Its chain must equal the child's remaining chain.
    const parent = t.ancestors[0] && byId.get(t.ancestors[0].id);
    if (parent && (parent.ancestors.length !== t.ancestors.length - 1 || parent.ancestors.some((ref, i) => ref.id !== t.ancestors[i + 1].id))) throw invalid();
    if (t.blockers.some(b => b.ticketKey !== undefined && !keys.has(b.ticketKey))) throw invalid();
    if (t.assignee) {
      const known = owners.get(t.assignee.id);
      if (known !== undefined && known !== t.assignee.name) throw invalid();
      owners.set(t.assignee.id, t.assignee.name);
    }
  }
  return { project: boardProject, tickets, revision: text(data.revision, true) };
}
async function getJSON(url: string, signal?: AbortSignal): Promise<unknown> {
  const response = await fetch(url, { signal, cache: 'no-store', headers: { Accept: 'application/json' } });
  if (!response.ok) throw new Error('Live service unavailable');
  return response.json();
}
export async function loadProjects(signal?: AbortSignal): Promise<Project[]> {
  return decodeProjects(await getJSON('/api/v1/projects', signal));
}
export function createLiveBoardSource(projectId: string): BoardSource {
  const base = `/api/v1/projects/${encodeURIComponent(projectId)}`;
  return {
    kind: 'live',
    async load(signal) { return decodeBoard(await getJSON(`${base}/board`, signal), projectId); },
    subscribe(onChange, onStatus) {
      let stopped = false;
      let timer: ReturnType<typeof setTimeout> | undefined;
      let closeCurrent = () => {};
      const connect = () => {
        if (stopped) return;
        onStatus('connecting');
        const stream = new EventSource(`${base}/events`);
        const opened = () => onStatus('connected');
        const disconnected = () => {
          onStatus('disconnected');
          // Native retry only continues in CONNECTING; HTTP failures can be terminal.
          if (stream.readyState === EventSource.CLOSED) failed();
        };
        const failed = () => {
          onStatus('error');
          closeCurrent();
          timer = setTimeout(connect, 3000);
        };
        const changed = (event: Event) => {
          try { text(object(JSON.parse((event as MessageEvent<string>).data)).revision, true); }
          catch { failed(); return; }
          onStatus('connected');
          onChange();
        };
        const listeners = { open: opened, error: disconnected, 'board.error': failed, 'board.changed': changed };
        for (const [type, listener] of Object.entries(listeners)) stream.addEventListener(type, listener);
        closeCurrent = () => {
          for (const [type, listener] of Object.entries(listeners)) stream.removeEventListener(type, listener);
          stream.close();
        };
      };
      connect();
      return () => {
        stopped = true;
        clearTimeout(timer);
        closeCurrent();
      };
    },
  };
}
