// Minimal service worker. MV3 requires one, but PickMem's popup does all
// the vault I/O and messaging directly, so this file exists mainly to
// register the extension with Chrome.
//
// If future features (context menu items, alarms, etc.) need a
// persistent background, they'd land here.

chrome.runtime.onInstalled.addListener(() => {
  // No-op for now. Reserved for future upgrade migrations.
});

export {};
