import { CopyField } from "./CopyField";
import { STAGES, StatusTimeline } from "./StatusTimeline";
import { api } from "../api";
import { formatAmount, networkLabel, pairLabel } from "../format";
import { useOrder } from "../hooks/useOrder";
import type { Catalog } from "../types";

interface Props {
  id: string;
  catalog: Catalog;
}

export function OrderPage({ id, catalog }: Props) {
  const { order, error } = useOrder(id);

  if (!order) {
    return (
      <div className="card order">
        <div className="order-head">
          <div>
            <span className="order-label">Order</span>
            <span className="order-id">{id}</span>
          </div>
          <span className="state">{error ? "Not found" : "Loading…"}</span>
        </div>
        {error ? <p className="error">{error}</p> : null}
      </div>
    );
  }

  const stage = STAGES[order.status];
  const awaiting = order.status === "awaiting_deposit";
  const depositNetwork = networkLabel(catalog.assets, order.from_chain);

  return (
    <div className="card order">
      <div className="order-head">
        <div>
          <span className="order-label">Order</span>
          <span className="order-id">{order.id}</span>
        </div>
        <span className="state" data-tone={stage.tone}>
          {stage.label}
        </span>
      </div>

      <StatusTimeline status={order.status} />

      {awaiting ? (
        <div className="deposit">
          <p className="deposit-ask">
            Send exactly{" "}
            <strong>
              {formatAmount(order.amount_in, order.from_asset)} {order.from_asset}
            </strong>{" "}
            to this address
          </p>
          <div className="deposit-body">
            <img
              className="qr"
              width={200}
              height={200}
              src={api.qrUrl(order.deposit_address)}
              alt="Deposit address QR code"
            />
            <div className="deposit-side">
              <CopyField value={order.deposit_address} />
              <p className="warn">
                Send only {order.from_asset} on {depositNetwork}. Coins sent on another network are
                unrecoverable.
              </p>
            </div>
          </div>
        </div>
      ) : null}

      <dl className="summary">
        <div>
          <dt>You send</dt>
          <dd>
            {formatAmount(order.amount_in, order.from_asset)}{" "}
            {pairLabel(catalog.assets, order.from_asset, order.from_chain)}
          </dd>
        </div>
        <div>
          <dt>You get (estimated)</dt>
          <dd>
            {formatAmount(order.estimated_out, order.to_asset)}{" "}
            {pairLabel(catalog.assets, order.to_asset, order.to_chain)}
          </dd>
        </div>
        <div>
          <dt>Service fee</dt>
          <dd>
            {formatAmount(order.fee, order.from_asset)} {order.from_asset}
          </dd>
        </div>
        <div>
          <dt>Recipient</dt>
          <dd className="mono">{order.dest_address}</dd>
        </div>
        {order.withdrawal_tx_id ? (
          <div>
            <dt>Payout tx</dt>
            <dd className="mono">{order.withdrawal_tx_id}</dd>
          </div>
        ) : null}
      </dl>

      {error ? <p className="error">{error}</p> : null}
      <p className="fineprint">Keep this link. It is the only way back to your order.</p>
    </div>
  );
}
