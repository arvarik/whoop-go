package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/arvarik/whoop-go/whoop"
)

// This example demonstrates how a developer running a webhook integration
// would securely parse WHOOP webhooks, instantly query the robust REST API
// specifically for that event's deeply nested metric payload, and sync the data out.
func main() {
	// Establish your core API Client setup
	client := whoop.NewClient(
		whoop.WithToken(os.Getenv("WHOOP_OAUTH_TOKEN")),
		whoop.WithMaxRetries(5),               // 5 resilient retries
		whoop.WithBackoffBase(1*time.Second),  // 1-second exponential base
		whoop.WithBackoffMax(120*time.Second), // 2-minute limit
	)

	webhookSecret := os.Getenv("WHOOP_WEBHOOK_SECRET")
	if webhookSecret == "" {
		log.Fatal("WHOOP_WEBHOOK_SECRET environment variable is required")
	}

	// Create a worker pool to process webhooks concurrently but with a limit
	// This prevents unbounded goroutine creation during traffic spikes
	jobQueue := make(chan string, 100)
	// Start 5 workers
	for i := 0; i < 5; i++ {
		go worker(client, jobQueue)
	}

	// Setup our HTTP receiver for the skinny webhook
	http.HandleFunc("/whoop/webhook", webhookHandler(client, webhookSecret, jobQueue))

	log.Println("Webhook Listener gracefully listening on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func webhookHandler(client *whoop.Client, webhookSecret string, jobQueue chan<- string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Securely parse and validate the HMAC-SHA256 skinny payload
		event, err := whoop.ParseWebhook(r, webhookSecret)
		if err != nil {
			log.Printf("Failed to process webhook securely: %v", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		log.Printf("Received genuine webhook event! Type: %s, ID: %s", event.Type, event.ID)

		// Acknowledge the webhook rapidly to WHOOP before processing the heavy REST pulls
		w.WriteHeader(http.StatusOK)

		// 2. We received a "workout.updated" payload on our webhook.
		// Go and fetch the explicit deeply-nested data array synchronously in the background.
		if event.Type == "workout.updated" {
			select {
			case jobQueue <- event.ID:
				// Successfully queued
			default:
				log.Printf("Worker pool full, dropping workout update for ID %s", event.ID)
			}
		}
	}
}

func worker(client *whoop.Client, jobQueue <-chan string) {
	for workoutID := range jobQueue {
		processWorkout(client, workoutID)
	}
}

// processWorkout utilizes the robust Client setup to fetch the WHOOP Workout by its ID
// dynamically triggered off a Webhook execution string.
func processWorkout(client *whoop.Client, workoutID string) {
	// 30 second context for standard API interactions
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	workout, err := client.Workout.GetByID(ctx, workoutID)
	if err != nil {
		log.Printf("[Webhook Worker] Failed to fetch REST API for Workout ID %s: %v", workoutID, err)
		return
	}

	if workout.Score != nil {
		var dist float64
		if workout.Score.DistanceMeter != nil {
			dist = *workout.Score.DistanceMeter
		}
		log.Printf("[Webhook Worker] Workout Processed: ID=%s, Strain=%.2f, MaxHR=%d, Distance=%.2fm",
			workout.ID,
			workout.Score.Strain,
			workout.Score.MaxHeartRate,
			dist,
		)
	} else {
		log.Printf("[Webhook Worker] Workout Processed: ID=%s (Processing Score...)", workout.ID)
	}

	// TODO: Save this `workout` payload locally to a database or generic store!
}
