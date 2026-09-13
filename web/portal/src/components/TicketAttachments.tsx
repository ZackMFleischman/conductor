import { useId, useMemo, useState } from 'react';
import { Alert, Box, Button, ButtonBase, Dialog, DialogContent, DialogTitle, Stack, Typography } from '@mui/material';
import AttachFile from '@mui/icons-material/AttachFile';
import InsertDriveFileOutlined from '@mui/icons-material/InsertDriveFileOutlined';
import type { BoardTicket } from '../board';
import { ticketAttachments, type TicketAttachment } from '../attachments';

function Thumbnail({ attachment }: { attachment: TicketAttachment }) {
  const [failed, setFailed] = useState(false);
  return !failed ? <Box component="img" src={attachment.url} alt={attachment.name} loading="lazy" referrerPolicy="no-referrer" onError={() => setFailed(true)} sx={{ display: 'block', width: '100%', height: '100%', objectFit: 'contain' }} /> : <Typography variant="caption">Preview unavailable</Typography>;
}

function ImageViewer({ attachment, onClose }: { attachment: TicketAttachment; onClose: () => void }) {
  const [actualSize, setActualSize] = useState(false);
  const [failed, setFailed] = useState(false);
  const titleId = useId();
  return <Dialog open fullScreen onClose={onClose} aria-labelledby={titleId} onClick={event => event.stopPropagation()}>
    <DialogTitle id={`${titleId}-bar`} sx={{ py: 1 }}><Stack direction="row" sx={{ alignItems: 'center', gap: 1, flexWrap: 'wrap' }}><Typography id={titleId} component="span" sx={{ flex: 1, overflowWrap: 'anywhere' }}>{attachment.name}</Typography><Button onClick={() => setActualSize(value => !value)}>{actualSize ? 'Fit to screen' : 'Actual size'}</Button><Button component="a" href={attachment.url} target="_blank" rel="noopener noreferrer">Open original</Button><Button onClick={onClose}>Close image</Button></Stack></DialogTitle>
    <DialogContent sx={{ p: 2, bgcolor: '#eff1f4', overflow: 'auto' }}>
      {failed ? <Alert severity="warning">Image unavailable. The source may have moved or been removed.</Alert> : <Box component="img" src={attachment.url} alt={attachment.name} referrerPolicy="no-referrer" onError={() => setFailed(true)} sx={{ display: 'block', margin: 'auto', ...(actualSize ? { maxWidth: 'none', maxHeight: 'none' } : { maxWidth: '100%', maxHeight: 'calc(100dvh - 120px)', objectFit: 'contain' }) }} />}
    </DialogContent>
  </Dialog>;
}

const fileSize = (bytes: number) => bytes < 1024 ? `${bytes} B` : bytes < 1024 * 1024 ? `${(bytes / 1024).toFixed(1)} KB` : `${(bytes / (1024 * 1024)).toFixed(1)} MB`;

export function TicketAttachments({ ticket, compact = false, onOpen }: { ticket: BoardTicket; compact?: boolean; onOpen?: () => void }) {
  const attachments = useMemo(() => ticketAttachments(ticket), [ticket]);
  const [selected, setSelected] = useState<string | null>(null);
  const image = attachments.find(a => a.image);
  const selectedAttachment = attachments.find(a => a.url === selected);
  if (!attachments.length) return null;
  return <Box sx={{ mt: compact ? 0.75 : 0 }}>
    {compact ? <>
      {image && <ButtonBase aria-label={`Preview image: ${image.name}`} onClick={() => setSelected(image.url)} sx={{ height: 104, width: '100%', bgcolor: '#eef1f5', borderRadius: 1, overflow: 'hidden', '&:focus-visible': { outline: '2px solid', outlineColor: 'primary.main' } }}><Thumbnail key={image.url} attachment={image} /></ButtonBase>}
      <Button size="small" startIcon={<AttachFile sx={{ fontSize: 14 }} />} onClick={onOpen} sx={{ p: 0, minWidth: 0, fontSize: '0.7rem' }}>{attachments.length} {attachments.length === 1 ? 'attachment' : 'attachments'}</Button>
    </> : <>
      <Typography component="h3" variant="h3" sx={{ mb: 1 }}>Attachments ({attachments.length})</Typography>
      <Stack spacing={1}>{attachments.map(a => <Box key={a.url} sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
        {a.image ? <ButtonBase aria-label={`Preview image: ${a.name}`} onClick={() => setSelected(a.url)} sx={{ width: 96, height: 64, flexShrink: 0, bgcolor: '#eef1f5', borderRadius: 1 }}><Thumbnail attachment={a} /></ButtonBase> : <InsertDriveFileOutlined sx={{ width: 32, mx: 4, color: 'text.secondary' }} />}
        <Box sx={{ minWidth: 0 }}><Button component="a" href={a.url} target="_blank" rel="noopener noreferrer" onClick={event => { if (a.image) { event.preventDefault(); setSelected(a.url); } }} sx={{ p: 0, minWidth: 0, textAlign: 'left', overflowWrap: 'anywhere' }}>{a.name}</Button><Typography variant="caption" component="div" color="text.secondary">{a.size === undefined ? 'Linked file' : fileSize(a.size)}</Typography></Box>
      </Box>)}</Stack>
    </>}
    {selectedAttachment && <ImageViewer key={selectedAttachment.url} attachment={selectedAttachment} onClose={() => setSelected(null)} />}
  </Box>;
}
