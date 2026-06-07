import { cn } from "@/lib/utils";

const COLORS: Record<string, string> = {
  openai: "from-emerald-400 to-emerald-600",
  anthropic: "from-orange-400 to-orange-600",
  google: "from-blue-400 to-blue-600",
  mistral: "from-amber-400 to-amber-600",
  groq: "from-rose-400 to-rose-600",
  openrouter: "from-purple-400 to-purple-600",
  cohere: "from-pink-400 to-pink-600",
  xai: "from-zinc-500 to-zinc-700",
  together: "from-cyan-400 to-cyan-600",
  deepseek: "from-indigo-400 to-indigo-600",
  ollama: "from-slate-400 to-slate-600",
  perplexity: "from-teal-400 to-teal-600",
  fireworks: "from-red-400 to-red-600",
  azure: "from-sky-400 to-sky-600",
  bedrock: "from-yellow-500 to-yellow-700",
  vertex: "from-violet-400 to-violet-600",
};

export function ProviderIcon({
  provider,
  size = 32,
  className,
}: {
  provider: string;
  size?: number;
  className?: string;
}) {
  const c = COLORS[provider] ?? "from-gray-400 to-gray-600";
  const initial = provider.slice(0, 2).toUpperCase();
  return (
    <div
      className={cn(
        "rounded-lg bg-gradient-to-br flex items-center justify-center text-white font-bold flex-shrink-0",
        c,
        className,
      )}
      style={{ width: size, height: size, fontSize: size * 0.4 }}
      title={provider}
    >
      {initial}
    </div>
  );
}
