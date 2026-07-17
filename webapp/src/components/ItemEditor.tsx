import { useState } from "react";
import type { Note, NoteInput } from "../types";
import { Modal, TagChip } from "./ui";

// ItemEditor is the add/edit form. Group is free-text with a datalist of
// known paths (type a new `a/b` path to create a folder). New items route to
// active by default; a de-emphasized toggle sends to the inbox instead.
export function ItemEditor({
  note,
  groups,
  suggestedTags,
  defaultGroup,
  onSave,
  onClose,
}: {
  note: Note | null;
  groups: string[];
  suggestedTags: string[];
  defaultGroup?: string;
  onSave: (input: NoteInput) => Promise<void>;
  onClose: () => void;
}) {
  const [label, setLabel] = useState(note?.label ?? "");
  const [group, setGroup] = useState(note?.group ?? defaultGroup ?? "");
  const [tags, setTags] = useState((note?.tags ?? []).join(", "));
  const [body, setBody] = useState(note?.body ?? "");
  const [toInbox, setToInbox] = useState(false);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const tagList = tags
    .split(",")
    .map((t) => t.trim())
    .filter(Boolean);

  // Clicking a suggested chip adds or removes that tag, keeping the free-text
  // field (the source of truth) in sync.
  const toggleTag = (t: string) => {
    const next = tagList.includes(t)
      ? tagList.filter((x) => x !== t)
      : [...tagList, t];
    setTags(next.join(", "));
  };

  const submit = async () => {
    if (!label.trim() || !group.trim()) {
      setErr("Label and group are required.");
      return;
    }
    setBusy(true);
    setErr(null);
    try {
      await onSave({
        label: label.trim(),
        group: group.trim(),
        tags: tagList,
        body: body.trimEnd(),
        to_inbox: note ? false : toInbox,
      });
      onClose();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
      setBusy(false);
    }
  };

  return (
    <Modal
      title={note ? "Edit memory" : "Add memory"}
      onClose={onClose}
      footer={
        <>
          {!note && (
            <label className="inbox-opt" title="Stage for review instead of activating now">
              <input
                type="checkbox"
                checked={toInbox}
                onChange={(e) => setToInbox(e.target.checked)}
              />
              Send to inbox instead
            </label>
          )}
          <span className="right" />
          <button className="ghost" onClick={onClose} disabled={busy}>
            Cancel
          </button>
          <button className="primary" onClick={submit} disabled={busy}>
            {busy ? "Saving…" : note ? "Save changes" : "Add memory"}
          </button>
        </>
      }
    >
      {err && <div className="form-error">{err}</div>}
      <label className="field">
        <span>Label</span>
        <input
          autoFocus
          value={label}
          onChange={(e) => setLabel(e.target.value)}
          placeholder="short name, e.g. Response style"
        />
      </label>
      <label className="field">
        <span>Memory</span>
        <textarea
          value={body}
          onChange={(e) => setBody(e.target.value)}
          placeholder="in your own words — what the assistant will read"
        />
      </label>
      <label className="field">
        <span>Group</span>
        <input
          list="known-groups"
          value={group}
          onChange={(e) => setGroup(e.target.value)}
          placeholder="finance/income"
        />
        <datalist id="known-groups">
          {groups.map((g) => (
            <option key={g} value={g} />
          ))}
        </datalist>
      </label>
      <label className="field">
        <span>Tags (comma-separated)</span>
        <input
          value={tags}
          onChange={(e) => setTags(e.target.value)}
          placeholder="optional"
        />
      </label>
      {suggestedTags.length > 0 && (
        <div className="tag-suggest">
          {suggestedTags.map((t) => (
            <button
              type="button"
              key={t}
              className={`tag-suggest-chip ${tagList.includes(t) ? "on" : ""}`}
              onClick={() => toggleTag(t)}
              title={tagList.includes(t) ? "Remove tag" : "Add tag"}
            >
              <TagChip tag={t} />
            </button>
          ))}
        </div>
      )}
    </Modal>
  );
}
