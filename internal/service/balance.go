package service

import (
	"context"
	"math"
	"strconv"
	"strings"
	"time"

	"munggonegg/credit-service-go/internal/config"
	"munggonegg/credit-service-go/internal/adapter/repository/mongodb"
	"munggonegg/credit-service-go/internal/core/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	EvtSubscribe    = "Subscribe"
	EvtTopup        = "Topup"
	EvtTokenUsed    = "Token Used"
	EvtExpired      = "Expired"
	EvtMainExpired  = "MainExpired"
	EvtTopupExpired = "TopupExpired"

	MainDeductionThreshold = 100
)

func RollupBalances(events []bson.M) (int, int, int) {
	main := 0
	topup := 0

	for _, ev := range events {
		eventType, ok := ev["eventType"].(string)
		if !ok {
			continue
		}
		eventType = strings.TrimSpace(eventType)

		var amount int
		switch v := ev["eggToken"].(type) {
		case int32:
			amount = int(math.Abs(float64(v)))
		case int64:
			amount = int(math.Abs(float64(v)))
		case float64:
			amount = int(math.Abs(v))
		case string:
			if val, err := strconv.Atoi(v); err == nil {
				amount = int(math.Abs(float64(val)))
			}
		default:
			continue
		}

		switch eventType {
		case EvtSubscribe:
			main += amount
		case EvtTopup:
			topup += amount
		case EvtTokenUsed:
			if main >= MainDeductionThreshold {
				if amount <= main {
					main -= amount
				} else {
					remainder := amount - main
					main = 0
					topup = int(math.Max(0, float64(topup-remainder)))
				}
			} else {
				if amount <= topup {
					topup -= amount
				} else {
					remainder := amount - topup
					topup = 0
					main = int(math.Max(0, float64(main-remainder)))
				}
			}
		case EvtExpired:
			if amount <= main {
				main -= amount
			} else {
				remainder := amount - main
				main = 0
				topup = int(math.Max(0, float64(topup-remainder)))
			}
		case EvtMainExpired:
			main = int(math.Max(0, float64(main-amount)))
		case EvtTopupExpired:
			topup = int(math.Max(0, float64(topup-amount)))
		}
	}

	return main, topup, main + topup
}

func RecomputeAndUpsertUserBalance(ctx context.Context, userID string) (*domain.UserBalance, error) {
	uueColl := mongodb.GetCollection(config.UsageEventColl)
	umpColl := mongodb.GetCollection(config.UserMainPackageColl)
	utpColl := mongodb.GetCollection(config.UserTopupPackageColl)
	pkgColl := mongodb.GetCollection(config.PackageMasterV3Coll)
	balColl := mongodb.GetCollection(config.UserBalanceColl)

	// Fetch events
	cursor, err := uueColl.Find(ctx, bson.M{"userId": userID}, options.Find().SetSort(bson.M{"eventTimeStamp": 1}))
	if err != nil {
		return nil, err
	}
	var events []bson.M
	if err = cursor.All(ctx, &events); err != nil {
		return nil, err
	}

	// Fetch main package
	var mainDoc bson.M
	_ = umpColl.FindOne(ctx, bson.M{"userId": userID, "status": "A"}).Decode(&mainDoc)

	// Fetch topup package
	var topupDoc bson.M
	_ = utpColl.FindOne(ctx, bson.M{"userId": userID, "status": "A"}).Decode(&topupDoc)

	// Calculate balances
	mainBal, topupBal, remainingBal := RollupBalances(events)

	// Get main package egg token
	mainEgg := 0
	if mainDoc != nil {
		if pkgID, ok := mainDoc["packageId"].(string); ok {
			var pkgDoc bson.M
			if err := pkgColl.FindOne(ctx, bson.M{"packageId": pkgID}).Decode(&pkgDoc); err == nil {
				if val, ok := pkgDoc["eggToken"]; ok {
					mainEgg = toInt(val)
				}
			}
		}
	}

	totalTopupToken := 0
	if topupDoc != nil {
		if val, ok := topupDoc["totalTopupToken"]; ok {
			totalTopupToken = toInt(val)
		}
	}

	totalToken := mainEgg + totalTopupToken
	now := time.Now()

	update := bson.M{
		"$set": bson.M{
			"userId":                userID,
			"totalToken":            totalToken,
			"mainTokenBalance":      mainBal,
			"topupTokenBalance":     topupBal,
			"remainingTokenBalance": remainingBal,
			"updatedAt":             now,
		},
		"$setOnInsert": bson.M{"createdAt": now},
	}

	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var updatedDoc domain.UserBalance
	err = balColl.FindOneAndUpdate(ctx, bson.M{"userId": userID}, update, opts).Decode(&updatedDoc)
	if err != nil {
		return nil, err
	}

	return &updatedDoc, nil
}

func RecomputeTotalTopupToken(ctx context.Context, userID string) (int, error) {
	tpeColl := mongodb.GetCollection(config.TopupPackageEventColl)
	pkgCollName := config.PackageMasterV3Coll

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"userId": userID, "status": "A"}}},
		{{Key: "$group", Value: bson.M{"_id": "$packageId", "cnt": bson.M{"$sum": 1}}}},
		{{Key: "$lookup", Value: bson.M{
			"from":         pkgCollName,
			"localField":   "_id",
			"foreignField": "packageId",
			"as":           "pkg",
		}}},
		{{Key: "$project", Value: bson.M{
			"cnt": 1,
			"tokenPerPack": bson.M{"$ifNull": bson.A{
				bson.M{"$arrayElemAt": bson.A{"$pkg.eggToken", 0}},
				0,
			}},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id": nil,
			"total": bson.M{
				"$sum": bson.M{"$multiply": bson.A{"$cnt", bson.M{"$toInt": "$tokenPerPack"}}},
			},
		}}},
	}

	cursor, err := tpeColl.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}
	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		return 0, err
	}

	if len(results) > 0 {
		return toInt(results[0]["total"]), nil
	}

	return 0, nil
}

func toInt(v interface{}) int {
	switch val := v.(type) {
	case int:
		return val
	case int32:
		return int(val)
	case int64:
		return int(val)
	case float64:
		return int(val)
	case string:
		i, _ := strconv.Atoi(val)
		return i
	default:
		return 0
	}
}
