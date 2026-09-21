import type { Catalog, CreatedOrder, Estimate, Order, SwapDraft } from "./types";

/** An error carrying the server's own message, which the UI shows verbatim. */
export class ApiError extends Error {}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: init?.body ? { "Content-Type": "application/json" } : undefined,
  });

  const body: unknown = await response.json().catch(() => null);
  if (!response.ok) {
    const message =
      body && typeof body === "object" && "error" in body && typeof body.error === "string"
        ? body.error
        : `Request failed (${response.status})`;
    throw new ApiError(message);
  }

  return body as T;
}

function pairPayload(draft: SwapDraft) {
  return {
    from_chain: draft.fromChain,
    to_chain: draft.toChain,
    from_asset: draft.fromAsset,
    to_asset: draft.toAsset,
    amount: Number(draft.amount) || 0,
  };
}

export const api = {
  catalog: (signal?: AbortSignal) => request<Catalog>("/assets", { signal }),

  quote: (draft: SwapDraft, signal?: AbortSignal) =>
    request<Estimate>("/quote", {
      method: "POST",
      body: JSON.stringify(pairPayload(draft)),
      signal,
    }),

  createSwap: (draft: SwapDraft) =>
    request<CreatedOrder>("/swap", {
      method: "POST",
      body: JSON.stringify({ ...pairPayload(draft), dest_address: draft.destAddress.trim() }),
    }),

  order: (id: string, signal?: AbortSignal) =>
    request<Order>(`/swap/${encodeURIComponent(id)}`, { signal }),

  qrUrl: (address: string, size = 200) =>
    `/qr?data=${encodeURIComponent(address)}&size=${size}`,
};
