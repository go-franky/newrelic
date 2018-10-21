package main

import (
	"log"
	"net/http"
	"os"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-franky/newrelic/insights"
	"github.com/joho/godotenv"
)

type query struct {
	Results []struct {
		Average float32 `json:"average"`
	} `json:"results"`
}

func main() {
	godotenv.Load()
	client := clientV1() // Can also use clientV2() here for different client instantiation

	result := &query{}

	if err := client.Query("SELECT average(duration) FROM Transaction", result); err != nil {
		log.Fatalf("Error: %v", err)
	}
	spew.Dump(result)
}

// clientV1 shows the most common way of instantiating a client
func clientV1() *insights.Client {
	client := insights.NewClient(nil,
		insights.AccountID(os.Getenv("INSIGHTS_ACCOUNT_ID")),
		insights.QueryKey(os.Getenv("INSIGHTS_QUERY_KEY")),
	)
	return client
}

// clientV2 shows an alternate way of instantiating a client
func clientV2() *insights.Client {
	client := insights.NewClient(
		&myClient{
			client: http.DefaultClient,
		},
		insights.AccountID(os.Getenv("INSIGHTS_ACCOUNT_ID")),
	)
	return client
}

type myClient struct {
	client *http.Client
}

func (c *myClient) Do(r *http.Request) (*http.Response, error) {
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Query-Key", os.Getenv("INSIGHTS_QUERY_KEY"))
	return c.client.Do(r)
}
