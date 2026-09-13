// One literal, Unicode-aware, case-insensitive rule for filtering and highlighting.
export function findSearchMatches(text: string, query: string): { start: number; end: number }[] {
  const term = query.trim();
  if (!term) return [];
  const escaped = term.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  return [...text.matchAll(new RegExp(escaped, 'giu'))].map(match => ({ start: match.index, end: match.index + match[0].length }));
}
