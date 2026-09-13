import { describe, expect, it } from 'vitest';
import { statuses } from '../board';
import { fixtureBoardSource } from './fixtureBoardSource';

describe('fixture board source', () => {
  it('provides clearly fictional, readable tickets across the board', async () => {
    const board = await fixtureBoardSource.load();
    expect(fixtureBoardSource.kind).toBe('fixture');
    expect(new Set(board.tickets.map(t => t.status))).toEqual(new Set(statuses));
    expect(new Set(board.tickets.map(t => t.id)).size).toBe(board.tickets.length);
    expect(board.tickets.every(t => t.key.startsWith('DEMO-') && t.description.length > 40)).toBe(true);
    expect(board.tickets.some(t => t.assignee === null)).toBe(true);
    expect(board.tickets.filter(t => t.status === 'blocked').every(t => t.blockers.length > 0)).toBe(true);
  });
  it('returns an independent snapshot on each load', async () => {
    const first = await fixtureBoardSource.load();
    first.project.name = 'changed';
    first.tickets.length = 0;
    const second = await fixtureBoardSource.load();
    expect(second.project.name).toBe('Conductor');
    expect(second.tickets.length).toBeGreaterThan(0);
  });
  it('honors an aborted request', async () => {
    await expect(fixtureBoardSource.load(AbortSignal.abort())).rejects.toMatchObject({ name: 'AbortError' });
  });
});
