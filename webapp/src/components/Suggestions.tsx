import { ComingSoonCard } from "./ui";

// Suggestions is the inert AI seam — present in the nav so the feature has a
// home, but clearly not yet active. When extraction ships, this becomes the
// feed of candidate memories flowing into the inbox.
export function Suggestions() {
  return (
    <div className="center-pane">
      <h1 className="vv-title">✨ Suggestions</h1>
      <ComingSoonCard title="Memories, suggested — with you in control">
        <p>
          Soon, PickMem will notice durable facts worth remembering as you work
          with an assistant and surface them here as candidates.
        </p>
        <p>
          Every suggestion lands in your <strong>Inbox</strong> as a pending
          item. You accept, edit, or discard — nothing enters your vault or
          becomes active without your say-so. Same rule as everything else in
          PickMem: <em>you decide what's remembered.</em>
        </p>
      </ComingSoonCard>
    </div>
  );
}
