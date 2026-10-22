import { setWrapLines, useWrapLines } from "../state/prefs";

/** Toolbar switch: wrap long lines in the editors and output, or scroll sideways. */
export function WrapToggle() {
  const wrap = useWrapLines();
  return (
    <button
      className={`toggle-button${wrap ? " on" : ""}`}
      aria-pressed={wrap}
      title={wrap ? "Long lines are wrapped — click to scroll sideways instead" : "Long lines scroll sideways — click to wrap them"}
      onClick={() => setWrapLines(!wrap)}
    >
      Wrap lines
    </button>
  );
}
