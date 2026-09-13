import { Fragment } from 'react';
import { findSearchMatches } from '../search';

export function HighlightedText({ text, query }: { text: string; query: string }) {
  const matches = findSearchMatches(text, query);
  let cursor = 0;
  return <>{matches.map(({ start, end }) => {
    const before = text.slice(cursor, start);
    cursor = end;
    return <Fragment key={start}>{before}<mark>{text.slice(start, end)}</mark></Fragment>;
  })}{text.slice(cursor)}</>;
}
