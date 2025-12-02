package http

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"

	"munggonegg/credit-service-go/internal/config"
	"munggonegg/credit-service-go/internal/adapter/repository/mongodb"
	"munggonegg/credit-service-go/internal/core/domain"
	"munggonegg/credit-service-go/internal/service"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/sync/errgroup"
)

func RecordTokenUsed(c *fiber.Ctx) error {
	var payload domain.TokenUsedIn
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	ctx := c.Context()
	umpColl := mongodb.GetCollection(config.UserMainPackageColl)
	balColl := mongodb.GetCollection(config.UserBalanceColl)
	pkgColl := mongodb.GetCollection(config.PackageMasterV3Coll)
	uueColl := mongodb.GetCollection(config.UsageEventColl)

	// 1. Parallel Fetching: UserMainPackage and UserBalance
	var ump domain.UserMainPackage
	var bal domain.UserBalance

	g, gCtx := errgroup.WithContext(ctx)

	// Fetch Main Package
	g.Go(func() error {
		if err := umpColl.FindOne(gCtx, bson.M{"userId": payload.UserID}).Decode(&ump); err != nil {
			return fmt.Errorf("User has no main package.")
		}
		return nil
	})

	// Fetch Balance
	g.Go(func() error {
		if err := balColl.FindOne(gCtx, bson.M{"userId": payload.UserID}).Decode(&bal); err != nil {
			return fmt.Errorf("No token balance remaining.")
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"detail": err.Error()})
	}

	// Check if balance is positive
	if bal.RemainingTokenBalance <= 0 {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"detail": "No token balance remaining."})
	}

	// 2. Call Portkey (Synchronous)
	now := time.Now().UTC()
	queryParams := fmt.Sprintf("trace_id=%s&workspace_slug=%s&time_of_generation_min=2025-08-01T00:00:00Z&time_of_generation_max=%s", payload.TraceID, config.AppConfig.PortkeyWorkspaceSlug, now.Format(time.RFC3339))
	req, err := http.NewRequest("GET", config.AppConfig.PortkeyURL+"?"+queryParams, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Failed to create request"})
	}
	req.Header.Set("x-portkey-api-key", config.AppConfig.PortkeyAPIKey)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"detail": fmt.Sprintf("Error connecting to Portkey: %v", err)})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorBody map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errorBody); err != nil {
			return c.Status(resp.StatusCode).JSON(fiber.Map{
				"detail": fmt.Sprintf("Portkey API error (status %d): unable to decode error response", resp.StatusCode),
			})
		}
		return c.Status(resp.StatusCode).JSON(fiber.Map{
			"detail": fmt.Sprintf("Portkey API error (status %d)", resp.StatusCode),
			"error":  errorBody,
		})
	}

	var portkeyResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&portkeyResp); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Failed to decode Portkey response"})
	}

	rows, ok := portkeyResp["data"].([]interface{})
	if !ok || len(rows) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"detail": "No Portkey cost found for traceId."})
	}

	totalCents := 0.0
	var aiModel string
	for _, row := range rows {
		r, ok := row.(map[string]interface{})
		if !ok {
			continue
		}
		if aiModel == "" {
			if m, ok := r["ai_model"].(string); ok {
				aiModel = m
			}
		}
		if cost, ok := r["cost"]; ok {
			// Handle cost type safely
			switch v := cost.(type) {
			case float64:
				totalCents += v
			case int:
				totalCents += float64(v)
			}
		}
	}

	if totalCents == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"detail": "No Portkey cost found for traceId."})
	}

	// 3. Fetch Package Master (for conversion ratio)
	var pkg domain.PackageMaster
	if err := pkgColl.FindOne(ctx, bson.M{"packageId": ump.PackageID}).Decode(&pkg); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Package not found"})
	}

	eggThbPrice := 1.0
	switch v := pkg.ConversionRatio.(type) {
	case int:
		eggThbPrice = float64(v)
	case int32:
		eggThbPrice = float64(v)
	case int64:
		eggThbPrice = float64(v)
	case float64:
		eggThbPrice = v
	case float32:
		eggThbPrice = float64(v)
	}
	if eggThbPrice == 0 {
		eggThbPrice = 1.0
	}

	// Calculate Tokens
	chatCost := totalCents / 100.0
	websearchCost := 0.0
	if payload.WebsearchCost != nil {
		websearchCost = *payload.WebsearchCost
	}

	totalCost := chatCost + websearchCost
	thb := totalCost * config.ThbPerUsd
	eggTokenFloat := thb / eggThbPrice
	eggTokenInt := -int(math.Ceil(eggTokenFloat)) // Negative for deduction

	chatTokenInt := -int(math.Ceil((chatCost * config.ThbPerUsd) / eggThbPrice))
	websearchTokenInt := 0
	if websearchCost > 0 {
		websearchTokenInt = -int(math.Ceil((websearchCost * config.ThbPerUsd) / eggThbPrice))
	}

	// 4. Calculate Split (Main vs Topup)
	// Logic from service.RollupBalances:
	// if main >= MainDeductionThreshold (100) -> deduct from main first
	// else -> deduct from topup first
	
	deduction := int(math.Abs(float64(eggTokenInt)))
	mainDeduction := 0
	topupDeduction := 0

	currentMain := bal.MainTokenBalance
	currentTopup := bal.TopupTokenBalance

	if currentMain >= service.MainDeductionThreshold {
		if deduction <= currentMain {
			mainDeduction = deduction
		} else {
			mainDeduction = currentMain
			topupDeduction = deduction - currentMain
		}
	} else {
		if deduction <= currentTopup {
			topupDeduction = deduction
		} else {
			topupDeduction = currentTopup
			mainDeduction = deduction - currentTopup
		}
	}

	// 5. Atomic Update
	update := bson.M{
		"$inc": bson.M{
			"totalToken":            eggTokenInt,
			"remainingTokenBalance": eggTokenInt,
			"mainTokenBalance":      -mainDeduction,
			"topupTokenBalance":     -topupDeduction,
		},
		"$set": bson.M{
			"updatedAt": time.Now(),
		},
	}

	// We use UpdateOne. If the user's balance changed concurrently, this update will still apply the deduction.
	// This might cause main/topup to go slightly negative if they were near 0, but Total/Remaining will be correct relative to usage.
	_, err = balColl.UpdateOne(ctx, bson.M{"userId": payload.UserID}, update)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": fmt.Sprintf("DB update failed: %v", err)})
	}

	// 6. Async Create Usage Event
	subID := ump.SubscriptionID
	pkgIDStr := ump.PackageID

	// Convert float64 to Decimal128
	totalCostDec, _ := primitive.ParseDecimal128(fmt.Sprintf("%.6f", totalCost))
	chatCostDec, _ := primitive.ParseDecimal128(fmt.Sprintf("%.6f", chatCost))
	websearchCostDec, _ := primitive.ParseDecimal128(fmt.Sprintf("%.6f", websearchCost))

	doc := domain.UsageEventOut{
		EventTimeStamp:   time.Now(),
		UserID:           payload.UserID,
		EventType:        "Token Used",
		SubscriptionID:   &subID,
		PackageID:        &pkgIDStr,
		EggToken:         eggTokenInt,
		ChatToken:        &chatTokenInt,
		WebsearchToken:   &websearchTokenInt,
		TotalCostUSD:     &totalCostDec,
		ChatCostUSD:      &chatCostDec,
		WebsearchCostUSD: &websearchCostDec,
		TraceID:          &payload.TraceID,
		AIModel:          &aiModel,
		AgentID:          payload.AgentID,
	}

	// Fire and forget
	// Synchronous Create Usage Event with Timeout
	// We use a detached context (context.Background) with a timeout to ensure the insert attempts to complete
	// even if the client disconnects, as the balance has already been deducted.
	bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := uueColl.InsertOne(bgCtx, doc); err != nil {
		// Log error but do not fail the request since deduction succeeded
		fmt.Printf("Failed to insert usage event: %v\n", err)
	}

	return c.Status(fiber.StatusCreated).JSON(doc)
}
