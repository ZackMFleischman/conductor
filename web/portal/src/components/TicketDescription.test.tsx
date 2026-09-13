import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, expect, it, vi } from 'vitest';
import { TicketDescription } from './TicketDescription';

afterEach(() => vi.restoreAllMocks());

const richText = `## Plan

Use **bold** and *emphasis* with [the docs](https://example.com/docs).

- First item
- Second item

- [x] Finished
- [ ] Remaining

> A quotation

\`inline code\`

\`\`\`ts
const code = 1;
\`\`\`

| Name | Status |
| --- | --- |
| Code | ~~Old~~ |
`;

it('renders Markdown and GFM as semantic formatted content', () => {
  const { container } = render(<TicketDescription text={richText} query="" />);
  expect(screen.getByRole('heading', { name: 'Plan' })).toBeInTheDocument();
  expect(container.querySelector('strong')).toHaveTextContent('bold');
  expect(container.querySelector('em')).toHaveTextContent('emphasis');
  expect(screen.getByRole('link', { name: 'the docs' })).toHaveAttribute('href', 'https://example.com/docs');
  expect(screen.getAllByRole('listitem')).toHaveLength(4);
  expect(screen.getAllByRole('checkbox')[0]).toBeChecked();
  expect(screen.getAllByRole('checkbox')[1]).toBeDisabled();
  expect(container.querySelector('blockquote')).toHaveTextContent('A quotation');
  expect(container.querySelector('pre code')).toHaveTextContent('const code = 1;');
  expect(screen.getByRole('table')).toHaveTextContent('NameStatusCodeOld');
  expect(container.querySelector('del')).toHaveTextContent('Old');
});

it('highlights rendered text in formatting, links, code and tables without changing link targets', () => {
  const { container } = render(<TicketDescription text={'**code** [code](https://example.com/code) `code`\n\n| code |\n| --- |\n| CODE |'} query="code" />);
  expect([...container.querySelectorAll('mark')].map(node => node.textContent)).toEqual(['code', 'code', 'code', 'code', 'CODE']);
  expect(screen.getByRole('link')).toHaveAttribute('href', 'https://example.com/code');
  expect(container.querySelector('strong mark')).toHaveTextContent('code');
});

it('keeps raw HTML inert and rejects executable Markdown URLs', () => {
  const { container } = render(<TicketDescription text={'<img src=x onerror=alert(1)>\n\n<script>alert(1)</script>\n\n[unsafe](javascript:alert%281%29)\n\n[safe](https://example.com)'} query="" />);
  expect(container.querySelector('img, script')).toBeNull();
  expect(screen.getByText('unsafe').closest('a')).not.toHaveAttribute('href', expect.stringMatching(/^javascript:/));
  expect(screen.getByRole('link', { name: 'safe' })).toHaveAttribute('href', 'https://example.com');
});

it('highlights a visible phrase spanning inline Markdown nodes', () => {
  const { container } = render(<TicketDescription text={'Use **bold** [text](https://example.com)'} query="Use bold text" />);
  expect([...container.querySelectorAll('mark')].map(node => node.textContent).join('')).toBe('Use bold text');
  expect(container.querySelector('strong mark')).toHaveTextContent('bold');
});

it('gives footnotes distinct labels and targets across descriptions', () => {
  const text = 'A note[^1]\n\n[^1]: Detail';
  const { container } = render(<><TicketDescription text={text} query="" /><TicketDescription text={text} query="" /></>);
  const ids = [...container.querySelectorAll('[id]')].map(element => element.id);
  expect(new Set(ids).size).toBe(ids.length);
  for (const reference of container.querySelectorAll('a[data-footnote-ref]')) {
    expect(document.getElementById(reference.getAttribute('aria-describedby')!)).toHaveTextContent('Footnotes');
    expect(document.getElementById(reference.getAttribute('href')!.slice(1))).toHaveTextContent('Detail');
  }
});

it('expands and collapses rich content with an accessible control', async () => {
  vi.spyOn(HTMLElement.prototype, 'scrollHeight', 'get').mockReturnValue(400);
  const user = userEvent.setup();
  render(<TicketDescription text={richText} query="" />);
  const button = screen.getByRole('button', { name: 'Show more' });
  expect(button).toHaveAttribute('aria-expanded', 'false');
  expect(document.getElementById(button.getAttribute('aria-controls')!)).toContainElement(screen.getByRole('table'));
  await user.click(button);
  expect(screen.getByRole('button', { name: 'Show less' })).toHaveAttribute('aria-expanded', 'true');
  await user.click(screen.getByRole('button', { name: 'Show less' }));
  expect(button).toHaveAttribute('aria-expanded', 'false');
});
