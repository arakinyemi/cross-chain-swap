import { useCallback, useEffect, useState } from "react";

export type Route = { name: "exchange" } | { name: "order"; id: string };

function parse(pathname: string): Route {
  const match = /^\/order\/([\w-]+)/.exec(pathname);

  return match?.[1] ? { name: "order", id: match[1] } : { name: "exchange" };
}

/**
 * Two views is not worth a router dependency: read the path, listen to
 * popstate, and push when the user creates an order.
 */
export function useRoute(): [Route, (path: string) => void] {
  const [route, setRoute] = useState<Route>(() => parse(window.location.pathname));

  useEffect(() => {
    const onPop = () => setRoute(parse(window.location.pathname));
    window.addEventListener("popstate", onPop);

    return () => window.removeEventListener("popstate", onPop);
  }, []);

  const navigate = useCallback((path: string) => {
    window.history.pushState({}, "", path);
    setRoute(parse(path));
  }, []);

  return [route, navigate];
}
