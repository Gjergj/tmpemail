import { Helmet } from 'react-helmet-async';

export function API() {
  return (
    <>
      <Helmet>
        <title>API Documentation - tmpemail.xyz</title>
        <meta
          name="description"
          content="Free API for temporary email testing. Generate addresses, fetch emails, and filter by sender, subject, or date."
        />
        <meta property="og:title" content="API Documentation - tmpemail.xyz" />
        <meta
          property="og:description"
          content="Free API for temporary email testing."
        />
        <meta property="og:type" content="website" />
        <meta property="og:url" content="https://tmpemail.xyz/api" />
        <meta name="twitter:card" content="summary" />
        <meta name="twitter:title" content="API Documentation - tmpemail.xyz" />
        <meta
          name="twitter:description"
          content="Free API for temporary email testing."
        />
        <link rel="canonical" href="https://tmpemail.xyz/api" />
      </Helmet>

      <header className="text-center mb-10">
        <h1 className="text-3xl font-bold text-slate-800 mb-2">API Documentation</h1>
        <p className="text-slate-500">
          Free API for temporary email testing
        </p>
      </header>

      <div className="space-y-8">
        {/* Rate Limiting Notice */}
        <div className="bg-amber-50 border border-amber-200 rounded-lg p-4">
          <div className="flex items-start gap-3">
            <svg className="w-5 h-5 text-amber-600 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
            <div>
              <h3 className="text-sm font-semibold text-amber-900 mb-1">Rate Limited for Testing</h3>
              <p className="text-sm text-amber-700">
                This API is rate limited and intended for testing purposes only. Generate endpoint: 10 requests/min, API endpoints: 60 requests/min, WebSocket: 5 connections/min.
              </p>
            </div>
          </div>
        </div>

        {/* Base URL */}
        <section className="bg-white rounded-lg p-6 shadow-sm border border-slate-200">
          <h2 className="text-xl font-semibold text-slate-800 mb-3">Base URL</h2>
          <code className="block bg-slate-100 text-slate-800 px-4 py-2 rounded text-sm">
            https://api.tmpemail.xyz/api/v1
          </code>
        </section>

        {/* Endpoints */}
        <section className="bg-white rounded-lg p-6 shadow-sm border border-slate-200">
          <h2 className="text-xl font-semibold text-slate-800 mb-4">Endpoints</h2>

          <div className="space-y-6">
            {/* Generate Address */}
            <div className="border-l-4 border-blue-500 pl-4">
              <div className="flex items-center gap-2 mb-2">
                <span className="text-xs font-bold bg-blue-100 text-blue-700 px-2 py-1 rounded">GET</span>
                <code className="text-sm text-slate-700">/generate</code>
              </div>
              <p className="text-sm text-slate-600 mb-2">Generate a new temporary email address.</p>
              <div className="bg-slate-50 rounded p-3 text-xs">
                <div className="text-slate-500 mb-1">Response:</div>
                <pre className="text-slate-800 overflow-x-auto">{`{
  "address": "bright-dolphin-873008@tmpemail.xyz",
  "expires_at": "2026-01-19T15:30:00Z"
}`}</pre>
              </div>
            </div>

            {/* List Emails */}
            <div className="border-l-4 border-green-500 pl-4">
              <div className="flex items-center gap-2 mb-2">
                <span className="text-xs font-bold bg-green-100 text-green-700 px-2 py-1 rounded">GET</span>
                <code className="text-sm text-slate-700">/emails/:address</code>
              </div>
              <p className="text-sm text-slate-600 mb-2">Retrieve all emails for an address.</p>
              <div className="bg-slate-50 rounded p-3 text-xs">
                <div className="text-slate-500 mb-1">Example:</div>
                <pre className="text-slate-800 overflow-x-auto">{`GET /emails/bright-dolphin-873008@tmpemail.xyz`}</pre>
              </div>
            </div>

            {/* Filter Emails - NEW */}
            <div className="border-l-4 border-purple-500 pl-4">
              <div className="flex items-center gap-2 mb-2">
                <span className="text-xs font-bold bg-purple-100 text-purple-700 px-2 py-1 rounded">GET</span>
                <code className="text-sm text-slate-700">/emails/:address/filter</code>
                <span className="text-xs bg-purple-50 text-purple-600 px-2 py-0.5 rounded-full font-medium">NEW</span>
              </div>
              <p className="text-sm text-slate-600 mb-3">Filter emails by sender, subject, or date.</p>

              <div className="space-y-3">
                <div>
                  <div className="text-xs font-semibold text-slate-700 mb-1">Query Parameters (all optional):</div>
                  <ul className="text-xs text-slate-600 space-y-1 ml-4">
                    <li><code className="bg-slate-100 px-1 rounded">from</code> - Filter by sender email (exact match)</li>
                    <li><code className="bg-slate-100 px-1 rounded">subject</code> - Filter by subject containing text (case-insensitive)</li>
                    <li><code className="bg-slate-100 px-1 rounded">since</code> - Filter by date (RFC3339 format: 2026-01-18T00:00:00Z)</li>
                  </ul>
                </div>

                <div className="bg-slate-50 rounded p-3">
                  <div className="text-slate-500 mb-1 text-xs">Examples:</div>
                  <pre className="text-slate-800 text-xs space-y-1">
{`# Filter by sender
GET /emails/address@tmpemail.xyz/filter?from=test@example.com

# Filter by subject
GET /emails/address@tmpemail.xyz/filter?subject=invoice

# Filter by date
GET /emails/address@tmpemail.xyz/filter?since=2026-01-15T00:00:00Z

# Combine filters
GET /emails/address@tmpemail.xyz/filter?from=test@example.com&subject=urgent&since=2026-01-18T00:00:00Z`}
                  </pre>
                </div>

                <div className="bg-slate-50 rounded p-3">
                  <div className="text-slate-500 mb-1 text-xs">Response:</div>
                  <pre className="text-slate-800 text-xs overflow-x-auto">{`{
  "emails": [
    {
      "id": "01JKHM3X...",
      "from": "test@example.com",
      "subject": "Test Email",
      "preview": "This is a test email body...",
      "received_at": "2026-01-18T10:30:00Z",
      "has_attachments": false
    }
  ]
}`}</pre>
                </div>
              </div>
            </div>

            {/* Get Email Content */}
            <div className="border-l-4 border-green-500 pl-4">
              <div className="flex items-center gap-2 mb-2">
                <span className="text-xs font-bold bg-green-100 text-green-700 px-2 py-1 rounded">GET</span>
                <code className="text-sm text-slate-700">/email/:address/:emailID</code>
              </div>
              <p className="text-sm text-slate-600 mb-2">Get full email content including HTML and text body.</p>
            </div>

            {/* Get Attachments */}
            <div className="border-l-4 border-green-500 pl-4">
              <div className="flex items-center gap-2 mb-2">
                <span className="text-xs font-bold bg-green-100 text-green-700 px-2 py-1 rounded">GET</span>
                <code className="text-sm text-slate-700">/email/:address/:emailID/attachments</code>
              </div>
              <p className="text-sm text-slate-600 mb-2">List all attachments for an email.</p>
            </div>

            {/* Download Attachment */}
            <div className="border-l-4 border-green-500 pl-4">
              <div className="flex items-center gap-2 mb-2">
                <span className="text-xs font-bold bg-green-100 text-green-700 px-2 py-1 rounded">GET</span>
                <code className="text-sm text-slate-700">/email/:address/:emailID/attachments/:attachmentID</code>
              </div>
              <p className="text-sm text-slate-600 mb-2">Download a specific attachment.</p>
            </div>
          </div>
        </section>

        {/* WebSocket */}
        <section className="bg-white rounded-lg p-6 shadow-sm border border-slate-200">
          <h2 className="text-xl font-semibold text-slate-800 mb-4">Real-time Updates</h2>

          <div className="border-l-4 border-indigo-500 pl-4">
            <div className="flex items-center gap-2 mb-2">
              <span className="text-xs font-bold bg-indigo-100 text-indigo-700 px-2 py-1 rounded">WS</span>
              <code className="text-sm text-slate-700">wss://api.tmpemail.xyz/ws?address=:address</code>
            </div>
            <p className="text-sm text-slate-600 mb-3">Connect to receive real-time email notifications.</p>

            <div className="bg-slate-50 rounded p-3 text-xs">
              <div className="text-slate-500 mb-1">Message format:</div>
              <pre className="text-slate-800 overflow-x-auto">{`{
  "type": "new_email",
  "email": {
    "id": "...",
    "from": "sender@example.com",
    "subject": "New Email",
    "preview": "Email preview text...",
    "received_at": "2026-01-18T10:30:00Z"
  }
}`}</pre>
            </div>
          </div>
        </section>

        {/* CURL Examples */}
        <section className="bg-white rounded-lg p-6 shadow-sm border border-slate-200">
          <h2 className="text-xl font-semibold text-slate-800 mb-4">CURL Examples</h2>

          <div className="space-y-4">
            <div>
              <div className="text-sm font-medium text-slate-700 mb-2">Generate address:</div>
              <pre className="bg-slate-900 text-slate-100 p-3 rounded text-xs overflow-x-auto">
{`curl https://api.tmpemail.xyz/api/v1/generate`}
              </pre>
            </div>

            <div>
              <div className="text-sm font-medium text-slate-700 mb-2">Filter emails by sender:</div>
              <pre className="bg-slate-900 text-slate-100 p-3 rounded text-xs overflow-x-auto">
{`curl "https://api.tmpemail.xyz/api/v1/emails/address@tmpemail.xyz/filter?from=test@example.com"`}
              </pre>
            </div>

            <div>
              <div className="text-sm font-medium text-slate-700 mb-2">Filter by subject and date:</div>
              <pre className="bg-slate-900 text-slate-100 p-3 rounded text-xs overflow-x-auto">
{`curl "https://api.tmpemail.xyz/api/v1/emails/address@tmpemail.xyz/filter?subject=invoice&since=2026-01-15T00:00:00Z"`}
              </pre>
            </div>
          </div>
        </section>

        {/* Error Responses */}
        <section className="bg-white rounded-lg p-6 shadow-sm border border-slate-200">
          <h2 className="text-xl font-semibold text-slate-800 mb-4">Error Responses</h2>

          <div className="space-y-3 text-sm">
            <div className="flex gap-3">
              <code className="text-slate-700 font-mono">400</code>
              <span className="text-slate-600">Bad Request - Invalid parameters</span>
            </div>
            <div className="flex gap-3">
              <code className="text-slate-700 font-mono">404</code>
              <span className="text-slate-600">Not Found - Address doesn't exist</span>
            </div>
            <div className="flex gap-3">
              <code className="text-slate-700 font-mono">410</code>
              <span className="text-slate-600">Gone - Address expired</span>
            </div>
            <div className="flex gap-3">
              <code className="text-slate-700 font-mono">429</code>
              <span className="text-slate-600">Too Many Requests - Rate limit exceeded</span>
            </div>
            <div className="flex gap-3">
              <code className="text-slate-700 font-mono">500</code>
              <span className="text-slate-600">Internal Server Error</span>
            </div>
          </div>
        </section>
      </div>
    </>
  );
}
