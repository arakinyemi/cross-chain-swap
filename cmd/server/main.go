package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"crosschain/internal/bitnob"
	"crosschain/internal/config"
	"crosschain/internal/db"
	"crosschain/internal/swap"
	"crosschain/internal/webhook"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("no .env file loaded: %v", err)
	}

	cfg := config.Load()
	if cfg.BitnobClientID == "" || cfg.BitnobClientSecret == "" || cfg.DatabaseURL == "" {
		log.Fatal("BITNOB_CLIENT_ID, BITNOB_CLIENT_SECRET, and DATABASE_URL are required")
	}

	store, err := db.NewPostgresStore(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer store.Close()

	bitnobClient := bitnob.NewClient(cfg.BitnobClientID, cfg.BitnobClientSecret, cfg.BitnobBaseURL)
	if err := bitnobClient.Whoami(); err != nil {
		log.Fatalf("verify bitnob credentials: %v", err)
	}

	swapService := swap.NewService(bitnobClient, store, cfg)
	webhookHandler := webhook.NewHandler(swapService, store, cfg.BitnobWebhookSecret)

	r := chi.NewRouter()
	r.Use(cors)
	r.Post("/swap", createSwapHandler(swapService))
	r.Get("/swap/{id}", getSwapHandler(store))
	r.Get("/prices", pricesHandler(bitnobClient))
	r.Post("/webhooks/bitnob", webhookHandler.Handle)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("server listening on :%s", cfg.Port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server error: %v", err)
	}
}

func createSwapHandler(svc *swap.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			FromChain   string  `json:"from_chain"`
			ToChain     string  `json:"to_chain"`
			FromAsset   string  `json:"from_asset"`
			ToAsset     string  `json:"to_asset"`
			DestAddress string  `json:"dest_address"`
			Amount      float64 `json:"amount"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		sw, err := svc.CreateSwap(req.FromChain, req.ToChain, req.FromAsset, req.ToAsset, req.DestAddress, req.Amount)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"swap_id":         sw.ID,
			"deposit_address": sw.DepositAddress,
			"from_chain":      sw.FromChain,
			"to_chain":        sw.ToChain,
			"from_asset":      sw.FromAsset,
			"to_asset":        sw.ToAsset,
			"amount_in":       sw.AmountIn,
			"estimated_out":   sw.AmountOut,
			"fee":             sw.Fee,
		})
	}
}

func getSwapHandler(store swap.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		sw, err := store.GetSwapByID(id)
		if err != nil {
			writeError(w, http.StatusNotFound, "swap not found")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"id":               sw.ID,
			"status":           sw.Status,
			"deposit_address":  sw.DepositAddress,
			"dest_address":     sw.DestAddress,
			"amount_in":        sw.AmountIn,
			"estimated_out":    sw.AmountOut,
			"fee":              sw.Fee,
			"from_chain":       sw.FromChain,
			"to_chain":         sw.ToChain,
			"from_asset":       sw.FromAsset,
			"to_asset":         sw.ToAsset,
			"btc_amount":       sw.BTCAmount,
			"trade1_order_id":  sw.Trade1OrderID,
			"trade2_order_id":  sw.Trade2OrderID,
			"withdrawal_tx_id": sw.WithdrawalTxID,
			"created_at":       sw.CreatedAt,
		})
	}
}

func pricesHandler(client *bitnob.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		prices, err := client.GetPrices()
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, prices)
	}
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("write JSON response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
