import { useState } from "react";
import {
  setCheckUpdates,
  setThemeSetting,
  useCheckUpdates,
  useThemeSetting,
  type ThemeSetting,
} from "../state/prefs";
import { Icon } from "./Icon";
import { Modal } from "./Modal";

const THEMES: { value: ThemeSetting; label: string }[] = [
  { value: "system", label: "System" },
  { value: "light", label: "Light" },
  { value: "dark", label: "Dark" },
];

/** Toolbar entry point to the settings dialog. */
export function SettingsButton() {
  const [open, setOpen] = useState(false);
  return (
    <>
      <button className="icon-btn settings-button" title="Settings" aria-label="Settings" onClick={() => setOpen(true)}>
        <Icon name="settings" size={16} />
      </button>
      {open && <SettingsModal onDismiss={() => setOpen(false)} />}
    </>
  );
}

function SettingsModal({ onDismiss }: { onDismiss: () => void }) {
  const theme = useThemeSetting();
  const checkUpdates = useCheckUpdates();
  return (
    <Modal onDismiss={onDismiss} className="settings-modal">
      <div className="licenses-modal-header">
        <span className="modal-title">Settings</span>
        <button className="licenses-close" onClick={onDismiss} aria-label="Close">
          ✕
        </button>
      </div>
      <div className="settings-row">
        <div className="settings-label">
          Appearance
          <span className="settings-hint">System follows the light or dark mode of your computer.</span>
        </div>
        <div className="seg" role="radiogroup" aria-label="Appearance">
          {THEMES.map((t) => (
            <button
              key={t.value}
              role="radio"
              aria-checked={theme === t.value}
              className={theme === t.value ? "on" : ""}
              onClick={() => setThemeSetting(t.value)}
            >
              {t.label}
            </button>
          ))}
        </div>
      </div>
      <div className="settings-row">
        <label className="settings-label" htmlFor="settings-updates">
          Check for updates
          <span className="settings-hint">
            Asks GitHub for a newer release at start and every six hours. It is the app's only network request.
          </span>
        </label>
        <input
          id="settings-updates"
          type="checkbox"
          checked={checkUpdates}
          onChange={(e) => setCheckUpdates(e.target.checked)}
        />
      </div>
    </Modal>
  );
}
