import { useState, useEffect } from 'react';

interface EmailDisplayProps {
  address: string;
  expiresAt: string;
}

export function EmailDisplay({ address, expiresAt }: EmailDisplayProps) {
  const [copied, setCopied] = useState(false);
  const [timeLeft, setTimeLeft] = useState('');

  useEffect(() => {
    const updateCountdown = () => {
      const now = new Date();
      const expires = new Date(expiresAt);
      const diff = expires.getTime() - now.getTime();

      if (diff <= 0) {
        setTimeLeft('Expired');
        return;
      }

      const hours = Math.floor(diff / (1000 * 60 * 60));
      const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
      const seconds = Math.floor((diff % (1000 * 60)) / 1000);

      setTimeLeft(`${hours}h ${minutes}m ${seconds}s`);
    };

    updateCountdown();
    const interval = setInterval(updateCountdown, 1000);

    return () => clearInterval(interval);
  }, [expiresAt]);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(address);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  return (
    <div className="p-5 bg-gray-100 rounded-lg mb-5">
      <h2 className="m-0 mb-4 text-lg text-gray-800">Your Temporary Email</h2>
      <div className="flex items-center gap-2.5 p-4 bg-white rounded-md border-2 border-gray-200">
        <code className="flex-1 text-base text-blue-600 font-mono break-all">
          {address}
        </code>
        <button
          onClick={handleCopy}
          className="px-4 py-2 bg-blue-600 text-white border-none rounded cursor-pointer text-sm whitespace-nowrap hover:bg-blue-700 transition-colors"
        >
          {copied ? '✓ Copied!' : 'Copy'}
        </button>
      </div>
      <div className="mt-2.5 text-sm text-gray-600">
        Expires in: <strong>{timeLeft}</strong>
      </div>
    </div>
  );
}
