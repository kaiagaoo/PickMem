import { useState } from "react";
import { api } from "../api";
import type { State } from "../types";
import { DOMAINS } from "../onboardingTemplates";

// The guided first-run flow: pick domains (→ groups), fill in as many
// suggested memories as you like (each with a guiding question + an example
// you can prefill), review, and create. It talks to the API directly (the
// vault context isn't mounted until onboarding finishes) and hands the final
// State back so the app drops into Home with a "Starter" lens active.

type Step = "welcome" | "domains" | "seed" | "review";

export function Onboarding({
  onComplete,
  onError,
}: {
  onComplete: (state: State) => void;
  onError: (msg: string) => void;
}) {
  const [step, setStep] = useState<Step>("welcome");
  const [chosen, setChosen] = useState<Set<string>>(new Set());
  const [answers, setAnswers] = useState<Record<string, string>>({});
  const [vaultName, setVaultName] = useState("");
  const [busy, setBusy] = useState(false);

  const chosenDomains = DOMAINS.filter((d) => chosen.has(d.key));
  const seededItems = chosenDomains
    .flatMap((d) => d.templates)
    .map((t) => ({ ...t, body: (answers[t.field] || "").trim() }))
    .filter((t) => t.body.length > 0);

  const toggleDomain = (key: string) => {
    setChosen((prev) => {
      const next = new Set(prev);
      next.has(key) ? next.delete(key) : next.add(key);
      return next;
    });
  };

  const setAnswer = (field: string, value: string) =>
    setAnswers((a) => ({ ...a, [field]: value }));

  // Fill every empty template in a domain with its example — the "give me a
  // thorough starter" shortcut. Existing answers are left untouched.
  const fillDomainExamples = (key: string) => {
    const d = DOMAINS.find((x) => x.key === key);
    if (!d) return;
    setAnswers((a) => {
      const next = { ...a };
      for (const t of d.templates) {
        if (t.example && !(next[t.field] || "").trim()) next[t.field] = t.example;
      }
      return next;
    });
  };

  const importExisting = async (file: File) => {
    setBusy(true);
    try {
      const parsed = JSON.parse(await file.text());
      const state = await api.importVault(parsed);
      onComplete(state);
    } catch (e) {
      onError("Import failed: " + (e instanceof Error ? e.message : String(e)));
      setBusy(false);
    }
  };

  const create = async () => {
    setBusy(true);
    try {
      if (vaultName.trim()) await api.setVaultName(vaultName.trim());

      // Ensure a folder for every group in the chosen domains, even ones the
      // user seeded no item for (empty-path guard → a scaffolded vault).
      const groups = new Set<string>();
      for (const d of chosenDomains) d.templates.forEach((t) => groups.add(t.group));
      for (const g of groups) await api.createGroup(g);

      // Add each seeded item, capturing its new id by diffing note ids.
      let latest: State | null = null;
      const seen = new Set<string>();
      (await api.getState()).notes.forEach((n) => seen.add(n.id));
      const seededIds: string[] = [];
      for (const item of seededItems) {
        latest = await api.addNote({
          label: item.label,
          group: item.group,
          body: item.body,
          tags: [],
        });
        for (const n of latest.notes) {
          if (!seen.has(n.id)) {
            seen.add(n.id);
            seededIds.push(n.id);
          }
        }
      }

      if (seededIds.length > 0) {
        await api.saveLens("Starter", seededIds);
        latest = await api.useLens("Starter");
      }
      onComplete(latest ?? (await api.getState()));
    } catch (e) {
      onError(e instanceof Error ? e.message : String(e));
      setBusy(false);
    }
  };

  return (
    <div className="onboard">
      <div className="onboard-card">
        <StepDots step={step} />

        {step === "welcome" && (
          <div className="ob-step">
            <div className="ob-hero">📌</div>
            <h1>You pick what your AI remembers.</h1>
            <p className="ob-lead">
              PickMem is a memory vault you control. You store small memories,
              file them into groups, and <strong>pick</strong> which ones an
              assistant sees — everything stays on your device.
            </p>
            <div className="ob-actions">
              <button className="primary big" onClick={() => setStep("domains")}>
                Build my vault
              </button>
              <label className="link-btn">
                I already have a vault
                <input
                  type="file"
                  accept="application/json"
                  style={{ display: "none" }}
                  onChange={(e) => {
                    const f = e.target.files?.[0];
                    if (f) void importExisting(f);
                  }}
                />
              </label>
            </div>
          </div>
        )}

        {step === "domains" && (
          <div className="ob-step">
            <h1>What should your memory help with?</h1>
            <p className="ob-lead">
              Pick the areas that fit. Each becomes a group with a few suggested
              memories to fill in — you can skip any of them.
            </p>
            <div className="chip-grid">
              {DOMAINS.map((d) => (
                <button
                  key={d.key}
                  className={`chip ${chosen.has(d.key) ? "on" : ""}`}
                  onClick={() => toggleDomain(d.key)}
                >
                  <span className="chip-label">{d.chip}</span>
                  <span className="chip-hint">
                    {d.hint ? `${d.hint} · ` : ""}
                    {d.templates.length} templates
                  </span>
                </button>
              ))}
            </div>
            <div className="ob-nav">
              <button className="ghost" onClick={() => setStep("welcome")}>
                Back
              </button>
              <button
                className="primary"
                onClick={() => setStep(chosen.size ? "seed" : "review")}
              >
                {chosen.size ? "Next" : "Skip for now"}
              </button>
            </div>
          </div>
        )}

        {step === "seed" && (
          <div className="ob-step">
            <h1>Fill in your memories</h1>
            <p className="ob-lead">
              Answer what fits and skip the rest. Tap <em>use example</em> to
              prefill, then edit. Each answer becomes a memory card.
            </p>
            <div className="seed-list">
              {chosenDomains.map((d) => {
                const filled = d.templates.filter((t) =>
                  (answers[t.field] || "").trim(),
                ).length;
                return (
                  <div className="seed-domain" key={d.key}>
                    <div className="seed-domain-head">
                      <span className="seed-domain-title">{d.chip}</span>
                      <span className="seed-domain-count">
                        {filled}/{d.templates.length}
                      </span>
                      <button
                        className="link right"
                        onClick={() => fillDomainExamples(d.key)}
                      >
                        fill all with examples
                      </button>
                    </div>
                    {d.templates.map((t) => (
                      <div className="seed-row" key={t.field}>
                        <div className="seed-row-head">
                          <span className="seed-row-label">{t.question}</span>
                          {t.example && (
                            <button
                              className="ex-btn"
                              onClick={() => setAnswer(t.field, t.example!)}
                            >
                              use example
                            </button>
                          )}
                        </div>
                        <input
                          value={answers[t.field] || ""}
                          onChange={(e) => setAnswer(t.field, e.target.value)}
                          placeholder={t.example ? `e.g. ${t.example}` : "optional"}
                        />
                      </div>
                    ))}
                  </div>
                );
              })}
            </div>
            <div className="ob-nav">
              <button className="ghost" onClick={() => setStep("domains")}>
                Back
              </button>
              <button className="primary" onClick={() => setStep("review")}>
                Next ({seededItems.length})
              </button>
            </div>
          </div>
        )}

        {step === "review" && (
          <div className="ob-step">
            <h1>Review & create</h1>
            {seededItems.length === 0 ? (
              <p className="ob-lead">
                No memories yet — that's fine. We'll set up your groups and you
                can add memories any time.
              </p>
            ) : (
              <p className="ob-lead">
                {seededItems.length} memor{seededItems.length === 1 ? "y" : "ies"}{" "}
                ready. They'll start picked, as a lens called <strong>Starter</strong>.
              </p>
            )}
            <div className="review-cards">
              {seededItems.map((it) => (
                <div className="memory-card preview" key={it.field}>
                  <span className="pick-toggle on">
                    <span className="dot" />
                  </span>
                  <div className="mc-main">
                    <div className="mc-top">
                      <span className="mc-label">{it.label}</span>
                    </div>
                    <div className="mc-body">{it.body}</div>
                    <div className="mc-path mono">{it.group}</div>
                  </div>
                </div>
              ))}
            </div>
            <label className="field">
              <span>Name your vault (optional)</span>
              <input
                value={vaultName}
                onChange={(e) => setVaultName(e.target.value)}
                placeholder="My memory"
              />
            </label>
            <div className="ob-nav">
              <button
                className="ghost"
                onClick={() => setStep(chosen.size ? "seed" : "domains")}
                disabled={busy}
              >
                Back
              </button>
              <button className="primary big" onClick={create} disabled={busy}>
                {busy ? "Creating…" : "Create vault"}
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

function StepDots({ step }: { step: Step }) {
  const order: Step[] = ["welcome", "domains", "seed", "review"];
  const idx = order.indexOf(step);
  return (
    <div className="step-dots">
      {order.map((s, i) => (
        <span key={s} className={`dot ${i <= idx ? "on" : ""}`} />
      ))}
    </div>
  );
}
