import type { ReactNode } from "react";

type PanelProps = {
  title: string;
  description?: string;
  children: ReactNode;
};

export function Panel({ title, description, children }: PanelProps) {
  return (
    <section className="rounded-md border border-zinc-200 bg-white">
      <div className="border-b border-zinc-200 px-5 py-4">
        <h3 className="text-base font-semibold text-zinc-950">{title}</h3>
        {description ? <p className="mt-1 text-sm leading-6 text-zinc-500">{description}</p> : null}
      </div>
      <div className="p-5">{children}</div>
    </section>
  );
}

type MetricCardProps = {
  label: string;
  value: string;
  detail: string;
  tone?: "sky" | "emerald" | "amber" | "rose" | "zinc";
};

const toneClasses = {
  amber: "bg-amber-500",
  emerald: "bg-emerald-500",
  rose: "bg-rose-500",
  sky: "bg-sky-500",
  zinc: "bg-zinc-400"
};

export function MetricCard({ label, value, detail, tone = "zinc" }: MetricCardProps) {
  return (
    <article className="rounded-md border border-zinc-200 bg-white p-5">
      <div className="flex items-center justify-between gap-3">
        <h3 className="text-sm font-semibold text-zinc-700">{label}</h3>
        <span className={`h-2.5 w-2.5 rounded-full ${toneClasses[tone]}`} />
      </div>
      <p className="mt-4 text-2xl font-semibold tracking-normal text-zinc-950">{value}</p>
      <p className="mt-2 text-sm leading-6 text-zinc-500">{detail}</p>
    </article>
  );
}

type StatusPillProps = {
  children: ReactNode;
  tone?: "neutral" | "good" | "warn" | "bad";
};

const pillClasses = {
  bad: "border-rose-200 bg-rose-50 text-rose-700",
  good: "border-emerald-200 bg-emerald-50 text-emerald-700",
  neutral: "border-zinc-200 bg-zinc-50 text-zinc-700",
  warn: "border-amber-200 bg-amber-50 text-amber-700"
};

export function StatusPill({ children, tone = "neutral" }: StatusPillProps) {
  return (
    <span className={`inline-flex rounded-md border px-2 py-1 text-xs font-semibold ${pillClasses[tone]}`}>
      {children}
    </span>
  );
}

type ProgressBarProps = {
  label: string;
  value: number;
};

export function ProgressBar({ label, value }: ProgressBarProps) {
  return (
    <div>
      <div className="mb-2 flex items-center justify-between gap-3 text-sm">
        <span className="font-medium text-zinc-700">{label}</span>
        <span className="font-semibold text-zinc-950">{value}%</span>
      </div>
      <div className="h-2 rounded-full bg-zinc-100">
        <div className="h-2 rounded-full bg-zinc-950" style={{ width: `${value}%` }} />
      </div>
    </div>
  );
}
