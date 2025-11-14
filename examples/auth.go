//go:build example_auth
// +build example_auth

// This is a standalone example program. Each example file has its own main function
// and should be run individually: go run -tags example_auth auth_example.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/caio-campos/vecto"
)

func main() {
	fmt.Println("=== Vecto Auth Examples ===")
	fmt.Println()

	basicAuthExample()
	bearerTokenExample()
	requestLevelAuthExample()
}

func basicAuthExample() {
	fmt.Println("1. Basic Authentication Example")
	fmt.Println("--------------------------------")

	client, err := vecto.New(vecto.Config{
		BaseURL: "https://httpbin.org",
		Timeout: 30 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	client.SetBasicAuth("user", "password")

	ctx := context.Background()
	res, err := client.Get(ctx, "/basic-auth/user/password", nil)
	if err != nil {
		log.Printf("Request failed: %v", err)
		return
	}

	if res.IsSuccess() {
		var result map[string]interface{}
		if err := res.JSON(&result); err != nil {
			log.Printf("Failed to parse JSON: %v", err)
			return
		}

		prettyJSON, _ := json.MarshalIndent(result, "", "  ")
		fmt.Printf("Success! Response:\n%s\n\n", prettyJSON)
	} else {
		fmt.Printf("Authentication failed with status: %d\n\n", res.StatusCode)
	}
}

func bearerTokenExample() {
	fmt.Println("2. Bearer Token Authentication Example")
	fmt.Println("---------------------------------------")

	client, err := vecto.New(vecto.Config{
		BaseURL: "https://httpbin.org",
		Timeout: 30 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	client.SetBearerToken("my-secure-token-123")

	ctx := context.Background()
	res, err := client.Get(ctx, "/bearer", nil)
	if err != nil {
		log.Printf("Request failed: %v", err)
		return
	}

	if res.IsSuccess() {
		var result map[string]interface{}
		if err := res.JSON(&result); err != nil {
			log.Printf("Failed to parse JSON: %v", err)
			return
		}

		prettyJSON, _ := json.MarshalIndent(result, "", "  ")
		fmt.Printf("Success! Response:\n%s\n\n", prettyJSON)
	} else {
		fmt.Printf("Authentication failed with status: %d\n\n", res.StatusCode)
	}
}

func requestLevelAuthExample() {
	fmt.Println("3. Request-Level Authentication Example")
	fmt.Println("----------------------------------------")

	client, err := vecto.New(vecto.Config{
		BaseURL: "https://api.github.com",
		Timeout: 30 * time.Second,
		Headers: map[string]string{
			"Accept": "application/vnd.github.v3+json",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	fmt.Println("Making request to GitHub API without auth...")
	res1, err := client.Get(ctx, "/user", nil)
	if err != nil {
		log.Printf("Request failed: %v", err)
	} else {
		fmt.Printf("Status: %d (expected 401)\n", res1.StatusCode)
	}

	fmt.Println("\nSwitching authentication methods per request...")

	res2, err := client.Get(ctx, "/", &vecto.RequestOptions{
		Headers: map[string]string{
			"Authorization": "token your-github-token-here",
		},
	})
	if err != nil {
		log.Printf("Request failed: %v", err)
		return
	}

	if res2.IsSuccess() {
		fmt.Printf("Authenticated request succeeded with status: %d\n", res2.StatusCode)
	}

	fmt.Println("\n=== Examples completed ===")
}
