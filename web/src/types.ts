// These mirror the JSON the Go server returns. Keep them in step with
// internal/swap/catalog.go, internal/swap/models.go and cmd/server/main.go.

export interface Network {
  id: string;
  name: string;
  short: string;
}

export interface Asset {
  symbol: string;
  name: string;
  networks: Network[];
}

export interface Catalog {
  assets: Asset[];
  min_amount: number;
  fee: number;
}

export interface Estimate {
  from_chain: string;
  to_chain: string;
  from_asset: string;
  to_asset: string;
  amount_in: number;
  amount_out: number;
  fee: number;
  rate: number;
  min_amount: number;
}

export interface CreatedOrder {
  swap_id: string;
  deposit_address: string;
  from_chain: string;
  to_chain: string;
  from_asset: string;
  to_asset: string;
  amount_in: number;
  estimated_out: number;
  fee: number;
}

export const SWAP_STATUSES = [
  "awaiting_deposit",
  "deposit_confirmed",
  "trade1_done",
  "trade2_done",
  "sending",
  "completed",
  "failed",
] as const;

export type SwapStatus = (typeof SWAP_STATUSES)[number];

export interface Order {
  id: string;
  status: SwapStatus;
  deposit_address: string;
  dest_address: string;
  amount_in: number;
  estimated_out: number;
  fee: number;
  from_chain: string;
  to_chain: string;
  from_asset: string;
  to_asset: string;
  btc_amount: string;
  trade1_order_id: string;
  trade2_order_id: string;
  withdrawal_tx_id: string;
  created_at: string;
}

/** What the exchange form holds before an order exists. */
export interface SwapDraft {
  fromAsset: string;
  fromChain: string;
  toAsset: string;
  toChain: string;
  amount: string;
  destAddress: string;
}
