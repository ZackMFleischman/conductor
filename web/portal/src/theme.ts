import { createTheme } from '@mui/material/styles';
export const theme = createTheme({
  palette: { primary: { main: '#22675b' }, background: { default: '#f7f8fa', paper: '#ffffff' }, text: { primary: '#1f2937', secondary: '#5d6878' }, divider: '#e1e5eb' },
  typography: { fontFamily: '"Segoe UI", Inter, Arial, sans-serif', h1: { fontSize: '1.5rem', fontWeight: 700, letterSpacing: '-0.045em' }, h2: { fontSize: '0.9rem', fontWeight: 650 }, h3: { fontSize: '0.95rem', fontWeight: 650, lineHeight: 1.45 }, body2: { fontSize: '0.82rem', lineHeight: 1.5 }, button: { textTransform: 'none', fontWeight: 600 } },
  shape: { borderRadius: 10 },
  components: {
    MuiButton: { defaultProps: { disableElevation: true } },
    MuiChip: { styleOverrides: { root: { fontWeight: 600 } } },
    MuiCssBaseline: { styleOverrides: { body: { minWidth: 320 }, '*': { boxSizing: 'border-box' }, '::selection': { background: '#c9e7df' } } },
  },
});
