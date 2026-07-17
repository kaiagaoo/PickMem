import { useState } from "react";
import { useVault } from "../store";
import { ComingSoonCard, ConfirmDialog, PromptDialog, TypeTag } from "./ui";

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
  const [newType, setNewType] = useState("");
  const [renamingType, setRenamingType] = useState<string | null>(null);

  const saveName = async () => {
    await actions.setVaultName(name.trim());
    setSavedName(true);
    setTimeout(() => setSavedName(false), 1500);
  };

  const addType = () => {
    const t = newType.trim().toLowerCase();
    if (!t || state.note_types.includes(t)) {
      setNewType("");
      return;
    }
    void actions.setNoteTypes([...state.note_types, t]);
    setNewType("");
  };
  const removeType = (t: string) =>
    void actions.setNoteTypes(state.note_types.filter((x) => x !== t));

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
        <h3>Note types</h3>
        <p className="muted">
          The kinds a memory can be, so you can pick “my ideas about X” apart
          from stable facts. Add your own, or click a type to rename it (notes
          using it update too). <code>fact</code> is the default and stays.
          Removing a type just hides it from the picker.
        </p>
        <div className="type-chips">
          {state.note_types.map((t) => (
            <span key={t} className="type-chip">
              {t === "fact" ? (
                <TypeTag type={t} />
              ) : (
                <button
                  className="type-rename"
                  title={`Rename “${t}”`}
                  onClick={() => setRenamingType(t)}
                >
                  <TypeTag type={t} />
                </button>
              )}
              {t !== "fact" && (
                <button
                  className="type-remove"
                  title={`Remove “${t}”`}
                  onClick={() => removeType(t)}
                >
                  ✕
                </button>
              )}
            </span>
          ))}
        </div>
        <div className="inline-add">
          <input
            value={newType}
            onChange={(e) => setNewType(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && addType()}
            placeholder="add a type… (e.g. task, question)"
          />
          <button className="ghost" onClick={addType} disabled={!newType.trim()}>
            Add
          </button>
        </div>
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

      {renamingType && (
        <PromptDialog
          title="Rename type"
          label={`New name for “${renamingType}”`}
          defaultValue={renamingType}
          confirmLabel="Rename"
          validate={(v) => {
            const t = v.trim().toLowerCase();
            if (t === renamingType) return "That's the same name.";
            if (state.note_types.includes(t)) return "That type already exists.";
            return null;
          }}
          onSubmit={(v) => void actions.renameNoteType(renamingType, v.trim().toLowerCase())}
          onClose={() => setRenamingType(null)}
        />
      )}
    </div>
  );
}
