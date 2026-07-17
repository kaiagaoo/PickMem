import { useState } from "react";
import type { Note, NoteInput } from "../types";
import { Modal, TypeTag } from "./ui";

// ItemEditor is the add/edit form with a live card preview, so the form
// visibly teaches the object model. Group is free-text with a datalist of
// known paths (type a new `a/b` path to create a folder). New items route to
// active by default; a de-emphasized toggle sends to the inbox instead.
export function ItemEditor({
  note,
  groups,
  noteTypes,
  defaultGroup,
  onSave,
  onClose,
}: {
  note: Note | null;
  groups: string[];
  noteTypes: string[];
  defaultGroup?: string;
  onSave: (input: NoteInput) => Promise<void>;
  onClose: () => void;
}) {
  const [label, setLabel] = useState(note?.label ?? "");
  const [group, setGroup] = useState(note?.group ?? defaultGroup ?? "");
  const [type, setType] = useState(note?.type ?? "fact");
  const [tags, setTags] = useState((note?.tags ?? []).join(", "));
  const [body, setBody] = useState(note?.body ?? "");
  const [toInbox, setToInbox] = useState(false);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const tagList = tags
    .split(",")
    .map((t) => t.trim())
    .filter(Boolean);

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
        type,
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
      wide
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
      <div className="editor-grid">
        <div>
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
          <div className="two-col">
            <label className="field">
              <span>Type</span>
              <select value={type} onChange={(e) => setType(e.target.value)}>
                {noteTypes.map((t) => (
                  <option key={t} value={t}>
                    {t}
                  </option>
                ))}
              </select>
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
          </div>
          <label className="field">
            <span>Tags (comma-separated)</span>
            <input
              value={tags}
              onChange={(e) => setTags(e.target.value)}
              placeholder="optional"
            />
          </label>
        </div>

        <div className="preview-col">
          <div className="preview-label">Preview</div>
          <div className="memory-card preview">
            <span className="pick-toggle" />
            <div className="mc-main">
              <div className="mc-top">
                <span className="mc-label">{label || "Label"}</span>
                <TypeTag type={type} />
              </div>
              <div className="mc-body">
                {body || "The memory text will appear here."}
              </div>
              {tagList.length > 0 && (
                <div className="mc-tags">
                  {tagList.map((t) => (
                    <span key={t} className="mc-tag">
                      #{t}
                    </span>
                  ))}
                </div>
              )}
            </div>
          </div>
          <div className="preview-path mono">{group || "no group"}</div>
        </div>
      </div>
    </Modal>
  );
}
