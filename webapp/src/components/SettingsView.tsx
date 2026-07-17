import { useState } from "react";
import { useVault } from "../store";
import { ComingSoonCard, ConfirmDialog, TagChip } from "./ui";

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
  const [newTag, setNewTag] = useState("");

  const saveName = async () => {
    await actions.setVaultName(name.trim());
    setSavedName(true);
    setTimeout(() => setSavedName(false), 1500);
  };

  const addTag = () => {
    const t = newTag.trim().toLowerCase();
    if (!t || state.suggested_tags.includes(t)) {
      setNewTag("");
      return;
    }
    void actions.setSuggestedTags([...state.suggested_tags, t]);
    setNewTag("");
  };
  const removeTag = (t: string) =>
    void actions.setSuggestedTags(state.suggested_tags.filter((x) => x !== t));

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
        <h3>Suggested tags</h3>
        <p className="muted">
          Quick-pick chips offered when you tag a memory — a fast way to mark
          it an <code>idea</code>, a <code>thought</code>, a{" "}
          <code>reference</code>, or anything you like. They're ordinary tags,
          so you can still type any tag freely. Removing one just drops it from
          the quick-pick row; notes already tagged with it keep the tag.
        </p>
        <div className="type-chips">
          {state.suggested_tags.map((t) => (
            <span key={t} className="type-chip">
              <TagChip tag={t} />
              <button
                className="type-remove"
                title={`Remove “${t}”`}
                onClick={() => removeTag(t)}
              >
                ✕
              </button>
            </span>
          ))}
        </div>
        <div className="inline-add">
          <input
            value={newTag}
            onChange={(e) => setNewTag(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && addTag()}
            placeholder="add a suggested tag… (e.g. task, question)"
          />
          <button className="ghost" onClick={addTag} disabled={!newTag.trim()}>
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
    </div>
  );
}
