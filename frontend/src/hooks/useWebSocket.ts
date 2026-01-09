import { useEffect, useRef, useState } from 'react';

interface WebSocketMessage {
  type: string;
  data: {
    id: string;
    from: string;
    subject: string;
    preview: string;
    received_at: string;
  };
}

interface UseWebSocketResult {
  messages: WebSocketMessage[];
  isConnected: boolean;
  error: string | null;
}

const WS_BASE_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:8080';
const MAX_RECONNECT_DELAY_MS = 30000;

export function useWebSocket(address: string | null): UseWebSocketResult {
  const [messages, setMessages] = useState<WebSocketMessage[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const addressRef = useRef(address);

  // Keep address ref in sync
  useEffect(() => {
    addressRef.current = address;
  }, [address]);

  useEffect(() => {
    if (!address) return;

    function connect() {
      const currentAddress = addressRef.current;
      if (!currentAddress) return;

      try {
        const ws = new WebSocket(
          `${WS_BASE_URL}/ws?address=${encodeURIComponent(currentAddress)}`
        );
        wsRef.current = ws;

        ws.onopen = () => {
          console.log('WebSocket connected');
          setIsConnected(true);
          setError(null);
          reconnectAttemptsRef.current = 0;
        };

        ws.onmessage = (event) => {
          try {
            const message: WebSocketMessage = JSON.parse(event.data);
            if (message.type === 'new_email') {
              setMessages((prev) => [message, ...prev]);
            }
          } catch (err) {
            console.error('Failed to parse WebSocket message:', err);
          }
        };

        ws.onerror = (event) => {
          console.error('WebSocket error:', event);
          setError('WebSocket connection error');
        };

        ws.onclose = () => {
          console.log('WebSocket disconnected');
          setIsConnected(false);
          wsRef.current = null;

          const backoff = Math.min(
            1000 * Math.pow(2, reconnectAttemptsRef.current),
            MAX_RECONNECT_DELAY_MS
          );
          reconnectAttemptsRef.current += 1;

          console.log(`Reconnecting in ${backoff}ms...`);
          reconnectTimeoutRef.current = setTimeout(connect, backoff);
        };
      } catch (err) {
        console.error('Failed to create WebSocket:', err);
        setError('Failed to establish WebSocket connection');
      }
    }

    connect();

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, [address]);

  return { messages, isConnected, error };
}
