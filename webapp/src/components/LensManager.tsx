import { useVault } from "../store";
import type { View } from "../types";
import { EmptyState } from "./ui";

// LensManager lists saved lenses with membership counts and per-lens actions:
// activate, rename, duplicate, delete. "Edit membership" activates the lens
// and jumps to the vault so the user edits the pick directly.
export function LensManager({ setView }: { setView: (v: View) => void }) {
  const { state, actions } = useVault();
  const activeLens = state.active.active_lens;

  const rename = async (name: string) => {
    const next = prompt(`Rename lens “${name}” to:`, name);
    if (!next || !next.trim() || next.trim() === name) return;
    const lens = state.lenses.find((l) => l.name === name);
    if (!lens) return;
    await actions.saveLens(next.trim(), lens.item_ids);
    await actions.deleteLens(name);
  };

  const duplicate = async (name: string) => {
    const lens = state.lenses.find((l) => l.name === name);
    if (!lens) return;
    await actions.saveLens(`${name} copy`, lens.item_ids);
  };

  return (
    <div className="center-pane">
      <h1 className="vv-title">Lenses</h1>
      <p className="pane-sub">
        A lens is a saved pick you can re-activate in one click.
      </p>
      {state.lenses.length === 0 ? (
        <EmptyState
          title="No lenses yet"
          hint="Pick some memories on the Home screen, then “Save pick as lens” to create your first one."
          action={
            <button className="primary" onClick={() => setView("vault")}>
              Go pick memories
            </button>
          }
        />
      ) : (
        <div className="lens-table">
          {state.lenses.map((l) => (
            <div className={`lens-manage-row ${activeLens === l.name ? "active" : ""}`} key={l.name}>
              <div className="lm-main">
                <span className="lm-name">{l.name}</span>
                {activeLens === l.name && <span className="lm-active">active</span>}
                <span className="lm-count">
                  {l.count} item{l.count === 1 ? "" : "s"}
                </span>
              </div>
              <div className="lm-actions">
                <button className="ghost" onClick={() => actions.useLens(l.name)}>
                  Activate
                </button>
                <button
                  className="ghost"
                  onClick={() => {
                    actions.useLens(l.name).then(() => setView("vault"));
                  }}
                >
                  Edit membership
                </button>
                <button className="ghost" onClick={() => rename(l.name)}>
                  Rename
                </button>
                <button className="ghost" onClick={() => duplicate(l.name)}>
                  Duplicate
                </button>
                <button className="danger" onClick={() => actions.deleteLens(l.name)}>
                  Delete
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
