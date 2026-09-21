import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { App } from './App';
import './styles.css';

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App mockMode={import.meta.env.VITE_MOCK_MODE === 'true'} />
  </StrictMode>,
);
