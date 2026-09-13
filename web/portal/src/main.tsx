import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { App } from './App';
import { fixtureBoardSource } from './data/fixtureBoardSource';
createRoot(document.getElementById('root')!).render(<StrictMode><App source={fixtureBoardSource} /></StrictMode>);
