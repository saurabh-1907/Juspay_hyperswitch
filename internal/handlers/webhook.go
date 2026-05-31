package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/saurabh/payment-routing-layer/internal/db"
	"github.com/saurabh/payment-routing-layer/internal/models"
	"go.mongodb.org/mongo-driver/bson"
)

func verifyWebhookSignature(payload []byte, signature string) bool {
	secret := os.Getenv("HYPERSWITCH_WEBHOOK_SECRET")
	if secret == "" {
		// If no secret configured, skip verification (dev mode)
		return true
	}
	if signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

func HandleHyperswitchWebhook(c *gin.Context) {
	// Read raw body for signature verification BEFORE binding
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Verify webhook signature
	sig := c.GetHeader("X-Webhook-Signature-512")
	if !verifyWebhookSignature(rawBody, sig) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid webhook signature"})
		return
	}

	var event map[string]interface{}
	if err := json.Unmarshal(rawBody, &event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook payload"})
		return
	}

	eventType, ok := event["type"].(string)
	if !ok {
		c.String(http.StatusOK, "Webhook received (unknown type)")
		return
	}

	fmt.Printf("[Webhook] Received event: %s\n", eventType)

	switch {
	case strings.HasPrefix(eventType, "payment_intent."):
		handlePaymentIntentEvent(c, event)
	case strings.HasPrefix(eventType, "refund."):
		fmt.Printf("[Webhook] Refund event received: %s\n", eventType)
		c.String(http.StatusOK, "Webhook received")
	default:
		fmt.Printf("[Webhook] Unhandled event type: %s\n", eventType)
		c.String(http.StatusOK, "Webhook received (unhandled type)")
	}
}

func handlePaymentIntentEvent(c *gin.Context, event map[string]interface{}) {
	dataObj, dataOk := event["data"].(map[string]interface{})
	if !dataOk {
		c.String(http.StatusBadRequest, "Missing data object in webhook payload")
		return
	}

	paymentIntent, piOk := dataObj["object"].(map[string]interface{})
	if !piOk {
		c.String(http.StatusBadRequest, "Missing object in webhook data")
		return
	}

	paymentID, _ := paymentIntent["payment_id"].(string)
	status, _ := paymentIntent["status"].(string)
	errMsg, _ := paymentIntent["error_message"].(string)

	if paymentID == "" {
		c.String(http.StatusBadRequest, "No payment_id found in webhook payload")
		return
	}

	coll := db.PaymentDB.Collection("transactions")
	var transaction models.Transaction
	err := coll.FindOne(context.Background(), bson.M{"hyperswitchPaymentId": paymentID}).Decode(&transaction)
	if err != nil {
		fmt.Printf("[Webhook] Transaction %s not found in DB.\n", paymentID)
		c.String(http.StatusOK, "Webhook received (transaction not found locally)")
		return
	}

	updateData := bson.M{
		"status":    status,
		"updatedAt": time.Now(),
	}

	if errMsg != "" {
		if transaction.Metadata == nil {
			transaction.Metadata = make(map[string]interface{})
		}
		transaction.Metadata["error"] = errMsg
		updateData["metadata"] = transaction.Metadata
	}

	_, updateErr := coll.UpdateOne(context.Background(), bson.M{"_id": transaction.ID}, bson.M{"$set": updateData})
	if updateErr != nil {
		fmt.Printf("[Webhook] Error updating transaction: %v\n", updateErr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transaction"})
		return
	}

	fmt.Printf("[Webhook] Updated transaction %s → status: %s\n", transaction.OrderID, status)
	c.String(http.StatusOK, "Webhook received successfully")
}
