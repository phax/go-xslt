// Small per-user view preferences, kept outside the app reducer (like buffers
// and results) so flipping one re-renders only the components that read it.
// Stored in localStorage; every access is guarded, since a locked-down webview
// may refuse it, and the app has to work without.
import { useSyncExternalStore } from "react";

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
