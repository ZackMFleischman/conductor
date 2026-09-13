import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { App } from './App';
import { fixtureBoardSource } from './data/fixtureBoardSource';
import { LivePortal } from './LivePortal';
const fixtureMode = new URLSearchParams(window.location.search).get('mode') === 'fixture';
createRoot(document.getElementById('root')!).render(<StrictMode>{fixtureMode ? <App source={fixtureBoardSource} /> : <LivePortal />}</StrictMode>);
