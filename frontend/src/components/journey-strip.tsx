import type { JourneyPhaseState } from "@/lib/journey";

export function JourneyStrip({
  phases,
  nextHint,
}: {
  phases: JourneyPhaseState[];
  nextHint?: string;
}) {
  return (
    <div>
      <ol className="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
        {phases.map((p) => {
          const tone =
            p.state === "done"
              ? "border-leaf/40 bg-leaf/5 text-ink"
              : p.state === "current"
                ? "border-leaf bg-white text-ink"
                : p.state === "locked"
                  ? "border-[color:var(--line)] bg-paper-2/50 text-muted"
                  : "border-[color:var(--line)] bg-white/40 text-muted";
          return (
            <li key={p.id} className={`rounded-md border px-3 py-3 ${tone}`}>
              <p className="text-[10px] font-semibold uppercase tracking-wide">
                {p.state === "done" ? "Done" : p.state === "current" ? "Now" : p.state === "locked" ? "Later" : "Next"}
              </p>
              <p className="mt-1 text-sm font-semibold">{p.title}</p>
              <p className="mt-1 text-xs leading-snug opacity-80">{p.summary}</p>
            </li>
          );
        })}
      </ol>
      {nextHint && <p className="mt-3 text-sm text-muted">{nextHint}</p>}
    </div>
  );
}
