import { ExchangeForm } from "./components/ExchangeForm";
import { OrderPage } from "./components/OrderPage";
import { useCatalog } from "./hooks/useCatalog";
import { useRoute } from "./hooks/useRoute";

export function App() {
  const [route, navigate] = useRoute();
  const { catalog, error } = useCatalog();

  const openExchange = (event: React.MouseEvent) => {
    event.preventDefault();
    navigate("/");
  };

  const trackOrder = (event: React.MouseEvent) => {
    event.preventDefault();
    const id = window.prompt("Enter your order ID");
    if (id?.trim()) navigate(`/order/${id.trim()}`);
  };

  return (
    <>
      <header className="topbar">
        <a className="brand" href="/" onClick={openExchange}>
          <span className="brand-mark">⇄</span>
          <span className="brand-name">Swap</span>
        </a>
        <nav className="topnav">
          <a href="/" onClick={openExchange}>
            Exchange
          </a>
          <a href="#" onClick={trackOrder}>
            Track order
          </a>
        </nav>
      </header>

      <main>
        {error ? (
          <div className="card">
            <p className="error">{error} Is the server running?</p>
          </div>
        ) : !catalog ? (
          <div className="card">
            <p className="fineprint">Loading markets…</p>
          </div>
        ) : route.name === "order" ? (
          <OrderPage id={route.id} catalog={catalog} />
        ) : (
          <ExchangeForm catalog={catalog} onCreated={(id) => navigate(`/order/${id}`)} />
        )}
      </main>

      <footer className="footer">Non-custodial cross-chain swaps · no account · no KYC</footer>
    </>
  );
}
