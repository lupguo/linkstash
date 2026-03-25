import { h } from 'preact';

export function ScoreFilter({ value, onChange }) {
  return (
    <div class="flex items-center gap-3">
      <label class="text-terminal-gray text-sm whitespace-nowrap">Min Score:</label>
      <input
        type="range"
        min="0"
        max="1"
        step="0.05"
        value={value}
        onInput={(e) => onChange(parseFloat(e.target.value))}
        class="flex-1 accent-green-500"
      />
      <span class="text-terminal-green text-sm w-10 text-right">{value.toFixed(2)}</span>
    </div>
  );
}
