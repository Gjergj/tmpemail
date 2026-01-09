1. ✅ DONE - if api calls fail it should just log the error but not break the operation. Just log errors.
   - Modified processEmail() to log errors instead of returning them when API store fails
   - Email is still saved to filesystem even if API call fails

2. ✅ DONE - Fix Add Health Check Endpoints
   - Added HTTP health server on port 8081 (configurable via TMPEMAIL_HEALTH_PORT)
   - GET /health - Simple liveness check (always returns ok if server is running)
   - GET /readiness - Readiness check (verifies SMTP server is ready + API connectivity)

3. ✅ DONE - Fix Default Paths Require Root
   - Changed default StoragePath from /var/mail/tmpemail to ./mail

4. SMTP server package evaluation:
   
   **Current: go-smtp (emersion/go-smtp)** ✅ RECOMMENDED TO KEEP
   - Library designed for embedding SMTP functionality in custom applications
   - Lightweight, supports ESMTP, AUTH, PIPELINING, UTF-8, LMTP
   - Perfect for our use case: receiving emails in a custom application
   
   **Evaluated alternatives:**
   
   | Package | Type | Verdict |
   |---------|------|---------|
   | **chasquid** | Full SMTP server | ❌ Not suitable - standalone server to replace Postfix, not a library |
   | **maddy** | All-in-one mail server | ❌ Not suitable - full mail server (SMTP+IMAP+security), overkill |
   | **mailpit** | Email testing tool | ❌ Not suitable - designed for dev/testing, not production |
   | **mox** | Comprehensive mail server | ❌ Not suitable - full server with webmail, spam filtering, etc. |
   
   **Conclusion:** The current `go-smtp` library is the correct choice. The other packages are 
   complete mail servers meant for standalone deployment, not libraries for embedding SMTP 
   functionality in custom applications.

5. ✅ DONE - MIME Parsing improvements
   
   **Migrated from:** Standard library (mime/multipart) - ~60 lines of manual parsing
   **Migrated to:** `github.com/jhillyerd/enmime` v1.3.0
   
   **Benefits gained:**
   - Simpler API with direct access to text, HTML, and attachments
   - Better handling of nested multipart messages  
   - Automatic charset decoding
   - More robust error handling for malformed emails (captures errors instead of failing)
   - Proper handling of inline attachments (embedded images, etc.)
   - Removed ~60 lines of manual MIME parsing code
   
   **Code change:**
   ```go
   // Before (~60 lines of manual parsing)
   msg, _ := mail.ReadMessage(bytes.NewReader(rawEmail))
   bodyText, bodyHTML, attachments := parseEmailBody(msg, logger)
   
   // After (simple and robust)
   env, _ := enmime.ReadEnvelope(bytes.NewReader(rawEmail))
   bodyText := env.Text
   bodyHTML := env.HTML  
   attachments := env.Attachments  // Already parsed!
   inlines := env.Inlines          // Inline attachments too!
   ```


1. Disk Exhaustion (DoS)
  Rate limiting at SMTP level (currently none)


No SPF/DKIM/DMARC validation (accepts mail from any sender)