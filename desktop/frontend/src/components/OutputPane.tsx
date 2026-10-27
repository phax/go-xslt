import { useMemo, useState } from "react";
import { ReadOnlyViewer } from "./Editor";
import { formatJson, formatXml } from "../lib/format";
import type { Diagnostic, TransformResult } from "../lib/api";
import { DiagnosticsList, MessagesList, SecondaryOutputsList } from "./ResultParts";
import { useT } from "../i18n";
import { runsMessages } from "../i18n/messages/runs";

interface OutputPaneProps {
  result: TransformResult | null;
  running: boolean;
  onDiagnosticClick?: (d: Diagnostic) => void;
}

export function OutputPane({ result, running, onDiagnosticClick }: OutputPaneProps) {
  const t = useT(runsMessages);
  const [tab, setTab] = useState<"output" | "rendered">("output");
  const [pretty, setPretty] = useState(true);
  const diags = result?.diagnostics ?? [];
  const method = result?.method || "xml";
  const canRender = method === "html";
  const isMarkup = method === "xml" || method === "html";
  const isJson = method === "json";
  const formattable = isMarkup || isJson;
  const messages = result?.messages ?? [];
  const secondary = result?.secondaryOutputs ?? [];

  const raw = result?.output ?? "";
  const shown = useMemo(
    () => {
      if (!pretty) return raw;
      if (isMarkup) return formatXml(raw);
      if (isJson) return formatJson(raw) ?? raw;
      return raw;
    },
    [raw, pretty, isMarkup, isJson],
  );

  return (
    <div className="pane output-pane">
      <div className="pane-header">
        <span className="pane-title">{t("paneOutput")}</span>
        <div className="tabs">
          <button className={tab === "output" ? "tab active" : "tab"} onClick={() => setTab("output")}>
            {t("tabSource")}
          </button>
          {canRender && (
            <button className={tab === "rendered" ? "tab active" : "tab"} onClick={() => setTab("rendered")}>
              {t("tabRendered")}
            </button>
          )}
        </div>
        {formattable && tab === "output" && (
          <label className="auto" title={t("formatTitle")}>
            <input type="checkbox" checked={pretty} onChange={(e) => setPretty(e.target.checked)} /> {t("format")}
          </label>
        )}
        <span className="spacer" />
        {result && (
          <span className="meta">
            {method} · {result.durationMs} ms
          </span>
        )}
      </div>

      <div className="output-body">
        {running && <div className="placeholder">{t("running")}</div>}
        {!running && tab === "output" && (
          <ReadOnlyViewer value={shown} language={isJson ? "json" : "xml"} />
        )}
        {!running && tab === "rendered" && canRender && (
          <iframe className="render-frame" title={t("renderedFrame")} sandbox="" srcDoc={raw} />
        )}
      </div>

      <MessagesList messages={messages} />
      <SecondaryOutputsList outputs={secondary} />
      <DiagnosticsList diagnostics={diags} onClick={onDiagnosticClick} />
    </div>
  );
}
