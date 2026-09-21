import { useEffect, useState } from "react";
import { api } from "../api";
import type { Estimate, SwapDraft } from "../types";

interface QuoteState {
  estimate: Estimate | null;
  error: string | null;
  loading: boolean;
}

const IDLE: QuoteState = { estimate: null, error: null, loading: false };

/**
 * Debounced pricing. Each edit aborts the request in flight, so a slow reply
 * for an old amount can never overwrite a newer one.
 */
export function useQuote(draft: SwapDraft): QuoteState {
  const [state, setState] = useState<QuoteState>(IDLE);
  const { fromAsset, fromChain, toAsset, toChain, amount } = draft;

  useEffect(() => {
    if (!Number(amount)) {
      setState(IDLE);
      return;
    }

    const controller = new AbortController();
    setState((previous) => ({ ...previous, loading: true }));

    const timer = window.setTimeout(() => {
      api
        .quote({ fromAsset, fromChain, toAsset, toChain, amount, destAddress: "" }, controller.signal)
        .then((estimate) => setState({ estimate, error: null, loading: false }))
        .catch((err: unknown) => {
          if (controller.signal.aborted) return;
          setState({
            estimate: null,
            loading: false,
            error: err instanceof Error ? err.message : "Rate unavailable",
          });
        });
    }, 350);

    return () => {
      controller.abort();
      window.clearTimeout(timer);
    };
  }, [fromAsset, fromChain, toAsset, toChain, amount]);

  return state;
}
