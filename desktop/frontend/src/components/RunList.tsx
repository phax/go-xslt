import { useMemo, useState } from "react";
import { RunKind, type SavedRun } from "../lib/api";
import { RUN_KIND_BADGE } from "../lib/labels";
import { useContextMenu } from "../state/overlays";
import { useProjectActions } from "../state/projectActions";
import { useT } from "../i18n";
import { shellMessages } from "../i18n/messages/shell";
import { useAppState } from "../state/store";
import type { ProjectState } from "../state/types";
import { Icon } from "./Icon";

export function RunList({ project }: { project: ProjectState }) {
  const state = useAppState();
  const actions = useProjectActions();
  const openMenu = useContextMenu();
  const t = useT(shellMessages);
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});

  const runs = project.manifest.runs ?? [];
  const groups = useMemo(() => {
    const map = new Map<string, SavedRun[]>();
    for (const r of runs) {
      const key = r.folder || "";
      if (!map.has(key)) map.set(key, []);
      map.get(key)!.push(r);
    }
    // Ungrouped runs (folder "") always render first, above any named group.
    return [...map.entries()].sort(([a], [b]) => (a === "" ? -1 : b === "" ? 1 : a.localeCompare(b)));
  }, [runs]);

  const activeTab = state.activeTabId ? state.tabs[state.activeTabId] : null;

  const newRunMenu = (e: { clientX: number; clientY: number; preventDefault: () => void }) =>
    openMenu(e, [
      { id: "t", label: t("newTransformRun"), onSelect: () => void actions.newRun(project.id, RunKind.RunTransform) },
      { id: "v", label: t("newValidateRun"), onSelect: () => void actions.newRun(project.id, RunKind.RunValidate) },
      { id: "x", label: t("newXpathRun"), onSelect: () => void actions.newRun(project.id, RunKind.RunXPath) },
    ]);

  return (
    <>
      <div className="sidebar-section-header">
        <span>{t("runs")}</span>
        <button className="icon-btn sidebar-add-btn" onClick={(e) => newRunMenu(e)} title={t("newRun")}>
          <Icon name="plus" />
        </button>
      </div>
      <div className="run-list">
        {runs.length === 0 && <div className="sidebar-empty">{t("noRuns")}</div>}
        {groups.map(([folder, items]) => (
          <div key={folder || "__root"}>
            {folder && (
              <div className="run-group-header" onClick={() => setCollapsed((c) => ({ ...c, [folder]: !c[folder] }))}>
                <Icon name="chevron-right" className={`run-group-caret${collapsed[folder] ? " collapsed" : ""}`} />
                <span>{folder}</span>
              </div>
            )}
            {(!folder || !collapsed[folder]) &&
              items.map((run) => (
                <div
                  key={run.id}
                  className={`run-item${activeTab?.kind === "run" && activeTab.projectId === project.id && activeTab.runId === run.id ? " active" : ""}`}
                  onClick={() => actions.openRun(project.id, run.id)}
                  onContextMenu={(e) =>
                    openMenu(e, [
                      { id: "rename", label: t("renameMenu"), onSelect: () => void actions.renameRun(project.id, run.id) },
                      { id: "duplicate", label: t("duplicate"), onSelect: () => void actions.duplicateRun(project.id, run.id) },
                      {
                        id: "delete",
                        label: t("deleteMenu"),
                        danger: true,
                        separatorBefore: true,
                        onSelect: () => void actions.confirmDeleteRun(project.id, run.id, run.name),
                      },
                    ])
                  }
                >
                  <span className={`run-item-icon kind-${run.kind}`}>{RUN_KIND_BADGE[run.kind]}</span>
                  <span className="run-item-name">{run.name}</span>
                </div>
              ))}
          </div>
        ))}
      </div>
    </>
  );
}
