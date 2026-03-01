package whoop_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/arvarik/whoop-go/whoop"
)

// Create a client with default settings.
func ExampleNewClient() {
	client := whoop.NewClient(
		whoop.WithToken(os.Getenv("WHOOP_OAUTH_TOKEN")),
	)

	profile, err := client.User.GetBasicProfile(context.Background())
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("Hello,", profile.FirstName)
}

// Customize backoff, retries, and base URL using functional options.
func ExampleNewClient_withOptions() {
	client := whoop.NewClient(
		whoop.WithToken("your_token"),
		whoop.WithMaxRetries(5),
		whoop.WithBackoffBase(1*time.Second),
		whoop.WithBackoffMax(2*time.Minute),
		whoop.WithBaseURL("https://custom-proxy.example.com"),
	)
	_ = client
}

// Fetch the authenticated user's basic profile.
func ExampleUserService_GetBasicProfile() {
	client := whoop.NewClient(whoop.WithToken("your_token"))
	profile, err := client.User.GetBasicProfile(context.Background())
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("User: %s %s (ID: %d)\n", profile.FirstName, profile.LastName, profile.UserID)
}

// Fetch the authenticated user's body measurements.
func ExampleUserService_GetBodyMeasurement() {
	client := whoop.NewClient(whoop.WithToken("your_token"))
	body, err := client.User.GetBodyMeasurement(context.Background())
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("Height: %.2fm, Weight: %.1fkg, Max HR: %d\n",
		body.HeightMeter, body.WeightKilogram, body.MaxHeartRate)
}

// Iterate through all physiological cycles using cursor-based pagination.
func ExampleCycleService_List() {
	client := whoop.NewClient(whoop.WithToken("your_token"))
	ctx := context.Background()

	page, err := client.Cycle.List(ctx, &whoop.ListOptions{Limit: 25})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	for {
		for _, c := range page.Records {
			fmt.Printf("Cycle %d: Strain=%.1f, ScoreState=%s\n",
				c.ID, c.Score.Strain, c.ScoreState)
		}

		page, err = page.NextPage(ctx)
		if err != nil {
			if errors.Is(err, whoop.ErrNoNextPage) {
				break // All pages consumed
			}
			fmt.Println("error:", err)
			return
		}
	}
}

// Fetch a single cycle by its numeric ID.
func ExampleCycleService_GetByID() {
	client := whoop.NewClient(whoop.WithToken("your_token"))
	cycle, err := client.Cycle.GetByID(context.Background(), 12345)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("Cycle %d: Strain=%.1f\n", cycle.ID, cycle.Score.Strain)
}

// List recent workouts with date filtering.
func ExampleWorkoutService_List() {
	client := whoop.NewClient(whoop.WithToken("your_token"))
	ctx := context.Background()

	start := time.Now().AddDate(0, 0, -7) // Last 7 days
	page, err := client.Workout.List(ctx, &whoop.ListOptions{
		Limit: 10,
		Start: &start,
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	for _, w := range page.Records {
		fmt.Printf("Workout %s: %s, Strain=%.1f\n", w.ID, w.SportName, w.Score.Strain)
	}
}

// Fetch a single workout by its UUID.
func ExampleWorkoutService_GetByID() {
	client := whoop.NewClient(whoop.WithToken("your_token"))
	workout, err := client.Workout.GetByID(context.Background(), "abc-def-123")
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Printf("Workout: %s (%s)\n", workout.SportName, workout.ID)
	if workout.Score != nil && workout.Score.ZoneDuration != nil {
		fmt.Printf("  Zone 5 Time: %dms\n", workout.Score.ZoneDuration.ZoneFiveMilli)
	}
}

// Iterate through sleep events with pagination.
func ExampleSleepService_List() {
	client := whoop.NewClient(whoop.WithToken("your_token"))
	ctx := context.Background()

	page, err := client.Sleep.List(ctx, &whoop.ListOptions{Limit: 10})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	for _, s := range page.Records {
		fmt.Printf("Sleep %s: Nap=%v, ScoreState=%s\n", s.ID, s.Nap, s.ScoreState)
		if s.Score != nil {
			fmt.Printf("  Performance: %.0f%%, Efficiency: %.0f%%\n",
				s.Score.SleepPerformancePercentage,
				s.Score.SleepEfficiencyPercentage)
		}
	}
}

// Fetch a single sleep event by its UUID.
func ExampleSleepService_GetByID() {
	client := whoop.NewClient(whoop.WithToken("your_token"))
	sleep, err := client.Sleep.GetByID(context.Background(), "abc-def-456")
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Printf("Sleep %s: Nap=%v\n", sleep.ID, sleep.Nap)
	if sleep.Score != nil && sleep.Score.StageSummary != nil {
		rem := time.Duration(sleep.Score.StageSummary.TotalRemSleepTimeMilli) * time.Millisecond
		fmt.Printf("  REM sleep: %s\n", rem)
	}
}

// List recovery scores across recent cycles.
func ExampleRecoveryService_List() {
	client := whoop.NewClient(whoop.WithToken("your_token"))
	ctx := context.Background()

	page, err := client.Recovery.List(ctx, &whoop.ListOptions{Limit: 7})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	for _, r := range page.Records {
		if r.Score != nil {
			fmt.Printf("Cycle %d: Recovery=%.0f%%, HRV=%.1f, RHR=%.0f\n",
				r.CycleID,
				r.Score.RecoveryScore,
				r.Score.HrvRmssdMilli,
				r.Score.RestingHeartRate)
		}
	}
}

// Fetch recovery data for a specific cycle.
func ExampleRecoveryService_GetByID() {
	client := whoop.NewClient(whoop.WithToken("your_token"))
	recovery, err := client.Recovery.GetByID(context.Background(), 12345)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	if recovery.Score != nil {
		fmt.Printf("Recovery: %.0f%%, SpO2: %.0f%%\n",
			recovery.Score.RecoveryScore,
			recovery.Score.Spo2Percentage)
	}
}

// Securely verify and parse incoming WHOOP webhook payloads.
func ExampleParseWebhook() {
	http.HandleFunc("/whoop/webhook", func(w http.ResponseWriter, r *http.Request) {
		event, err := whoop.ParseWebhook(r, os.Getenv("WHOOP_WEBHOOK_SECRET"))
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		fmt.Printf("Event: type=%s, id=%s, user=%d\n",
			event.Type, event.ID, event.UserID)
		w.WriteHeader(http.StatusOK)
	})
}
