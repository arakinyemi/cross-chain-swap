import type { Asset } from "./types";

const decimalsFor = (asset: string) => (asset === "BTC" ? 8 : 2);

export function formatAmount(value: number | undefined, asset: string): string {
  if (typeof value !== "number" || !Number.isFinite(value)) return "—";

  return value.toLocaleString("en-US", {
    minimumFractionDigits: 0,
    maximumFractionDigits: decimalsFor(asset),
  });
}

export function networkLabel(assets: Asset[], chain: string): string {
  for (const asset of assets) {
    const match = asset.networks.find((network) => network.id === chain);
    if (match) return match.short || match.name;
  }

  return chain;
}

/** "0.007 BTC · BTC" reads as a mistake, so drop a network that repeats the asset. */
export function pairLabel(assets: Asset[], asset: string, chain: string): string {
  const network = networkLabel(assets, chain);

  return network === asset ? asset : `${asset} · ${network}`;
}
