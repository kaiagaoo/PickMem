// Persists the user's granted FileSystemDirectoryHandle across popup
// sessions using IndexedDB. Directory handles are `structuredClone`-able
// per the File System Access API, so IDB stores them natively — we
// don't have to reconstruct anything by path.
//
// Permission model: the user grants read-write once via
// showDirectoryPicker(). We re-verify on every popup open via
// queryPermission; if it degrades to "prompt", we re-request with a user
// gesture (the popup button click satisfies that requirement).

const DB_NAME = "pickmem";
const DB_VERSION = 1;
const STORE = "handles";
const HANDLE_KEY = "vault";

function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, DB_VERSION);
    req.onupgradeneeded = () => {
      const db = req.result;
      if (!db.objectStoreNames.contains(STORE)) {
        db.createObjectStore(STORE);
      }
    };
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  });
}

async function idbGet<T>(key: string): Promise<T | undefined> {
  const db = await openDB();
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE, "readonly");
    const req = tx.objectStore(STORE).get(key);
    req.onsuccess = () => resolve(req.result as T | undefined);
    req.onerror = () => reject(req.error);
  });
}

async function idbPut(key: string, val: unknown): Promise<void> {
  const db = await openDB();
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE, "readwrite");
    tx.objectStore(STORE).put(val, key);
    tx.oncomplete = () => resolve();
    tx.onerror = () => reject(tx.error);
  });
}

async function idbDelete(key: string): Promise<void> {
  const db = await openDB();
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE, "readwrite");
    tx.objectStore(STORE).delete(key);
    tx.oncomplete = () => resolve();
    tx.onerror = () => reject(tx.error);
  });
}

/** Returns the persisted vault handle, or undefined if none was saved
 * yet. Does NOT verify permission — callers should call ensurePermission
 * next. */
export async function loadVaultHandle(): Promise<FileSystemDirectoryHandle | undefined> {
  return idbGet<FileSystemDirectoryHandle>(HANDLE_KEY);
}

/** Prompts the user to grant a directory. Called only from a user-
 * gesture handler (button click). */
export async function grantVault(): Promise<FileSystemDirectoryHandle> {
  // showDirectoryPicker isn't in stock lib.dom.d.ts across every TS
  // version, so we grab it off window with a narrowed cast.
  const w = window as unknown as {
    showDirectoryPicker: (o?: {
      mode?: "read" | "readwrite";
      startIn?: string;
    }) => Promise<FileSystemDirectoryHandle>;
  };
  const handle = await w.showDirectoryPicker({
    mode: "readwrite",
    // A vault is a personal document tree — hint the picker to open the
    // user's Documents by default.
    startIn: "documents",
  });
  await idbPut(HANDLE_KEY, handle);
  return handle;
}

/** Verifies the popup has read-write permission on the stored handle.
 * If not, silently re-requests (requires a user gesture — the caller
 * must be inside a click handler). Returns true if permission is
 * granted; false if the user declined. */
export async function ensurePermission(
  handle: FileSystemDirectoryHandle
): Promise<boolean> {
  const opts = { mode: "readwrite" } as const;
  // Types for these methods aren't included in stock lib.dom.d.ts across
  // TS versions, so treat handle as `any` for the two method calls only.
  const h = handle as unknown as {
    queryPermission: (o: { mode: "readwrite" }) => Promise<PermissionState>;
    requestPermission: (o: { mode: "readwrite" }) => Promise<PermissionState>;
  };
  let state = await h.queryPermission(opts);
  if (state === "granted") return true;
  state = await h.requestPermission(opts);
  return state === "granted";
}

/** Forget the persisted handle. Used when the user picks a different
 * vault or explicitly disconnects. */
export async function clearVaultHandle(): Promise<void> {
  await idbDelete(HANDLE_KEY);
}
