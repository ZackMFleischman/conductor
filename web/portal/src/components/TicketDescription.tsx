import { memo, useContext, useId, useLayoutEffect, useRef, useState } from 'react';
import { Box, Button, Typography } from '@mui/material';
import Markdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { highlightMarkdown } from './highlightMarkdown';
import { linkTicketReferences, TicketLinksContext } from './TicketLinks';

export const TicketDescription = memo(function TicketDescription({ text, query, full = false, hideImages = false }: { text: string; query: string; full?: boolean; hideImages?: boolean }) {
  const { keys, open } = useContext(TicketLinksContext);
  const id = useId();
  const element = useRef<HTMLDivElement>(null);
  const [expanded, setExpanded] = useState(false);
  const [overflows, setOverflows] = useState(false);
  useLayoutEffect(() => {
    const node = element.current;
    if (!node) return;
    const measure = () => {
      const lineHeight = Number.parseFloat(getComputedStyle(node).lineHeight);
      setOverflows(node.scrollHeight > lineHeight * 4 + 1);
    };
    measure();
    if (typeof ResizeObserver === 'undefined') {
      window.addEventListener('resize', measure);
      return () => window.removeEventListener('resize', measure);
    }
    const observer = new ResizeObserver(measure);
    observer.observe(node);
    return () => observer.disconnect();
  }, [text, query]);
  return <>
    <Typography component="div" id={id} variant="body2" color="text.secondary" sx={{
      maxHeight: full || expanded ? 'none' : '5.6em', overflow: 'hidden', lineHeight: 1.4,
    }}>
      <Box ref={element} sx={{
        overflowWrap: 'anywhere',
        '& > :first-child': { mt: 0 }, '& > :last-child': { mb: 0 },
        '& p, & ul, & ol, & blockquote, & pre, & table': { my: 0.75 },
        '& h1, & h2, & h3, & h4, & h5, & h6': { fontSize: '1.08em', lineHeight: 1.4, fontWeight: 700, color: 'text.primary', mt: 1, mb: 0.5 },
        '& ul, & ol': { pl: 2.25 }, '& li > p': { my: 0.25 },
        '& blockquote': { ml: 0, mr: 0, pl: 1, borderLeft: '3px solid', borderColor: 'divider' },
        '& a': { color: 'primary.main', textDecoration: 'underline' },
        '& code': { fontFamily: 'Consolas, monospace', fontSize: '0.95em', bgcolor: '#eef1f5', px: 0.25, borderRadius: 0.5 },
        '& pre': { p: 1, bgcolor: '#eef1f5', borderRadius: 1, overflowX: 'auto', whiteSpace: 'pre' },
        '& pre code': { p: 0 },
        '& th, & td': { border: '1px solid', borderColor: 'divider', px: 0.75, py: 0.5 },
        '& th': { fontWeight: 700, bgcolor: '#eef1f5' },
        '& table': { borderCollapse: 'collapse', width: '100%', minWidth: 360, tableLayout: 'fixed' },
        '& img': { maxWidth: '100%', height: 'auto' },
        '& .contains-task-list': { listStyle: 'none', pl: 0 },
        '& input[type="checkbox"]': { mr: 0.5 },
      }}>
        <Markdown remarkPlugins={[remarkGfm]} remarkRehypeOptions={{ clobberPrefix: `${id}-` }} rehypePlugins={[[linkTicketReferences, { keys }], [highlightMarkdown, { query }]]} components={{
          ...(hideImages ? { img: () => null } : {}),
          a: ({ node: _node, ...props }) => <a {...props} onClick={event => {
            event.stopPropagation();
            const key = props.href?.startsWith('#ticket=') ? props.href.slice(8) : props.href?.startsWith('#') ? props.href.slice(1) : '';
            if (key && keys.has(key)) { event.preventDefault(); open(key); }
          }} aria-describedby={props['aria-describedby'] === 'footnote-label' ? `${id}-footnote-label` : props['aria-describedby']} tabIndex={!full && !expanded && overflows ? -1 : undefined} />,
          h2: ({ node: _node, ...props }) => <h2 {...props} id={props.id === 'footnote-label' ? `${id}-footnote-label` : props.id} />,
          table: ({ node: _node, ...props }) => <Box sx={{ overflowX: 'auto' }}><table {...props} /></Box>,
        }}>{text}</Markdown>
      </Box>
    </Typography>
    {!full && overflows && <Button size="small" aria-expanded={expanded} aria-controls={id}
      onClick={() => setExpanded(value => !value)} sx={{ minWidth: 0, p: 0, mt: 0.25, fontSize: '0.68rem' }}>
      {expanded ? 'Show less' : 'Show more'}
    </Button>}
  </>;
});
