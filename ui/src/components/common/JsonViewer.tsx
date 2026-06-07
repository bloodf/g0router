export function JsonViewer({ value }: { value: unknown }) {
  return (
    <pre className="bg-surface-2 border border-border rounded-lg p-4 text-xs overflow-auto custom-scrollbar font-mono max-h-[60vh]">
      {JSON.stringify(value, null, 2)}
    </pre>
  );
}
