import type { Asset } from "../types";

interface Props {
  label: string;
  assets: Asset[];
  asset: string;
  chain: string;
  onChange: (next: { asset: string; chain: string }) => void;
}

/**
 * Coin + network pair. Changing the coin re-homes the network to the first one
 * that coin actually lives on, so the selection is never an unsupported pair.
 */
export function AssetPicker({ label, assets, asset, chain, onChange }: Props) {
  const networks = assets.find((item) => item.symbol === asset)?.networks ?? [];

  const selectAsset = (symbol: string) => {
    const available = assets.find((item) => item.symbol === symbol)?.networks ?? [];
    const keep = available.some((network) => network.id === chain);
    onChange({ asset: symbol, chain: keep ? chain : (available[0]?.id ?? "") });
  };

  return (
    <div className="picker">
      <select
        className="asset"
        aria-label={`${label} asset`}
        value={asset}
        onChange={(event) => selectAsset(event.target.value)}
      >
        {assets.map((item) => (
          <option key={item.symbol} value={item.symbol}>
            {item.symbol}
          </option>
        ))}
      </select>

      <select
        className="network"
        aria-label={`${label} network`}
        value={chain}
        disabled={networks.length <= 1}
        onChange={(event) => onChange({ asset, chain: event.target.value })}
      >
        {networks.map((network) => (
          <option key={network.id} value={network.id}>
            {network.short && network.short !== network.name
              ? `${network.name} · ${network.short}`
              : network.name}
          </option>
        ))}
      </select>
    </div>
  );
}
