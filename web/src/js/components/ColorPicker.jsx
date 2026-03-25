import { h } from 'preact';

const COLORS = [
  { name: '', label: 'Default', css: 'bg-gray-500' },
  { name: 'green', label: 'Green', css: 'bg-green-500' },
  { name: 'red', label: 'Red', css: 'bg-red-500' },
  { name: 'cyan', label: 'Cyan', css: 'bg-cyan-500' },
  { name: 'yellow', label: 'Yellow', css: 'bg-yellow-500' },
  { name: 'purple', label: 'Purple', css: 'bg-purple-500' },
  { name: 'orange', label: 'Orange', css: 'bg-orange-500' },
  { name: 'blue', label: 'Blue', css: 'bg-blue-500' },
];

export function ColorPicker({ value, onChange }) {
  return (
    <div class="flex items-center gap-2">
      {COLORS.map(color => (
        <button
          key={color.name}
          type="button"
          title={color.label}
          class={`w-6 h-6 rounded-full ${color.css} cursor-pointer transition-all ${value === color.name ? 'ring-2 ring-white scale-110' : 'ring-1 ring-terminal-border hover:ring-terminal-gray'}`}
          onClick={() => onChange(color.name)}
        />
      ))}
    </div>
  );
}
