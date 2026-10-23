// Small per-user view preferences, kept outside the app reducer (like buffers
// and results) so flipping one re-renders only the components that read it.
// Stored in localStorage; every access is guarded, since a locked-down webview
// may refuse it, and the app has to work without.
import { useSyncExternalStore } from "react";
import type { Language } from "../i18n";

const WRAP_KEY = "go-xslt.wrapLines";

function readWrap(): boolean {
  try {
    return window.localStorage.getItem(WRAP_KEY) === "1";
  } catch {
    return false;
  }
}

// XSLT lines are long and wrapping them breaks the layout, so the default is
// to scroll sideways instead.
let wrapLines = readWrap();
const listeners = new Set<() => void>();

export function setWrapLines(on: boolean) {
  wrapLines = on;
  try {
    window.localStorage.setItem(WRAP_KEY, on ? "1" : "0");
  } catch {
    /* not persisted; the choice still holds for this session */
  }
  listeners.forEach((l) => l());
}

/** Whether the editors wrap long lines (default: no, they scroll sideways). */
export function useWrapLines(): boolean {
  return useSyncExternalStore(
    (cb) => {
      listeners.add(cb);
      return () => listeners.delete(cb);
    },
    () => wrapLines,
  );
}

// ---- update check ----

const UPDATES_KEY = "go-xslt.checkUpdates";

function readUpdates(): boolean {
  try {
    return window.localStorage.getItem(UPDATES_KEY) !== "0";
  } catch {
    return true;
  }
}

// On by default: the app asks GitHub at start whether a newer release exists.
// It is the app's only network request, and this switch turns it off.
let checkUpdates = readUpdates();
const updateListeners = new Set<() => void>();

export function setCheckUpdates(on: boolean) {
  checkUpdates = on;
  try {
    window.localStorage.setItem(UPDATES_KEY, on ? "1" : "0");
  } catch {
    /* not persisted; the choice still holds for this session */
  }
  updateListeners.forEach((l) => l());
}

/** Whether the app checks GitHub for a newer release at start. */
export function useCheckUpdates(): boolean {
  return useSyncExternalStore(
    (cb) => {
      updateListeners.add(cb);
      return () => updateListeners.delete(cb);
    },
    () => checkUpdates,
  );
}

// ---- theme ----

const THEME_KEY = "go-xslt.theme";

/** The theme setting: follow the operating system, or a fixed one. */
export type ThemeSetting = "system" | "light" | "dark";
/** The theme in effect. */
export type Theme = "light" | "dark";

function readTheme(): ThemeSetting {
  try {
    const v = window.localStorage.getItem(THEME_KEY);
    return v === "light" || v === "dark" ? v : "system";
  } catch {
    return "system";
  }
}

const systemDark = window.matchMedia?.("(prefers-color-scheme: dark)");

let themeSetting = readTheme();
let theme: Theme = resolveTheme();
const themeListeners = new Set<() => void>();

function resolveTheme(): Theme {
  if (themeSetting !== "system") return themeSetting;
  return systemDark && !systemDark.matches ? "light" : "dark";
}

// The theme in effect is a data attribute on <html>, which tokens.css
// redefines the colors for.
function applyTheme() {
  theme = resolveTheme();
  document.documentElement.dataset.theme = theme;
  themeListeners.forEach((l) => l());
}
applyTheme();
// With "system", the app follows a change of the OS appearance while it runs.
systemDark?.addEventListener("change", () => {
  if (themeSetting === "system") applyTheme();
});

export function setThemeSetting(t: ThemeSetting) {
  themeSetting = t;
  try {
    window.localStorage.setItem(THEME_KEY, t);
  } catch {
    /* not persisted; the choice still holds for this session */
  }
  applyTheme();
}

function subscribeTheme(cb: () => void) {
  themeListeners.add(cb);
  return () => themeListeners.delete(cb);
}

/** The theme setting: "system" (default), "light" or "dark". */
export function useThemeSetting(): ThemeSetting {
  return useSyncExternalStore(subscribeTheme, () => themeSetting);
}

/** The theme in effect, after resolving "system". */
export function useTheme(): Theme {
  return useSyncExternalStore(subscribeTheme, () => theme);
}

// ---- language ----

const LANGUAGE_KEY = "go-xslt.language";

/** The language setting: follow the operating system, or a fixed language. */
export type LanguageSetting = "system" | Language;

const SUPPORTED: Language[] = ["en", "de", "fr", "it"];

function readLanguage(): LanguageSetting {
  try {
    const v = window.localStorage.getItem(LANGUAGE_KEY);
    return v && (SUPPORTED as string[]).includes(v) ? (v as Language) : "system";
  } catch {
    return "system";
  }
}

// The first of the system's preferred languages the app offers; English
// when none is.
function systemLanguage(): Language {
  for (const tag of navigator.languages ?? [navigator.language]) {
    const base = tag.toLowerCase().split("-")[0] as Language;
    if (SUPPORTED.includes(base)) return base;
  }
  return "en";
}

let languageSetting = readLanguage();
let language: Language = languageSetting === "system" ? systemLanguage() : languageSetting;
const languageListeners = new Set<() => void>();
document.documentElement.lang = language;

export function setLanguageSetting(l: LanguageSetting) {
  languageSetting = l;
  language = l === "system" ? systemLanguage() : l;
  document.documentElement.lang = language;
  try {
    window.localStorage.setItem(LANGUAGE_KEY, l);
  } catch {
    /* not persisted; the choice still holds for this session */
  }
  languageListeners.forEach((fn) => fn());
}

function subscribeLanguage(cb: () => void) {
  languageListeners.add(cb);
  return () => languageListeners.delete(cb);
}

/** The language setting: "system" (default) or a language. */
export function useLanguageSetting(): LanguageSetting {
  return useSyncExternalStore(subscribeLanguage, () => languageSetting);
}

/** The language in effect, after resolving "system". */
export function useLanguage(): Language {
  return useSyncExternalStore(subscribeLanguage, () => language);
}

/** The language in effect, for code outside React. */
export function currentLanguage(): Language {
  return language;
}
