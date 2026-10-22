import { useEffect, useState } from "react";
import { Browser } from "@wailsio/runtime";
import { checkForUpdate, type UpdateInfo } from "../lib/api";
import { setCheckUpdates, useCheckUpdates } from "../state/prefs";

// Checked at every start and then every six hours while the app runs: a new
// release matters most in its first hours, so the answer is never cached
// across starts. That is far below GitHub's 60 unauthenticated requests per
// hour and address.
const RECHECK_MS = 6 * 60 * 60 * 1000;

// The key earlier versions cached the answer under; removed so it cannot go
// stale in the browser storage.
const LEGACY_KEY = "go-xslt.lastUpdateCheck";

/** Toolbar: the update-check switch, and a link to the new release when
 * GitHub has one newer than this build. */
export function UpdateNotice() {
  const enabled = useCheckUpdates();
  const [info, setInfo] = useState<UpdateInfo | null>(null);

  useEffect(() => {
    try {
      window.localStorage.removeItem(LEGACY_KEY);
    } catch {
      /* nothing stored, or storage unavailable */
    }
    if (!enabled) {
      setInfo(null);
      return;
    }
    let live = true;
    const run = async () => {
      const fresh = await checkForUpdate();
      // A failed check (offline, rate limit) keeps the last good answer.
      if (live && fresh && !fresh.error) setInfo(fresh);
    };
    void run();
    const timer = window.setInterval(() => void run(), RECHECK_MS);
    return () => {
      live = false;
      window.clearInterval(timer);
    };
  }, [enabled]);

  return (
    <>
      {info?.newer && (
        <button
          className="update-notice"
          title={`You have ${info.current}. Open the release page of ${info.latest}.`}
          onClick={() => void Browser.OpenURL(info.url)}
        >
          Update: v{info.latest}
        </button>
      )}
      <button
        className={`toggle-button${enabled ? " on" : ""}`}
        aria-pressed={enabled}
        title={
          enabled
            ? "Checks GitHub for a newer release at start and every six hours — click to turn off"
            : "No update check — click to check GitHub for a newer release"
        }
        onClick={() => setCheckUpdates(!enabled)}
      >
        Update check
      </button>
    </>
  );
}
