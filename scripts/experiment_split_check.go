package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"time"

	"github.com/umakantv/go-utils/httpclient"
)

type createExperimentRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	StartDate   string                 `json:"start_date"`
	EndDate     string                 `json:"end_date"`
	Variants    []createVariantRequest `json:"variants"`
}

type createVariantRequest struct {
	Name              string  `json:"name"`
	Description       string  `json:"description"`
	TrafficPercentage float64 `json:"traffic_percentage"`
}

type experimentResponse struct {
	ID       int               `json:"id"`
	Variants []variantResponse `json:"variants"`
}

type variantResponse struct {
	ID                int     `json:"id"`
	Name              string  `json:"name"`
	TrafficPercentage float64 `json:"traffic_percentage"`
}

type evaluateRequest struct {
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
}

type evaluateResponse struct {
	VariantName string `json:"variant_name"`
}

type updateExperimentRequest struct {
	Variants []updateVariantRequest `json:"variants"`
}

type updateVariantRequest struct {
	ID                int      `json:"id"`
	TrafficPercentage *float64 `json:"traffic_percentage,omitempty"`
}

func main() {
	baseURL := "http://localhost:8080"
	client := httpclient.New(httpclient.ClientConfig{
		BaseHeaders: map[string]string{
			"Authorization": "Bearer secret-token",
		},
	})

	headers := map[string]string{
		"Authorization": "Bearer secret-token",
		"Content-Type":  "application/json",
	}

	createPayload := createExperimentRequest{
		Name:        "homepage-layout-test-" + strconv.Itoa(int(time.Now().Unix())),
		Description: "Testing different homepage layouts",
		StartDate:   "2026-01-01T00:00:00Z",
		EndDate:     "2026-04-15T23:59:59Z",
		Variants: []createVariantRequest{
			{
				Name:              "control",
				Description:       "Current layout",
				TrafficPercentage: 50,
			},
			{
				Name:              "treatment-a",
				Description:       "New layout with hero section",
				TrafficPercentage: 30,
			},
			{
				Name:              "treatment-b",
				Description:       "New layout with grid layout",
				TrafficPercentage: 20,
			},
		},
	}

	createBody, err := json.Marshal(createPayload)
	if err != nil {
		log.Fatalf("failed to marshal create payload: %v", err)
	}

	var created experimentResponse
	if err := client.PostJSON(fmt.Sprintf("%s/experiments", baseURL), createBody, &created, httpclient.WithHeaders(headers)); err != nil {
		log.Fatalf("create experiment failed: %v", err)
	}

	if created.ID == 0 || len(created.Variants) == 0 {
		log.Fatalf("unexpected create response: %+v", created)
	}

	variantIDs := map[string]int{}
	for _, v := range created.Variants {
		variantIDs[v.Name] = v.ID
	}

	fmt.Printf("Created experiment ID: %d\n", created.ID)
	fmt.Println("Initial variant IDs:")
	printVariants(created.Variants)

	fmt.Println("\nInitial split for user IDs 1-100:")
	splitCounts := evaluateRange(client, baseURL, headers, created.ID, 1, 500)
	printSplitCounts(splitCounts)

	updatedSplits := map[string]float64{
		"control":     20,
		"treatment-a": 50,
		"treatment-b": 30,
	}

	updatePayload := updateExperimentRequest{
		Variants: []updateVariantRequest{
			{ID: variantIDs["control"], TrafficPercentage: floatPtr(updatedSplits["control"])},
			{ID: variantIDs["treatment-a"], TrafficPercentage: floatPtr(updatedSplits["treatment-a"])},
			{ID: variantIDs["treatment-b"], TrafficPercentage: floatPtr(updatedSplits["treatment-b"])},
		},
	}

	updateBody, err := json.Marshal(updatePayload)
	if err != nil {
		log.Fatalf("failed to marshal update payload: %v", err)
	}

	if err := client.PutJSON(fmt.Sprintf("%s/experiments/%d", baseURL, created.ID), updateBody, nil, httpclient.WithHeaders(headers)); err != nil {
		log.Fatalf("update experiment failed: %v", err)
	}

	fmt.Println("\nUpdated split for user IDs 101-200:")
	updatedCounts := evaluateRange(client, baseURL, headers, created.ID, 1001, 1500)
	printSplitCounts(updatedCounts)
}

func evaluateRange(client *httpclient.Client, baseURL string, headers map[string]string, experimentID int, startID int, endID int) map[string]int {
	counts := map[string]int{}
	for userID := startID; userID <= endID; userID++ {
		payload := evaluateRequest{
			EntityType: "user",
			EntityID:   fmt.Sprintf("%d", userID),
		}
		body, err := json.Marshal(payload)
		if err != nil {
			log.Fatalf("failed to marshal evaluate payload: %v", err)
		}

		var eval evaluateResponse
		if err := client.PostJSON(fmt.Sprintf("%s/experiments/%d/evaluate", baseURL, experimentID), body, &eval, httpclient.WithHeaders(headers)); err != nil {
			log.Fatalf("evaluate request failed for user %d: %v", userID, err)
		}

		counts[eval.VariantName]++
	}

	return counts
}

func printVariants(variants []variantResponse) {
	sort.Slice(variants, func(i, j int) bool { return variants[i].ID < variants[j].ID })
	for _, variant := range variants {
		fmt.Printf("- %s (ID %d, %0.1f%%)\n", variant.Name, variant.ID, variant.TrafficPercentage)
	}
}

func printSplitCounts(counts map[string]int) {
	keys := make([]string, 0, len(counts))
	for variant := range counts {
		keys = append(keys, variant)
	}
	sort.Strings(keys)
	for _, variant := range keys {
		fmt.Printf("- %s: %d\n", variant, counts[variant])
	}
}

func floatPtr(value float64) *float64 {
	return &value
}
