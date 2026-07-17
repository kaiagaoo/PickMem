import { useState } from "react";
import { useVault } from "../store";
import type { Note } from "../types";
import { EmptyState } from "./ui";

// InboxView is the review surface for staged (pending) items. It's mostly
// empty in v1 — but it's the exact pipeline AI-suggested memories will flow
// into, so the empty state names that future.
export function InboxView({ onEdit }: { onEdit: (n: Note) => void }) {
  const { state } = useVault();
  if (state.pending.length === 0) {
    return (
      <div className="center-pane">
        <h1 className="vv-title">Inbox</h1>
        <EmptyState
          title="Nothing to review"
          hint="Later, memories suggested from your conversations will show up here for your approval before entering the vault. For now, you can send items here from the editor's “Send to inbox” option."
        />
      </div>
    );
  }
  return (
    <div className="center-pane">
      <h1 className="vv-title">
        Inbox <span className="muted">· {state.pending.length} to review</span>
      </h1>
      <div className="inbox-stack">
        {state.pending.map((n) => (
          <PendingCard key={n.id} note={n} onEdit={() => onEdit(n)} />
        ))}
      </div>
    </div>
  );
}

function PendingCard({ note, onEdit }: { note: Note; onEdit: () => void }) {
  const { state, actions } = useVault();
  const [group, setGroup] = useState(note.group || state.groups[0] || "");
  const [busy, setBusy] = useState(false);

  const run = async (fn: () => Promise<void>) => {
    setBusy(true);
    try {
      await fn();
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="pending-card">
      <div className="pc-top">
        <span className="pc-label">{note.label}</span>
      </div>
      {note.body && <div className="pc-body">{note.body}</div>}
      <div className="pc-meta mono">source: {note.source}</div>
      <div className="pc-actions">
        <span className="muted">file into</span>
        <select value={group} onChange={(e) => setGroup(e.target.value)}>
          {[group, ...state.groups.filter((g) => g !== group)]
            .filter(Boolean)
            .map((g) => (
              <option key={g} value={g}>
                {g}
              </option>
            ))}
        </select>
        <button
          className="primary"
          disabled={busy || !group}
          onClick={() => run(() => actions.acceptInbox(note.id, group))}
        >
          Accept
        </button>
        <button className="ghost" onClick={onEdit} disabled={busy}>
          Edit
        </button>
        <button
          className="danger right"
          disabled={busy}
          onClick={() => run(() => actions.rejectInbox(note.id))}
        >
          Discard
        </button>
      </div>
    </div>
  );
}
