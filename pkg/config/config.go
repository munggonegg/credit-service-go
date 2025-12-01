package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	MongoURL      string
	MongoDBName   string
	PortkeyAPIKey string
	PortkeyURL    string
}

var AppConfig Config

const (
	PaymentColl           = "payment_transactions"
	SubsColl              = "subscription_transactions"
	UsageEventColl        = "user_usage_event"
	UserBalanceColl       = "user_balance"
	PackageMasterV3Coll   = "package_master_v3"
	ProviderMasterColl    = "provider_master"
	ProviderModelsColl    = "provider_models"
	PackageBestModelsColl = "package_best_models"
	UserMainPackageColl   = "user_main_package"
	UserTopupPackageColl  = "user_topup_package"
	PackageScheduleColl   = "user_package_schedule"
	B2BScheduleColl       = "b2b_package_schedule"
	TopupPackageEventColl = "topup_package_event"
	SubsPackageEventColl  = "subscription_package_event"
)

func LoadConfig() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	AppConfig = Config{
		MongoURL:      os.Getenv("MONGO_URL"),
		MongoDBName:   os.Getenv("MONGO_DB_NAME"),
		PortkeyAPIKey: os.Getenv("PORTKEY_API_KEY"),
		PortkeyURL:    os.Getenv("PORTKEY_URL"),
	}

	if AppConfig.MongoURL == "" || AppConfig.MongoDBName == "" {
		log.Fatal("Please set MONGO_URL and MONGO_DB_NAME env vars.")
	}
}
