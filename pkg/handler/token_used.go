package handler

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"munggonegg/credit-service-go/pkg/config"
	"munggonegg/credit-service-go/pkg/database"
	"munggonegg/credit-service-go/pkg/model"
	"munggonegg/credit-service-go/pkg/service"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func RecordTokenUsed(c *fiber.Ctx) error {
	var payload model.TokenUsedIn
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	ctx := c.Context()
	umpColl := database.GetCollection(config.UserMainPackageColl)
	balColl := database.GetCollection(config.UserBalanceColl)
	pkgColl := database.GetCollection(config.PackageMasterV3Coll)
	uueColl := database.GetCollection(config.UsageEventColl)

	// Check main package
	var ump bson.M
	if err := umpColl.FindOne(ctx, bson.M{"userId": payload.UserID}).Decode(&ump); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"detail": "User has no main package."})
	}

	// Check balance
	var bal bson.M
	if err := balColl.FindOne(ctx, bson.M{"userId": payload.UserID}).Decode(&bal); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"detail": "No token balance remaining."})
	}
	remainingBal := 0
	if val, ok := bal["remainingTokenBalance"]; ok {
		remainingBal = toInt(val)
	}
	if remainingBal <= 0 {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"detail": "No token balance remaining."})
	}

	// Call Portkey
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
			totalCents += toFloat(cost)
		}
	}

	if totalCents == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"detail": "No Portkey cost found for traceId."})
	}

	// Get package conversion ratio
	var pkg bson.M
	pkgIDStr := toString(ump["packageId"])
	if err := pkgColl.FindOne(ctx, bson.M{"packageId": pkgIDStr}).Decode(&pkg); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Package not found"})
	}

	eggThbPrice := toFloat(pkg["conversionRatio"])
	if eggThbPrice == 0 {
		eggThbPrice = 1
	}

	chatCost := totalCents / 100.0
	websearchCost := 0.0
	if payload.WebsearchCost != nil {
		websearchCost = *payload.WebsearchCost
	}

	totalCost := chatCost + websearchCost

	thb := totalCost * config.ThbPerUsd
	eggTokenFloat := thb / eggThbPrice
	eggTokenInt := -int(math.Ceil(eggTokenFloat))

	chatTokenInt := -int(math.Ceil((chatCost * config.ThbPerUsd) / eggThbPrice))

	websearchTokenInt := 0
	if websearchCost > 0 {
		websearchTokenInt = -int(math.Ceil((websearchCost * config.ThbPerUsd) / eggThbPrice))
	}

	// Create Usage Event
	subID := toString(ump["subscriptionId"])

	// Convert float64 to Decimal128 for precise storage
	totalCostDec, err := primitive.ParseDecimal128(fmt.Sprintf("%.6f", totalCost))
	if err != nil {
		totalCostDec = primitive.NewDecimal128(0, 0)
	}
	chatCostDec, err := primitive.ParseDecimal128(fmt.Sprintf("%.6f", chatCost))
	if err != nil {
		chatCostDec = primitive.NewDecimal128(0, 0)
	}
	websearchCostDec, err := primitive.ParseDecimal128(fmt.Sprintf("%.6f", websearchCost))
	if err != nil {
		websearchCostDec = primitive.NewDecimal128(0, 0)
	}

	doc := model.UsageEventOut{
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

	// Insert and Recompute
	_, err = uueColl.InsertOne(ctx, doc)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": fmt.Sprintf("DB insert failed: %v", err)})
	}

	_, err = service.RecomputeAndUpsertUserBalance(ctx, payload.UserID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": fmt.Sprintf("DB update failed: %v", err)})
	}

	return c.Status(fiber.StatusCreated).JSON(doc)
}

// Helper functions
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
	case primitive.Decimal128:
		f, _ := strconv.ParseFloat(val.String(), 64)
		return int(f)
	default:
		return 0
	}
}

func toFloat(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	case primitive.Decimal128:
		f, _ := strconv.ParseFloat(val.String(), 64)
		return f
	default:
		return 0
	}
}

func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
