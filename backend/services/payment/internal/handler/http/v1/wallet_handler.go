package v1

import (
	"net/http"

	authmw "project/pkg/auth/middleware"
	"project/pkg/response"
	"project/services/payment/internal/usecase"

	"github.com/gin-gonic/gin"
)

// WalletHandler serves customer wallet endpoints (auth required).
type WalletHandler struct {
	walletUC usecase.WalletUsecase
	topupUC  usecase.TopupUsecase
}

func NewWalletHandler(walletUC usecase.WalletUsecase, topupUC usecase.TopupUsecase) *WalletHandler {
	return &WalletHandler{walletUC: walletUC, topupUC: topupUC}
}

// GetMyWallet returns the authenticated customer's wallet balance and ledger history.
// Response follows repo PascalCase convention (no json tags on entity structs).
func (h *WalletHandler) GetMyWallet(c *gin.Context) {
	ctx := c.Request.Context()
	customerID := authmw.UserIDFromContext(ctx)
	if customerID == "" {
		response.BadRequest(c, "missing customer id")
		return
	}

	wallet, entries, err := h.walletUC.GetWalletWithHistory(ctx, customerID, 20)
	if err != nil {
		response.InternalError(c)
		return
	}

	// PascalCase response per repo convention — struct fields serve as JSON keys.
	type ledgerRow struct {
		ID           string `json:"ID"`
		EntryType    string `json:"EntryType"`
		Amount       int64  `json:"Amount"`
		RefType      string `json:"RefType"`
		BalanceAfter int64  `json:"BalanceAfter"`
		CreatedAt    string `json:"CreatedAt"`
	}
	type walletResp struct {
		ID        string      `json:"ID"`
		OwnerType string      `json:"OwnerType"`
		Balance   int64       `json:"Balance"`
		Ledger    []ledgerRow `json:"Ledger"`
	}

	rows := make([]ledgerRow, len(entries))
	for i, e := range entries {
		rows[i] = ledgerRow{
			ID:           e.ID,
			EntryType:    string(e.EntryType),
			Amount:       e.Amount,
			RefType:      string(e.RefType),
			BalanceAfter: e.BalanceAfter,
			CreatedAt:    e.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		}
	}

	response.Success(c, walletResp{
		ID:        wallet.ID,
		OwnerType: string(wallet.OwnerType),
		Balance:   wallet.Balance,
		Ledger:    rows,
	})
}

// InitiateTopup creates a MoMo topup intent and returns pay_url.
// Request body: {"amount": 50000}
func (h *WalletHandler) InitiateTopup(c *gin.Context) {
	ctx := c.Request.Context()
	customerID := authmw.UserIDFromContext(ctx)
	if customerID == "" {
		response.BadRequest(c, "missing customer id")
		return
	}

	var req struct {
		Amount int64 `json:"amount"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Amount <= 0 {
		response.BadRequest(c, "amount must be a positive integer (VND)")
		return
	}

	result, err := h.topupUC.InitiateTopup(ctx, customerID, req.Amount)
	if err != nil {
		response.InternalError(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  200,
		"message": "topup initiated",
		"data": gin.H{
			"TopupID": result.TopupID,
			"PayUrl":  result.PayURL,
		},
	})
}
