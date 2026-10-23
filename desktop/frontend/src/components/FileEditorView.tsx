import { lazy, Suspense, useEffect, useRef, useState } from "react";
import { buffers, useBuffer } from "../state/buffers";
import { useProjectActions } from "../state/projectActions";
import { useAppState } from "../state/store";
import type { FileTab } from "../state/types";
import { Editor } from "./Editor";
import { dirname } from "../state/paths";
import { Icon } from "./Icon";
import { useT } from "../i18n";
import { viewsMessages } from "../i18n/messages/views";

// Only needed once someone switches an .xsd to the structure view.
const SchemaStructureView = lazy(() => import("./SchemaStructureView").then((m) => ({ default: m.SchemaStructureView })));

/** A plain project file opened for editing — any extension, not just the
 * ones a run references. Autosaves through the shared buffer store. */
export function FileEditorView({ tab }: { tab: FileTab }) {
  const state = useAppState();
  const actions = useProjectActions();
  const t = useT(viewsMessages);
  const project = state.projects[tab.projectId];
  const buf = useBuffer(tab.projectId, tab.path);
  const isXsd = /\.xsd$/i.test(tab.path);
  const [mode, setMode] = useState<"code" | "structure">("code");
  const [xsdVersion, setXsdVersion] = useState("1.1");

  useEffect(() => {
    void buffers.ensureLoaded(tab.projectId, tab.path);
  }, [tab.projectId, tab.path]);

  // A file recreated by editing a "missing" tab (autosave writes it back to
  // disk) still shows the missing banner until the tree catches up — nudge
  // one rescan right after that first successful save.
  const wasSaving = useRef(false);
  useEffect(() => {
    if (wasSaving.current && !buf.saving && !buf.error && tab.missing) void actions.rescan(tab.projectId);
    wasSaving.current = buf.saving;
  }, [buf.saving, buf.error, tab.missing, tab.projectId, actions]);

  if (!project) return null;

  const dirty = buf.version !== buf.savedVersion;

  return (
    <div className="pane">
      {tab.missing && (
        <div className="missing-banner">
          <Icon name="warning" />
          <span>{t("fileMissing")}</span>
        </div>
      )}
      <div className="pane-header">
        <span className="pane-title">{tab.path.split("/").pop()}</span>
        <span className="meta">{tab.path}</span>
        <span className="spacer" />
        {isXsd && (
          <>
            {mode === "structure" && (
              <select value={xsdVersion} onChange={(e) => setXsdVersion(e.target.value)} title={t("xsdVersionTitle")}>
                <option value="1.1">XSD 1.1</option>
                <option value="1.0">XSD 1.0</option>
              </select>
            )}
            <div className="seg">
              <button className={mode === "code" ? "on" : ""} onClick={() => setMode("code")}>{t("modeCode")}</button>
              <button className={mode === "structure" ? "on" : ""} onClick={() => setMode("structure")}>{t("modeStructure")}</button>
            </div>
          </>
        )}
        {buf.error && <span className="meta" style={{ color: "var(--error)" }}>{t("saveFailed", { error: buf.error })}</span>}
        {!buf.error && buf.saving && <span className="meta">{t("saving")}</span>}
        {!buf.error && !buf.saving && dirty && <span className="meta">{t("unsaved")}</span>}
        {!buf.error && !buf.saving && !dirty && <span className="meta">{t("saved")}</span>}
      </div>
      <div className="editor-wrap">
        {isXsd && mode === "structure" ? (
          <Suspense fallback={<div className="struct-empty">{t("loading")}</div>}>
            <SchemaStructureView
              schemas={[buf.text]}
              baseDir={tab.path.includes("/") ? `${project.dir}/${dirname(tab.path)}` : project.dir}
              projectDir={project.dir}
              version={xsdVersion}
            />
          </Suspense>
        ) : (
          <Editor value={buf.text} onChange={(v) => buffers.setText(tab.projectId, tab.path, v)} />
        )}
      </div>
    </div>
  );
}
