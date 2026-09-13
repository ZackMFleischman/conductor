import { useId, useLayoutEffect, useRef, useState } from 'react';
import { Button, Typography } from '@mui/material';
import { HighlightedText } from './HighlightedText';

export function TicketDescription({ text, query }: { text: string; query: string }) {
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
    <Typography component="div" ref={element} id={id} variant="body2" color="text.secondary" sx={{
      whiteSpace: 'pre-wrap', display: '-webkit-box', WebkitBoxOrient: 'vertical',
      WebkitLineClamp: expanded ? 'unset' : 4, overflow: expanded ? 'visible' : 'hidden',
    }}><HighlightedText text={text} query={query} /></Typography>
    {overflows && <Button size="small" aria-expanded={expanded} aria-controls={id}
      onClick={() => setExpanded(value => !value)} sx={{ minWidth: 0, p: 0, mt: 0.25, fontSize: '0.68rem' }}>
      {expanded ? 'Show less' : 'Show more'}
    </Button>}
  </>;
}
