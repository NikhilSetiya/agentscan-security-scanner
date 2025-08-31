import { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import { apiClient, ApiResponse, ApiError, Pagination } from '../services/api';

// Cache management
interface CacheEntry<T> {
  data: T;
  timestamp: number;
  ttl: number;
}

class ApiCache {
  private cache = new Map<string, CacheEntry<any>>();
  private subscribers = new Map<string, Set<() => void>>();

  set<T>(key: string, data: T, ttl: number = 300000): void { // 5 minutes default TTL
    this.cache.set(key, {
      data,
      timestamp: Date.now(),
      ttl,
    });
    this.notifySubscribers(key);
  }

  get<T>(key: string): T | null {
    const entry = this.cache.get(key);
    if (!entry) return null;

    const now = Date.now();
    if (now - entry.timestamp > entry.ttl) {
      this.cache.delete(key);
      this.notifySubscribers(key);
      return null;
    }

    return entry.data;
  }

  invalidate(key: string): void {
    this.cache.delete(key);
    this.notifySubscribers(key);
  }

  invalidatePattern(pattern: string): void {
    const regex = new RegExp(pattern);
    for (const key of this.cache.keys()) {
      if (regex.test(key)) {
        this.cache.delete(key);
        this.notifySubscribers(key);
      }
    }
  }

  subscribe(key: string, callback: () => void): () => void {
    if (!this.subscribers.has(key)) {
      this.subscribers.set(key, new Set());
    }
    this.subscribers.get(key)!.add(callback);

    return () => {
      const keySubscribers = this.subscribers.get(key);
      if (keySubscribers) {
        keySubscribers.delete(callback);
        if (keySubscribers.size === 0) {
          this.subscribers.delete(key);
        }
      }
    };
  }

  private notifySubscribers(key: string): void {
    const keySubscribers = this.subscribers.get(key);
    if (keySubscribers) {
      keySubscribers.forEach(callback => callback());
    }
  }

  clear(): void {
    this.cache.clear();
    this.subscribers.forEach(subscribers => {
      subscribers.forEach(callback => callback());
    });
  }

  getStats() {
    return {
      size: this.cache.size,
      keys: Array.from(this.cache.keys()),
    };
  }
}

// Global cache instance
const apiCache = new ApiCache();

// Generic API hook state
interface ApiState<T> {
  data: T | null;
  loading: boolean;
  error: ApiError | null;
}

// Generic API hook options
interface UseApiOptions<T = any> {
  immediate?: boolean;
  onSuccess?: (data: T) => void;
  onError?: (error: ApiError) => void;
  cacheKey?: string;
  cacheTTL?: number;
  staleWhileRevalidate?: boolean;
  retryOnError?: boolean;
  retryCount?: number;
  retryDelay?: number;
  dependencies?: any[];
}

// Loading states
type LoadingState = 'idle' | 'loading' | 'revalidating' | 'error' | 'success';

// Enhanced API state
interface EnhancedApiState<T> {
  data: T | null;
  loading: boolean;
  loadingState: LoadingState;
  error: ApiError | null;
  isStale: boolean;
  lastFetch: number | null;
  retryCount: number;
}

// Enhanced API hook with caching and advanced features
export function useApi<T>(
  apiCall: () => Promise<ApiResponse<T>>,
  options: UseApiOptions<T> = {}
) {
  const {
    immediate = true,
    onSuccess,
    onError,
    cacheKey,
    cacheTTL = 300000, // 5 minutes
    staleWhileRevalidate = false,
    retryOnError = false,
    retryCount = 3,
    retryDelay = 1000,
    dependencies = []
  } = options;

  const [state, setState] = useState<EnhancedApiState<T>>({
    data: null,
    loading: false,
    loadingState: 'idle',
    error: null,
    isStale: false,
    lastFetch: null,
    retryCount: 0,
  });

  const mountedRef = useRef(true);
  const apiCallRef = useRef(apiCall);
  const retryTimeoutRef = useRef<NodeJS.Timeout>();

  // Update the ref when apiCall changes
  useEffect(() => {
    apiCallRef.current = apiCall;
  });

  useEffect(() => {
    return () => {
      mountedRef.current = false;
      if (retryTimeoutRef.current) {
        clearTimeout(retryTimeoutRef.current);
      }
    };
  }, []);

  // Cache subscription
  useEffect(() => {
    if (!cacheKey) return;

    const unsubscribe = apiCache.subscribe(cacheKey, () => {
      const cachedData = apiCache.get<T>(cacheKey);
      if (cachedData && mountedRef.current) {
        setState(prev => ({
          ...prev,
          data: cachedData,
          isStale: false,
        }));
      }
    });

    return unsubscribe;
  }, [cacheKey]);

  const executeWithRetry = useCallback(async (isRetry = false, currentRetryCount = 0): Promise<void> => {
    if (!mountedRef.current) return;

    // Check cache first
    if (cacheKey && !isRetry) {
      const cachedData = apiCache.get<T>(cacheKey);
      if (cachedData) {
        setState(prev => ({
          ...prev,
          data: cachedData,
          loading: false,
          loadingState: 'success',
          error: null,
          isStale: false,
          lastFetch: Date.now(),
        }));

        if (staleWhileRevalidate) {
          // Continue with background revalidation
          setState(prev => ({ ...prev, loadingState: 'revalidating' }));
        } else {
          return;
        }
      }
    }

    if (!staleWhileRevalidate || !state.data) {
      setState(prev => ({
        ...prev,
        loading: true,
        loadingState: 'loading',
        error: null,
        retryCount: currentRetryCount,
      }));
    }

    try {
      const response = await apiCallRef.current();

      if (!mountedRef.current) return;

      if (response.error) {
        if (retryOnError && currentRetryCount < retryCount) {
          // Retry after delay
          retryTimeoutRef.current = setTimeout(() => {
            executeWithRetry(true, currentRetryCount + 1);
          }, retryDelay * Math.pow(2, currentRetryCount)); // Exponential backoff
          return;
        }

        setState(prev => ({
          ...prev,
          loading: false,
          loadingState: 'error',
          error: response.error!,
          retryCount: currentRetryCount,
        }));
        onError?.(response.error);
      } else {
        // Cache the successful response
        if (cacheKey && response.data) {
          apiCache.set(cacheKey, response.data, cacheTTL);
        }

        setState(prev => ({
          ...prev,
          loading: false,
          loadingState: 'success',
          data: response.data!,
          error: null,
          isStale: false,
          lastFetch: Date.now(),
          retryCount: 0,
        }));
        onSuccess?.(response.data);
      }
    } catch (error) {
      if (!mountedRef.current) return;

      const apiError: ApiError = {
        code: 'UNKNOWN_ERROR',
        message: error instanceof Error ? error.message : 'Unknown error',
      };

      if (retryOnError && currentRetryCount < retryCount) {
        // Retry after delay
        retryTimeoutRef.current = setTimeout(() => {
          executeWithRetry(true, currentRetryCount + 1);
        }, retryDelay * Math.pow(2, currentRetryCount));
        return;
      }

      setState(prev => ({
        ...prev,
        loading: false,
        loadingState: 'error',
        error: apiError,
        retryCount: currentRetryCount,
      }));
      onError?.(apiError);
    }
  }, [cacheKey, cacheTTL, staleWhileRevalidate, retryOnError, retryCount, retryDelay, onSuccess, onError]);

  const execute = useCallback(() => {
    executeWithRetry(false, 0);
  }, [executeWithRetry]);

  const reset = useCallback(() => {
    setState({
      data: null,
      loading: false,
      loadingState: 'idle',
      error: null,
      isStale: false,
      lastFetch: null,
      retryCount: 0,
    });
    if (cacheKey) {
      apiCache.invalidate(cacheKey);
    }
  }, [cacheKey]);

  const invalidate = useCallback(() => {
    if (cacheKey) {
      apiCache.invalidate(cacheKey);
    }
    setState(prev => ({ ...prev, isStale: true }));
  }, [cacheKey]);

  const refetch = useCallback(() => {
    execute();
  }, [execute]);

  // Execute on mount or dependency changes
  useEffect(() => {
    if (immediate) {
      execute();
    }
  }, [immediate, ...dependencies]);

  return {
    ...state,
    execute,
    refetch,
    reset,
    invalidate,
    isLoading: state.loading,
    isRevalidating: state.loadingState === 'revalidating',
    isError: state.loadingState === 'error',
    isSuccess: state.loadingState === 'success',
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

// Specialized hooks for different data types

// Dashboard data hook with caching
export function useDashboardStats(options: Omit<UseApiOptions, 'cacheKey'> = {}) {
  return useApi(() => apiClient.getDashboardStats(), {
    cacheKey: 'dashboard-stats',
    cacheTTL: 60000, // 1 minute cache for dashboard
    staleWhileRevalidate: true,
    ...options,
  });
}

// User data hooks
export function useCurrentUser(options: Omit<UseApiOptions, 'cacheKey'> = {}) {
  return useApi(() => apiClient.getCurrentUser(), {
    cacheKey: 'current-user',
    cacheTTL: 300000, // 5 minutes
    staleWhileRevalidate: true,
    ...options,
  });
}

// Repository hooks with enhanced caching
export function useRepository(id: string | undefined, options: Omit<UseApiOptions, 'cacheKey'> = {}) {
  return useApi(
    () => {
      if (!id) throw new Error('Repository ID is required');
      return apiClient.getRepository(id);
    },
    {
      immediate: !!id,
      cacheKey: id ? `repository-${id}` : undefined,
      cacheTTL: 300000, // 5 minutes
      dependencies: [id],
      ...options,
    }
  );
}

// Scan job hooks with real-time updates
export function useScanJob(id: string | undefined, options: Omit<UseApiOptions, 'cacheKey'> = {}) {
  return useApi(
    () => {
      if (!id) throw new Error('Scan job ID is required');
      return apiClient.getScanJob(id);
    },
    {
      immediate: !!id,
      cacheKey: id ? `scan-job-${id}` : undefined,
      cacheTTL: 30000, // 30 seconds for active scans
      staleWhileRevalidate: true,
      dependencies: [id],
      ...options,
    }
  );
}

export function useScanJobWithDetails(id: string | undefined, options: Omit<UseApiOptions, 'cacheKey'> = {}) {
  return useApi(
    () => {
      if (!id) throw new Error('Scan job ID is required');
      return apiClient.getScanJobWithDetails(id);
    },
    {
      immediate: !!id,
      cacheKey: id ? `scan-job-details-${id}` : undefined,
      cacheTTL: 60000, // 1 minute for detailed results
      dependencies: [id],
      ...options,
    }
  );
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

// Enhanced list hooks with caching and filtering
export function useRepositories(
  params: { search?: string; page?: number; limit?: number; organization_id?: string } = {},
  options: Omit<UseApiOptions, 'cacheKey'> = {}
) {
  const cacheKey = useMemo(() => {
    const searchParams = new URLSearchParams();
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined) {
        searchParams.set(key, String(value));
      }
    });
    return `repositories-${searchParams.toString()}`;
  }, [params]);

  return usePaginatedApi(() => apiClient.getRepositories(params), {
    cacheKey,
    cacheTTL: 120000, // 2 minutes
    staleWhileRevalidate: true,
    dependencies: [JSON.stringify(params)],
    ...options,
  });
}

export function useScanJobs(
  params: { repository_id?: string; status?: string; page?: number; limit?: number; user_id?: string } = {},
  options: Omit<UseApiOptions, 'cacheKey'> = {}
) {
  const cacheKey = useMemo(() => {
    const searchParams = new URLSearchParams();
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined) {
        searchParams.set(key, String(value));
      }
    });
    return `scan-jobs-${searchParams.toString()}`;
  }, [params]);

  return usePaginatedApi(() => apiClient.getScanJobs(params), {
    cacheKey,
    cacheTTL: 30000, // 30 seconds for active job lists
    staleWhileRevalidate: true,
    dependencies: [JSON.stringify(params)],
    ...options,
  });
}

