package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"crosschain/internal/swap"
)

type Handler struct {
	swapService   *swap.Service
	store         swap.Store
	webhookSecret string
}

func NewHandler(swapService *swap.Service, store swap.Store, webhookSecret string) *Handler {
	return &Handler{
		swapService:   swapService,
		store:         store,
		webhookSecret: webhookSecret,
	}
}

type event struct {
	Event string          `json:"event"`
	Type  string          `json:"type"`
	Data  json.RawMessage `json:"data"`
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("read webhook body: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	if !h.validSignature(r, body) {
		log.Printf("invalid bitnob webhook signature")
		w.WriteHeader(http.StatusOK)
		return
	}

	var evt event
	if err := json.Unmarshal(body, &evt); err != nil {
		log.Printf("decode bitnob webhook: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	eventType := evt.Event
	if eventType == "" {
		eventType = evt.Type
	}

	switch eventType {
	case "address.deposit.confirmed":
		h.handleDepositConfirmed(evt.Data)
	case "transfer.success":
		h.handleTransferSuccess(evt.Data)
	default:
		log.Printf("unhandled bitnob webhook event: %s", eventType)
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) handleDepositConfirmed(data json.RawMessage) {
	var payload struct {
		Address string `json:"address"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		log.Printf("decode deposit webhook data: %v", err)
		return
	}
	if payload.Address == "" {
		log.Printf("deposit webhook missing address")
		return
	}

	sw, err := h.store.GetSwapByDepositAddress(payload.Address)
	if err != nil {
		log.Printf("swap not found for deposit address %s: %v", payload.Address, err)
		return
	}
	if sw.Status != swap.StatusAwaitingDeposit {
		log.Printf("ignoring deposit webhook for swap %s with status %s", sw.ID, sw.Status)
		return
	}

	go func() {
		if err := h.swapService.ExecuteTrades(sw); err != nil {
			log.Printf("execute trades for swap %s: %v", sw.ID, err)
		}
	}()
}

func (h *Handler) handleTransferSuccess(data json.RawMessage) {
	var payload struct {
		Reference string `json:"reference"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		log.Printf("decode transfer webhook data: %v", err)
		return
	}
	if payload.Reference == "" {
		log.Printf("transfer webhook missing reference")
		return
	}

	ref := strings.TrimSuffix(payload.Reference, "-withdrawal")
	sw, err := h.store.GetSwapByReference(ref)
	if err != nil {
		log.Printf("swap not found for withdrawal reference %s: %v", payload.Reference, err)
		return
	}
	if err := h.store.UpdateSwapStatus(sw.ID, swap.StatusCompleted); err != nil {
		log.Printf("mark swap %s completed: %v", sw.ID, err)
	}
}

func (h *Handler) validSignature(r *http.Request, body []byte) bool {
	if h.webhookSecret == "" {
		return true
	}

	signature := r.Header.Get("X-Bitnob-Signature")
	if signature == "" {
		signature = r.Header.Get("X-Webhook-Signature")
	}
	if signature == "" {
		return false
	}

	signature = strings.TrimPrefix(signature, "sha256=")
	mac := hmac.New(sha256.New, []byte(h.webhookSecret))
	_, _ = mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expected))
}
