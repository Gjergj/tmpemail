interface EmailAddressData {
  address: string;
  expiresAt: string;
}

const EMAIL_STORAGE_KEY = 'tmpemail_address';

export function saveEmailAddress(address: string, expiresAt: string): void {
  const data: EmailAddressData = { address, expiresAt };
  localStorage.setItem(EMAIL_STORAGE_KEY, JSON.stringify(data));
}

export function getEmailAddress(): string | null {
  const stored = localStorage.getItem(EMAIL_STORAGE_KEY);
  if (!stored) return null;

  try {
    const data: EmailAddressData = JSON.parse(stored);
    const expiresAt = new Date(data.expiresAt);
    const now = new Date();

    if (now >= expiresAt) {
      clearEmailAddress();
      return null;
    }

    return data.address;
  } catch {
    clearEmailAddress();
    return null;
  }
}

export function getEmailExpiry(): string | null {
  const stored = localStorage.getItem(EMAIL_STORAGE_KEY);
  if (!stored) return null;

  try {
    const data: EmailAddressData = JSON.parse(stored);
    return data.expiresAt;
  } catch {
    return null;
  }
}

export function clearEmailAddress(): void {
  localStorage.removeItem(EMAIL_STORAGE_KEY);
}
