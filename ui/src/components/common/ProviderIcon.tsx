import { cn } from "@/lib/utils";
import { useMemo, useState } from "react";

// Provider icon assets borrowed from 9Router under the MIT License.
// See /providers/LICENSE-9Router-icons.txt for attribution.
const PROVIDER_ICONS: Record<string, string> = {
  openai: "/providers/openai.png",
  anthropic: "/providers/anthropic.png",
  azure: "/providers/azure.png",
  bedrock: "/providers/aws-polly.png",
  cerebras: "/providers/cerebras.png",
  cohere: "/providers/cohere.png",
  deepseek: "/providers/deepseek.png",
  fireworks: "/providers/fireworks.png",
  gemini: "/providers/gemini.png",
  groq: "/providers/groq.png",
  huggingface: "/providers/huggingface.png",
  mistral: "/providers/mistral.png",
  nebius: "/providers/nebius.png",
  nvidia: "/providers/nvidia.png",
  ollama: "/providers/ollama.png",
  openrouter: "/providers/openrouter.png",
  perplexity: "/providers/perplexity.png",
  replicate: "/providers/huggingface.png",
  together: "/providers/together.png",
  vertex: "/providers/vertex.png",
  antigravity: "/providers/antigravity.png",
  "github-copilot": "/providers/copilot.png",
  cursor: "/providers/cursor.png",
  "gitlab-duo": "/providers/github.png",
  kimi: "/providers/kimi.png",
  kiro: "/providers/kiro.png",
  xai: "/providers/xai.png",
  xiaomi: "/providers/xiaomi-mimo.png",
  alibaba: "/providers/alicode.png",
  minimax: "/providers/minimax.png",
  zhipu: "/providers/glm.png",
  "cloudflare-ai-gateway": "/providers/cloudflare-ai.png",
  kagi: "/providers/brave-search.png",
  kilo: "/providers/kilocode.png",
  litellm: "/providers/openrouter.png",
  "lm-studio": "/providers/ollama.png",
  "ollama-cloud": "/providers/ollama.png",
  opencode: "/providers/opencode.png",
  qwen: "/providers/qwen.png",
  tavily: "/providers/tavily.png",
  vllm: "/providers/ollama.png",
};

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
  iconUrl,
  size = 32,
  className,
}: {
  provider: string;
  iconUrl?: string;
  size?: number;
  className?: string;
}) {
  const [error, setError] = useState(false);
  const src = useMemo(
    () => iconUrl || PROVIDER_ICONS[provider] || undefined,
    [iconUrl, provider],
  );

  if (src && !error) {
    return (
      <img
        src={src}
        alt={provider}
        title={provider}
        className={cn(
          "rounded-lg object-contain flex-shrink-0 bg-white",
          className,
        )}
        style={{ width: size, height: size }}
        onError={() => setError(true)}
      />
    );
  }

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
