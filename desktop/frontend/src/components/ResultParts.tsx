import type { Diagnostic, SecondaryOutput } from "../lib/api";
import { useT } from "../i18n";
import { runsMessages } from "../i18n/messages/runs";

/** Shared across all three run kinds: a run's diagnostics list, clickable to
 * jump to the offending line when the diagnostic targets an open editor. */
export function DiagnosticsList({ diagnostics, onClick }: { diagnostics: Diagnostic[]; onClick?: (d: Diagnostic) => void }) {
  const t = useT(runsMessages);
  if (diagnostics.length === 0) return null;
  const errors = diagnostics.filter((d) => d.severity === "error");
  const warnings = diagnostics.length - errors.length;
  return (
    <div className="diagnostics">
      <div className="diagnostics-header">
        {t("diagSummary", {
          errors: t(errors.length === 1 ? "errorsOne" : "errorsMany", { count: errors.length }),
          warnings: t(warnings === 1 ? "warningsOne" : "warningsMany", { count: warnings }),
        })}
      </div>
      <ul>
        {diagnostics.map((d, i) => (
          <li
            key={i}
            className={`diag diag-${d.severity}`}
            onClick={() => onClick?.(d)}
            title={d.line > 0 ? t("goToLine", { line: d.line }) : undefined}
          >
            <span className="diag-loc">{d.line > 0 ? `${d.line}:${d.col || 1}` : "—"}</span>
            <span className="diag-code">{d.code || d.phase}</span>
            <span className="diag-msg">{d.message}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

/** xsl:message output from a transform run. */
export function MessagesList({ messages }: { messages: string[] }) {
  const t = useT(runsMessages);
  if (messages.length === 0) return null;
  return (
    <div className="messages">
      <div className="messages-header">
        {t(messages.length === 1 ? "messagesOne" : "messagesMany", { count: messages.length })}
      </div>
      <ul>
        {messages.map((m, i) => (
          <li key={i} className="msg-item">
            {m}
          </li>
        ))}
      </ul>
    </div>
  );
}

/** xsl:result-document outputs from a transform run. */
export function SecondaryOutputsList({ outputs }: { outputs: SecondaryOutput[] }) {
  const t = useT(runsMessages);
  if (outputs.length === 0) return null;
  return (
    <div className="secondary">
      <div className="secondary-header">
        {t(outputs.length === 1 ? "resultDocsOne" : "resultDocsMany", { count: outputs.length })}
      </div>
      {outputs.map((s, i) => (
        <details key={i} className="secondary-doc">
          <summary>
            {s.href || t("resultN", { n: i + 1 })} · {s.method || "xml"}
          </summary>
          <pre className="output-text">{s.content}</pre>
        </details>
      ))}
    </div>
  );
}
