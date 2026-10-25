import React, { useState, useEffect, useMemo, useCallback } from 'react';
import { SimRouter } from '../core/sim/sim-router';
import { apiClient as defaultApiClient } from '../core/api/client';
import { APP_CONFIG } from '../core/constants';
import { toast } from 'sonner';
import { BackendContext, type BackendContextValue } from './backend-context';

export const BackendProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [mode, setModeState] = useState<'live' | 'sim'>('sim');
  const [isConnected, setIsConnected] = useState<boolean>(false);
  const [isChecking, setIsChecking] = useState<boolean>(true);
  const [backendUrl, setBackendUrlState] = useState<string>(APP_CONFIG.defaultRouterUrl);
  const [lastUpdate, setLastUpdate] = useState<number>(0);

  // Pure memoized instances of SimRouter and ApiClient
  const simRouter = useMemo(() => new SimRouter(APP_CONFIG.defaultReplicas), []);
  const apiClient = useMemo(() => defaultApiClient, []);

  const triggerUpdate = useCallback(() => {
    setLastUpdate((prev) => prev + 1);
  }, []);

  const checkConnection = useCallback(async (): Promise<boolean> => {
    setIsChecking(true);
    try {
      const healthy = await apiClient.healthCheck();
      setIsConnected(healthy);
      setIsChecking(false);
      return healthy;
    } catch {
      setIsConnected(false);
      setIsChecking(false);
      return false;
    }
  }, [apiClient]);

  // Initial connection check on app load
  useEffect(() => {
    let active = true;

    async function initialCheck() {
      const healthy = await apiClient.healthCheck();
      if (!active) return;
      setIsConnected(healthy);
      setIsChecking(false);
      if (healthy) {
        setModeState('live');
        toast.success('Connected to Go Distributed Router on :9000', {
          description: 'Live Cluster Mode is active.',
        });
      } else {
        setModeState('sim');
      }
    }

    const timer = setTimeout(() => {
      void initialCheck();
    }, 0);

    return () => {
      active = false;
      clearTimeout(timer);
    };
  }, [apiClient]);

  const setMode = useCallback((newMode: 'live' | 'sim') => {
    setModeState(newMode);
    toast.info(`Switched to ${newMode === 'live' ? 'Live Cluster' : 'Web Simulator'} Mode`, {
      description: newMode === 'live'
        ? `Directing requests to Go backend at ${backendUrl}`
        : 'All hashing, routing, and eviction execute locally in browser.',
    });
  }, [backendUrl]);

  const setBackendUrl = useCallback((url: string) => {
    setBackendUrlState(url);
    apiClient.setBaseUrl(url);
    void checkConnection();
  }, [apiClient, checkConnection]);

  const value = useMemo<BackendContextValue>(() => ({
    mode,
    setMode,
    isConnected,
    isChecking,
    simRouter,
    apiClient,
    backendUrl,
    setBackendUrl,
    checkConnection,
    lastUpdate,
    triggerUpdate,
  }), [mode, setMode, isConnected, isChecking, simRouter, apiClient, backendUrl, setBackendUrl, checkConnection, lastUpdate, triggerUpdate]);

  return (
    <BackendContext.Provider value={value}>
      {children}
    </BackendContext.Provider>
  );
};
