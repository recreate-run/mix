# TanStack Pacer React Guide

A comprehensive guide to using TanStack Pacer in React applications for managing debouncing, throttling, rate limiting, queuing, and batching.

## Table of Contents

- [Installation](#installation)
- [Core Concepts](#core-concepts)
- [Debouncing](#debouncing)
- [Throttling](#throttling)
- [Rate Limiting](#rate-limiting)
- [Queuing](#queuing)
- [Batching](#batching)
- [Async Operations](#async-operations)
- [Hook Architecture](#hook-architecture)
- [Best Practices](#best-practices)
- [Common Patterns](#common-patterns)

## Installation

```bash
npm install @tanstack/react-pacer
# or
yarn add @tanstack/react-pacer
```

The React adapter re-exports all core Pacer utilities, so you don't need to install the core package separately.

## Core Concepts

TanStack Pacer provides execution control techniques to manage when and how functions execute:

- **Debouncing**: Wait for a quiet period before executing
- **Throttling**: Execute at most once per time window with consistent spacing
- **Rate Limiting**: Hard limits within time windows
- **Queuing**: Sequential processing without losing operations
- **Batching**: Group multiple operations for bulk processing

### Hook Architecture

Each technique offers three levels of abstraction:

1. **Simple Callback Hooks**: Basic wrapper around core functions
2. **State-Integrated Hooks**: Built-in React state management
3. **Low-Level Instance Hooks**: Direct access to underlying instances

## Debouncing

Debouncing delays execution until calls stop coming in. Perfect for search inputs and form validation.

### Basic Debounced Callback

```tsx
import { useDebouncedCallback } from '@tanstack/react-pacer';

function SearchComponent() {
  const debouncedSearch = useDebouncedCallback(
    (query: string) => {
      console.log('Searching for:', query);
      // Perform search
    },
    { wait: 500 } // Wait 500ms after last call
  );

  return (
    <input
      type="text"
      onChange={(e) => debouncedSearch(e.target.value)}
      placeholder="Search..."
    />
  );
}
```

### Debounced State

```tsx
import { useDebouncedState } from '@tanstack/react-pacer';

function FormComponent() {
  const [searchQuery, setSearchQuery, debouncer] = useDebouncedState('', {
    wait: 300,
    onExecute: (query) => {
      console.log('Executing search for:', query);
      // Perform API call
    }
  });

  return (
    <div>
      <input
        value={searchQuery}
        onChange={(e) => setSearchQuery(e.target.value)}
        placeholder="Type to search..."
      />
      <p>Current query: {searchQuery}</p>
      <button onClick={() => debouncer.cancel()}>
        Cancel Search
      </button>
    </div>
  );
}
```

### Debounced Value (Auto-debouncing)

```tsx
import { useDebouncedValue } from '@tanstack/react-pacer';

function AutoDebouncedComponent({ externalValue }) {
  const debouncedValue = useDebouncedValue(externalValue, {
    wait: 200,
    onExecute: (value) => console.log('Value stabilized:', value)
  });

  return <div>Debounced: {debouncedValue}</div>;
}
```

### Advanced Debouncing with Leading Edge

```tsx
import { useDebouncer } from '@tanstack/react-pacer';

function AdvancedDebounceComponent() {
  const debouncer = useDebouncer(
    (action: string) => console.log('Action:', action),
    {
      wait: 500,
      leading: true,  // Execute immediately on first call
      trailing: true, // Also execute after wait period
      onReject: () => console.log('Call was debounced')
    }
  );

  return (
    <div>
      <button onClick={() => debouncer.maybeExecute('click')}>
        Click Me
      </button>
      <p>Executions: {debouncer.getExecutionCount()}</p>
    </div>
  );
}
```

## Throttling

Throttling ensures consistent spacing between executions. Ideal for scroll handlers and mouse events.

### Basic Throttled Callback

```tsx
import { useThrottledCallback } from '@tanstack/react-pacer';

function ScrollComponent() {
  const throttledScroll = useThrottledCallback(
    (scrollY: number) => {
      console.log('Scroll position:', scrollY);
      // Update UI based on scroll
    },
    { wait: 100 } // Execute at most once per 100ms
  );

  useEffect(() => {
    const handleScroll = () => throttledScroll(window.scrollY);
    window.addEventListener('scroll', handleScroll);
    return () => window.removeEventListener('scroll', handleScroll);
  }, [throttledScroll]);

  return <div style={{ height: '200vh' }}>Scroll me!</div>;
}
```

### Throttled State Updates

```tsx
import { useThrottledState } from '@tanstack/react-pacer';

function MouseTracker() {
  const [position, setPosition, throttler] = useThrottledState(
    { x: 0, y: 0 },
    {
      wait: 50, // Update position at most every 50ms
      onExecute: (pos) => console.log('Position updated:', pos)
    }
  );

  return (
    <div
      style={{ height: '100vh', background: '#f0f0f0' }}
      onMouseMove={(e) => setPosition({ x: e.clientX, y: e.clientY })}
    >
      <p>Mouse at: {position.x}, {position.y}</p>
      <button onClick={() => throttler.reset()}>
        Reset Throttler
      </button>
    </div>
  );
}
```

### Leading vs Trailing Throttling

```tsx
import { useThrottler } from '@tanstack/react-pacer';

function ThrottlingModes() {
  const leadingThrottler = useThrottler(
    () => console.log('Leading execution'),
    { wait: 1000, leading: true, trailing: false }
  );

  const trailingThrottler = useThrottler(
    () => console.log('Trailing execution'),
    { wait: 1000, leading: false, trailing: true }
  );

  return (
    <div>
      <button onClick={() => leadingThrottler.maybeExecute()}>
        Leading Throttle (executes immediately)
      </button>
      <button onClick={() => trailingThrottler.maybeExecute()}>
        Trailing Throttle (executes after wait)
      </button>
    </div>
  );
}
```

## Rate Limiting

Rate limiting provides hard limits within time windows. Essential for API calls and resource management.

### Basic Rate Limited Callback

```tsx
import { useRateLimitedCallback } from '@tanstack/react-pacer';

function ApiComponent() {
  const rateLimitedApi = useRateLimitedCallback(
    async (data: any) => {
      const response = await fetch('/api/data', {
        method: 'POST',
        body: JSON.stringify(data)
      });
      return response.json();
    },
    {
      limit: 5,        // 5 calls
      window: 60000,   // per minute
      windowType: 'sliding', // or 'fixed'
      onReject: () => alert('Rate limit exceeded! Please wait.')
    }
  );

  return (
    <button onClick={() => rateLimitedApi({ action: 'save' })}>
      Save Data (Rate Limited)
    </button>
  );
}
```

### Rate Limited State with Feedback

```tsx
import { useRateLimitedState, useRateLimiter } from '@tanstack/react-pacer';

function RateLimitedForm() {
  const [formData, setFormData] = useRateLimitedState(
    { name: '', email: '' },
    {
      limit: 3,
      window: 30000, // 30 seconds
      onExecute: (data) => console.log('Form updated:', data),
      onReject: () => console.log('Update rate limited')
    }
  );

  // Access the underlying rate limiter for status
  const rateLimiter = useRateLimiter(() => {}, {
    limit: 3,
    window: 30000
  });

  return (
    <div>
      <input
        value={formData.name}
        onChange={(e) => setFormData({ ...formData, name: e.target.value })}
        placeholder="Name"
      />
      <input
        value={formData.email}
        onChange={(e) => setFormData({ ...formData, email: e.target.value })}
        placeholder="Email"
      />
      <p>
        Executions: {rateLimiter.getExecutionCount()} / {rateLimiter.options.limit}
      </p>
    </div>
  );
}
```

### Fixed vs Sliding Windows

```tsx
import { useRateLimiter } from '@tanstack/react-pacer';

function WindowTypesDemo() {
  const fixedWindow = useRateLimiter(
    () => console.log('Fixed window execution'),
    {
      limit: 3,
      window: 10000,
      windowType: 'fixed' // Resets completely every 10 seconds
    }
  );

  const slidingWindow = useRateLimiter(
    () => console.log('Sliding window execution'),
    {
      limit: 3,
      window: 10000,
      windowType: 'sliding' // Rolling 10-second window
    }
  );

  return (
    <div>
      <button onClick={() => fixedWindow.maybeExecute()}>
        Fixed Window (resets completely)
      </button>
      <button onClick={() => slidingWindow.maybeExecute()}>
        Sliding Window (rolling limit)
      </button>
    </div>
  );
}
```

## Queuing

Queuing ensures every operation is processed without loss. Perfect for critical operations and sequential processing.

### Basic Queued State

```tsx
import { useQueuedState } from '@tanstack/react-pacer';

function TaskProcessor() {
  const [tasks, addTask, queuer] = useQueuedState<string>([], {
    onExecute: (task) => {
      console.log('Processing task:', task);
      // Process the task
      return new Promise(resolve => setTimeout(resolve, 1000));
    },
    concurrency: 2 // Process 2 tasks simultaneously
  });

  return (
    <div>
      <button onClick={() => addTask(`Task ${Date.now()}`)}>
        Add Task
      </button>
      <button onClick={() => queuer.start()}>Start Processing</button>
      <button onClick={() => queuer.pause()}>Pause</button>
      <button onClick={() => queuer.stop()}>Stop</button>
      
      <div>
        <h3>Pending Tasks: {tasks.length}</h3>
        <ul>
          {tasks.map((task, index) => (
            <li key={index}>{task}</li>
          ))}
        </ul>
      </div>
    </div>
  );
}
```

### Priority Queuing

```tsx
import { useQueuer } from '@tanstack/react-pacer';
import { useState } from 'react';

interface PriorityTask {
  id: string;
  message: string;
  priority: number;
}

function PriorityTaskManager() {
  const [tasks, setTasks] = useState<PriorityTask[]>([]);
  
  const queuer = useQueuer<PriorityTask>({
    onExecute: async (task) => {
      console.log(`Processing priority ${task.priority} task:`, task.message);
      await new Promise(resolve => setTimeout(resolve, 1000));
    },
    onItemsChange: (queue) => setTasks(queue.peekAllItems()),
    getPriority: (task) => task.priority, // Higher numbers = higher priority
    order: 'desc' // Process highest priority first
  });

  const addTask = (message: string, priority: number) => {
    queuer.add({
      id: Date.now().toString(),
      message,
      priority
    });
  };

  return (
    <div>
      <div>
        <button onClick={() => addTask('Low priority task', 1)}>
          Add Low Priority
        </button>
        <button onClick={() => addTask('High priority task', 5)}>
          Add High Priority
        </button>
        <button onClick={() => addTask('Critical task', 10)}>
          Add Critical
        </button>
      </div>
      
      <button onClick={() => queuer.start()}>Start Processing</button>
      
      <div>
        <h3>Task Queue ({tasks.length} items):</h3>
        {tasks.map(task => (
          <div key={task.id}>
            Priority {task.priority}: {task.message}
          </div>
        ))}
      </div>
    </div>
  );
}
```

### LIFO Queue with Expiration

```tsx
import { useQueuedValue } from '@tanstack/react-pacer';

function RecentActionsProcessor() {
  const queuedActions = useQueuedValue<string>({
    onExecute: (action) => {
      console.log('Processing recent action:', action);
      return new Promise(resolve => setTimeout(resolve, 500));
    },
    order: 'desc', // LIFO - process most recent first
    expireAfter: 30000, // Expire actions after 30 seconds
    onExpire: (action) => console.log('Action expired:', action)
  });

  return (
    <div>
      <button onClick={() => queuedActions.add('User clicked button')}>
        Click Action
      </button>
      <button onClick={() => queuedActions.add('User scrolled page')}>
        Scroll Action
      </button>
      <button onClick={() => queuedActions.add('User typed text')}>
        Type Action
      </button>
    </div>
  );
}
```

## Batching

Batching groups multiple operations for efficient bulk processing.

### Time-Based Batching

```tsx
import { useBatcher } from '@tanstack/react-pacer';
import { useState } from 'react';

function LogBatcher() {
  const [logs, setLogs] = useState<string[]>([]);
  
  const batcher = useBatcher<string>({
    onBatch: (items) => {
      console.log('Sending batch of logs:', items);
      // Send to logging service
      setLogs(prev => [...prev, ...items]);
    },
    batchSize: 5,      // Send when 5 items collected
    maxWait: 3000,     // Or every 3 seconds, whichever comes first
    onItemsChange: (items) => console.log('Batch has', items.length, 'items')
  });

  const addLog = (message: string) => {
    batcher.add(`${new Date().toISOString()}: ${message}`);
  };

  return (
    <div>
      <button onClick={() => addLog('User action logged')}>
        Log Action
      </button>
      <button onClick={() => addLog('Error occurred')}>
        Log Error
      </button>
      <button onClick={() => batcher.flush()}>
        Force Send Batch
      </button>
      
      <div>
        <h3>Sent Logs ({logs.length}):</h3>
        {logs.map((log, index) => (
          <div key={index}>{log}</div>
        ))}
      </div>
    </div>
  );
}
```

### Conditional Batching

```tsx
import { useBatcher } from '@tanstack/react-pacer';

interface AnalyticsEvent {
  type: string;
  data: any;
  timestamp: number;
}

function AnalyticsBatcher() {
  const batcher = useBatcher<AnalyticsEvent>({
    onBatch: (events) => {
      // Send to analytics service
      console.log('Sending analytics batch:', events);
      fetch('/api/analytics', {
        method: 'POST',
        body: JSON.stringify({ events })
      });
    },
    shouldBatch: (items) => {
      // Custom batching logic
      const criticalEvents = items.filter(e => e.type === 'error');
      const totalSize = JSON.stringify(items).length;
      
      // Batch if we have critical events or size is large
      return criticalEvents.length > 0 || totalSize > 1024;
    },
    maxWait: 5000 // Force send every 5 seconds regardless
  });

  const trackEvent = (type: string, data: any) => {
    batcher.add({
      type,
      data,
      timestamp: Date.now()
    });
  };

  return (
    <div>
      <button onClick={() => trackEvent('click', { button: 'submit' })}>
        Track Click
      </button>
      <button onClick={() => trackEvent('error', { message: 'Failed' })}>
        Track Error
      </button>
      <button onClick={() => trackEvent('view', { page: 'home' })}>
        Track View
      </button>
    </div>
  );
}
```

## Async Operations

TanStack Pacer provides specialized hooks for async operations with return value handling.

### Async Debounced API Calls

```tsx
import { useAsyncDebouncer } from '@tanstack/react-pacer';
import { useState } from 'react';

interface SearchResult {
  id: string;
  title: string;
  description: string;
}

function AsyncSearchComponent() {
  const [results, setResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const debouncedSearch = useAsyncDebouncer(
    async (query: string) => {
      if (!query) return [];
      
      const response = await fetch(`/api/search?q=${encodeURIComponent(query)}`);
      if (!response.ok) throw new Error('Search failed');
      
      const data = await response.json();
      return data.results;
    },
    {
      wait: 300,
      onExecute: () => {
        setLoading(true);
        setError(null);
      },
      onSuccess: (results) => {
        setResults(results);
        setLoading(false);
      },
      onError: (err) => {
        setError(err.message);
        setLoading(false);
        setResults([]);
      }
    }
  );

  return (
    <div>
      <input
        type="text"
        onChange={(e) => debouncedSearch.maybeExecute(e.target.value)}
        placeholder="Search..."
      />
      
      {loading && <div>Searching...</div>}
      {error && <div>Error: {error}</div>}
      
      <ul>
        {results.map(result => (
          <li key={result.id}>
            <h4>{result.title}</h4>
            <p>{result.description}</p>
          </li>
        ))}
      </ul>
    </div>
  );
}
```

### Async Rate Limited Operations

```tsx
import { useAsyncRateLimiter } from '@tanstack/react-pacer';
import { useState } from 'react';

function ApiUploader() {
  const [uploads, setUploads] = useState<string[]>([]);
  const [status, setStatus] = useState<string>('');

  const rateLimitedUpload = useAsyncRateLimiter(
    async (file: File) => {
      const formData = new FormData();
      formData.append('file', file);
      
      const response = await fetch('/api/upload', {
        method: 'POST',
        body: formData
      });
      
      if (!response.ok) throw new Error('Upload failed');
      
      const result = await response.json();
      return result.url;
    },
    {
      limit: 3,
      window: 60000, // 3 uploads per minute
      onExecute: () => setStatus('Uploading...'),
      onSuccess: (url, file) => {
        setUploads(prev => [...prev, `${file.name}: ${url}`]);
        setStatus('Upload successful');
      },
      onError: (err) => setStatus(`Upload failed: ${err.message}`),
      onReject: () => setStatus('Rate limit exceeded. Please wait.')
    }
  );

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      rateLimitedUpload.maybeExecute(file);
    }
  };

  return (
    <div>
      <input type="file" onChange={handleFileSelect} />
      <div>Status: {status}</div>
      <div>Executions: {rateLimitedUpload.getExecutionCount()}</div>
      
      <h3>Uploaded Files:</h3>
      <ul>
        {uploads.map((upload, index) => (
          <li key={index}>{upload}</li>
        ))}
      </ul>
    </div>
  );
}
```

### Async Queued Processing

```tsx
import { useAsyncQueuedState } from '@tanstack/react-pacer';

interface ProcessingJob {
  id: string;
  data: any;
  status: 'pending' | 'processing' | 'completed' | 'failed';
}

function JobProcessor() {
  const [jobs, addJob, queuer] = useAsyncQueuedState<ProcessingJob>([], {
    onExecute: async (job) => {
      job.status = 'processing';
      
      try {
        // Simulate processing
        await new Promise(resolve => setTimeout(resolve, 2000));
        
        // Simulate random failure
        if (Math.random() < 0.2) {
          throw new Error('Processing failed');
        }
        
        job.status = 'completed';
        console.log('Job completed:', job.id);
      } catch (error) {
        job.status = 'failed';
        console.error('Job failed:', job.id, error);
        throw error; // Re-throw to trigger onError
      }
    },
    concurrency: 3, // Process up to 3 jobs simultaneously
    onError: (error, job) => {
      console.error('Job processing error:', error, job);
    },
    retries: 2, // Retry failed jobs twice
    retryDelay: 1000 // Wait 1 second between retries
  });

  const createJob = () => {
    const job: ProcessingJob = {
      id: `job-${Date.now()}`,
      data: { value: Math.random() },
      status: 'pending'
    };
    addJob(job);
  };

  return (
    <div>
      <button onClick={createJob}>Add Job</button>
      <button onClick={() => queuer.start()}>Start Processing</button>
      <button onClick={() => queuer.pause()}>Pause</button>
      <button onClick={() => queuer.stop()}>Stop</button>
      
      <div>
        <h3>Jobs Queue ({jobs.length} items):</h3>
        {jobs.map(job => (
          <div key={job.id} style={{ 
            padding: '8px',
            margin: '4px',
            background: job.status === 'completed' ? '#d4edda' : 
                       job.status === 'failed' ? '#f8d7da' : 
                       job.status === 'processing' ? '#fff3cd' : '#e9ecef'
          }}>
            {job.id} - Status: {job.status}
          </div>
        ))}
      </div>
    </div>
  );
}
```

## Best Practices

### 1. Choose the Right Technique

- **Debouncing**: User input (search, form validation), resize events
- **Throttling**: Scroll handlers, mouse movement, progress updates
- **Rate Limiting**: API calls with strict limits, external service calls
- **Queuing**: Critical operations that can't be lost, sequential processing
- **Batching**: Bulk operations, logging, analytics events

### 2. Hook Selection Guidelines

```tsx
// Use simple callback hooks for basic needs
const debouncedFn = useDebouncedCallback(fn, options);

// Use state hooks when you need built-in React state
const [value, setValue, controller] = useDebouncedState(initial, options);

// Use low-level hooks for custom state management integration
const controller = useDebouncer(fn, options);
```

### 3. Error Handling

```tsx
// Always provide error handlers for async operations
const asyncHook = useAsyncDebouncer(asyncFn, {
  onError: (error) => {
    console.error('Operation failed:', error);
    // Handle error appropriately
  },
  throwOnError: false // Prevent unhandled promise rejections
});
```

### 4. Performance Optimization

```tsx
// Use React.useCallback to prevent unnecessary re-renders
const stableCallback = useCallback((data) => {
  // Your logic here
}, [dependencies]);

const debouncedFn = useDebouncedCallback(stableCallback, options);
```

### 5. State Management Integration

```tsx
// Example with Zustand
const useStore = create((set) => ({
  items: [],
  setItems: (items) => set({ items })
}));

function Component() {
  const { items, setItems } = useStore();
  
  const queuer = useQueuer({
    onItemsChange: (queue) => setItems(queue.peekAllItems())
  });
}
```

## Common Patterns

### Search with Debouncing and Rate Limiting

```tsx
function SmartSearch() {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState([]);
  
  // Debounce user input
  const debouncedSearch = useAsyncDebouncer(
    async (searchQuery: string) => {
      const response = await fetch(`/api/search?q=${searchQuery}`);
      return response.json();
    },
    { wait: 300 }
  );
  
  // Rate limit API calls
  const rateLimitedSearch = useAsyncRateLimiter(
    debouncedSearch.maybeExecute,
    {
      limit: 10,
      window: 60000 // 10 searches per minute
    }
  );
  
  const handleSearch = (value: string) => {
    setQuery(value);
    if (value) {
      rateLimitedSearch.maybeExecute(value)
        .then(setResults)
        .catch(console.error);
    }
  };
  
  return (
    <div>
      <input
        value={query}
        onChange={(e) => handleSearch(e.target.value)}
        placeholder="Search..."
      />
      {/* Render results */}
    </div>
  );
}
```

### Progressive Enhancement with Multiple Techniques

```tsx
function AdvancedDataProcessor() {
  // Debounce user input
  const debouncedInput = useDebouncedValue(userInput, { wait: 500 });
  
  // Throttle UI updates
  const throttledUpdate = useThrottledCallback(updateUI, { wait: 100 });
  
  // Rate limit API calls
  const rateLimitedAPI = useRateLimitedCallback(apiCall, {
    limit: 5,
    window: 60000
  });
  
  // Queue critical operations
  const [criticalTasks, addCriticalTask] = useQueuedState([], {
    onExecute: processCriticalTask
  });
  
  // Batch analytics events
  const analyticsBatcher = useBatcher({
    onBatch: sendAnalyticsEvents,
    batchSize: 10,
    maxWait: 5000
  });
  
  // Coordinate all techniques
  useEffect(() => {
    if (debouncedInput) {
      throttledUpdate(debouncedInput);
      rateLimitedAPI(debouncedInput);
      analyticsBatcher.add({ type: 'search', query: debouncedInput });
    }
  }, [debouncedInput]);
}
```

### Error Recovery and Resilience

```tsx
function ResilientComponent() {
  const [retryCount, setRetryCount] = useState(0);
  
  const resilientOperation = useAsyncDebouncer(
    async (data) => {
      try {
        return await riskyApiCall(data);
      } catch (error) {
        if (retryCount < 3) {
          setRetryCount(prev => prev + 1);
          throw error; // Will be caught by onError
        }
        throw new Error('Max retries exceeded');
      }
    },
    {
      wait: 1000,
      onError: (error) => {
        if (retryCount < 3) {
          // Retry after increasing delay
          setTimeout(() => {
            resilientOperation.maybeExecute(lastData);
          }, 1000 * Math.pow(2, retryCount));
        }
      },
      onSuccess: () => setRetryCount(0)
    }
  );
}
```

This comprehensive guide covers all major aspects of using TanStack Pacer in React applications. The library provides powerful, flexible tools for managing execution timing and flow control, enabling you to build more responsive and efficient user interfaces.