// Legacy compatibility
export function useScans(params?: { repository_id?: string; status?: string; page?: number; limit?: number }) {
  return useScanJobs(params);
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

// Enhanced mutations with cache invalidation and optimistic updates
export function useCreateScanJob(options: UseApiOptions = {}) {
  return useMutation(apiClient.createScanJob.bind(apiClient), {
    onSuccess: (data) => {
      // Invalidate scan job lists
      apiCache.invalidatePattern('scan-jobs-.*');
      // Invalidate dashboard stats
      apiCache.invalidate('dashboard-stats');
      options.onSuccess?.(data);
    },
    ...options,
  });
}

export function useCreateRepository(options: UseApiOptions = {}) {
  return useMutation(apiClient.createRepository.bind(apiClient), {
    onSuccess: (data) => {
      // Invalidate repository lists
      apiCache.invalidatePattern('repositories-.*');
      // Invalidate dashboard stats
      apiCache.invalidate('dashboard-stats');
      options.onSuccess?.(data);
    },
    ...options,
  });
}

export function useUpdateRepository(options: UseApiOptions = {}) {
  return useMutation(
    ({ id, updates }: { id: string; updates: any }) => apiClient.updateRepository(id, updates),
    {
      onSuccess: (data, variables) => {
        // Invalidate specific repository cache
        apiCache.invalidate(`repository-${variables.id}`);
        // Invalidate repository lists
        apiCache.invalidatePattern('repositories-.*');
        options.onSuccess?.(data);
      },
      ...options,
    }
  );
}

export function useDeleteRepository(options: UseApiOptions = {}) {
  return useMutation(apiClient.deleteRepository.bind(apiClient), {
    onSuccess: (data, repositoryId) => {
      // Invalidate specific repository cache
      apiCache.invalidate(`repository-${repositoryId}`);
      // Invalidate repository lists
      apiCache.invalidatePattern('repositories-.*');
      // Invalidate related scan jobs
      apiCache.invalidatePattern('scan-jobs-.*');
      options.onSuccess?.(data);
    },
    ...options,
  });
}

// Scan job control mutations
export function useStartScanJob(options: UseApiOptions = {}) {
  return useMutation(apiClient.startScanJob.bind(apiClient), {
    onSuccess: (data, scanJobId) => {
      // Invalidate specific scan job cache
      apiCache.invalidate(`scan-job-${scanJobId}`);
      apiCache.invalidate(`scan-job-details-${scanJobId}`);
      // Invalidate scan job lists
      apiCache.invalidatePattern('scan-jobs-.*');
      options.onSuccess?.(data);
    },
    ...options,
  });
}

export function useCancelScanJob(options: UseApiOptions = {}) {
  return useMutation(apiClient.cancelScanJob.bind(apiClient), {
    onSuccess: (data, scanJobId) => {
      // Invalidate specific scan job cache
      apiCache.invalidate(`scan-job-${scanJobId}`);
      apiCache.invalidate(`scan-job-details-${scanJobId}`);
      // Invalidate scan job lists
      apiCache.invalidatePattern('scan-jobs-.*');
      options.onSuccess?.(data);
    },
    ...options,
  });
}

export function useRetryScanJob(options: UseApiOptions = {}) {
  return useMutation(apiClient.retryScanJob.bind(apiClient), {
    onSuccess: (data, scanJobId) => {
      // Invalidate specific scan job cache
      apiCache.invalidate(`scan-job-${scanJobId}`);
      apiCache.invalidate(`scan-job-details-${scanJobId}`);
      // Invalidate scan job lists
      apiCache.invalidatePattern('scan-jobs-.*');
      options.onSuccess?.(data);
    },
    ...options,
  });
}

// Legacy compatibility
export function useSubmitScan(options: UseApiOptions = {}) {
  return useCreateScanJob(options);
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

// Cache management utilities
export const cacheUtils = {
  // Invalidate specific cache key
  invalidate: (key: string) => apiCache.invalidate(key),
  
  // Invalidate cache keys matching pattern
  invalidatePattern: (pattern: string) => apiCache.invalidatePattern(pattern),
  
  // Clear all cache
  clear: () => apiCache.clear(),
  
  // Get cache stats
  getStats: () => apiCache.getStats(),
  
  // Manually set cache data
  set: <T>(key: string, data: T, ttl?: number) => apiCache.set(key, data, ttl),
  
  // Get cached data
  get: <T>(key: string) => apiCache.get<T>(key),
  
  // Invalidate all repository-related cache
  invalidateRepositories: () => {
    apiCache.invalidatePattern('repositories-.*');
    apiCache.invalidatePattern('repository-.*');
  },
  
  // Invalidate all scan job-related cache
  invalidateScanJobs: () => {
    apiCache.invalidatePattern('scan-jobs-.*');
    apiCache.invalidatePattern('scan-job-.*');
  },
  
  // Invalidate dashboard cache
  invalidateDashboard: () => {
    apiCache.invalidate('dashboard-stats');
  },
  
  // Refresh all active data
  refreshAll: () => {
    apiCache.clear();
  },
};

// Connection state hook
export function useConnectionState() {
  const [connectionState, setConnectionState] = useState(apiClient.getConnectionState());

  useEffect(() => {
    const unsubscribe = apiClient.onConnectionStateChange(setConnectionState);
    return unsubscribe;
  }, []);

  return {
    connectionState,
    isConnected: connectionState === 'connected',
    isDisconnected: connectionState === 'disconnected',
    isReconnecting: connectionState === 'reconnecting',
  };
}

// Optimistic update hook
export function useOptimisticUpdate<T>() {
  const [optimisticData, setOptimisticData] = useState<T | null>(null);
  const [isOptimistic, setIsOptimistic] = useState(false);

  const setOptimistic = useCallback((data: T) => {
    setOptimisticData(data);
    setIsOptimistic(true);
  }, []);

  const clearOptimistic = useCallback(() => {
    setOptimisticData(null);
    setIsOptimistic(false);
  }, []);

  const revert = useCallback(() => {
    clearOptimistic();
  }, [clearOptimistic]);

  return {
    optimisticData,
    isOptimistic,
    setOptimistic,
    clearOptimistic,
    revert,
  };
}