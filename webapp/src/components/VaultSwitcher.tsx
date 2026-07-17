import { useEffect, useRef, useState } from "react";
import { useVault } from "../store";
import { pickFolder } from "../api";
import { Modal } from "./ui";

type Dialog = "open" | "create" | "import" | null;

// VaultSwitcher sits at the top of the sidebar: shows the current vault and
// drops down a list of recent vaults to switch between, plus actions to open
// an existing folder, create a new vault, or import a whole vault from a
// pickmem-vault.json. All vaults live locally on disk.
export function VaultSwitcher() {
  const { state, actions } = useVault();
  const [open, setOpen] = useState(false);
  const [dialog, setDialog] = useState<Dialog>(null);
  const ref = useRef<HTMLDivElement>(null);

  const current = state.vaults.find((v) => v.current);

  useEffect(() => {
    if (!open) return;
    const onDoc = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
  }, [open]);

  return (
    <div className="vault-switcher" ref={ref}>
      <button className="vs-current" onClick={() => setOpen((o) => !o)}>
        <span className="pin">📌</span>
        <span className="vs-name">{current?.name ?? state.vault_name ?? "PickMem"}</span>
        <span className="vs-caret">▾</span>
      </button>

      {open && (
        <div className="vs-pop">
          <div className="vs-section-label">Switch vault</div>
          {state.vaults.map((v) => (
            <div key={v.path} className={`vs-item ${v.current ? "current" : ""}`}>
              <button
                className="vs-item-main"
                disabled={!v.exists}
                onClick={() => {
                  setOpen(false);
                  if (!v.current) void actions.switchVault(v.path);
                }}
                title={v.path}
              >
                <span className="vs-item-name">{v.name}</span>
                <span className="vs-item-path mono">{v.path}</span>
                {!v.exists && <span className="vs-missing">missing</span>}
              </button>
              {!v.current && (
                <button
                  className="vs-forget"
                  title="Remove from list"
                  onClick={() => actions.forgetVault(v.path)}
                >
                  ✕
                </button>
              )}
            </div>
          ))}
          <div className="vs-divider" />
          <button className="vs-action" onClick={() => { setOpen(false); setDialog("open"); }}>
            Open a folder…
          </button>
          <button className="vs-action" onClick={() => { setOpen(false); setDialog("create"); }}>
            Create new vault…
          </button>
          <button className="vs-action" onClick={() => { setOpen(false); setDialog("import"); }}>
            Import a vault…
          </button>
        </div>
      )}

      {dialog && <VaultDialog kind={dialog} onClose={() => setDialog(null)} />}
    </div>
  );
}

function VaultDialog({ kind, onClose }: { kind: Exclude<Dialog, null>; onClose: () => void }) {
  const { actions } = useVault();
  const [path, setPath] = useState("");
  const [name, setName] = useState("");
  const [blob, setBlob] = useState<unknown | null>(null);
  const [blobName, setBlobName] = useState("");
  const [busy, setBusy] = useState(false);
  const [browsing, setBrowsing] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  // browse pops the OS folder chooser (server-side) and drops the chosen
  // absolute path into the field. A cancel yields "" and leaves it alone; a
  // failure (e.g. no picker on this OS) just surfaces as a hint — the text
  // field still works for typing a path by hand.
  const browse = async () => {
    setErr(null);
    setBrowsing(true);
    try {
      const picked = await pickFolder();
      if (picked) setPath(picked);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBrowsing(false);
    }
  };

  const titles = {
    open: "Open a folder as a vault",
    create: "Create a new vault",
    import: "Import a vault",
  };

  const readFile = async (f: File) => {
    try {
      const parsed = JSON.parse(await f.text());
      setBlob(parsed);
      setBlobName(f.name);
      if (!name && parsed?.vault_name) setName(parsed.vault_name);
    } catch (e) {
      setErr("Not a valid vault JSON: " + (e instanceof Error ? e.message : String(e)));
    }
  };

  const submit = async () => {
    if (!path.trim()) {
      setErr("Enter a folder path.");
      return;
    }
    if (kind === "import" && !blob) {
      setErr("Choose a pickmem-vault.json file to import.");
      return;
    }
    setBusy(true);
    setErr(null);
    try {
      if (kind === "open") await actions.switchVault(path.trim());
      else if (kind === "create") await actions.createVault(path.trim(), name.trim());
      else await actions.importVaultAsNew(path.trim(), name.trim(), blob);
      onClose();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
      setBusy(false);
    }
  };

  return (
    <Modal
      title={titles[kind]}
      onClose={onClose}
      footer={
        <>
          <button className="ghost" onClick={onClose} disabled={busy}>
            Cancel
          </button>
          <button className="primary" onClick={submit} disabled={busy}>
            {busy ? "Working…" : kind === "open" ? "Open" : kind === "create" ? "Create" : "Import"}
          </button>
        </>
      }
    >
      {err && <div className="form-error">{err}</div>}
      {kind === "import" && (
        <label className="field">
          <span>Vault file (pickmem-vault.json)</span>
          <input
            type="file"
            accept="application/json"
            onChange={(e) => {
              const f = e.target.files?.[0];
              if (f) void readFile(f);
            }}
          />
          {blobName && <div className="muted" style={{ marginTop: 4 }}>Loaded {blobName}</div>}
        </label>
      )}
      <label className="field">
        <span>
          {kind === "open" ? "Folder path" : "New folder path (created if missing)"}
        </span>
        <div className="path-row">
          <input
            autoFocus
            value={path}
            onChange={(e) => setPath(e.target.value)}
            placeholder="~/vaults/work"
          />
          <button
            type="button"
            className="ghost path-browse"
            onClick={browse}
            disabled={browsing || busy}
          >
            {browsing ? "Choosing…" : "Browse…"}
          </button>
        </div>
      </label>
      {kind !== "open" && (
        <label className="field">
          <span>Vault name (optional)</span>
          <input value={name} onChange={(e) => setName(e.target.value)} placeholder="Work memory" />
        </label>
      )}
      <p className="muted" style={{ fontSize: 13 }}>
        Vaults are plain folders on this machine. Paths are resolved on the
        computer running <code>pickmem web</code>.
      </p>
    </Modal>
  );
}
