import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { api } from "./api";
import type { NoteInput, State } from "./types";

interface Actions {
  addNote: (input: NoteInput) => Promise<void>;
  editNote: (id: string, input: NoteInput) => Promise<void>;
  deleteNote: (id: string) => Promise<void>;
  setActive: (ids: string[], lens?: string) => Promise<void>;
  toggleNote: (id: string) => void;
  toggleGroup: (ids: string[], select: boolean) => void;
  clearPick: () => Promise<void>;
  acceptInbox: (id: string, group: string) => Promise<void>;
  rejectInbox: (id: string) => Promise<void>;
  saveLens: (name: string, ids?: string[]) => Promise<void>;
  useLens: (name: string) => Promise<void>;
  deleteLens: (name: string) => Promise<void>;
  createGroup: (path: string) => Promise<void>;
  renameGroup: (from: string, to: string) => Promise<void>;
  deleteGroup: (path: string) => Promise<void>;
  setVaultName: (name: string) => Promise<void>;
  setSuggestedTags: (tags: string[]) => Promise<void>;
  importVault: (blob: unknown) => Promise<void>;
  clearVault: () => Promise<void>;
  switchVault: (path: string) => Promise<void>;
  createVault: (path: string, name: string) => Promise<void>;
  importVaultAsNew: (path: string, name: string, vault: unknown) => Promise<void>;
  forgetVault: (path: string) => Promise<void>;
}

interface Ctx {
  state: State;
  selected: Set<string>;
  actions: Actions;
}

const VaultCtx = createContext<Ctx | null>(null);

export function useVault(): Ctx {
  const ctx = useContext(VaultCtx);
  if (!ctx) throw new Error("useVault must be used within VaultProvider");
  return ctx;
}

export function VaultProvider({
  initial,
  onError,
  children,
}: {
  initial: State;
  onError: (msg: string) => void;
  children: ReactNode;
}) {
  const [state, setState] = useState<State>(initial);

  const run = useCallback(
    async (p: Promise<State>) => {
      try {
        setState(await p);
      } catch (e) {
        onError(e instanceof Error ? e.message : String(e));
        throw e;
      }
    },
    [onError],
  );

  const selected = useMemo(
    () => new Set(state.active.item_ids),
    [state.active.item_ids],
  );

  const setActive = useCallback(
    (ids: string[], lens = "") => run(api.setActive(ids, lens)),
    [run],
  );

  const actions = useMemo<Actions>(
    () => ({
      addNote: (input) => run(api.addNote(input)),
      editNote: (id, input) => run(api.editNote(id, input)),
      deleteNote: (id) => run(api.deleteNote(id)),
      setActive,
      toggleNote: (id) => {
        const next = new Set(state.active.item_ids);
        next.has(id) ? next.delete(id) : next.add(id);
        void setActive([...next]);
      },
      toggleGroup: (ids, select) => {
        const next = new Set(state.active.item_ids);
        for (const id of ids) select ? next.add(id) : next.delete(id);
        void setActive([...next]);
      },
      clearPick: () => setActive([]),
      acceptInbox: (id, group) => run(api.acceptInbox(id, group)),
      rejectInbox: (id) => run(api.rejectInbox(id)),
      saveLens: (name, ids) => run(api.saveLens(name, ids)),
      useLens: (name) => run(api.useLens(name)),
      deleteLens: (name) => run(api.deleteLens(name)),
      createGroup: (path) => run(api.createGroup(path)),
      renameGroup: (from, to) => run(api.renameGroup(from, to)),
      deleteGroup: (path) => run(api.deleteGroup(path)),
      setVaultName: (name) => run(api.setVaultName(name)),
      setSuggestedTags: (tags) => run(api.setSuggestedTags(tags)),
      importVault: (blob) => run(api.importVault(blob)),
      clearVault: () => run(api.clearVault()),
      switchVault: (path) => run(api.switchVault(path)),
      createVault: (path, name) => run(api.createVault(path, name)),
      importVaultAsNew: (path, name, v) => run(api.importVaultAsNew(path, name, v)),
      forgetVault: (path) => run(api.forgetVault(path)),
    }),
    [run, setActive, state.active.item_ids],
  );

  return (
    <VaultCtx.Provider value={{ state, selected, actions }}>
      {children}
    </VaultCtx.Provider>
  );
}
