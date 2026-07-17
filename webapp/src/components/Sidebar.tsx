import { useVault } from "../store";
import type { Nav, View } from "../types";
import { VaultSwitcher } from "./VaultSwitcher";
import { VaultTree, type TreeHandlers } from "./VaultTree";

// Sidebar is the left navigation zone: the vault switcher, the full
// Obsidian-style tree (groups → note titles), then lenses, inbox, settings,
// and the inert Suggestions seam.
export function Sidebar({
  view,
  setView,
  nav,
  handlers,
}: {
  view: View;
  setView: (v: View) => void;
  nav: Nav;
  handlers: TreeHandlers;
}) {
  const { state, selected, actions } = useVault();
  const activeLens = state.active.active_lens;

  return (
    <nav className="sidebar">
      <VaultSwitcher />

      <div className="sb-section grow">
        <div className="sb-heading">
          <button
            className={`sb-home ${view === "vault" && nav.kind === "group" && nav.path === "" ? "active" : ""}`}
            onClick={() => handlers.onNavGroup("")}
            title="Back to all memories"
          >
            All memories
          </button>
          <button className="link right" onClick={() => handlers.onNewSubgroup("")}>
            + group
          </button>
        </div>
        <VaultTree
          groups={state.groups}
          notes={state.notes}
          picked={selected}
          nav={view === "vault" ? nav : { kind: "group", path: "__none__" }}
          {...handlers}
        />
      </div>

      <div className="sb-section">
        <div className="sb-heading">
          Lenses
          <button className="link right" onClick={() => setView("lenses")}>
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
