import { createContext, useContext, type ReactNode } from 'react';
import type { Root, RootContent, ElementContent } from 'hast';

export const TicketLinksContext = createContext<{ keys: ReadonlySet<string>; open: (key: string) => void }>({ keys: new Set(), open: () => {} });
export const ticketHref = (key: string) => `#ticket=${encodeURIComponent(key)}`;

export function TicketLink({ ticketKey, children }: { ticketKey: string; children?: ReactNode }) {
  const { keys, open } = useContext(TicketLinksContext);
  if (!keys.has(ticketKey)) return <>{children ?? ticketKey}</>;
  return <a href={ticketHref(ticketKey)} onClick={event => { event.preventDefault(); event.stopPropagation(); open(ticketKey); }} style={{ color: '#285d4f', textDecoration: 'underline' }}>{children ?? ticketKey}</a>;
}

// Link only known project keys in prose. Existing links and code stay intact.
export function linkTicketReferences({ keys }: { keys: ReadonlySet<string> }) {
  return (tree: Root) => {
    function visit(node: Root | RootContent) {
      if (node.type !== 'root' && node.type !== 'element') return;
      if (node.type === 'element' && ['a', 'code', 'pre'].includes(node.tagName)) return;
      node.children = node.children.flatMap((child): ElementContent[] => {
        if (child.type !== 'text') { visit(child); return [child as ElementContent]; }
        const result: ElementContent[] = [];
        let cursor = 0;
        for (const match of child.value.matchAll(/\b[A-Za-z][A-Za-z0-9_]*-\d+\b/g)) {
          if (!keys.has(match[0])) continue;
          result.push({ type: 'text', value: child.value.slice(cursor, match.index) });
          result.push({ type: 'element', tagName: 'a', properties: { href: ticketHref(match[0]) }, children: [{ type: 'text', value: match[0] }] });
          cursor = match.index + match[0].length;
        }
        result.push({ type: 'text', value: child.value.slice(cursor) });
        return result;
      });
    }
    visit(tree);
  };
}
