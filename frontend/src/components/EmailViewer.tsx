import { useEffect, useState } from 'react';
import DOMPurify from 'dompurify';
import { getEmailContent } from '../services/api';
import type { EmailContentResponse } from '../services/api';

interface EmailViewerProps {
  address: string;
  emailId: string;
  onClose: () => void;
}

export function EmailViewer({ address, emailId, onClose }: EmailViewerProps) {
  const [email, setEmail] = useState<EmailContentResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [viewMode, setViewMode] = useState<'html' | 'text'>('html');

  useEffect(() => {
    const fetchEmail = async () => {
      try {
        setLoading(true);
        const data = await getEmailContent(address, emailId);
        setEmail(data);

        if (!data.body_html && data.body_text) {
          setViewMode('text');
        }
      } catch (err) {
        console.error('Failed to fetch email:', err);
        setError('Failed to load email content');
      } finally {
        setLoading(false);
      }
    };

    fetchEmail();
  }, [address, emailId]);

  const handleOverlayClick = (e: React.MouseEvent) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  if (loading) {
    return (
      <div
        className="fixed inset-0 bg-black/50 flex items-center justify-center z-[1000] p-5"
        onClick={handleOverlayClick}
      >
        <div className="bg-white rounded-lg max-w-3xl w-full max-h-[90vh] overflow-hidden flex flex-col">
          <div className="p-15 text-center text-slate-500">Loading...</div>
        </div>
      </div>
    );
  }

  if (error || !email) {
    return (
      <div
        className="fixed inset-0 bg-black/50 flex items-center justify-center z-[1000] p-5"
        onClick={handleOverlayClick}
      >
        <div className="bg-white rounded-lg max-w-3xl w-full max-h-[90vh] overflow-hidden flex flex-col">
          <div className="p-5 text-red-600">{error || 'Email not found'}</div>
          <button
            onClick={onClose}
            className="mx-5 mb-5 px-5 py-2.5 bg-blue-600 text-white border-none rounded cursor-pointer hover:bg-blue-700 transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    );
  }

  const sanitizedHTML = DOMPurify.sanitize(email.body_html);

  return (
    <div
      className="fixed inset-0 bg-black/50 flex items-center justify-center z-[1000] p-5"
      onClick={handleOverlayClick}
    >
      <div className="bg-white rounded-lg max-w-3xl w-full max-h-[90vh] overflow-hidden flex flex-col">
        <div className="p-5 border-b border-gray-200 flex justify-between items-start">
          <div className="flex-1">
            <h2 className="m-0 mb-2.5 text-xl text-gray-800">
              {email.subject || '(No Subject)'}
            </h2>
            <div className="text-sm text-slate-500 mb-1">From: {email.from}</div>
            <div className="text-[13px] text-slate-400">
              {new Date(email.received_at).toLocaleString()}
            </div>
          </div>
          <button
            onClick={onClose}
            className="bg-transparent border-none text-[32px] cursor-pointer text-slate-400 p-0 w-8 h-8 leading-8 hover:text-slate-600"
          >
            ×
          </button>
        </div>

        {email.attachments && email.attachments.length > 0 && (
          <div className="px-5 py-4 bg-slate-50 border-b border-gray-200 text-sm">
            <strong>Attachments: </strong>
            {email.attachments.map((att) => (
              <span key={att.id} className="ml-2.5 text-blue-600">
                📎 {att.filename}
              </span>
            ))}
          </div>
        )}

        {email.body_html && email.body_text && (
          <div className="flex gap-1 px-5 py-2.5 border-b border-gray-200">
            <button
              onClick={() => setViewMode('html')}
              className={`px-3 py-1.5 border rounded text-[13px] cursor-pointer transition-colors ${viewMode === 'html'
                  ? 'border-blue-600 bg-blue-600 text-white'
                  : 'border-gray-200 bg-white text-gray-800 hover:bg-gray-50'
                }`}
            >
              HTML
            </button>
            <button
              onClick={() => setViewMode('text')}
              className={`px-3 py-1.5 border rounded text-[13px] cursor-pointer transition-colors ${viewMode === 'text'
                  ? 'border-blue-600 bg-blue-600 text-white'
                  : 'border-gray-200 bg-white text-gray-800 hover:bg-gray-50'
                }`}
            >
              Plain Text
            </button>
          </div>
        )}

        <div className="flex-1 overflow-y-auto p-5">
          {viewMode === 'html' && email.body_html ? (
            <div
              dangerouslySetInnerHTML={{ __html: sanitizedHTML }}
              className="leading-relaxed"
            />
          ) : (
            <pre className="whitespace-pre-wrap break-words font-mono text-[13px] leading-relaxed m-0">
              {email.body_text || '(No content)'}
            </pre>
          )}
        </div>
      </div>
    </div>
  );
}
