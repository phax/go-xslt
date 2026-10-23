// The app's user interface in English, German, French and Italian.
//
// Texts live in i18n/messages/<area>.ts, one file per area of the app, each
// declared with defineMessages: English is the reference, and TypeScript
// refuses to build when another language lacks a key (or has one English
// does not), so no translation can be forgotten. A component reads them with
// useT(messages); code outside React (toasts, dialogs started from actions)
// with translate(messages, …), which uses the language in effect.
//
// Placeholders are written {name} and filled from the params object.
import { currentLanguage, useLanguage } from "../state/prefs";

export type Language = "en" | "de" | "fr" | "it";

/** The languages the app offers, each named in its own language. */
export const LANGUAGES: { value: Language; label: string }[] = [
  { value: "en", label: "English" },
  { value: "de", label: "Deutsch" },
  { value: "fr", label: "Français" },
  { value: "it", label: "Italiano" },
];

type Catalog<K extends string> = Record<K, string>;

/** The texts of one area in all four languages; see the file comment. */
export type Messages<K extends string> = {
  en: Catalog<K>;
  de: Catalog<K>;
  fr: Catalog<K>;
  it: Catalog<K>;
};

export function defineMessages<E extends Record<string, string>>(m: {
  en: E;
  de: Record<keyof E, string>;
  fr: Record<keyof E, string>;
  it: Record<keyof E, string>;
}): Messages<Extract<keyof E, string>> {
  return m as Messages<Extract<keyof E, string>>;
}

export type Params = Record<string, string | number>;

function fill(text: string, params?: Params): string {
  if (!params) return text;
  return text.replace(/\{(\w+)\}/g, (whole, name: string) => (name in params ? String(params[name]) : whole));
}

/** The text for key in the language in effect, for code outside React. */
export function translate<K extends string>(messages: Messages<K>, key: K, params?: Params): string {
  return fill(messages[currentLanguage()][key] ?? messages.en[key], params);
}

/** A t(key, params) for one area, re-rendering when the language changes. */
export function useT<K extends string>(messages: Messages<K>): (key: K, params?: Params) => string {
  const lang = useLanguage();
  return (key, params) => fill(messages[lang][key] ?? messages.en[key], params);
}
