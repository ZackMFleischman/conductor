import { describe, expect, it } from 'vitest';
import { decodeBoard, decodeProjects } from './liveBoardSource';

export const liveSnapshot = (projectId = 'project-a', title = 'Child') => ({
  revision: 'opaque:1', project: { id: projectId, name: projectId, description: '' },
  tickets: [
    { id: 'root', key: 'A-1', title: 'Root', description: '', status: 'ready', assignee: null, blockers: [], ancestors: [] },
    { id: 'child', key: 'A-2', title, description: 'Full description', status: 'blocked', assignee: { id: 'owner', name: 'Owner' }, blockers: [{ ticketKey: 'A-1', reason: 'Waiting' }], ancestors: [{ id: 'root', key: 'A-1', title: 'Root' }] },
  ],
});

describe('live JSON decoding', () => {
  it('accepts a complete snapshot including null assignment and opaque revision', () => {
    expect(decodeBoard(liveSnapshot(), 'project-a').tickets[0].assignee).toBeNull();
    expect(decodeBoard(liveSnapshot(), 'project-a').revision).toBe('opaque:1');
    expect(decodeProjects({ projects: [] })).toEqual([]);
  });
  it.each([
    ['missing revision', (v: any) => { delete v.revision; }],
    ['wrong project', (v: any) => { v.project.id = 'other'; }],
    ['invalid status', (v: any) => { v.tickets[0].status = 'draft'; }],
    ['missing null assignment', (v: any) => { delete v.tickets[0].assignee; }],
    ['missing array', (v: any) => { delete v.tickets[0].ancestors; }],
    ['duplicate id', (v: any) => { v.tickets[1].id = 'root'; }],
    ['duplicate key', (v: any) => { v.tickets[1].key = 'A-1'; }],
    ['unknown ancestor', (v: any) => { v.tickets[1].ancestors[0].id = 'missing'; }],
    ['inconsistent reference', (v: any) => { v.tickets[1].ancestors[0].title = 'Wrong'; }],
    ['cyclic ancestor', (v: any) => { v.tickets[0].ancestors = [{ id: 'root', key: 'A-1', title: 'Root' }]; }],
    ['unknown blocker', (v: any) => { v.tickets[1].blockers[0].ticketKey = 'OTHER-1'; }],
    ['inconsistent assignee', (v: any) => { v.tickets[0].assignee = { id: 'owner', name: 'Other' }; }],
    ['truncated ancestors', (v: any) => { v.tickets.push({ ...v.tickets[1], id: 'leaf', key: 'A-3', ancestors: [{ id: 'child', key: 'A-2', title: 'Child' }] }); }],
  ])('rejects %s', (_, mutate) => {
    const value = liveSnapshot(); mutate(value);
    expect(() => decodeBoard(value, 'project-a')).toThrow();
  });
  it('rejects malformed or duplicate projects', () => {
    expect(() => decodeProjects({ projects: null })).toThrow();
    expect(() => decodeProjects({ projects: [{ id: 'a', name: 'A' }] })).toThrow();
    const project = { id: 'a', name: 'A', description: '' };
    expect(() => decodeProjects({ projects: [project, project] })).toThrow();
  });
});
