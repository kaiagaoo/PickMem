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

type PermissionHandle = FileSystemDirectoryHandle & {
  queryPermission: (o: { mode: "readwrite" }) => Promise<PermissionState>;
  requestPermission: (o: { mode: "readwrite" }) => Promise<PermissionState>;
};

/** Read-only check: does the popup currently have read-write permission
 * on the stored handle? Safe to call from DOMContentLoaded — queryPermission
 * does NOT require a user gesture, unlike requestPermission. */
export async function hasPermission(
  handle: FileSystemDirectoryHandle
): Promise<boolean> {
  const h = handle as PermissionHandle;
  const state = await h.queryPermission({ mode: "readwrite" });
  return state === "granted";
}

/** Re-requests permission on the stored handle. MUST be called from
 * inside a user-gesture handler (a click), or Chrome throws
 * SecurityError: "User activation is required to request permissions."
 * Returns true if the user granted it. */
export async function requestPermission(
  handle: FileSystemDirectoryHandle
): Promise<boolean> {
  const h = handle as PermissionHandle;
  const state = await h.requestPermission({ mode: "readwrite" });
  return state === "granted";
}

/** Forget the persisted handle. Used when the user picks a different
 * vault or explicitly disconnects. */
export async function clearVaultHandle(): Promise<void> {
  await idbDelete(HANDLE_KEY);
}
