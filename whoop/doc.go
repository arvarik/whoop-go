// Package whoop provides a production-grade Go client for the WHOOP Developer API (v2).
//
// The client handles authentication, rate limiting (100 req/min via token bucket),
// automatic retries with exponential backoff on 429 responses, webhook signature
// verification via HMAC-SHA256, and cursor-based pagination.
//
// # Quick Start
//
//	client := whoop.NewClient(
//	    whoop.WithToken("your_oauth2_token"),
//	)
//
//	profile, err := client.User.GetBasicProfile(ctx)
//
// # Pagination
//
// List methods return page objects with a NextPage iterator:
//
//	page, _ := client.Cycle.List(ctx, &whoop.ListOptions{Limit: 25})
//	for {
//	    for _, c := range page.Records { /* process cycle */ }
//	    page, err = page.NextPage(ctx)
//	    if errors.Is(err, whoop.ErrNoNextPage) {
//	        break
//	    }
//	}
//
// # Webhooks
//
// Use ParseWebhook to validate and decode incoming WHOOP webhook payloads:
//
//	event, err := whoop.ParseWebhook(r, "webhook_secret")
package whoop
