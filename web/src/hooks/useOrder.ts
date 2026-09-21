import { useEffect, useState } from "react";
import { api } from "../api";
import type { Order } from "../types";

const POLL_MS = 5000;
const SETTLED = new Set(["completed", "failed"]);

/** Polls an order until it settles, then stops. */
export function useOrder(id: string) {
  const [order, setOrder] = useState<Order | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const controller = new AbortController();
    let timer = 0;
    let stopped = false;

    const tick = async () => {
      try {
        const next = await api.order(id, controller.signal);
        if (stopped) return;
        setOrder(next);
        setError(null);
        if (SETTLED.has(next.status)) return;
      } catch (err: unknown) {
        if (stopped || controller.signal.aborted) return;
        setError(err instanceof Error ? err.message : "Could not load this order.");
      }
      timer = window.setTimeout(tick, POLL_MS);
    };

    void tick();

    return () => {
      stopped = true;
      controller.abort();
      window.clearTimeout(timer);
    };
  }, [id]);

  return { order, error };
}
