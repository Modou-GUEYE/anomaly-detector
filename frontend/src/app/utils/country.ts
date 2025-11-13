import * as countries from 'i18n-iso-countries';
import fr from 'i18n-iso-countries/langs/fr.json';

let localeRegistered = false;

/**
 * Register French locale once for i18n-iso-countries
 */
export function registerLocaleOnce() {
  if (!localeRegistered) {
    try {
      countries.registerLocale(fr as any);
      localeRegistered = true;
    } catch {
      // ignore if already registered
      localeRegistered = true;
    }
  }
}

/**
 * Convert ISO alpha-2 country code to localized name (fr), with ISO guard
 */
export function getCountryNameFromCode(code?: string): string {
  if (!code) return '';
  const cleaned = code.trim().toUpperCase();
  if (!/^[A-Z]{2}$/.test(cleaned)) return cleaned;
  registerLocaleOnce();
  try {
    const name = (countries as any).getName(cleaned, 'fr');
    return name || cleaned;
  } catch {
    return cleaned;
  }
}

/**
 * Get all country names in French as a sorted array
 */
export function getAllCountriesFR(): string[] {
  registerLocaleOnce();
  const names = (countries as any).getNames('fr') || {};
  return Object.values(names as Record<string, string>).sort((a, b) => a.localeCompare(b));
}
