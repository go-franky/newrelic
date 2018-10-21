package main

import (
	"log"
	"os"
	"time"

	"github.com/go-franky/newrelic/insights"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	client := insights.NewClient(nil,
		insights.AccountID(os.Getenv("INSIGHTS_ACCOUNT_ID")),
		insights.InsertKey(os.Getenv("INSIGHTS_INSERT_KEY")),
	)

	data := map[string]interface{}{
		"bool":     true,
		"duration": time.Duration(3123456),
		"float":    3.2,
		"int":      3,
		"string":   "foo",
		"time":     time.Now(),
	}
	if err := client.Publish("testEvent", data); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
