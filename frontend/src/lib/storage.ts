const PREFIX = "ds:";

export function getItem<T>(key: string, fallback: T): T {
  try {
    const raw = localStorage.getItem(PREFIX + key);
    if (raw === null) return fallback;
    return JSON.parse(raw) as T;
  } catch {
    return fallback;
  }
}

export function setItem<T>(key: string, value: T): boolean {
  try {
    localStorage.setItem(PREFIX + key, JSON.stringify(value));
    return true;
  } catch (err) {
    // The most common failure here is QuotaExceededError when the origin
    // exceeds ~5MB. Callers receive `false` and decide whether to skip
    // the cache or surface a user-facing notice — they must not crash
    // the surrounding flow.
    console.warn("storage: setItem failed", key, err);
    return false;
  }
}

export function removeItem(key: string): void {
  localStorage.removeItem(PREFIX + key);
}
