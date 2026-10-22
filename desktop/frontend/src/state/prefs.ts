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
