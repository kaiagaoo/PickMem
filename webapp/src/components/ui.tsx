import { useEffect, useState, type ReactNode } from "react";

export function TypeTag({ type }: { type: string }) {
  return <span className={`type-tag t-${type}`}>{type}</span>;
}

// PickToggle is the accent moment of the app — on = "I chose this."
export function PickToggle({
  on,
  onClick,
  label,
}: {
  on: boolean;
  onClick: () => void;
  label?: string;
}) {
  return (
    <button
      className={`pick-toggle ${on ? "on" : ""}`}
      role="checkbox"
      aria-checked={on}
      aria-label={label ? `${on ? "Unpick" : "Pick"} ${label}` : "pick"}
      onClick={(e) => {
        e.stopPropagation();
        onClick();
      }}
    >
      <span className="dot" />
    </button>
  );
}

export function TokenMeter({ count, tokens }: { count: number; tokens: number }) {
  return (
    <div className="token-meter">
      <span className="big">{count}</span>{" "}
      <span className="muted">
        item{count === 1 ? "" : "s"} · ~{tokens} tokens
      </span>
    </div>
  );
}

export function EmptyState({
  title,
  hint,
  action,
}: {
  title: string;
  hint?: string;
  action?: ReactNode;
}) {
  return (
    <div className="empty-state">
      <div className="es-title">{title}</div>
      {hint && <div className="es-hint">{hint}</div>}
      {action && <div className="es-action">{action}</div>}
    </div>
  );
}

export function ComingSoonCard({
  title,
  children,
}: {
  title: string;
  children: ReactNode;
}) {
  return (
    <div className="coming-soon">
      <div className="cs-head">
        <span className="spark">✨</span>
        <span className="cs-title">{title}</span>
        <span className="cs-pill">coming soon</span>
      </div>
      <div className="cs-body">{children}</div>
    </div>
  );
}

// Modal is a lightweight centered overlay used by the editor and dialogs.
export function Modal({
  title,
  onClose,
  children,
  footer,
  wide,
}: {
  title: string;
  onClose: () => void;
  children: ReactNode;
  footer?: ReactNode;
  wide?: boolean;
}) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);
  return (
    <div className="modal-backdrop" onMouseDown={onClose}>
      <div
        className={`modal ${wide ? "wide" : ""}`}
        onMouseDown={(e) => e.stopPropagation()}
      >
        <div className="modal-head">
          <h2>{title}</h2>
          <button className="ghost" onClick={onClose} aria-label="Close">
            ✕
          </button>
        </div>
        <div className="modal-body">{children}</div>
        {footer && <div className="modal-foot">{footer}</div>}
      </div>
    </div>
  );
}

// ConfirmDialog gates destructive actions. When `typeToConfirm` is set, the
// confirm button stays disabled until the user types that exact string.
export function ConfirmDialog({
  title,
  message,
  confirmLabel = "Confirm",
  danger,
  typeToConfirm,
  onConfirm,
  onClose,
}: {
  title: string;
  message: ReactNode;
  confirmLabel?: string;
  danger?: boolean;
  typeToConfirm?: string;
  onConfirm: () => void;
  onClose: () => void;
}) {
  const [typed, setTyped] = useState("");
  const ready = !typeToConfirm || typed === typeToConfirm;
  return (
    <Modal
      title={title}
      onClose={onClose}
      footer={
        <>
          <button className="ghost" onClick={onClose}>
            Cancel
          </button>
          <button
            className={danger ? "danger-solid" : "primary"}
            disabled={!ready}
            onClick={() => {
              onConfirm();
              onClose();
            }}
          >
            {confirmLabel}
          </button>
        </>
      }
    >
      <div className="confirm-msg">{message}</div>
      {typeToConfirm && (
        <label className="field" style={{ marginTop: 12 }}>
          <span>
            Type <code>{typeToConfirm}</code> to confirm
          </span>
          <input value={typed} onChange={(e) => setTyped(e.target.value)} autoFocus />
        </label>
      )}
    </Modal>
  );
}
