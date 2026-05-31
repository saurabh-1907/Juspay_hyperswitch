package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Transaction struct {
	ID                   primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	OrderID              string                 `bson:"orderId" json:"orderId"`
	Amount               int                    `bson:"amount" json:"amount"`
	Currency             string                 `bson:"currency" json:"currency"`
	Status               string                 `bson:"status" json:"status"`
	HyperswitchPaymentID string                 `bson:"hyperswitchPaymentId,omitempty" json:"hyperswitchPaymentId,omitempty"`
	PaymentMethod        string                 `bson:"paymentMethod,omitempty" json:"paymentMethod,omitempty"`
	CustomerEmail        string                 `bson:"customerEmail,omitempty" json:"customerEmail,omitempty"`
	CustomerPhone        string                 `bson:"customerPhone,omitempty" json:"customerPhone,omitempty"`
	Metadata             map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
	Reconciled           bool                   `bson:"reconciled" json:"reconciled"`
	CreatedAt            time.Time              `bson:"createdAt" json:"createdAt"`
	UpdatedAt            time.Time              `bson:"updatedAt" json:"updatedAt"`
}
