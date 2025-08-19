import { useState, useEffect, useCallback, useRef } from 'react';
import { apiClient, ApiResponse, ApiError, Pagination } from '../services/api';

// Generic API hook state
interface ApiState<T> {
  data: T | null;
  loading: boolean;
  error: ApiError | null;
}

// Generic API hook options
interface UseApiOptions {
  immediate?: boolean;
  onSuccess?: (data: any) => void;
  onError?: (error: ApiError) => void;
}

// Generic API hook
export function useApi<T>(
  apiCall: () => Promise<ApiResponse<T>>,
  options: UseApiOptions = {}
) {
  const { immediate = true, onSuccess, onError } = options;
  const [state, setState] = useState<ApiState<T>>({
    data: null,
    loading: false,
    error: null,
  });

  const mountedRef = useRef(true);
  const apiCallRef = useRef(apiCall);

  // Update the ref when apiCall changes
  useEffect(() => {
    apiCallRef.current = apiCall;
  });

  useEffect(() => {
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const execute = useCallback(async () => {
    setState(prev => ({ ...prev, loading: true, error: null }));

    try {
      const response = await apiCallRef.current();

      if (!mountedRef.current) return;

      if (response.error) {
        setState(prev => ({ ...prev, loading: false, error: response.error! }));
        onError?.(response.error);
      } else {
        setState(prev => ({ ...prev, loading: false, data: response.data! }));
        onSuccess?.(response.data);
      }
    } catch (error) {
      if (!mountedRef.current) return;

      const apiError: ApiError = {
        code: 'UNKNOWN_ERROR',
        message: error instanceof Error ? error.message : 'Unknown error',
      };
      setState(prev => ({ ...prev, loading: false, error: apiError }));
      onError?.(apiError);
    }
  }, [onSuccess, onError]); // Stable dependencies

  const reset = useCallback(() => {
    setState({
      data: null,
      loading: false,
      error: null,
    });
  }, []);

  useEffect(() => {
    if (immediate) {
      execute();
    }
  }, [immediate]); // Removed execute from dependencies since it's stable now

  return {
    ...state,
    execute,
    reset,
  };
}

// Mutation hook for POST/PUT/DELETE operations
export function useMutation<TData, TVariables = void>(
  mutationFn: (variables: TVariables) => Promise<ApiResponse<TData>>,
  options: UseApiOptions = {}
) {
  const { onSuccess, onError } = options;
  const [state, setState] = useState<ApiState<TData>>({
    data: null,
    loading: false,
    error: null,
  });

  const mountedRef = useRef(true);
  const mutationFnRef = useRef(mutationFn);

  // Update the ref when mutationFn changes
  useEffect(() => {
    mutationFnRef.current = mutationFn;
  });

  useEffect(() => {
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const mutate = useCallback(async (variables: TVariables) => {
    setState(prev => ({ ...prev, loading: true, error: null }));

    try {
      const response = await mutationFnRef.current(variables);

      if (!mountedRef.current) return;

      if (response.error) {
        setState(prev => ({ ...prev, loading: false, error: response.error! }));
        onError?.(response.error);
        return { success: false, error: response.error };
      } else {
        setState(prev => ({ ...prev, loading: false, data: response.data! }));
        onSuccess?.(response.data);
        return { success: true, data: response.data };
      }
    } catch (error) {
      if (!mountedRef.current) return;

      const apiError: ApiError = {
        code: 'UNKNOWN_ERROR',
        message: error instanceof Error ? error.message : 'Unknown error',
      };
      setState(prev => ({ ...prev, loading: false, error: apiError }));
      onError?.(apiError);
      return { success: false, error: apiError };
    }
  }, [onSuccess, onError]); // Stable dependencies

  const reset = useCallback(() => {
    setState({
      data: null,
      loading: false,
      error: null,
    });
  }, []);

  return {
    ...state,
    mutate,
    reset,
  };
}

// Specific hooks for common API operations

// Dashboard data hook
export function useDashboardStats() {
  return useApi(() => apiClient.getDashboardStats());
}

// Enhanced API state with pagination
interface PaginatedApiState<T> extends ApiState<T> {
  pagination?: Pagination;
}

// Paginated API hook
export function usePaginatedApi<T>(
  apiCall: () => Promise<ApiResponse<T>>,
  options: UseApiOptions = {}
) {
  const { immediate = true, onSuccess, onError } = options;
  const [state, setState] = useState<PaginatedApiState<T>>({
    data: null,
    loading: false,
    error: null,
    pagination: undefined,
  });

  const mountedRef = useRef(true);
  const apiCallRef = useRef(apiCall);

  // Update the ref when apiCall changes
  useEffect(() => {
    apiCallRef.current = apiCall;
  });

  useEffect(() => {
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const execute = useCallback(async () => {
    setState(prev => ({ ...prev, loading: true, error: null }));

    try {
      const response = await apiCallRef.current();

      if (!mountedRef.current) return;

      if (response.error) {
        setState(prev => ({ ...prev, loading: false, error: response.error! }));
        onError?.(response.error);
      } else {
        setState(prev => ({ 
          ...prev, 
          loading: false, 
          data: response.data!, 
          pagination: response.meta?.pagination 
        }));
        onSuccess?.(response.data);
      }
    } catch (error) {
      if (!mountedRef.current) return;

      const apiError: ApiError = {
        code: 'UNKNOWN_ERROR',
        message: error instanceof Error ? error.message : 'Unknown error',
      };
      setState(prev => ({ ...prev, loading: false, error: apiError }));
      onError?.(apiError);
    }
  }, [onSuccess, onError]);

  const reset = useCallback(() => {
    setState({
      data: null,
      loading: false,
      error: null,
      pagination: undefined,
    });
  }, []);

  useEffect(() => {
    if (immediate) {
      execute();
    }
  }, [immediate]);

  return {
    ...state,
    execute,
    reset,
  };
}

// Repositories hook
export function useRepositories(params?: { search?: string; page?: number; limit?: number }) {
  return usePaginatedApi(() => apiClient.getRepositories(params || {}), {
    immediate: true,
  });
}

// Scans hook
export function useScans(params?: { repository_id?: string; status?: string; page?: number; limit?: number }) {
  return usePaginatedApi(() => apiClient.getScans(params || {}), {
    immediate: true,
  });
}

// Single scan hook
export function useScan(scanId: string | undefined) {
  return useApi(
    () => {
      if (!scanId) throw new Error('Scan ID is required');
      return apiClient.getScan(scanId);
    },
    { immediate: !!scanId }
  );
}

// Scan results hook
export function useScanResults(scanId: string | undefined) {
  return useApi(
    () => {
      if (!scanId) throw new Error('Scan ID is required');
      return apiClient.getScanResults(scanId);
    },
    { immediate: !!scanId }
  );
}

// Submit scan mutation
export function useSubmitScan() {
  return useMutation(apiClient.submitScan.bind(apiClient));
}

// Create repository mutation
export function useCreateRepository() {
  return useMutation(apiClient.createRepository.bind(apiClient));
}

// Polling hook for real-time updates
export function usePolling<T>(
  apiCall: () => Promise<ApiResponse<T>>,
  interval: number = 5000,
  enabled: boolean = true
) {
  const { data, loading, error, execute } = useApi(apiCall, { immediate: enabled });
  const intervalRef = useRef<number>();

  useEffect(() => {
    if (enabled && !loading) {
      intervalRef.current = setInterval(() => {
        execute();
      }, interval) as unknown as number;
    }

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, [enabled, loading, interval]); // Removed execute from dependencies since it's now stable

  return { data, loading, error, execute };
}

// WebSocket hook for real-time updates
export function useWebSocket(url: string, enabled: boolean = true) {
  const [socket, setSocket] = useState<WebSocket | null>(null);
  const [connectionState, setConnectionState] = useState<'connecting' | 'connected' | 'disconnected'>('disconnected');
  const [lastMessage, setLastMessage] = useState<any>(null);

  useEffect(() => {
    if (!enabled) return;

    const ws = new WebSocket(url);
    setSocket(ws);
    setConnectionState('connecting');

    ws.onopen = () => {
      setConnectionState('connected');
    };

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        setLastMessage(data);
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error);
      }
    };

    ws.onclose = () => {
      setConnectionState('disconnected');
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
      setConnectionState('disconnected');
    };

    return () => {
      ws.close();
    };
  }, [url, enabled]);

  const sendMessage = useCallback((message: any) => {
    if (socket && socket.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify(message));
    }
  }, [socket]);

  return {
    socket,
    connectionState,
    lastMessage,
    sendMessage,
    isConnected: connectionState === 'connected',
  };
}