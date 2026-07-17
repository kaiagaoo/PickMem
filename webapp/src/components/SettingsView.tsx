import { useRef, useState } from "react";
import { useVault } from "../store";
import { ComingSoonCard, ConfirmDialog } from "./ui";

export type Theme = "light" | "dark" | "system";

// SettingsView: vault name, export/import (the portability story), theme,
// the destructive danger zone, and the inert AI card that reserves space for
// the future extraction feature.
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
  const [importMsg, setImportMsg] = useState<string | null>(null);
  const fileRef = useRef<HTMLInputElement>(null);

  const saveName = async () => {
    await actions.setVaultName(name.trim());
    setSavedName(true);
    setTimeout(() => setSavedName(false), 1500);
  };

  const doExport = async () => {
    const blob = await actions.exportVault();
    const url = URL.createObjectURL(
      new Blob([JSON.stringify(blob, null, 2)], { type: "application/json" }),
    );
    const a = document.createElement("a");
    a.href = url;
    a.download = "pickmem-vault.json";
    a.click();
    URL.revokeObjectURL(url);
  };

  const doImport = async (file: File) => {
    setImportMsg(null);
    try {
      const parsed = JSON.parse(await file.text());
      await actions.importVault(parsed);
      setImportMsg("Imported. New items were added to your vault.");
    } catch (e) {
      setImportMsg("Import failed: " + (e instanceof Error ? e.message : String(e)));
    }
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
        <h3>Backup & portability</h3>
        <p className="muted">
          Your vault is your data. Export it as a single JSON file, or import to
          add items from another export.
        </p>
        <div className="setting-actions">
          <button className="primary" onClick={doExport}>
            Export vault
          </button>
          <button className="ghost" onClick={() => fileRef.current?.click()}>
            Import…
          </button>
          <input
            ref={fileRef}
            type="file"
            accept="application/json"
            style={{ display: "none" }}
            onChange={(e) => {
              const f = e.target.files?.[0];
              if (f) void doImport(f);
              e.target.value = "";
            }}
          />
        </div>
        {importMsg && <div className="import-msg">{importMsg}</div>}
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
