import { createContext, useCallback, useContext, useEffect, useRef, type ReactNode } from "react";
import { DockviewReact, type DockviewApi, type DockviewReadyEvent, type IDockviewPanelProps } from "dockview-react";
import "dockview-react/dist/styles/dockview.css";
import { useTheme } from "../state/prefs";
import { useT } from "../i18n";
import { runsMessages } from "../i18n/messages/runs";

/** The panes of a run view, arranged the way the user likes: drag a pane's tab
 * to dock it left, right, above or below another one, or drop it on the same
 * tab bar to stack them as tabs — much like VS Code's editor groups. The
 * arrangement is remembered per view kind (`storageKey`) and "Reset layout"
 * restores `arrange`'s default. The panes themselves stay ordinary React
 * elements owned by the run view (they close over its state): `panes` hands
 * their current content to the dock through a context, on every render. */

export interface DockPane {
  id: string;
  title: string;
  content: ReactNode;
}

const PaneContents = createContext<Record<string, ReactNode>>({});

function PaneHost({ api }: IDockviewPanelProps) {
  const contents = useContext(PaneContents);
  return <>{contents[api.id]}</>;
}
const components = { pane: PaneHost };

const STORAGE_PREFIX = "go-xslt.dock.";

function readLayout(key: string): unknown | null {
  try {
    const raw = window.localStorage.getItem(STORAGE_PREFIX + key);
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
}

function writeLayout(key: string, layout: unknown) {
  try {
    window.localStorage.setItem(STORAGE_PREFIX + key, JSON.stringify(layout));
  } catch {
    /* not persisted; the arrangement still holds for this session */
  }
}

/** Gives the panels' groups these shares of the dock's width, left to right. */
export function setWidths(api: DockviewApi, ids: string[], shares: number[]) {
  const total = api.width;
  if (!total) return;
  const sum = shares.reduce((a, b) => a + b, 0);
  ids.forEach((id, i) => api.getPanel(id)?.group.api.setSize({ width: Math.round((total * shares[i]) / sum) }));
}

export function DockLayout({
  storageKey,
  panes,
  arrange,
}: {
  storageKey: string;
  panes: DockPane[];
  /** Builds the default arrangement on an empty dock (addPanel with ids from `panes`). */
  arrange: (api: DockviewApi) => void;
}) {
  const apiRef = useRef<DockviewApi | null>(null);
  const theme = useTheme();
  const t = useT(runsMessages);
  const saveTimer = useRef<number | undefined>(undefined);
  const titles = useRef(panes);
  titles.current = panes;

  const addDefault = useCallback(
    (api: DockviewApi) => {
      api.clear();
      arrange(api);
      // titles come from the panes, so the layout code only has to place ids
      for (const p of titles.current) api.getPanel(p.id)?.api.setTitle(p.title);
    },
    [arrange],
  );

  const onReady = useCallback(
    (event: DockviewReadyEvent) => {
      const api = event.api;
      apiRef.current = api;
      let restored = false;
      const saved = readLayout(storageKey);
      if (saved) {
        try {
          api.fromJSON(saved as never);
          // a stored layout is only good if it holds exactly the current panes
          const have = api.panels.map((p) => p.id).sort().join("|");
          const want = titles.current.map((p) => p.id).sort().join("|");
          restored = have === want;
        } catch {
          restored = false;
        }
      }
      if (!restored) addDefault(api);
      // a stored layout keeps the titles of its day (and language): show today's
      else for (const p of titles.current) api.getPanel(p.id)?.api.setTitle(p.title);
      api.onDidLayoutChange(() => {
        window.clearTimeout(saveTimer.current);
        saveTimer.current = window.setTimeout(() => writeLayout(storageKey, api.toJSON()), 300);
      });
    },
    [storageKey, addDefault],
  );

  useEffect(() => () => window.clearTimeout(saveTimer.current), []);

  // the titles follow the app's language
  const titleKey = panes.map((p) => p.title).join("|");
  useEffect(() => {
    const api = apiRef.current;
    if (!api) return;
    for (const p of titles.current) {
      const panel = api.getPanel(p.id);
      if (panel && panel.title !== p.title) panel.api.setTitle(p.title);
    }
  }, [titleKey]);

  const contents: Record<string, ReactNode> = {};
  for (const p of panes) contents[p.id] = p.content;

  return (
    <div className="dock-root">
      <div className="dock-bar">
        <button
          className="dock-reset"
          title={t("resetLayoutTitle")}
          onClick={() => apiRef.current && addDefault(apiRef.current)}
        >
          {t("resetLayout")}
        </button>
      </div>
      <div className="dock-body">
        <PaneContents.Provider value={contents}>
          <DockviewReact className={`dock-theme ${theme === "light" ? "dockview-theme-light" : "dockview-theme-dark"}`} components={components} onReady={onReady} disableFloatingGroups />
        </PaneContents.Provider>
      </div>
    </div>
  );
}
