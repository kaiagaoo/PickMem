import { useMemo, useState } from "react";
import { useVault } from "../store";
import { buildContext, pickedNotes, type CopyFormat } from "../format";
import { PromptDialog, TokenMeter, TypeTag } from "./ui";

// ActiveTray is the payoff zone: the current pick, its size, and the actions
// that turn it into something you hand an assistant. The lens dropdown
// switches saved selections; when the pick has diverged from the active
// lens, an "unsaved changes" hint offers Update / Save-as-new.
export function ActiveTray() {
  const { state, actions } = useVault();
  const picked = pickedNotes(state);
  const activeLens = state.active.active_lens;

  const [fmt, setFmt] = useState<CopyFormat>("text");
  const [copied, setCopied] = useState(false);
  const [newLensName, setNewLensName] = useState("");
  const [saving, setSaving] = useState(false);
  const [savingAsNew, setSavingAsNew] = useState(false);

  // Has the pick diverged from the lens it claims to be?
  const dirty = useMemo(() => {
    if (!activeLens) return false;
    const lens = state.lenses.find((l) => l.name === activeLens);
    if (!lens) return false;
    const a = new Set(lens.item_ids);
    const b = state.active.item_ids;
    return a.size !== b.length || b.some((id) => !a.has(id));
  }, [activeLens, state.lenses, state.active.item_ids]);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(buildContext(state, fmt));
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      /* clipboard blocked; block is visible for manual copy */
    }
  };

  const saveNewLens = async () => {
    const name = newLensName.trim();
    if (!name) return;
    setSaving(true);
    try {
      await actions.saveLens(name, state.active.item_ids);
      await actions.useLens(name);
      setNewLensName("");
    } finally {
      setSaving(false);
    }
  };

  return (
    <aside className="tray">
      <div className="tray-head">
        <div className="tray-title">Active memory</div>
        <select
          className="lens-select"
          value={activeLens || "__adhoc__"}
          onChange={(e) => {
            const v = e.target.value;
            if (v === "__adhoc__") return;
            void actions.useLens(v);
          }}
        >
          <option value="__adhoc__">
            {activeLens ? activeLens : "Ad-hoc selection"}
          </option>
          {state.lenses
            .filter((l) => l.name !== activeLens)
            .map((l) => (
              <option key={l.name} value={l.name}>
                {l.name} ({l.count})
              </option>
            ))}
        </select>
      </div>

      {dirty && (
        <div className="dirty-hint">
          <span>Pick changed from “{activeLens}”.</span>
          <div className="dirty-actions">
            <button
              className="link"
              onClick={() => actions.saveLens(activeLens, state.active.item_ids)}
            >
              Update lens
            </button>
            <span className="sep">·</span>
            <button className="link" onClick={() => setSavingAsNew(true)}>
              Save as new
            </button>
          </div>
        </div>
      )}

      <div className="tray-list">
        {picked.length === 0 ? (
          <div className="tray-empty">
            Nothing picked yet. Toggle memories on the left to build the context
            you'll hand your assistant.
          </div>
        ) : (
          picked.map((n) => (
            <div className="tray-row" key={n.id}>
              <div className="tr-main">
                <span className="tr-label">{n.label}</span>
                <TypeTag type={n.type} />
              </div>
              <button
                className="tr-remove"
                title="Unpick"
                onClick={() => actions.toggleNote(n.id)}
              >
                ✕
              </button>
            </div>
          ))
        )}
      </div>

      <div className="tray-foot">
        <TokenMeter count={picked.length} tokens={state.tokens} />

        <div className="copy-block">
          <div className="fmt-row">
            {(["text", "markdown", "json"] as CopyFormat[]).map((f) => (
              <button
                key={f}
                className={`fmt-btn ${fmt === f ? "on" : ""}`}
                onClick={() => setFmt(f)}
              >
                {f}
              </button>
            ))}
          </div>
          <button className="primary block" onClick={copy} disabled={picked.length === 0}>
            {copied ? "Copied ✓" : "Copy context"}
          </button>
        </div>

        <div className="save-lens-row">
          <input
            value={newLensName}
            onChange={(e) => setNewLensName(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && saveNewLens()}
            placeholder="save pick as lens…"
          />
          <button
            className="ghost"
            onClick={saveNewLens}
            disabled={picked.length === 0 || !newLensName.trim() || saving}
          >
            Save
          </button>
        </div>

        <button
          className="ghost block"
          onClick={() => actions.clearPick()}
          disabled={picked.length === 0}
        >
          Clear pick
        </button>
      </div>

      {savingAsNew && (
        <PromptDialog
          title="Save as a new lens"
          label="Lens name"
          placeholder="e.g. Advice mode"
          confirmLabel="Save lens"
          onSubmit={(name) =>
            void actions.saveLens(name, state.active.item_ids).then(() => actions.useLens(name))
          }
          onClose={() => setSavingAsNew(false)}
        />
      )}
    </aside>
  );
}
