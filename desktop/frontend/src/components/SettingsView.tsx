import { useState } from "react";
import { LANGUAGES, useT } from "../i18n";
import { settingsMessages } from "../i18n/messages/settings";
import {
  setCheckUpdates,
  setLanguageSetting,
  setThemeSetting,
  useCheckUpdates,
  useLanguageSetting,
  useThemeSetting,
  type LanguageSetting,
  type ThemeSetting,
} from "../state/prefs";
import { Icon } from "./Icon";
import { Modal } from "./Modal";

/** Toolbar entry point to the settings dialog. */
export function SettingsButton() {
  const t = useT(settingsMessages);
  const [open, setOpen] = useState(false);
  return (
    <>
      <button className="icon-btn settings-button" title={t("button")} aria-label={t("button")} onClick={() => setOpen(true)}>
        <Icon name="settings" size={16} />
      </button>
      {open && <SettingsModal onDismiss={() => setOpen(false)} />}
    </>
  );
}

function SettingsModal({ onDismiss }: { onDismiss: () => void }) {
  const t = useT(settingsMessages);
  const theme = useThemeSetting();
  const language = useLanguageSetting();
  const checkUpdates = useCheckUpdates();
  const themes: { value: ThemeSetting; label: string }[] = [
    { value: "system", label: t("themeSystem") },
    { value: "light", label: t("themeLight") },
    { value: "dark", label: t("themeDark") },
  ];
  return (
    <Modal onDismiss={onDismiss} className="settings-modal">
      <div className="licenses-modal-header">
        <span className="modal-title">{t("title")}</span>
        <button className="licenses-close" onClick={onDismiss} aria-label={t("close")}>
          ✕
        </button>
      </div>
      <div className="settings-row">
        <div className="settings-label">
          {t("appearance")}
          <span className="settings-hint">{t("appearanceHint")}</span>
        </div>
        <div className="seg" role="radiogroup" aria-label={t("appearance")}>
          {themes.map((o) => (
            <button
              key={o.value}
              role="radio"
              aria-checked={theme === o.value}
              className={theme === o.value ? "on" : ""}
              onClick={() => setThemeSetting(o.value)}
            >
              {o.label}
            </button>
          ))}
        </div>
      </div>
      <div className="settings-row">
        <label className="settings-label" htmlFor="settings-language">
          {t("language")}
          <span className="settings-hint">{t("languageHint")}</span>
        </label>
        <select
          id="settings-language"
          value={language}
          onChange={(e) => setLanguageSetting(e.target.value as LanguageSetting)}
        >
          <option value="system">{t("languageSystem")}</option>
          {LANGUAGES.map((l) => (
            <option key={l.value} value={l.value}>
              {l.label}
            </option>
          ))}
        </select>
      </div>
      <div className="settings-row">
        <label className="settings-label" htmlFor="settings-updates">
          {t("updates")}
          <span className="settings-hint">{t("updatesHint")}</span>
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
