// Writes text to the clipboard. This is PickMem's low-permission
// fallback: it works on every site regardless of whether an adapter
// matches, and it doesn't need the File System Access grant either —
// clipboardWrite is declared in the manifest.
//
// Must be called from within a user-gesture handler (the popup button
// click). Chrome enforces that; a stray call from a timer will silently
// fail.

export async function copyToClipboard(text: string): Promise<void> {
  await navigator.clipboard.writeText(text);
}
