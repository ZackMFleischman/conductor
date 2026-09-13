export const statuses = ['ready', 'in_progress', 'blocked', 'review', 'done'] as const;
export type TicketStatus = (typeof statuses)[number];
export interface Assignee { id: string; name: string }
export interface Blocker { reason: string; ticketKey?: string }
export interface BoardTicket {
  id: string;
  key: string;
  title: string;
  description: string;
  status: TicketStatus;
  assignee: Assignee | null;
  blockers: Blocker[];
}
export interface BoardSnapshot {
  project: { id: string; name: string; description: string };
  tickets: BoardTicket[];
}
export interface BoardSource {
  readonly kind: 'fixture' | 'live';
  load(signal?: AbortSignal): Promise<BoardSnapshot>;
}
