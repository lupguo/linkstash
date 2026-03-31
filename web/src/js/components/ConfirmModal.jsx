import { h } from 'preact';
import { useEffect } from 'preact/hooks';

export function ConfirmModal({ open, title, message, confirmText = 'Delete', cancelText = 'Cancel', onConfirm, onCancel }) {
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
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-xs transition-opacity duration-150"
      onClick={onCancel}
    >
      <div
        class="bg-bg-surface border border-border-hi rounded-lg p-6 max-w-[400px] w-[90vw] shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <h3 class="text-red-400 font-semibold text-lg mb-3">
          {title}
        </h3>
        <p class="text-text-secondary text-sm mb-6 leading-relaxed">
          {message}
        </p>
        <div class="flex items-center justify-end gap-3">
          <button type="button" class="btn text-sm px-4 py-1.5" onClick={onCancel}>
            {cancelText}
          </button>
          <button type="button" class="btn btn-danger text-sm px-4 py-1.5" onClick={onConfirm}>
            {confirmText}
          </button>
        </div>
      </div>
    </div>
  );
}
