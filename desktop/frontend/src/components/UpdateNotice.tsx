import { useEffect, useState } from "react";
import { Browser } from "@wailsio/runtime";
import { appVersion, checkForUpdate, type UpdateInfo } from "../lib/api";
import { setCheckUpdates, useCheckUpdates } from "../state/prefs";

const LAST_KEY = "go-xslt.lastUpdateCheck";
// GitHub allows 60 unauthenticated API requests per hour and address; once a
// day is plenty to notice a release.
const CHECK_EVERY_MS = 24 * 60 * 60 * 1000;

type Cached = { at: number; info: UpdateInfo };

function readCached(): Cached | null {
  try {
    const raw = window.localStorage.getItem(LAST_KEY);
    return raw ? (JSON.parse(raw) as Cached) : null;
  } catch {
    return null;
  }
}

function writeCached(c: Cached) {
  try {
    window.localStorage.setItem(LAST_KEY, JSON.stringify(c));
  } catch {
    /* the next start simply checks again */
  }
}

/** Toolbar: the update-check switch, and a link to the new release when
 * GitHub has one newer than this build. */
export function UpdateNotice() {
  const enabled = useCheckUpdates();
  const [info, setInfo] = useState<UpdateInfo | null>(null);

  useEffect(() => {
    if (!enabled) {
      setInfo(null);
      return;
    }
    let live = true;
    const cached = readCached();
    // The cached answer is only good for the build that asked.
    if (cached && Date.now() - cached.at < CHECK_EVERY_MS) {
      void appVersion().then((v) => {
        if (!live) return;
        if (v === cached.info.current) {
          setInfo(cached.info);
        } else {
          void run();
        }
      });
    } else {
      void run();
    }
    async function run() {
      const fresh = await checkForUpdate();
      if (!live || !fresh) return;
      if (!fresh.error) writeCached({ at: Date.now(), info: fresh });
      setInfo(fresh);
    }
    return () => {
      live = false;
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
            ? "Checks GitHub once a day for a newer release — click to turn off"
            : "No update check — click to check GitHub for a newer release"
        }
        onClick={() => setCheckUpdates(!enabled)}
      >
        Update check
      </button>
    </>
  );
}
