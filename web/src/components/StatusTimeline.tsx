import type { SwapStatus } from "../types";

const STEPS = ["Awaiting deposit", "Confirming", "Exchanging", "Sending", "Done"] as const;

interface Stage {
  step: number;
  label: string;
  tone?: "ok" | "bad";
}

export const STAGES: Record<SwapStatus, Stage> = {
  awaiting_deposit: { step: 0, label: "Awaiting your deposit" },
  deposit_confirmed: { step: 1, label: "Deposit confirmed" },
  trade1_done: { step: 2, label: "Exchanging" },
  trade2_done: { step: 2, label: "Exchanging" },
  sending: { step: 3, label: "Sending to you" },
  completed: { step: 4, label: "Completed", tone: "ok" },
  failed: { step: -1, label: "Failed", tone: "bad" },
};

export function StatusTimeline({ status }: { status: SwapStatus }) {
  const stage = STAGES[status];
  const finished = status === "completed";

  return (
    <ol className="steps">
      {STEPS.map((label, index) => (
        <li
          key={label}
          className={[finished || stage.step > index ? "done" : "", !finished && stage.step === index ? "active" : ""]
            .filter(Boolean)
            .join(" ")}
        >
          <span className="dot" />
          {label}
        </li>
      ))}
    </ol>
  );
}
