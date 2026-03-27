import { h } from 'preact';
import { useEffect } from 'preact/hooks';

export function ConfirmModal({ open, title, message, confirmText = 'Delete', cancelText = 'Cancel', onConfirm, onCancel }) {
  // Close on Escape key
  useEffect(() => {
    if (!open) return;
    function handleKey(e) {
      if (e.key === 'Escape') onCancel();
    }
    window.addEventListener('keydown', handleKey);
    return () => window.removeEventListener('keydown', handleKey);
  }, [open, onCancel]);

  if (!open) return null;

  return (
    <div
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm transition-opacity duration-150"
      onClick={onCancel}
    >
      <div
        class="bg-[var(--bg-surface)] border border-terminal-border rounded-lg p-6 max-w-[400px] w-[90vw] shadow-2xl transform transition-all duration-150 scale-100"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Title */}
        <h3 class="text-terminal-red font-mono text-lg mb-3">
          ⚠ {title}
        </h3>

        {/* Message */}
        <p class="text-terminal-gray text-sm mb-6 leading-relaxed">
          {message}
        </p>

        {/* Actions */}
        <div class="flex items-center justify-end gap-3">
          <button
            type="button"
            class="terminal-btn text-xs px-4 py-1.5"
            onClick={onCancel}
          >
            {cancelText}
          </button>
          <button
            type="button"
            class="terminal-btn terminal-btn-danger text-xs px-4 py-1.5"
            onClick={onConfirm}
          >
            {confirmText}
          </button>
        </div>
      </div>
    </div>
  );
}
