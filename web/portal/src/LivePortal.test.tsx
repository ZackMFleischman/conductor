import { act, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { LivePortal } from './LivePortal';

class Stream extends EventTarget {
  static CLOSED = 2;
  static all: Stream[] = [];
  readyState = 1;
  closed = false;
  constructor(public url: string) { super(); Stream.all.push(this); }
  close() { this.closed = true; this.readyState = Stream.CLOSED; }
  emit(type: string, data = '{"revision":"new"}') { this.dispatchEvent(new MessageEvent(type, { data })); }
}
const project = (id: string) => ({ id, name: id, description: '' });
const snapshot = (id: string, title: string) => ({ revision: 'r1', project: project(id), tickets: [{ id: 't', key: 'A-1', title, description: 'Description', status: 'ready', assignee: null, blockers: [], ancestors: [] }] });
const response = (body: unknown) => new Response(JSON.stringify(body), { status: 200 });
let fetcher: ReturnType<typeof vi.fn>;
beforeEach(() => {
  history.replaceState(null, '', '/');
  Stream.all = [];
  vi.stubGlobal('EventSource', Stream);
  fetcher = vi.fn(async (url: string) => response(url === '/api/v1/projects' ? { projects: [project('a'), project('b')] } : snapshot(url.includes('/a/') ? 'a' : 'b', 'First')));
  vi.stubGlobal('fetch', fetcher);
});
afterEach(() => { vi.unstubAllGlobals(); vi.useRealTimers(); });

it('highlights only new or changed live tickets, renews the cue, and clears it', async () => {
  render(<LivePortal />);
  await screen.findByText('First');
  const card = () => screen.getByRole('article', { name: /First|Updated/ });
  expect(card()).not.toHaveAttribute('data-live-updated', 'true');
  vi.useFakeTimers();
  await act(async () => { Stream.all[0].emit('board.changed'); });
  expect(card()).not.toHaveAttribute('data-live-updated', 'true');
  const changed = snapshot('a', 'Updated');
  changed.tickets[0].status = 'in_progress';
  changed.tickets.push({ ...changed.tickets[0], id: 'new', key: 'A-2', title: 'New ticket' });
  fetcher.mockImplementation(async () => response(changed));
  await act(async () => { Stream.all[0].emit('board.changed'); });
  expect(card()).toHaveAttribute('data-live-updated', 'true');
  expect(screen.getByRole('article', { name: 'New ticket' })).toHaveAttribute('data-live-updated', 'true');
  await act(async () => { vi.advanceTimersByTime(1000); });
  changed.tickets[0].description = 'Another update';
  await act(async () => { Stream.all[0].emit('board.changed'); });
  await act(async () => { vi.advanceTimersByTime(1000); });
  expect(card()).toHaveAttribute('data-live-updated', 'true');
  expect(screen.getByRole('article', { name: 'New ticket' })).not.toHaveAttribute('data-live-updated', 'true');
  await act(async () => { vi.advanceTimersByTime(2000); });
  expect(card()).not.toHaveAttribute('data-live-updated', 'true');
});

it('keeps the project in the heading and tab title across project switches and invalid selections', async () => {
  const user = userEvent.setup();
  render(<LivePortal />);
  await screen.findByRole('heading', { level: 1, name: 'a · Work board' });
  expect(document.title).toBe('a · Conductor');
  await user.selectOptions(screen.getByRole('combobox', { name: 'Project' }), 'b');
  await screen.findByRole('heading', { level: 1, name: 'b · Work board' });
  expect(document.title).toBe('b · Conductor');
  await act(async () => {
    history.pushState(null, '', '/?project=missing');
    window.dispatchEvent(new PopStateEvent('popstate'));
  });
  expect(document.title).toBe('Conductor');
});

it('initializes the selected project from the URL and preserves unrelated URL state', async () => {
  history.replaceState(null, '', '/?project=b&keep=1#anchor');
  render(<LivePortal />);
  expect(await screen.findByRole('combobox', { name: 'Project' })).toHaveValue('b');
  await waitFor(() => expect(Stream.all[0]?.url).toBe('/api/v1/projects/b/events'));
  expect(location.search).toBe('?project=b&keep=1');
  expect(location.hash).toBe('#anchor');
});

it.each([
  ['unknown', '/?project=missing', 'Unavailable project: missing'],
  ['empty', '/?project=', 'Unavailable project: (empty)'],
  ['duplicate', '/?project=a&project=b', 'Unavailable project: a'],
])('shows recovery for an %s explicit project without starting a board source', async (_case, url, label) => {
  history.replaceState(null, '', url);
  const { unmount } = render(<LivePortal />);
  expect(await screen.findByRole('alert')).toHaveTextContent(/choose a registered project/i);
  expect((screen.getByRole('option', { name: label }) as HTMLOptionElement).selected).toBe(true);
  expect(screen.queryByText('No projects registered')).not.toBeInTheDocument();
  expect(fetcher.mock.calls.filter(([requestUrl]) => requestUrl !== '/api/v1/projects')).toHaveLength(0);
  expect(Stream.all).toHaveLength(0);
  unmount();
});

it('pushes selector changes into history and retains the URL selection on remount', async () => {
  const user = userEvent.setup();
  history.replaceState(null, '', '/?keep=1#anchor');
  const first = render(<LivePortal />);
  const select = await screen.findByRole('combobox', { name: 'Project' });
  expect(select).toHaveValue('a');
  expect(location.search).toBe('?keep=1&project=a');
  await user.selectOptions(select, 'b');
  expect(location.search).toBe('?keep=1&project=b');
  expect(location.hash).toBe('#anchor');
  await waitFor(() => expect(Stream.all.at(-1)?.url).toBe('/api/v1/projects/b/events'));
  first.unmount();
  render(<LivePortal />);
  expect(await screen.findByRole('combobox', { name: 'Project' })).toHaveValue('b');
  expect(Stream.all.at(-1)?.url).toBe('/api/v1/projects/b/events');
});

it('uses back and forward popstate URLs and rejects a project removed from the loaded inventory', async () => {
  history.replaceState(null, '', '/?project=a');
  render(<LivePortal />);
  expect(await screen.findByRole('combobox', { name: 'Project' })).toHaveValue('a');
  history.pushState(null, '', '/?project=b');
  window.dispatchEvent(new PopStateEvent('popstate'));
  await waitFor(() => expect(screen.getByRole('combobox', { name: 'Project' })).toHaveValue('b'));
  await waitFor(() => expect(Stream.all.at(-1)?.url).toBe('/api/v1/projects/b/events'));
  history.replaceState(null, '', '/?project=a');
  window.dispatchEvent(new PopStateEvent('popstate'));
  await waitFor(() => expect(screen.getByRole('combobox', { name: 'Project' })).toHaveValue('a'));
  await waitFor(() => expect(Stream.all.at(-1)?.url).toBe('/api/v1/projects/a/events'));
  history.pushState(null, '', '/?project=removed');
  window.dispatchEvent(new PopStateEvent('popstate'));
  expect(await screen.findByRole('alert')).toHaveTextContent(/choose a registered project/i);
  expect(Stream.all.every(stream => stream.closed)).toBe(true);
  expect(Stream.all).toHaveLength(3);
});

it('loads live projects and refetches on invalidation and reconnect', async () => {
  render(<LivePortal />);
  expect(await screen.findByText('First')).toBeInTheDocument();
  expect(screen.queryByText('Fixture data')).not.toBeInTheDocument();
  expect(Stream.all[0].url).toBe('/api/v1/projects/a/events');
  fetcher.mockResolvedValue(response(snapshot('a', 'Updated')));
  await act(async () => { Stream.all[0].emit('open'); Stream.all[0].emit('board.changed'); });
  expect(await screen.findByText('Updated')).toBeInTheDocument();
  await act(async () => { Stream.all[0].emit('error'); });
  expect(screen.getByRole('alert')).toHaveTextContent(/stale/i);
  fetcher.mockResolvedValue(response(snapshot('a', 'Reconnected')));
  await act(async () => { Stream.all[0].emit('open'); });
  expect(screen.getByRole('alert')).toHaveTextContent(/stale/i);
  await act(async () => { Stream.all[0].emit('board.changed'); });
  expect(await screen.findByText('Reconnected')).toBeInTheDocument();
  expect(screen.queryByRole('alert')).not.toBeInTheDocument();
});

it('reopens after a server error and cancels scheduled reconnects on unmount', async () => {
  const { unmount } = render(<LivePortal />);
  await screen.findByText('First');
  vi.useFakeTimers();
  await act(async () => { Stream.all[0].emit('board.error'); });
  expect(Stream.all[0].closed).toBe(true);
  await act(async () => { vi.advanceTimersByTime(3000); });
  expect(Stream.all).toHaveLength(2);
  expect(screen.getByRole('alert')).toHaveTextContent(/stale/i);
  await act(async () => { Stream.all[1].emit('board.error'); });
  unmount();
  await act(async () => { vi.advanceTimersByTime(6000); });
  expect(Stream.all).toHaveLength(2);
});

it('replaces terminal native SSE failures such as a 503 response', async () => {
  const { unmount } = render(<LivePortal />);
  await screen.findByText('First');
  vi.useFakeTimers();
  await act(async () => { Stream.all[0].readyState = 0; Stream.all[0].emit('error'); });
  await act(async () => { vi.advanceTimersByTime(6000); });
  expect(Stream.all).toHaveLength(1);
  await act(async () => { Stream.all[0].readyState = Stream.CLOSED; Stream.all[0].emit('error'); });
  expect(screen.getByRole('alert')).toHaveTextContent(/stale/i);
  await act(async () => { vi.advanceTimersByTime(3000); });
  expect(Stream.all).toHaveLength(2);
  await act(async () => { Stream.all[1].readyState = Stream.CLOSED; Stream.all[1].emit('error'); });
  unmount();
  await act(async () => { vi.advanceTimersByTime(6000); });
  expect(Stream.all).toHaveLength(2);
});

it('coalesces refreshes and never lets an obsolete snapshot overwrite the latest data', async () => {
  render(<LivePortal />);
  await screen.findByText('First');
  let finish!: (response: Response) => void;
  fetcher.mockImplementationOnce(() => new Promise<Response>(resolve => { finish = resolve; }));
  await act(async () => { Stream.all[0].emit('board.changed'); });
  const count = fetcher.mock.calls.length;
  await act(async () => { Stream.all[0].emit('board.changed'); Stream.all[0].emit('board.changed'); });
  expect(fetcher.mock.calls.length).toBe(count);
  fetcher.mockResolvedValue(response(snapshot('a', 'Latest')));
  await act(async () => { finish(response(snapshot('a', 'Obsolete'))); });
  expect(await screen.findByText('Latest')).toBeInTheDocument();
  expect(screen.queryByText('Obsolete')).not.toBeInTheDocument();
  expect(fetcher.mock.calls.length).toBe(count + 1);
});

it('does not mark a pre-disconnect request fresh after the connection reopens', async () => {
  render(<LivePortal />);
  await screen.findByText('First');
  let finish!: (response: Response) => void;
  fetcher.mockImplementationOnce(() => new Promise<Response>(resolve => { finish = resolve; }));
  await act(async () => { Stream.all[0].emit('open'); Stream.all[0].emit('board.changed'); });
  await act(async () => { Stream.all[0].readyState = 0; Stream.all[0].emit('error'); Stream.all[0].emit('open'); });
  await act(async () => { finish(response(snapshot('a', 'Obsolete'))); });
  expect(screen.getByRole('alert')).toHaveTextContent(/stale/i);
  expect(screen.queryByText('Obsolete')).not.toBeInTheDocument();
  fetcher.mockResolvedValue(response(snapshot('a', 'Fresh')));
  await act(async () => { Stream.all[0].emit('board.changed'); });
  expect(await screen.findByText('Fresh')).toBeInTheDocument();
  expect(screen.queryByRole('alert')).not.toBeInTheDocument();
});

it('aborts old project loads and removes its stream listeners on switch and unmount', async () => {
  const user = userEvent.setup();
  const { unmount } = render(<LivePortal />);
  await screen.findByText('First');
  let finish!: (response: Response) => void;
  fetcher.mockImplementationOnce(() => new Promise<Response>(resolve => { finish = resolve; }));
  await act(async () => { Stream.all[0].emit('board.changed'); });
  const oldSignal = fetcher.mock.calls.at(-1)![1].signal as AbortSignal;
  await user.selectOptions(screen.getByRole('combobox', { name: 'Project' }), 'b');
  await screen.findByText('First');
  expect(oldSignal.aborted).toBe(true);
  expect(Stream.all[0].closed).toBe(true);
  const count = fetcher.mock.calls.length;
  await act(async () => { Stream.all[0].emit('board.changed'); finish(response(snapshot('a', 'Obsolete'))); });
  expect(fetcher.mock.calls.length).toBe(count);
  expect(screen.queryByText('Obsolete')).not.toBeInTheDocument();
  const signal = fetcher.mock.calls.at(-1)![1].signal as AbortSignal;
  unmount();
  expect(signal.aborted).toBe(true);
  expect(Stream.all[1].closed).toBe(true);
});

it('retains stale data on refresh failure and supports explicit retry after board.error', async () => {
  const user = userEvent.setup();
  render(<LivePortal />);
  await screen.findByText('First');
  fetcher.mockRejectedValue(new Error('private path'));
  await act(async () => { Stream.all[0].emit('board.changed'); });
  expect(await screen.findByRole('alert')).toHaveTextContent(/stale/i);
  expect(screen.getByText('First')).toBeInTheDocument();
  expect(screen.queryByText('private path')).not.toBeInTheDocument();
  await act(async () => { Stream.all[0].emit('board.error', '{"code":"STORE_UNAVAILABLE"}'); });
  expect(Stream.all[0].closed).toBe(true);
  fetcher.mockResolvedValue(response(snapshot('a', 'Recovered')));
  await user.click(screen.getByRole('button', { name: 'Try again' }));
  expect(await screen.findByText('Recovered')).toBeInTheDocument();
  expect(Stream.all).toHaveLength(2);
});

it('shows safe project errors, retries, and distinguishes an empty registry', async () => {
  const user = userEvent.setup();
  fetcher.mockRejectedValue(new Error('private path'));
  render(<LivePortal />);
  expect(await screen.findByRole('alert')).toHaveTextContent(/projects/i);
  expect(screen.queryByText('Fixture data')).not.toBeInTheDocument();
  fetcher.mockResolvedValue(response({ projects: [] }));
  await user.click(screen.getByRole('button', { name: 'Try again' }));
  await waitFor(() => expect(screen.getByText('No projects registered')).toBeInTheDocument());
  expect(Stream.all).toHaveLength(0);
});
