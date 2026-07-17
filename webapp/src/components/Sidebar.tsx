import { useVault } from "../store";
import type { View } from "../types";
import { GroupTree } from "./GroupTree";

// Sidebar is the left navigation zone: groups (filter the vault), lenses
// (activate), inbox (with count), settings, and the inert Suggestions seam
// where AI-extracted candidates will land later.
export function Sidebar({
  view,
  setView,
  groupFilter,
  setGroupFilter,
  onRenameGroup,
  onDeleteGroup,
  onAddInGroup,
}: {
  view: View;
  setView: (v: View) => void;
  groupFilter: string | null;
  setGroupFilter: (g: string | null) => void;
  onRenameGroup: (path: string) => void;
  onDeleteGroup: (path: string) => void;
  onAddInGroup: (path: string) => void;
}) {
  const { state, selected, actions } = useVault();
  const activeLens = state.active.active_lens;

  return (
    <nav className="sidebar">
      <div className="sb-brand">
        <span className="pin">📌</span>
        <span className="sb-name">{state.vault_name || "PickMem"}</span>
      </div>

      <div className="sb-section">
        <div className="sb-heading">Groups</div>
        <GroupTree
          groups={state.groups}
          notes={state.notes}
          selected={view === "vault" ? groupFilter : null}
          picked={selected}
          onSelect={(g) => {
            setGroupFilter(g);
            setView("vault");
          }}
          onToggleSubtree={actions.toggleGroup}
          onRename={onRenameGroup}
          onDelete={onDeleteGroup}
          onAddHere={onAddInGroup}
        />
      </div>

      <div className="sb-section">
        <div className="sb-heading">
          Lenses
          <button
            className="link right"
            onClick={() => setView("lenses")}
            title="Manage lenses"
          >
            manage
          </button>
        </div>
        {state.lenses.length === 0 ? (
          <div className="sb-muted">No lenses yet</div>
        ) : (
          state.lenses.map((l) => (
            <button
              key={l.name}
              className={`nav-item lens ${activeLens === l.name ? "active" : ""}`}
              onClick={() => actions.useLens(l.name)}
            >
              <span className="ni-label">{l.name}</span>
              <span className="ni-count">{l.count}</span>
            </button>
          ))
        )}
      </div>

      <div className="sb-section">
        <button
          className={`nav-item flat ${view === "inbox" ? "active" : ""}`}
          onClick={() => setView("inbox")}
        >
          <span className="ni-label">Inbox</span>
          <span className={`ni-count ${state.pending.length ? "badge" : ""}`}>
            {state.pending.length}
          </span>
        </button>
        <button
          className={`nav-item flat ${view === "settings" ? "active" : ""}`}
          onClick={() => setView("settings")}
        >
          <span className="ni-label">⚙ Settings</span>
        </button>
      </div>

      <div className="sb-section seam">
        <button
          className={`nav-item flat soon ${view === "suggestions" ? "active" : ""}`}
          onClick={() => setView("suggestions")}
        >
          <span className="ni-label">✨ Suggestions</span>
          <span className="ni-soon">soon</span>
        </button>
      </div>
    </nav>
  );
}
