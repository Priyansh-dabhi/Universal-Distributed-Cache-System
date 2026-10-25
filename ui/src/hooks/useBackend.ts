import { useContext } from 'react';
import { BackendContext, type BackendContextValue } from '../context/backend-context';

export function useBackend(): BackendContextValue {
  const ctx = useContext(BackendContext);
  if (!ctx) {
    throw new Error('useBackend must be used within a BackendProvider');
  }
  return ctx;
}
