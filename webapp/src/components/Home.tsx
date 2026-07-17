import { useEffect, useRef, useState } from "react";
import { useVault } from "../store";
import type { Nav, Note, View } from "../types";
import { Sidebar } from "./Sidebar";
import { GroupView } from "./GroupView";
import { NoteDetail } from "./NoteDetail";
import { ActiveTray } from "./ActiveTray";
import { InboxView } from "./InboxView";
import { LensManager } from "./LensManager";
import { SettingsView, type Theme } from "./SettingsView";
import { Suggestions } from "./Suggestions";
import { ItemEditor } from "./ItemEditor";
import { ConfirmDialog, PromptDialog } from "./ui";
import type { TreeHandlers } from "./VaultTree";

interface EditorState {
  note: Note | null;
  defaultGroup?: string;
}

interface PromptState {
  title: string;
  label: string;
  defaultValue?: string;
  placeholder?: string;
  confirmLabel?: string;
  options?: string[];
  validate?: (v: string) => string | null;
  onSubmit: (value: string) => void;
}

// Home is the three-zone shell: navigate (left tree) · browse & pick
// (drill-down center) · active memory (right). The tray shows on the vault
// browser; other views take the full center width.
export function Home({ theme, setTheme }: { theme: Theme; setTheme: (t: Theme) => void }) {
  const { state, actions } = useVault();
  const [view, setView] = useState<View>("vault");
  const [nav, setNav] = useState<Nav>({ kind: "group", path: "" });
  const [editor, setEditor] = useState<EditorState | null>(null);
  const [pendingDelete, setPendingDelete] = useState<Note | null>(null);
  const [pendingGroupDelete, setPendingGroupDelete] = useState<string | null>(null);
  const [prompt, setPrompt] = useState<PromptState | null>(null);

  // Reset navigation to the root when the vault changes underneath us.
  const vaultPath = state.vault_path;
  const prevVault = useRef(vaultPath);
  useEffect(() => {
    if (prevVault.current !== vaultPath) {
      prevVault.current = vaultPath;
      setNav({ kind: "group", path: "" });
      setView("vault");
    }
  }, [vaultPath]);

  const goGroup = (path: string) => {
    setNav({ kind: "group", path });
    setView("vault");
  };
  const goNote = (id: string) => {
    setNav({ kind: "note", id });
    setView("vault");
  };

  const newSubgroup = (parent: string) => {
    setPrompt({
      title: parent ? `New subgroup in ${parent}` : "New group",
      label: parent ? `Name of the subgroup under “${parent}”` : "Group name",
      placeholder: parent ? "e.g. income" : "e.g. finance/income",
      confirmLabel: "Create",
      onSubmit: (name) => {
        const path = parent ? `${parent}/${name}` : name;
        void actions.createGroup(path).then(() => goGroup(path));
      },
    });
  };
  const renameGroup = (path: string) => {
    setPrompt({
      title: "Rename group",
      label: `New name for “${path}”`,
      defaultValue: path,
      confirmLabel: "Rename",
      validate: (v) => (v === path ? "That's the same name." : null),
      onSubmit: (to) => {
        void actions.renameGroup(path, to).then(() => {
          if (nav.kind === "group" && (nav.path === path || nav.path.startsWith(path + "/"))) {
            goGroup(to + nav.path.slice(path.length));
          }
        });
      },
    });
  };
  const groupNoteCount = (path: string) =>
    state.notes.filter((n) => n.group === path || n.group.startsWith(path + "/")).length;

  const handlers: TreeHandlers = {
    onNavGroup: goGroup,
    onNavNote: goNote,
    onToggleSubtree: actions.toggleGroup,
    onNewSubgroup: newSubgroup,
    onAddNote: (g) => {
      setView("vault");
      setEditor({ note: null, defaultGroup: g });
    },
    onRenameGroup: renameGroup,
    onDeleteGroup: (p) => setPendingGroupDelete(p),
    onEditNote: (n) => setEditor({ note: n }),
    onDeleteNote: (n) => setPendingDelete(n),
  };

  const center = () => {
    switch (view) {
      case "vault":
        return nav.kind === "note" ? (
          <NoteDetail id={nav.id} onNavGroup={goGroup} onEdit={handlers.onEditNote} onDelete={handlers.onDeleteNote} />
        ) : (
          <GroupView path={nav.path} h={handlers} />
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
      <Sidebar view={view} setView={setView} nav={nav} handlers={handlers} />
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
              Delete <strong>{pendingDelete.label}</strong>? This removes the note file.
            </>
          }
          onConfirm={() => {
            const g = pendingDelete.group;
            const delId = pendingDelete.id;
            void actions.deleteNote(delId);
            if (nav.kind === "note" && nav.id === delId) goGroup(g);
          }}
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
              Delete <strong className="mono">{pendingGroupDelete}</strong> and its subgroups?{" "}
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
          onConfirm={() => {
            const p = pendingGroupDelete;
            void actions.deleteGroup(p);
            if (nav.kind === "group" && (nav.path === p || nav.path.startsWith(p + "/"))) {
              goGroup("");
            }
          }}
          onClose={() => setPendingGroupDelete(null)}
        />
      )}

      {prompt && (
        <PromptDialog
          title={prompt.title}
          label={prompt.label}
          defaultValue={prompt.defaultValue}
          placeholder={prompt.placeholder}
          confirmLabel={prompt.confirmLabel}
          options={prompt.options}
          validate={prompt.validate}
          onSubmit={prompt.onSubmit}
          onClose={() => setPrompt(null)}
        />
      )}
    </div>
  );
}
