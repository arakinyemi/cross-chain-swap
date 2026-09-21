import { useState } from "react";
import { AssetPicker } from "./AssetPicker";
import { api } from "../api";
import { formatAmount, networkLabel } from "../format";
import { useQuote } from "../hooks/useQuote";
import type { Catalog, SwapDraft } from "../types";

interface Props {
  catalog: Catalog;
  onCreated: (id: string) => void;
}

export function ExchangeForm({ catalog, onCreated }: Props) {
  const [draft, setDraft] = useState<SwapDraft>(() => ({
    fromAsset: "USDT",
    fromChain: "tron",
    toAsset: "USDC",
    toChain: "solana",
    amount: String(Math.max(100, catalog.min_amount)),
    destAddress: "",
  }));
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const { estimate, error: quoteError, loading } = useQuote(draft);
  const patch = (next: Partial<SwapDraft>) => setDraft((current) => ({ ...current, ...next }));

  const flip = () =>
    patch({
      fromAsset: draft.toAsset,
      fromChain: draft.toChain,
      toAsset: draft.fromAsset,
      toChain: draft.fromChain,
    });

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!draft.destAddress.trim()) {
      setSubmitError("Enter the address that should receive your coins.");
      return;
    }

    setSubmitting(true);
    setSubmitError(null);
    try {
      const order = await api.createSwap(draft);
      onCreated(order.swap_id);
    } catch (err: unknown) {
      setSubmitError(err instanceof Error ? err.message : "Could not create the order.");
    } finally {
      setSubmitting(false);
    }
  };

  const receiveNetwork = networkLabel(catalog.assets, draft.toChain);
  const error = submitError ?? quoteError;

  return (
    <>
      <div className="hero">
        <h1>Swap crypto across chains</h1>
        <p>No account. No KYC. No custody of your keys.</p>
      </div>

      <form className="card" onSubmit={submit} autoComplete="off">
        <div className="legs">
          <div className="leg">
            <label className="leg-label" htmlFor="send-amount">
              You send
            </label>
            <div className="leg-body">
              <input
                id="send-amount"
                className="amount"
                type="text"
                inputMode="decimal"
                spellCheck={false}
                value={draft.amount}
                onChange={(event) => patch({ amount: event.target.value.replace(/[^\d.]/g, "") })}
              />
              <AssetPicker
                label="Send"
                assets={catalog.assets}
                asset={draft.fromAsset}
                chain={draft.fromChain}
                onChange={({ asset, chain }) => patch({ fromAsset: asset, fromChain: chain })}
              />
            </div>
          </div>

          <button type="button" className="flip" onClick={flip} title="Reverse direction" aria-label="Reverse direction">
            ⇅
          </button>

          <div className="leg">
            <label className="leg-label" htmlFor="receive-amount">
              You get
            </label>
            <div className="leg-body">
              <input
                id="receive-amount"
                className="amount"
                type="text"
                readOnly
                tabIndex={-1}
                value={loading ? "…" : estimate ? formatAmount(estimate.amount_out, estimate.to_asset) : "—"}
              />
              <AssetPicker
                label="Receive"
                assets={catalog.assets}
                asset={draft.toAsset}
                chain={draft.toChain}
                onChange={({ asset, chain }) => patch({ toAsset: asset, toChain: chain })}
              />
            </div>
          </div>
        </div>

        <div className="rate-row">
          <span className="badge" title="The final amount is settled at the market rate when your deposit confirms.">
            Float rate
          </span>
          <span className="rate-line">
            {estimate
              ? `1 ${estimate.from_asset} ≈ ${formatAmount(estimate.rate, estimate.to_asset)} ${estimate.to_asset} · fee ${formatAmount(estimate.fee, estimate.from_asset)} ${estimate.from_asset}`
              : quoteError
                ? "Rate unavailable"
                : `Minimum ${formatAmount(catalog.min_amount, draft.fromAsset)} — enter an amount to see the rate`}
          </span>
        </div>

        <label className="field" htmlFor="dest-address">
          <span className="field-label">
            Recipient {draft.toAsset} address ({receiveNetwork})
          </span>
          <input
            id="dest-address"
            type="text"
            spellCheck={false}
            placeholder={`Your ${draft.toAsset} address on ${receiveNetwork}`}
            value={draft.destAddress}
            onChange={(event) => patch({ destAddress: event.target.value })}
          />
        </label>

        {error ? <p className="error">{error}</p> : null}

        <button type="submit" className="primary" disabled={submitting}>
          {submitting ? "Creating order…" : "Exchange now"}
        </button>

        <p className="fineprint">
          By exchanging you agree that this service is non-custodial at rest and holds no personal data.
        </p>
      </form>

      <ul className="assurances">
        <li>
          <strong>No sign-up</strong>
          <span>An order is a deposit address and an ID. Nothing else.</span>
        </li>
        <li>
          <strong>No KYC</strong>
          <span>We never ask for identity documents.</span>
        </li>
        <li>
          <strong>Any chain</strong>
          <span>Stablecoins and BTC, routed automatically.</span>
        </li>
      </ul>
    </>
  );
}
