import type { TicketNotesPage } from './data/ticketNotes';
export const statuses = ['ready', 'in_progress', 'blocked', 'review', 'done'] as const;
export type TicketStatus = (typeof statuses)[number];
export interface Assignee { id: string; name: string }
export interface Blocker { reason: string; ticketKey?: string }
export interface TicketReference { id: string; key: string; title: string }
export interface Attachment { id: string; name: string; url: string; mediaType: string; size: number; modifiedAt?: string }
export interface BoardTicket {
  attachments?: Attachment[];
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
  completedAt?: string;
  status: TicketStatus;
  assignee: Assignee | null;
  blockers: Blocker[];
  ancestors: TicketReference[];
}
export interface BoardSnapshot {
  problems?: ProblemSummary[];
  project: { id: string; name: string; description: string };
  tickets: BoardTicket[];
}
export type ConnectionStatus = 'connecting' | 'connected' | 'disconnected' | 'error';
export interface BoardSource {
  loadTicketNotes?(id: string, before?: string, signal?: AbortSignal): Promise<TicketNotesPage>;
  loadProblem?(id: string, signal?: AbortSignal): Promise<ProblemDetail>;
  readonly kind: 'fixture' | 'live';
  load(signal?: AbortSignal): Promise<BoardSnapshot>;
  subscribe?(onChange: () => void, onStatus: (status: ConnectionStatus) => void): () => void;
}
export interface ProblemSummary { id: string; key: string; summary: string; ticketKey?: string; createdAt: string; updatedAt: string; noteCount: number }
export interface ProblemNote { id: string; body: string; createdAt: string; reporter: string }
export interface ProblemDetail extends ProblemSummary { expected: string; actual: string; correction: string; evidence: string; reporter: string; notes: ProblemNote[] }
