package v1

import (
	"log/slog"
	"net/http"

	"project/pkg/response"
	"project/services/payment/internal/usecase"

	"github.com/gin-gonic/gin"
)

// MoMoHandler serves MoMo IPN and return URL endpoints.
// IPN is PUBLIC (no JWT auth) — MoMo server calls it server-to-server.
type MoMoHandler struct {
	ipnUC usecase.IPNUsecase
}

func NewMoMoHandler(ipnUC usecase.IPNUsecase) *MoMoHandler {
	return &MoMoHandler{ipnUC: ipnUC}
}

// HandleIPN processes MoMo's server-to-server payment notification.
// HMAC-SHA256 signature is verified inside the usecase.
// POST /api/v1/payments/momo/ipn (no auth)
func (h *MoMoHandler) HandleIPN(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse MoMo's own field names (camelCase per MoMo v2 spec).
	var body struct {
		PartnerCode  string `json:"partnerCode"`
		AccessKey    string `json:"accessKey"`
		RequestID    string `json:"requestId"`
		Amount       int64  `json:"amount"`
		OrderID      string `json:"orderId"`
		OrderInfo    string `json:"orderInfo"`
		OrderType    string `json:"orderType"`
		TransID      int64  `json:"transId"`
		ResultCode   int    `json:"resultCode"`
		Message      string `json:"message"`
		PayType      string `json:"payType"`
		ResponseTime int64  `json:"responseTime"`
		ExtraData    string `json:"extraData"`
		Signature    string `json:"signature"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		slog.WarnContext(ctx, "ipn: malformed body", "err", err)
		response.BadRequest(c, "malformed body")
		return
	}

	err := h.ipnUC.HandleIPN(ctx, usecase.IPNRequest{
		PartnerCode:  body.PartnerCode,
		AccessKey:    body.AccessKey,
		RequestID:    body.RequestID,
		Amount:       body.Amount,
		OrderID:      body.OrderID,
		OrderInfo:    body.OrderInfo,
		OrderType:    body.OrderType,
		TransID:      body.TransID,
		ResultCode:   body.ResultCode,
		Message:      body.Message,
		PayType:      body.PayType,
		ResponseTime: body.ResponseTime,
		ExtraData:    body.ExtraData,
		Signature:    body.Signature,
	})
	if err != nil {
		slog.WarnContext(ctx, "ipn: handle failed", "order_id", body.OrderID, "err", err)
		// Return 400 for invalid signature; MoMo will not retry on 4xx.
		c.JSON(http.StatusBadRequest, gin.H{"resultCode": 1, "message": err.Error()})
		return
	}

	// MoMo expects a 200 with resultCode=0 to acknowledge the IPN.
	c.JSON(http.StatusOK, gin.H{"resultCode": 0, "message": "success"})
}

// HandleReturn is the browser redirect endpoint after the customer pays on MoMo.
// Non-authoritative — just acknowledges. Actual status is determined by IPN.
// GET /api/v1/payments/momo/return
func (h *MoMoHandler) HandleReturn(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  200,
		"message": "payment return received — check wallet for updated balance",
	})
}
