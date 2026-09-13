import { createTheme } from '@mui/material/styles';
export const theme = createTheme({
  palette: { primary: { main: '#22675b' }, background: { default: '#f7f8fa', paper: '#ffffff' }, text: { primary: '#1f2937', secondary: '#5d6878' }, divider: '#e1e5eb' },
  typography: { fontFamily: '"Segoe UI", Inter, Arial, sans-serif', h1: { fontSize: '1.5rem', fontWeight: 700, letterSpacing: '-0.045em' }, h2: { fontSize: '0.8rem', fontWeight: 650 }, h3: { fontSize: '0.82rem', fontWeight: 650, lineHeight: 1.35 }, body2: { fontSize: '0.75rem', lineHeight: 1.4 }, caption: { fontSize: '0.68rem', lineHeight: 1.4 }, button: { textTransform: 'none', fontWeight: 600 } },
  shape: { borderRadius: 10 },
  components: {
    MuiButton: { defaultProps: { disableElevation: true } },
    MuiChip: { styleOverrides: { root: { fontWeight: 600 } } },
    MuiCssBaseline: { styleOverrides: { body: { minWidth: 320 }, '*': { boxSizing: 'border-box' }, mark: { backgroundColor: '#ffe496', color: '#29220c', borderRadius: 2, boxDecorationBreak: 'clone' }, '::selection': { background: '#c9e7df' } } },
  },
});
