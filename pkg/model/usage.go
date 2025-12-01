package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UsageEventOut struct {
	EventTimeStamp   time.Time             `json:"eventTimeStamp" bson:"eventTimeStamp"`
	UserID           string                `json:"userId" bson:"userId"`
	EventType        string                `json:"eventType" bson:"eventType"`
	SubscriptionID   *string               `json:"subscriptionId,omitempty" bson:"subscriptionId,omitempty"`
	PackageID        *string               `json:"packageId,omitempty" bson:"packageId,omitempty"`
	EggToken         int                   `json:"eggToken" bson:"eggToken"`
	ChatToken        *int                  `json:"chatToken,omitempty" bson:"chatToken,omitempty"`
	WebsearchToken   *int                  `json:"websearchToken,omitempty" bson:"websearchToken,omitempty"`
	TotalCostUSD     *primitive.Decimal128 `json:"totalCostUsd,omitempty" bson:"totalCostUsd,omitempty"`
	ChatCostUSD      *primitive.Decimal128 `json:"chatCostUsd,omitempty" bson:"chatCostUsd,omitempty"`
	WebsearchCostUSD *primitive.Decimal128 `json:"websearchCostUsd,omitempty" bson:"websearchCostUsd,omitempty"`
	TraceID          *string               `json:"traceId,omitempty" bson:"traceId,omitempty"`
	AIModel          *string               `json:"aiModel,omitempty" bson:"aiModel,omitempty"`
	AgentID          *string               `json:"agentId,omitempty" bson:"agentId,omitempty"`
}

type TokenUsedIn struct {
	UserID        string   `json:"userId"`
	TraceID       string   `json:"traceId"`
	AgentID       *string  `json:"agentId,omitempty"`
	WebsearchCost *float64 `json:"websearchCost,omitempty"`
}

type UserBalance struct {
	UserID                string    `bson:"userId"`
	TotalToken            int       `bson:"totalToken"`
	MainTokenBalance      int       `bson:"mainTokenBalance"`
	TopupTokenBalance     int       `bson:"topupTokenBalance"`
	RemainingTokenBalance int       `bson:"remainingTokenBalance"`
	UpdatedAt             time.Time `bson:"updatedAt"`
	CreatedAt             time.Time `bson:"createdAt"`
}

type UserMainPackage struct {
	UserID         string    `bson:"userId"`
	SubscriptionID string    `bson:"subscriptionId"`
	PackageID      string    `bson:"packageId"`
	Status         string    `bson:"status"`
	StartDate      time.Time `bson:"startDate"`
	EndDate        time.Time `bson:"endDate"`
	CreatedAt      time.Time `bson:"createdAt"`
	UpdatedAt      time.Time `bson:"updatedAt"`
}

type UserTopupPackage struct {
	UserID          string    `bson:"userId"`
	Status          string    `bson:"status"`
	TotalTopupToken int       `bson:"totalTopupToken"`
	StartDate       time.Time `bson:"startDate"`
	EndDate         time.Time `bson:"endDate"`
	CreatedAt       time.Time `bson:"createdAt"`
	UpdatedAt       time.Time `bson:"updatedAt"`
}

type PackageMaster struct {
	PackageID       string      `bson:"packageId"`
	EggToken        int         `bson:"eggToken"`
	ConversionRatio interface{} `bson:"conversionRatio"`
}
