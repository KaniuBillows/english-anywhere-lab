export function getUserTimezone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone;
  } catch {
    return 'UTC';
  }
}

export function getUserLocale(): string {
  return navigator.language || 'en';
}

export function formatDate(dateStr: string): string {
  const d = new Date(dateStr);
  return d.toLocaleDateString(undefined, {
    weekday: 'short',
    month: 'short',
    day: 'numeric',
  });
}

export function formatShortDate(dateStr: string): string {
  const d = new Date(dateStr);
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
}

export function toISODateString(date: Date = new Date()): string {
  return date.toISOString().slice(0, 10);
}

/** Get Monday of the week containing the given date */
export function getWeekStart(date: Date = new Date()): string {
  const d = new Date(date);
  const day = d.getDay();
  const diff = d.getDate() - day + (day === 0 ? -6 : 1);
  d.setDate(diff);
  return toISODateString(d);
}

/** Get YYYY-MM for the given date */
export function getMonth(date: Date = new Date()): string {
  return toISODateString(date).slice(0, 7);
}

export function addDays(dateStr: string, days: number): string {
  const d = new Date(dateStr);
  d.setDate(d.getDate() + days);
  return toISODateString(d);
}

export function addMonths(monthStr: string, months: number): string {
  const d = new Date(monthStr + '-01');
  d.setMonth(d.getMonth() + months);
  return toISODateString(d).slice(0, 7);
}
