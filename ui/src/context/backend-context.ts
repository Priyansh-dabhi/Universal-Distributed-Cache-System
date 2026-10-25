import { createContext } from 'react';
import type { SimRouter } from '../core/sim/sim-router';
import type { ApiClient } from '../core/api/client';

export interface BackendContextValue {
  mode: 'live' | 'sim';
  setMode: (mode: 'live' | 'sim') => void;
  isConnected: boolean;
  isChecking: boolean;
  simRouter: SimRouter;
  apiClient: ApiClient;
  backendUrl: string;
  setBackendUrl: (url: string) => void;
  checkConnection: () => Promise<boolean>;
  lastUpdate: number;
  triggerUpdate: () => void;
}

export const BackendContext = createContext<BackendContextValue | null>(null);
