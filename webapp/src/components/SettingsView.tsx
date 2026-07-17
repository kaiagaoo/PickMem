import { useState } from "react";
import { useVault } from "../store";
import { ComingSoonCard, ConfirmDialog } from "./ui";

export type Theme = "light" | "dark" | "system";

// SettingsView: vault name, theme, the destructive danger zone, and the inert
// AI card that reserves space for the future extraction feature. Vault
// import/switch lives in the top-left vault switcher, not here.
export function SettingsView({
  theme,
  setTheme,
}: {
  theme: Theme;
  setTheme: (t: Theme) => void;
}) {
  const { state, actions } = useVault();
  const [name, setName] = useState(state.vault_name);
  const [savedName, setSavedName] = useState(false);
  const [clearing, setClearing] = useState(false);

  const saveName = async () => {
    await actions.setVaultName(name.trim());
    setSavedName(true);
    setTimeout(() => setSavedName(false), 1500);
  };

  return (
    <div className="center-pane settings">
      <h1 className="vv-title">Settings</h1>

      <section className="setting-card">
        <h3>Vault</h3>
        <div className="setting-row">
          <label className="field grow">
            <span>Vault name</span>
            <input value={name} onChange={(e) => setName(e.target.value)} placeholder="My memory" />
          </label>
          <button className="primary" onClick={saveName}>
            {savedName ? "Saved ✓" : "Save"}
          </button>
        </div>
        <div className="setting-path mono">{state.vault_path}</div>
      </section>

      <section className="setting-card">
        <h3>Appearance</h3>
        <div className="theme-row">
          {(["system", "light", "dark"] as Theme[]).map((t) => (
            <button
              key={t}
              className={`theme-btn ${theme === t ? "on" : ""}`}
              onClick={() => setTheme(t)}
            >
              {t}
            </button>
          ))}
        </div>
      </section>

      <section className="setting-card">
        <h3>AI extraction</h3>
        <ComingSoonCard title="Let PickMem suggest memories">
          PickMem will read what you share with an assistant and propose small
          memories — you approve each one in the inbox before it enters your
          vault. Nothing automatic, nothing hidden.
        </ComingSoonCard>
      </section>

      <section className="setting-card danger">
        <h3>Danger zone</h3>
        <div className="setting-row">
          <div>
            <div className="dz-title">Clear vault</div>
            <div className="muted">
              Deletes every PickMem note (active and inbox) and all lenses. Your
              other Obsidian notes are untouched. This cannot be undone.
            </div>
          </div>
          <button className="danger-solid" onClick={() => setClearing(true)}>
            Clear vault
          </button>
        </div>
      </section>

      {clearing && (
        <ConfirmDialog
          title="Clear the entire vault?"
          danger
          confirmLabel="Delete everything"
          typeToConfirm="DELETE"
          message="This permanently deletes every PickMem note and lens in this vault. There is no undo."
          onConfirm={() => actions.clearVault()}
          onClose={() => setClearing(false)}
        />
      )}
    </div>
  );
}
