import { useT } from "../i18n";
import { shellMessages } from "../i18n/messages/shell";
import { setWrapLines, useWrapLines } from "../state/prefs";

/** Toolbar switch: wrap long lines in the editors and output, or scroll sideways. */
export function WrapToggle() {
  const wrap = useWrapLines();
  const t = useT(shellMessages);
  return (
    <button
      className={`toggle-button${wrap ? " on" : ""}`}
      aria-pressed={wrap}
      title={wrap ? t("wrapOnTitle") : t("wrapOffTitle")}
      onClick={() => setWrapLines(!wrap)}
    >
      {t("wrapLines")}
    </button>
  );
}
