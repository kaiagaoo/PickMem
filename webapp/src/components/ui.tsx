import { useEffect, useRef, useState, type ReactNode } from "react";

export interface MenuItem {
  label: string;
  danger?: boolean;
  onClick: () => void;
}

// Menu is a ⋯ button that opens a small dropdown of actions. Closes on
// outside click or Escape. `trigger` lets callers style the button.
export function Menu({
  items,
  trigger = "⋯",
  className = "",
}: {
  items: MenuItem[];
  trigger?: ReactNode;
  className?: string;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!open) return;
    const onDoc = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && setOpen(false);
    document.addEventListener("mousedown", onDoc);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDoc);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);
  return (
    <div className={`menu-wrap ${className}`} ref={ref}>
      <button
        className="menu-btn"
        title="Actions"
        onClick={(e) => {
          e.stopPropagation();
          setOpen((o) => !o);
        }}
      >
        {trigger}
      </button>
      {open && (
        <div className="menu-pop" onClick={(e) => e.stopPropagation()}>
          {items.map((it, i) => (
            <button
              key={i}
              className={it.danger ? "danger" : ""}
              onClick={() => {
                setOpen(false);
                it.onClick();
              }}
            >
              {it.label}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

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

// PromptDialog is the in-app replacement for window.prompt: a single-field
// modal with autofocus, Enter-to-submit, optional validation, and an optional
// datalist of suggestions (used for picking an existing group).
export function PromptDialog({
  title,
  label,
  defaultValue = "",
  placeholder,
  confirmLabel = "Save",
  options,
  validate,
  onSubmit,
  onClose,
}: {
  title: string;
  label: string;
  defaultValue?: string;
  placeholder?: string;
  confirmLabel?: string;
  options?: string[];
  validate?: (v: string) => string | null;
  onSubmit: (value: string) => void;
  onClose: () => void;
}) {
  const [value, setValue] = useState(defaultValue);
  const [err, setErr] = useState<string | null>(null);
  const listId = "prompt-options";

  const submit = () => {
    const v = value.trim();
    if (!v) {
      setErr("This can't be empty.");
      return;
    }
    const e = validate?.(v);
    if (e) {
      setErr(e);
      return;
    }
    onSubmit(v);
    onClose();
  };

  return (
    <Modal
      title={title}
      onClose={onClose}
      footer={
        <>
          <button className="ghost" onClick={onClose}>
            Cancel
          </button>
          <button className="primary" onClick={submit}>
            {confirmLabel}
          </button>
        </>
      }
    >
      {err && <div className="form-error">{err}</div>}
      <label className="field">
        <span>{label}</span>
        <input
          autoFocus
          value={value}
          list={options ? listId : undefined}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              e.preventDefault();
              submit();
            }
          }}
          placeholder={placeholder}
        />
        {options && (
          <datalist id={listId}>
            {options.map((o) => (
              <option key={o} value={o} />
            ))}
          </datalist>
        )}
      </label>
    </Modal>
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
