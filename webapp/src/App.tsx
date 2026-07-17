import { useCallback, useEffect, useState } from "react";
import { api } from "./api";
import type { State } from "./types";
import { VaultProvider } from "./store";
import { Home } from "./components/Home";
import { Onboarding } from "./components/Onboarding";
import type { Theme } from "./components/SettingsView";

const THEME_KEY = "pickmem-theme";
const onboardedKey = (vaultPath: string) => `pickmem-onboarded:${vaultPath}`;

function applyTheme(theme: Theme) {
  const root = document.documentElement;
  if (theme === "system") {
    root.removeAttribute("data-theme");
  } else {
    root.setAttribute("data-theme", theme);
  }
}

export function App() {
  const [state, setState] = useState<State | null>(null);
  const [loadErr, setLoadErr] = useState<string | null>(null);
  const [toast, setToast] = useState<string | null>(null);
  const [needsOnboarding, setNeedsOnboarding] = useState(false);
  const [theme, setThemeState] = useState<Theme>(
    () => (localStorage.getItem(THEME_KEY) as Theme) || "system",
  );

  const pushToast = useCallback((msg: string) => {
    setToast(msg);
    setTimeout(() => setToast(null), 4000);
  }, []);

  const setTheme = useCallback((t: Theme) => {
    setThemeState(t);
    localStorage.setItem(THEME_KEY, t);
    applyTheme(t);
  }, []);

  useEffect(() => applyTheme(theme), [theme]);

  useEffect(() => {
    api
      .getState()
      .then((s) => {
        setState(s);
        const onboarded = localStorage.getItem(onboardedKey(s.vault_path));
        // Guide first-run only for a genuinely empty vault; returning users
        // (including CLI-created vaults with notes) go straight to Home.
        if (!onboarded && s.notes.length === 0) setNeedsOnboarding(true);
      })
      .catch((e) => setLoadErr(String(e)));
  }, []);

  const finishOnboarding = useCallback((finalState: State) => {
    localStorage.setItem(onboardedKey(finalState.vault_path), "1");
    setState(finalState);
    setNeedsOnboarding(false);
  }, []);

  if (loadErr) {
    return (
      <div className="fullscreen-msg">
        <div className="fs-card error">
          <h2>Can't reach the PickMem server</h2>
          <p className="mono">{loadErr}</p>
          <p className="muted">Is <code>pickmem web</code> still running?</p>
        </div>
      </div>
    );
  }
  if (!state) {
    return (
      <div className="fullscreen-msg">
        <div className="loader">Loading vault…</div>
      </div>
    );
  }

  return (
    <>
      {needsOnboarding ? (
        <Onboarding onComplete={finishOnboarding} onError={pushToast} />
      ) : (
        <VaultProvider key={state.vault_path} initial={state} onError={pushToast}>
          <Home theme={theme} setTheme={setTheme} />
        </VaultProvider>
      )}
      {toast && <div className="toast">{toast}</div>}
    </>
  );
}
