import { useState, useEffect } from 'react';
import { Helmet } from 'react-helmet-async';
import { EmailDisplay } from '../components/EmailDisplay';
import { EmailList } from '../components/EmailList';
import { EmailViewer } from '../components/EmailViewer';
import { useWebSocket } from '../hooks/useWebSocket';
import { generateEmail, getEmails } from '../services/api';
import type { EmailSummary } from '../services/api';
import {
  saveEmailAddress,
  getEmailAddress,
  getEmailExpiry,
  clearEmailAddress,
} from '../utils/localStorage';

export function Home() {
  const [address, setAddress] = useState<string | null>(null);
  const [expiresAt, setExpiresAt] = useState<string | null>(null);
  const [emails, setEmails] = useState<EmailSummary[]>([]);
  const [selectedEmail, setSelectedEmail] = useState<EmailSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const { messages, isConnected } = useWebSocket(address);

  useEffect(() => {
    const initializeAddress = async () => {
      try {
        const storedAddress = getEmailAddress();

        if (storedAddress) {
          setAddress(storedAddress);
          setExpiresAt(getEmailExpiry());
          const response = await getEmails(storedAddress);
          setEmails(response.emails);
        } else {
          const response = await generateEmail();
          setAddress(response.address);
          setExpiresAt(response.expires_at);
          saveEmailAddress(response.address, response.expires_at);
        }
      } catch (err) {
        console.error('Failed to initialize:', err);
        setError('Failed to initialize email address');
      } finally {
        setLoading(false);
      }
    };

    initializeAddress();
  }, []);

  useEffect(() => {
    if (messages.length === 0) return;

    const latestMessage = messages[0];
    if (latestMessage.type === 'new_email') {
      const newEmail: EmailSummary = {
        id: latestMessage.data.id,
        from: latestMessage.data.from,
        subject: latestMessage.data.subject,
        preview: latestMessage.data.preview,
        received_at: latestMessage.data.received_at,
        has_attachments: false,
      };
      setEmails((prev) => [newEmail, ...prev]);
    }
  }, [messages]);

  const handleGenerateNew = async () => {
    try {
      setLoading(true);
      clearEmailAddress();
      const response = await generateEmail();
      setAddress(response.address);
      setExpiresAt(response.expires_at);
      saveEmailAddress(response.address, response.expires_at);
      setEmails([]);
      setSelectedEmail(null);
    } catch (err) {
      console.error('Failed to generate new address:', err);
      setError('Failed to generate new email address');
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return <div className="text-center py-15 text-slate-500">Loading...</div>;
  }

  if (error) {
    return <div className="text-center py-15 text-red-600">{error}</div>;
  }

  return (
    <>
      <Helmet>
        <title>Free Temporary Email - tmpemail.xyz</title>
        <meta
          name="description"
          content="Get a free disposable email address instantly. Protect your privacy, avoid spam, and keep your inbox clean with tmpemail.xyz."
        />
        <meta property="og:title" content="Free Temporary Email - tmpemail.xyz" />
        <meta
          property="og:description"
          content="Get a free disposable email address instantly. Protect your privacy and avoid spam."
        />
        <meta property="og:type" content="website" />
        <meta property="og:url" content="https://tmpemail.xyz/" />
        <meta name="twitter:card" content="summary" />
        <meta name="twitter:title" content="Free Temporary Email - tmpemail.xyz" />
        <meta
          name="twitter:description"
          content="Get a free disposable email address instantly. Protect your privacy and avoid spam."
        />
        <link rel="canonical" href="https://tmpemail.xyz/" />
      </Helmet>

      <header className="text-center mb-8">
        <h1 className="text-3xl font-bold text-slate-800 mb-1">tmpemail.xyz</h1>
        <p className="text-slate-500">Temporary Email Service</p>
      </header>

      {address && expiresAt && (
        <>
          <EmailDisplay address={address} expiresAt={expiresAt} />
          <div className="mb-5 text-center">
            <button
              onClick={handleGenerateNew}
              className="px-5 py-2.5 bg-slate-500 text-white border-none rounded-md cursor-pointer text-sm font-medium hover:bg-slate-600 transition-colors"
            >
              Generate New Address
            </button>
          </div>
        </>
      )}

      <EmailList
        emails={emails}
        onSelectEmail={setSelectedEmail}
        isConnected={isConnected}
      />

      {selectedEmail && address && (
        <EmailViewer
          address={address}
          emailId={selectedEmail.id}
          onClose={() => setSelectedEmail(null)}
        />
      )}
    </>
  );
}
