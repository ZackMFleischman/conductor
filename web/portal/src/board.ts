export const statuses = ['ready', 'in_progress', 'blocked', 'review', 'done'] as const;
export type TicketStatus = (typeof statuses)[number];
export interface Assignee { id: string; name: string }
export interface Blocker { reason: string; ticketKey?: string }
export interface TicketReference { id: string; key: string; title: string }
export interface BoardTicket {
  id: string;
  key: string;
  title: string;
  description: string;
  kind?: string;
  summary?: string;
  evidence?: string;
  qa?: string;
  createdAt?: string;
  updatedAt?: string;
  status: TicketStatus;
  assignee: Assignee | null;
  blockers: Blocker[];
  ancestors: TicketReference[];
}
export interface BoardSnapshot {
  project: { id: string; name: string; description: string };
  tickets: BoardTicket[];
}
export type ConnectionStatus = 'connecting' | 'connected' | 'disconnected' | 'error';
export interface BoardSource {
  readonly kind: 'fixture' | 'live';
  load(signal?: AbortSignal): Promise<BoardSnapshot>;
  subscribe?(onChange: () => void, onStatus: (status: ConnectionStatus) => void): () => void;
}
