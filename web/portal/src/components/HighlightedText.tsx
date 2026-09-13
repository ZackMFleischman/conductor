import { Fragment } from 'react';
import { findSearchMatches } from '../search';
import { TicketLink } from './TicketLinks';

export function HighlightedText({ text, query }: { text: string; query: string }) {
  const matches = findSearchMatches(text, query);
  let offset = 0;
  return <>{text.split(/(\b[A-Za-z][A-Za-z0-9_]*-P?\d+\b)/g).map((part, i) => {
    const local = matches.filter(m => m.end > offset && m.start < offset + part.length).map(m => ({ start: Math.max(0, m.start - offset), end: Math.min(part.length, m.end - offset) }));
    offset += part.length;
    return <TicketLink key={i} ticketKey={part}><MarkedText text={part} matches={local} /></TicketLink>;
  })}</>;
}

function MarkedText({ text, matches }: { text: string; matches: { start: number; end: number }[] }) {
  let cursor = 0;
  return <>{matches.map(({ start, end }) => {
    const before = text.slice(cursor, start);
    cursor = end;
    return <Fragment key={start}>{before}<mark>{text.slice(start, end)}</mark></Fragment>;
  })}{text.slice(cursor)}</>;
}
