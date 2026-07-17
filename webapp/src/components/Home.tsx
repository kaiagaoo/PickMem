import { useState } from "react";
import { useVault } from "../store";
import type { Note, View } from "../types";
import { Sidebar } from "./Sidebar";
import { VaultView } from "./VaultView";
import { ActiveTray } from "./ActiveTray";
import { InboxView } from "./InboxView";
import { LensManager } from "./LensManager";
import { SettingsView, type Theme } from "./SettingsView";
import { Suggestions } from "./Suggestions";
import { ItemEditor } from "./ItemEditor";
import { ConfirmDialog } from "./ui";

interface EditorState {
  note: Note | null;
  defaultGroup?: string;
}

// Home is the three-zone shell: navigate (left) · browse & pick (center) ·
// active memory (right). The tray shows only on the vault view; other views
// take the full center width.
export function Home({
  theme,
  setTheme,
}: {
  theme: Theme;
  setTheme: (t: Theme) => void;
}) {
  const { state, actions } = useVault();
  const [view, setView] = useState<View>("vault");
  const [groupFilter, setGroupFilter] = useState<string | null>(null);
  const [editor, setEditor] = useState<EditorState | null>(null);
  const [pendingDelete, setPendingDelete] = useState<Note | null>(null);
  const [pendingGroupDelete, setPendingGroupDelete] = useState<string | null>(null);

  const renameGroup = (path: string) => {
    const next = window.prompt(`Rename group “${path}” to:`, path);
    if (next && next.trim() && next.trim() !== path) {
      void actions.renameGroup(path, next.trim());
    }
  };
  const groupNoteCount = (path: string) =>
    state.notes.filter((n) => n.group === path || n.group.startsWith(path + "/")).length;

  const center = () => {
    switch (view) {
      case "vault":
        return (
          <VaultView
            groupFilter={groupFilter}
            onAdd={(g) => setEditor({ note: null, defaultGroup: g })}
            onEdit={(n) => setEditor({ note: n })}
            onDelete={(n) => setPendingDelete(n)}
          />
        );
      case "inbox":
        return <InboxView onEdit={(n) => setEditor({ note: n })} />;
      case "lenses":
        return <LensManager setView={setView} />;
      case "settings":
        return <SettingsView theme={theme} setTheme={setTheme} />;
      case "suggestions":
        return <Suggestions />;
    }
  };

  return (
    <div className={`home ${view === "vault" ? "with-tray" : ""}`}>
      <Sidebar
        view={view}
        setView={setView}
        groupFilter={groupFilter}
        setGroupFilter={setGroupFilter}
        onRenameGroup={renameGroup}
        onDeleteGroup={(p) => setPendingGroupDelete(p)}
        onAddInGroup={(g) => {
          setView("vault");
          setEditor({ note: null, defaultGroup: g });
        }}
      />
      <main className="home-main">{center()}</main>
      {view === "vault" && <ActiveTray />}

      {editor && (
        <ItemEditor
          note={editor.note}
          groups={state.groups}
          noteTypes={state.note_types}
          defaultGroup={editor.defaultGroup}
          onSave={(input) =>
            editor.note ? actions.editNote(editor.note.id, input) : actions.addNote(input)
          }
          onClose={() => setEditor(null)}
        />
      )}

      {pendingDelete && (
        <ConfirmDialog
          title="Delete memory?"
          danger
          confirmLabel="Delete"
          message={
            <>
              Delete <strong>{pendingDelete.label}</strong>? This removes the
              note file from your vault.
            </>
          }
          onConfirm={() => actions.deleteNote(pendingDelete.id)}
          onClose={() => setPendingDelete(null)}
        />
      )}

      {pendingGroupDelete && (
        <ConfirmDialog
          title="Delete group?"
          danger
          confirmLabel="Delete group"
          typeToConfirm={groupNoteCount(pendingGroupDelete) > 0 ? "DELETE" : undefined}
          message={
            <>
              Delete <strong className="mono">{pendingGroupDelete}</strong> and its
              subgroups?{" "}
              {groupNoteCount(pendingGroupDelete) > 0 ? (
                <>
                  This removes{" "}
                  <strong>
                    {groupNoteCount(pendingGroupDelete)} note
                    {groupNoteCount(pendingGroupDelete) === 1 ? "" : "s"}
                  </strong>{" "}
                  and the folder. This cannot be undone.
                </>
              ) : (
                "The empty folder will be removed."
              )}
            </>
          }
          onConfirm={() => actions.deleteGroup(pendingGroupDelete)}
          onClose={() => setPendingGroupDelete(null)}
        />
      )}
    </div>
  );
}
