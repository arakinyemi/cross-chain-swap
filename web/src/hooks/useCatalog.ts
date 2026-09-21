import { useEffect, useState } from "react";
import { api } from "../api";
import type { Catalog } from "../types";

export function useCatalog() {
  const [catalog, setCatalog] = useState<Catalog | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const controller = new AbortController();
    api
      .catalog(controller.signal)
      .then(setCatalog)
      .catch((err: unknown) => {
        if (controller.signal.aborted) return;
        setError(err instanceof Error ? err.message : "Could not load the asset list.");
      });

    return () => controller.abort();
  }, []);

  return { catalog, error };
}
