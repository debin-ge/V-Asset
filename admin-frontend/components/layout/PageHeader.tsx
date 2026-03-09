import type { ReactNode } from "react";

export function PageHeader({
  eyebrow,
  title,
  description,
  actions,
}: {
  eyebrow?: string;
  title: string;
  description: string;
  actions?: ReactNode;
}) {
  return (
    <div className="relative overflow-hidden rounded-[32px] border border-white/50 bg-white/70 p-7 shadow-xl shadow-blue-950/5 backdrop-blur-xl xl:flex xl:items-end xl:justify-between">
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(59,130,246,0.18),transparent_28%),radial-gradient(circle_at_80%_20%,rgba(168,85,247,0.14),transparent_24%)]" />
      <div className="relative space-y-3">
        {eyebrow ? (
          <p className="text-xs font-semibold uppercase tracking-[0.24em] text-blue-600/80">{eyebrow}</p>
        ) : null}
        <div className="space-y-2">
          <h1 className="text-3xl font-semibold tracking-tight text-slate-950 md:text-4xl">{title}</h1>
          <p className="max-w-3xl text-sm leading-6 text-slate-600 md:text-base">{description}</p>
        </div>
      </div>
      {actions ? <div className="relative mt-5 flex flex-wrap items-center gap-2 xl:mt-0">{actions}</div> : null}
    </div>
  );
}
