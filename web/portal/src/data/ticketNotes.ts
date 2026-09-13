export type TicketNote = { id: string; author: string; body: string; createdAt: string };
export type TicketNotesPage = { ticketId: string; notes: TicketNote[]; total: number; nextCursor?: string };
export type TicketNotesLoader = (before?: string, signal?: AbortSignal) => Promise<TicketNotesPage>;

export function decodeTicketNotes(value: unknown, ticketId: string): TicketNotesPage {
 const invalid = () => new Error('Invalid ticket comments response');
 if (!value || typeof value !== 'object' || Array.isArray(value)) throw invalid();
 const data = value as Record<string, unknown>;
 if (data.ticketId !== ticketId || !Number.isSafeInteger(data.total) || (data.total as number) < 0 || !Array.isArray(data.notes)) throw invalid();
 const notes = data.notes.map(value => {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw invalid();
  const note = value as Record<string, unknown>;
  for (const key of ['id', 'author', 'body', 'createdAt']) if (typeof note[key] !== 'string') throw invalid();
  if (!note.id || !note.author) throw invalid();
  return { id: note.id as string, author: note.author as string, body: note.body as string, createdAt: note.createdAt as string };
 });
 if (new Set(notes.map(note => note.id)).size !== notes.length || notes.length > 20 || notes.length > (data.total as number)) throw invalid();
 if (data.nextCursor !== undefined && (typeof data.nextCursor !== 'string' || !data.nextCursor)) throw invalid();
 return { ticketId, notes, total: data.total as number, ...(data.nextCursor === undefined ? {} : { nextCursor: data.nextCursor as string }) };
}

export async function loadTicketNotes(projectId: string, ticketId: string, before?: string, signal?: AbortSignal): Promise<TicketNotesPage> {
 const query = before ? `?${new URLSearchParams({ before })}` : '';
 const response = await fetch(`/api/v1/projects/${encodeURIComponent(projectId)}/tickets/${encodeURIComponent(ticketId)}/notes${query}`, { signal, cache: 'no-store' });
 if (!response.ok) throw new Error('Comments could not be loaded');
 return decodeTicketNotes(await response.json(), ticketId);
}
