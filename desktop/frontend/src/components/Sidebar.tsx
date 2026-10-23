import { useT } from "../i18n";
import { shellMessages } from "../i18n/messages/shell";
import { useAppDispatch, useAppState } from "../state/store";
import { ProjectSwitcher } from "./ProjectSwitcher";
import { FileTree } from "./FileTree";
import { RunList } from "./RunList";

export function Sidebar() {
  const state = useAppState();
  const dispatch = useAppDispatch();
  const t = useT(shellMessages);
  const project = state.activeProjectId ? state.projects[state.activeProjectId] : null;

  return (
    <div className="sidebar">
      <ProjectSwitcher />
      {project ? (
        <>
          <div className="sidebar-section-tabs">
            <button
              className={`sidebar-section-tab${state.ui.sidebarSection === "files" ? " active" : ""}`}
              onClick={() => dispatch({ type: "SIDEBAR_SECTION", section: "files" })}
            >
              {t("files")}
            </button>
            <button
              className={`sidebar-section-tab${state.ui.sidebarSection === "runs" ? " active" : ""}`}
              onClick={() => dispatch({ type: "SIDEBAR_SECTION", section: "runs" })}
            >
              {t("runs")}
            </button>
          </div>
          <div className="sidebar-section">
            {state.ui.sidebarSection === "files" ? <FileTree project={project} /> : <RunList project={project} />}
          </div>
        </>
      ) : (
        <div className="sidebar-empty">{t("sidebarNoProject")}</div>
      )}
    </div>
  );
}
