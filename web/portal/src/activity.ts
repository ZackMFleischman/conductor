import type { BoardTicket, ProblemSummary } from './board';

export interface ActivityEntry { id: string; key: string; title: string; at: string; changes: string[] }
export function problemChanges(previous: readonly ProblemSummary[], next: readonly ProblemSummary[], at: string): ActivityEntry[] {
  const before = new Map(previous.map(p => [p.id, p]));
  return next.flatMap(problem => {
    const old = before.get(problem.id);
    if (old && JSON.stringify(old) === JSON.stringify(problem)) return [];
    return [{ id: `${at}:problem:${problem.id}`, key: problem.key, title: problem.summary, at, changes: [!old ? 'Problem reported' : problem.noteCount > old.noteCount ? 'Problem report update added' : 'Problem report updated'] }];
  });
}
const statusName = (status: string) => ({ ready: 'Ready', in_progress: 'In progress', blocked: 'Blocked', review: 'Review', done: 'Done' })[status] ?? status;

export function ticketChanges(previous: readonly BoardTicket[], next: readonly BoardTicket[], at: string): ActivityEntry[] {
  const before = new Map(previous.map(t => [t.id, t]));
  const entries: ActivityEntry[] = [];
  for (const ticket of next) {
    const old = before.get(ticket.id);
    before.delete(ticket.id);
    const changes: string[] = [];
    if (!old) changes.push('Ticket created');
    else {
      if (old.status !== ticket.status) changes.push(`${statusName(old.status)} → ${statusName(ticket.status)}`);
      if (old.title !== ticket.title) changes.push(`Title: ${old.title} → ${ticket.title}`);
      if (JSON.stringify(old.assignee) !== JSON.stringify(ticket.assignee)) changes.push(`Assignee: ${old.assignee?.name ?? 'Unassigned'} → ${ticket.assignee?.name ?? 'Unassigned'}`);
      if (old.kind !== ticket.kind) changes.push(`Type: ${old.kind ?? 'implementation'} → ${ticket.kind ?? 'implementation'}`);
      for (const [field, label] of [['description', 'Description'], ['summary', 'Summary'], ['evidence', 'Evidence'], ['qa', 'QA notes'], ['blockers', 'Blockers'], ['ancestors', 'Parents'], ['attachments', 'Attachments']] as const) {
        if (JSON.stringify(old[field]) !== JSON.stringify(ticket[field])) changes.push(`${label} updated`);
      }
      if (!changes.length && old.updatedAt !== ticket.updatedAt) changes.push('Ticket updated');
    }
    if (changes.length) entries.push({ id: `${at}:${ticket.id}`, key: ticket.key, title: ticket.title, at, changes });
  }
  for (const ticket of before.values()) entries.push({ id: `${at}:${ticket.id}`, key: ticket.key, title: ticket.title, at, changes: ['Ticket removed from board'] });
  return entries;
}
