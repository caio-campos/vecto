//go:build example_enhanced_request
// +build example_enhanced_request

// This is a standalone example program. Each example file has its own main function
// and should be run individually: go run -tags example_enhanced_request enhanced_request_example.go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/caio-campos/vecto"
)

func main() {
	fmt.Println("=== Vecto Enhanced Request Examples ===")
	fmt.Println()

	pathParamsExample()
	formDataExample()
	queryStructExample()
	enhancedCombinedExample()
}

func pathParamsExample() {
	fmt.Println("1. Path Parameters Example")
	fmt.Println("--------------------------")

	client, err := vecto.New(vecto.Config{
		BaseURL: "https://jsonplaceholder.typicode.com",
		Timeout: 30 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	fmt.Println("Fetching post with ID 1 using path params...")
	res, err := client.Get(ctx, "/posts/{id}", &vecto.RequestOptions{
		PathParams: map[string]string{
			"id": "1",
		},
	})

	if err != nil {
		log.Printf("Request failed: %v", err)
		return
	}

	if res.IsSuccess() {
		type Post struct {
			ID     int    `json:"id"`
			UserID int    `json:"userId"`
			Title  string `json:"title"`
		}

		var post Post
		if err := res.JSON(&post); err != nil {
			log.Printf("Failed to parse JSON: %v", err)
			return
		}

		fmt.Printf("✓ Post ID: %d, Title: %s\n", post.ID, post.Title)
	}
	fmt.Println()
}

func formDataExample() {
	fmt.Println("2. Form Data Example")
	fmt.Println("--------------------")

	client, err := vecto.New(vecto.Config{
		BaseURL: "https://httpbin.org",
		Timeout: 30 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	fmt.Println("Sending form data...")
	res, err := client.Post(ctx, "/post", &vecto.RequestOptions{
		FormData: map[string]string{
			"username": "john_doe",
			"email":    "john@example.com",
			"age":      "30",
		},
	})

	if err != nil {
		log.Printf("Request failed: %v", err)
		return
	}

	if res.IsSuccess() {
		fmt.Println("✓ Form data submitted successfully")
		fmt.Printf("  Response status: %d\n", res.StatusCode)
	}
	fmt.Println()
}

func queryStructExample() {
	fmt.Println("3. Query from Struct Example")
	fmt.Println("-----------------------------")

	client, err := vecto.New(vecto.Config{
		BaseURL: "https://jsonplaceholder.typicode.com",
		Timeout: 30 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	type PostsQuery struct {
		UserID int      `query:"userId"`
		Limit  int      `query:"_limit,omitempty"`
		Tags   []string `query:"tags,omitempty"`
	}

	query := PostsQuery{
		UserID: 1,
		Limit:  5,
	}

	ctx := context.Background()

	fmt.Printf("Fetching posts for user %d with limit %d...\n", query.UserID, query.Limit)
	res, err := client.Get(ctx, "/posts", &vecto.RequestOptions{
		QueryStruct: query,
	})

	if err != nil {
		log.Printf("Request failed: %v", err)
		return
	}

	if res.IsSuccess() {
		type Post struct {
			ID     int    `json:"id"`
			UserID int    `json:"userId"`
			Title  string `json:"title"`
		}

		var posts []Post
		if err := res.JSON(&posts); err != nil {
			log.Printf("Failed to parse JSON: %v", err)
			return
		}

		fmt.Printf("✓ Fetched %d posts\n", len(posts))
		for i, post := range posts {
			if i >= 3 {
				fmt.Printf("  ... and %d more\n", len(posts)-3)
				break
			}
			fmt.Printf("  - Post %d: %s\n", post.ID, post.Title)
		}
	}
	fmt.Println()
}

func enhancedCombinedExample() {
	fmt.Println("4. Combined Features Example")
	fmt.Println("-----------------------------")

	client, err := vecto.New(vecto.Config{
		BaseURL: "https://api.example.com",
		Timeout: 30 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	type FilterQuery struct {
		Status string `query:"status"`
		Limit  int    `query:"limit"`
		Offset int    `query:"offset,omitempty"`
	}

	ctx := context.Background()

	fmt.Println("Making complex request with multiple features...")
	_, err = client.Get(ctx, "/api/v1/users/{userId}/orders", &vecto.RequestOptions{
		PathParams: map[string]string{
			"userId": "123",
		},
		QueryStruct: FilterQuery{
			Status: "completed",
			Limit:  10,
		},
		Headers: map[string]string{
			"X-Custom-Header": "value",
		},
	})

	if err != nil {
		fmt.Printf("Request would be sent to: /api/v1/users/123/orders?status=completed&limit=10\n")
	}

	fmt.Println("\n=== Examples completed ===")
}
