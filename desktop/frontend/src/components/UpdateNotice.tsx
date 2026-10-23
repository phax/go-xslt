import { useEffect, useMemo, useState, type MouseEvent } from "react";
import { Browser } from "@wailsio/runtime";
import { checkForUpdate, type UpdateInfo } from "../lib/api";
import { useCheckUpdates } from "../state/prefs";
import { Modal } from "./Modal";

// Checked at every start and then every six hours while the app runs: a new
// release matters most in its first hours, so the answer is never cached
// across starts. That is far below GitHub's 60 unauthenticated requests per
// hour and address.
const RECHECK_MS = 6 * 60 * 60 * 1000;

// The key earlier versions cached the answer under; removed so it cannot go
// stale in the browser storage.
const LEGACY_KEY = "go-xslt.lastUpdateCheck";

// The release the user chose to skip: its dialog no longer opens by itself
// (the toolbar button still shows it).
const SKIPPED_KEY = "go-xslt.skippedRelease";

function readSkipped(): string {
  try {
    return window.localStorage.getItem(SKIPPED_KEY) ?? "";
  } catch {
    return "";
  }
}

function writeSkipped(version: string) {
  try {
    window.localStorage.setItem(SKIPPED_KEY, version);
  } catch {
    /* not remembered; the dialog opens again at the next start */
  }
}

/** Toolbar: a button for a newer release, and the dialog with that release's
 * notes, which opens by itself once per start when GitHub has a release newer
 * than this build. The check itself is switched in the settings. */
export function UpdateNotice() {
  const enabled = useCheckUpdates();
  const [info, setInfo] = useState<UpdateInfo | null>(null);
  const [open, setOpen] = useState(false);

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
    let shown = false; // the dialog opens by itself at most once per start
    const run = async () => {
      const fresh = await checkForUpdate();
      // A failed check (offline, rate limit) keeps the last good answer.
      if (!live || !fresh || fresh.error) return;
      setInfo(fresh);
      if (fresh.newer && !shown && readSkipped() !== fresh.latest) {
        shown = true;
        setOpen(true);
      }
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
          title={`You have ${info.current}. Show what is new in ${info.latest}.`}
          onClick={() => setOpen(true)}
        >
          Update: v{info.latest}
        </button>
      )}
      {open && info?.newer && (
        <UpdateDialog
          info={info}
          onClose={() => setOpen(false)}
          onSkip={() => {
            writeSkipped(info.latest);
            setOpen(false);
          }}
        />
      )}
    </>
  );
}

function UpdateDialog({ info, onClose, onSkip }: { info: UpdateInfo; onClose: () => void; onSkip: () => void }) {
  const notes = useMemo(() => sanitizeNotes(info.notesHtml), [info.notesHtml]);
  const published = info.publishedAt ? new Date(info.publishedAt).toLocaleDateString() : "";

  // A link in the notes opens in the browser, never inside the app window.
  const onNotesClick = (e: MouseEvent<HTMLDivElement>) => {
    const a = (e.target as HTMLElement).closest("a");
    if (!a) return;
    e.preventDefault();
    const href = a.getAttribute("href") ?? "";
    if (/^https?:\/\//i.test(href)) void Browser.OpenURL(href);
  };

  return (
    <Modal onDismiss={onClose} className="update-modal">
      <div className="licenses-modal-header">
        <span className="modal-title">
          go-xslt {info.latest} is available
        </span>
        <button className="licenses-close" onClick={onClose} aria-label="Close">
          ✕
        </button>
      </div>
      <div className="update-meta">
        You have {info.current}
        {published ? ` · released ${published}` : ""}
      </div>
      {notes ? (
        <div className="update-notes" onClick={onNotesClick} dangerouslySetInnerHTML={{ __html: notes }} />
      ) : (
        <div className="update-notes update-notes-empty">This release has no notes.</div>
      )}
      <div className="modal-actions">
        <button onClick={onSkip} title="Do not open this dialog again until a newer release">
          Skip this version
        </button>
        <button onClick={onClose}>Later</button>
        <button className="primary" onClick={() => void Browser.OpenURL(info.url)}>
          Download
        </button>
      </div>
    </Modal>
  );
}

// The notes come from GitHub already rendered and sanitized; this is a second
// line of defence before they go into the page: only plain formatting
// elements survive, every attribute but a link's href (http/https only) and
// a few presentational ones is dropped, and images are removed — the app
// makes no requests besides the update check itself.
const ALLOWED = new Set([
  "a", "p", "br", "hr", "strong", "b", "em", "i", "code", "tt", "pre", "blockquote",
  "ul", "ol", "li", "h1", "h2", "h3", "h4", "h5", "h6",
  "table", "thead", "tbody", "tr", "th", "td", "del", "sup", "sub", "details", "summary", "span", "div",
]);
const KEEP_ATTRS = new Set(["href", "colspan", "rowspan", "align"]);

function sanitizeNotes(html: string | undefined): string {
  if (!html) return "";
  const doc = new DOMParser().parseFromString(`<div>${html}</div>`, "text/html");
  const root = doc.body.firstElementChild;
  if (!root) return "";
  const walk = (el: Element) => {
    for (const child of Array.from(el.children)) {
      const tag = child.tagName.toLowerCase();
      if (!ALLOWED.has(tag)) {
        // Unknown structure keeps its text; scripts, styles, media and frames
        // go entirely.
        if (["script", "style", "img", "picture", "video", "audio", "iframe", "object", "embed", "svg", "math", "form", "input", "link", "meta"].includes(tag)) {
          child.remove();
        } else {
          child.replaceWith(...Array.from(child.childNodes));
        }
        continue;
      }
      for (const attr of Array.from(child.attributes)) {
        const name = attr.name.toLowerCase();
        if (!KEEP_ATTRS.has(name) || (name === "href" && !/^https?:\/\//i.test(attr.value.trim()))) {
          child.removeAttribute(attr.name);
        }
      }
      walk(child);
    }
  };
  // Repeat until stable: unwrapping a disallowed element can expose children
  // the walk has not seen.
  let before = "";
  while (before !== root.innerHTML) {
    before = root.innerHTML;
    walk(root);
  }
  return root.innerHTML;
}
