import type { EmailSummary } from '../services/api';

interface EmailListProps {
  emails: EmailSummary[];
  onSelectEmail: (email: EmailSummary) => void;
  isConnected: boolean;
}

function formatTimeAgo(timestamp: string): string {
  const now = new Date();
  const date = new Date(timestamp);
  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);
  const diffMin = Math.floor(diffSec / 60);
  const diffHour = Math.floor(diffMin / 60);
  const diffDay = Math.floor(diffHour / 24);

  if (diffSec < 60) return `${diffSec}s ago`;
  if (diffMin < 60) return `${diffMin}m ago`;
  if (diffHour < 24) return `${diffHour}h ago`;
  return `${diffDay}d ago`;
}

function ConnectionStatus({ isConnected }: { isConnected: boolean }) {
  return isConnected ? (
    <span className="text-green-500">● Connected</span>
  ) : (
    <span className="text-slate-400">○ Connecting...</span>
  );
}

export function EmailList({ emails, onSelectEmail, isConnected }: EmailListProps) {
  if (emails.length === 0) {
    return (
      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <div className="py-15 px-5 text-center">
          <div className="text-xs mb-4">
            <ConnectionStatus isConnected={isConnected} />
          </div>
          <p className="text-base text-gray-800 mb-2.5">Waiting for emails...</p>
          <p className="text-sm text-slate-400 max-w-md mx-auto">
            Send an email to your temporary address above and it will appear here in real-time.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
      <div className="flex justify-between items-center px-5 py-4 border-b border-gray-200 bg-gray-50">
        <h3 className="m-0 text-base font-semibold text-gray-800">Inbox</h3>
        <div className="text-xs">
          <ConnectionStatus isConnected={isConnected} />
        </div>
      </div>
      <div className="max-h-[600px] overflow-y-auto">
        {emails.map((email) => (
          <div
            key={email.id}
            className="px-5 py-4 border-b border-gray-200 cursor-pointer transition-colors hover:bg-gray-50"
            onClick={() => onSelectEmail(email)}
          >
            <div className="flex justify-between items-center mb-1">
              <span className="font-semibold text-sm text-gray-800">
                {email.from}
              </span>
              <span className="text-xs text-slate-400">
                {formatTimeAgo(email.received_at)}
              </span>
            </div>
            <div className="text-sm text-gray-800 mb-1 font-medium">
              {email.subject || '(No Subject)'}
              {email.has_attachments && (
                <span className="text-xs ml-1">📎</span>
              )}
            </div>
            <div className="text-[13px] text-slate-500 overflow-hidden text-ellipsis whitespace-nowrap">
              {email.preview}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
