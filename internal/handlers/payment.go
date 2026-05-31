package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/saurabh/payment-routing-layer/internal/db"
	"github.com/saurabh/payment-routing-layer/internal/models"
	"github.com/saurabh/payment-routing-layer/internal/services"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CreatePayment(c *gin.Context) {
	var req struct {
		Amount        int    `json:"amount" binding:"required"`
		Currency      string `json:"currency"`
		PaymentMethod string `json:"paymentMethod"`
		CustomerEmail string `json:"customerEmail"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	if req.Currency == "" {
		req.Currency = "INR"
	}

	orderID := "ORD_" + uuid.NewString()

	transaction := models.Transaction{
		ID:            primitive.NewObjectID(),
		OrderID:       orderID,
		Amount:        req.Amount,
		Currency:      req.Currency,
		PaymentMethod: req.PaymentMethod,
		CustomerEmail: req.CustomerEmail,
		Status:        "created",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	coll := db.PaymentDB.Collection("transactions")
	_, err := coll.InsertOne(context.Background(), transaction)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to create local transaction"})
		return
	}

	hyperswitchReq := services.PaymentIntentRequest{
		Amount:                   req.Amount,
		Currency:                 req.Currency,
		MerchantOrderReferenceID: orderID,
	}

	hsResp, err := services.CreatePaymentIntent(hyperswitchReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Hyperswitch payment creation failed: " + err.Error()})
		return
	}

	transaction.HyperswitchPaymentID = hsResp.PaymentID
	if hsResp.Status != "" {
		transaction.Status = hsResp.Status
	} else {
		transaction.Status = "pending"
	}
	transaction.UpdatedAt = time.Now()

	update := bson.M{
		"$set": bson.M{
			"hyperswitchPaymentId": transaction.HyperswitchPaymentID,
			"status":               transaction.Status,
			"updatedAt":            transaction.UpdatedAt,
		},
	}

	_, err = coll.UpdateOne(context.Background(), bson.M{"_id": transaction.ID}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to update transaction"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"transaction":          transaction,
			"clientSecret":         hsResp.ClientSecret,
			"hyperswitchPaymentId": hsResp.PaymentID,
		},
	})
}

func ReconcilePayment(c *gin.Context) {
	orderID := c.Param("orderId")

	coll := db.PaymentDB.Collection("transactions")
	var transaction models.Transaction

	err := coll.FindOne(context.Background(), bson.M{"orderId": orderID}).Decode(&transaction)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Transaction not found"})
		return
	}

	if transaction.HyperswitchPaymentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "No Hyperswitch Payment ID associated"})
		return
	}

	hsData, err := services.RetrievePaymentIntent(transaction.HyperswitchPaymentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Reconciliation failed: " + err.Error()})
		return
	}

	hsStatus, ok := hsData["status"].(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Invalid status from Hyperswitch"})
		return
	}

	updated := false
	if transaction.Status != hsStatus {
		update := bson.M{
			"$set": bson.M{
				"status":     hsStatus,
				"reconciled": true,
				"updatedAt":  time.Now(),
			},
		}
		_, err = coll.UpdateOne(context.Background(), bson.M{"_id": transaction.ID}, update)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to update reconciled transaction"})
			return
		}
		updated = true
	}

	msg := "Transaction already up-to-date"
	if updated {
		msg = "Transaction reconciled and updated"
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"message":       msg,
		"currentStatus": hsStatus,
	})
}
