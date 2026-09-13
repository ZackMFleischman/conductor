import { unified } from 'unified';
import remarkParse from 'remark-parse';
import type { Root, Content } from 'mdast';
import type { BoardTicket } from './board';

export interface TicketAttachment { url: string; name: string; image: boolean; size?: number }
const parser = unified().use(remarkParse);
const imageExtension = /\.(?:png|jpe?g|gif|webp|avif|bmp)$/i;
const fileExtension = /\.[a-z0-9]{1,16}$/i;
export function safeAttachmentURL(value: string): boolean {
  if (!(/^https?:\/\/[^\s\\]+$/i.test(value) || /^\/(?!\/)[^\s\\]+$/.test(value))) return false;
  try { const url = new URL(value, 'http://localhost'); return !!url.hostname && !url.username && !url.password; } catch { return false; }
}

export function ticketAttachments(ticket: BoardTicket): TicketAttachment[] {
  const result = new Map<string, TicketAttachment>();
  for (const a of ticket.attachments ?? []) {
    if (safeAttachmentURL(a.url)) result.set(a.url, { url: a.url, name: a.name, size: a.size, image: /^image\/(png|jpeg|gif|webp|avif|bmp)$/.test(a.mediaType) });
  }
  for (const text of [ticket.description, ticket.summary, ticket.evidence, ticket.qa]) {
    if (!text || !text.includes('[')) continue;
    const tree = parser.parse(text) as Root;
    const definitions = new Map<string, string>();
    const walk = (node: Root | Content, visit: (node: Root | Content) => void) => {
      visit(node);
      if ('children' in node) node.children.forEach(child => walk(child, visit));
    };
    walk(tree, node => { if (node.type === 'definition') definitions.set(node.identifier.toLowerCase(), node.url); });
    walk(tree, node => {
      if (!['image', 'link', 'imageReference', 'linkReference'].includes(node.type)) return;
      const url = 'url' in node ? node.url : 'identifier' in node ? definitions.get(node.identifier.toLowerCase()) : undefined;
      if (!url || !safeAttachmentURL(url) || result.has(url)) return;
      const pathname = new URL(url, 'http://localhost').pathname;
      const image = node.type === 'image' || node.type === 'imageReference' || imageExtension.test(pathname);
      if (!image && !fileExtension.test(pathname)) return;
      let name = 'alt' in node ? node.alt : 'children' in node ? node.children.map(child => 'value' in child ? child.value : '').join('') : '';
      if (!name) { try { name = decodeURIComponent(pathname.split('/').pop() || 'Attachment'); } catch { name = 'Attachment'; } }
      result.set(url, { url, name, image });
    });
  }
  return [...result.values()];
}
