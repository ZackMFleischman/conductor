import type { ElementContent, Root, RootContent, Text } from 'hast';
import { unified } from 'unified';
import remarkParse from 'remark-parse';
import remarkGfm from 'remark-gfm';
import remarkRehype from 'remark-rehype';
import { findSearchMatches } from '../search';

const parser = unified().use(remarkParse).use(remarkGfm).use(remarkRehype, { allowDangerousHtml: true });
const blocks = new Set(['p', 'div', 'pre', 'blockquote', 'ul', 'ol', 'li', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'table', 'tr', 'th', 'td', 'br', 'hr']);

// Keep inline runs together, with boundaries between blocks. Offsets still map
// to the original text nodes, so a phrase can be marked across emphasis/links.
function collectText(tree: Root) {
  let text = '';
  const offsets = new Map<Text, number>();
  function visit(node: Root | RootContent) {
    if (node.type === 'text' || node.type === 'raw') {
      // react-markdown displays raw HTML as inert text; match that same content.
      offsets.set(node as Text, text.length);
      text += node.value.replace(/\s/g, ' ');
    } else if (node.type === 'root' || node.type === 'element') {
      const boundary = node.type === 'element' && blocks.has(node.tagName);
      if (boundary) text += '\n';
      node.children.forEach(visit);
      if (boundary) text += '\n';
    }
  }
  visit(tree);
  return { text, offsets };
}

export function markdownSearchText(markdown: string) {
  return collectText(parser.runSync(parser.parse(markdown)) as Root).text;
}

export function highlightMarkdown({ query }: { query: string }) {
  return (tree: Root) => {
    const { text, offsets } = collectText(tree);
    const matches = findSearchMatches(text, query);
    if (!matches.length) return;
    function visit(node: Root | RootContent) {
      if (node.type !== 'root' && node.type !== 'element') return;
      for (let index = 0; index < node.children.length; index++) {
        const child = node.children[index];
        if (child.type !== 'text' && child.type !== 'raw') { visit(child); continue; }
        const offset = offsets.get(child as Text)!;
        const local = matches.filter(match => match.end > offset && match.start < offset + child.value.length);
        if (!local.length) continue;
        const replacement: ElementContent[] = [];
        let cursor = 0;
        for (const match of local) {
          const start = Math.max(0, match.start - offset);
          const end = Math.min(child.value.length, match.end - offset);
          if (start > cursor) replacement.push({ type: 'text', value: child.value.slice(cursor, start) });
          replacement.push({ type: 'element', tagName: 'mark', properties: {}, children: [{ type: 'text', value: child.value.slice(start, end) }] });
          cursor = end;
        }
        if (cursor < child.value.length) replacement.push({ type: 'text', value: child.value.slice(cursor) });
        node.children.splice(index, 1, ...replacement);
        index += replacement.length - 1;
      }
    }
    visit(tree);
  };
}
