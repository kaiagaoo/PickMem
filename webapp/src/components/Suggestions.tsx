import { useState, type FormEvent } from "react";
import { api } from "../api";
import { useVault } from "../store";
import type { Suggestion } from "../types";
import { TagChip } from "./ui";

// Ranking is read-only. Each result requires an explicit user click before it
// joins Active Memory, keeping candidate generation separate from disclosure.
export function Suggestions() {
  const { selected, actions } = useVault();
  const [query, setQuery] = useState("");
  const [items, setItems] = useState<Suggestion[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [ran, setRan] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    if (!query.trim() || loading) return;
    setLoading(true);
    setError("");
    try {
      setItems(await api.suggestions(query.trim()));
      setRan(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="center-pane suggestions-view">
      <h1 className="vv-title">✨ Context suggestions</h1>
      <p className="pane-sub">
        Describe the task. Ranking runs locally and reveals its matched terms;
        nothing becomes active until you approve it.
      </p>
      <form className="suggest-form" onSubmit={submit}>
        <textarea
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="e.g. Help me prepare for a backend interview focused on Go"
          aria-label="Task description"
        />
        <button className="primary" disabled={!query.trim() || loading}>
          {loading ? "Ranking…" : "Suggest context"}
        </button>
      </form>
      {error && <p className="suggest-error">{error}</p>}
      {ran && items.length === 0 && (
        <div className="empty-state">
          <div className="es-title">No relevant memory found</div>
          <div className="es-hint">Try a more specific task description.</div>
        </div>
      )}
      <div className="suggest-list">
        {items.map((item, index) => {
          const picked = selected.has(item.id);
          return (
            <article className={`suggest-card ${picked ? "picked" : ""}`} key={item.id}>
              <div className="suggest-rank">{index + 1}</div>
              <div className="suggest-main">
                <div className="suggest-title-row">
                  <strong>{item.label}</strong>
                  <span className="mono muted">{item.group}</span>
                </div>
                <p>{item.body}</p>
                <div className="suggest-evidence">
                  matched {item.matched_terms.map((term) => `“${term}”`).join(", ")}
                  <span>score {item.score.toFixed(3)}</span>
                </div>
                {item.tags.length > 0 && (
                  <div className="mc-tags">
                    {item.tags.map((tag) => <TagChip key={tag} tag={tag} />)}
                  </div>
                )}
              </div>
              <button
                className={picked ? "ghost" : "primary"}
                onClick={() => actions.toggleNote(item.id)}
              >
                {picked ? "Remove" : "Approve"}
              </button>
            </article>
          );
        })}
      </div>
    </div>
  );
}
