package github

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestConfigHTTPClientBehavior tests that configuration attributes actually affect HTTP client behavior
func TestConfigHTTPClientBehavior(t *testing.T) {
	t.Run("read_delay affects request timing", func(t *testing.T) {
		config := &Config{
			BaseURL:   "https://enterprise.example.com/",
			ReadDelay: 200 * time.Millisecond,
		}

		// Create a mock server that records request times
		requestTimes := []time.Time{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestTimes = append(requestTimes, time.Now())
			w.WriteHeader(200)
			if _, err := w.Write([]byte(`{"message": "ok"}`)); err != nil {
				t.Errorf("Failed to write response: %v", err)
			}
		}))
		defer server.Close()

		// Update config to use mock server
		config.BaseURL = server.URL + "/"

		client := config.AnonymousHTTPClient()

		// Make multiple GET requests (read requests)
		for range 3 {
			resp, err := client.Get(server.URL)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			resp.Body.Close()
		}

		// Verify delay between requests
		if len(requestTimes) < 2 {
			t.Fatal("Not enough requests recorded")
		}

		for i := 1; i < len(requestTimes); i++ {
			delay := requestTimes[i].Sub(requestTimes[i-1])
			// Allow some tolerance for timing variations
			if delay < 180*time.Millisecond || delay > 250*time.Millisecond {
				t.Errorf("Expected delay around 200ms, got %v", delay)
			}
		}
	})

	t.Run("write_delay affects POST request timing", func(t *testing.T) {
		config := &Config{
			BaseURL:    "https://enterprise.example.com/",
			WriteDelay: 300 * time.Millisecond,
		}

		requestTimes := []time.Time{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestTimes = append(requestTimes, time.Now())
			w.WriteHeader(201)
			if _, err := w.Write([]byte(`{"message": "created"}`)); err != nil {
				t.Errorf("Failed to write response: %v", err)
			}
		}))
		defer server.Close()

		config.BaseURL = server.URL + "/"
		client := config.AnonymousHTTPClient()

		// Make multiple POST requests (write requests)
		for range 3 {
			resp, err := client.Post(server.URL, "application/json", nil)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			resp.Body.Close()
		}

		// Verify delay between write requests
		if len(requestTimes) < 2 {
			t.Fatal("Not enough requests recorded")
		}

		for i := 1; i < len(requestTimes); i++ {
			delay := requestTimes[i].Sub(requestTimes[i-1])
			// Allow some tolerance for timing variations
			if delay < 280*time.Millisecond || delay > 350*time.Millisecond {
				t.Errorf("Expected delay around 300ms, got %v", delay)
			}
		}
	})

	t.Run("parallel_requests affects locking behavior", func(t *testing.T) {
		// Test serial requests (parallel_requests = false)
		serialConfig := &Config{
			BaseURL:          "https://enterprise.example.com/",
			ParallelRequests: false,
			WriteDelay:       50 * time.Millisecond,
		}

		requestTimes := []time.Time{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestTimes = append(requestTimes, time.Now())
			// Simulate some processing time
			time.Sleep(10 * time.Millisecond)
			w.WriteHeader(200)
			if _, err := w.Write([]byte(`{"message": "ok"}`)); err != nil {
				t.Errorf("Failed to write response: %v", err)
			}
		}))
		defer server.Close()

		serialConfig.BaseURL = server.URL + "/"
		serialClient := serialConfig.AnonymousHTTPClient()

		// Make concurrent requests - they should be serialized
		done := make(chan bool, 2)

		go func() {
			resp, _ := serialClient.Post(server.URL, "application/json", nil)
			if resp != nil {
				resp.Body.Close()
			}
			done <- true
		}()

		go func() {
			resp, _ := serialClient.Post(server.URL, "application/json", nil)
			if resp != nil {
				resp.Body.Close()
			}
			done <- true
		}()

		// Wait for completion
		<-done
		<-done

		// With serial requests, the second request should be delayed
		if len(requestTimes) >= 2 {
			delay := requestTimes[1].Sub(requestTimes[0])
			// Should include write delay + processing time
			if delay < 50*time.Millisecond {
				t.Errorf("Expected significant delay for serial requests, got %v", delay)
			}
		}
	})

	t.Run("retry configuration affects retry behavior", func(t *testing.T) {
		config := &Config{
			BaseURL:    "https://enterprise.example.com/",
			MaxRetries: 2,
			RetryDelay: 100 * time.Millisecond,
			RetryableErrors: map[int]bool{
				500: true,
				502: true,
			},
		}

		requestCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			if requestCount <= 2 {
				// Return 500 for first two requests
				w.WriteHeader(500)
				if _, err := w.Write([]byte(`{"message": "server error"}`)); err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			} else {
				// Succeed on third request
				w.WriteHeader(200)
				if _, err := w.Write([]byte(`{"message": "success"}`)); err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}
		}))
		defer server.Close()

		config.BaseURL = server.URL + "/"
		client := config.AnonymousHTTPClient()

		startTime := time.Now()
		resp, err := client.Get(server.URL)
		duration := time.Since(startTime)

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		// Should have made 3 requests (initial + 2 retries)
		if requestCount != 3 {
			t.Errorf("Expected 3 requests (1 + 2 retries), got %d", requestCount)
		}

		// Should have taken at least 2 * retry_delay (2 * 100ms)
		if duration < 200*time.Millisecond {
			t.Errorf("Expected at least 200ms duration for retries, got %v", duration)
		}

		// Final response should be successful
		if resp.StatusCode != 200 {
			t.Errorf("Expected final response to be 200, got %d", resp.StatusCode)
		}
	})

	t.Run("insecure mode skips TLS verification", func(t *testing.T) {
		// Create a TLS server with self-signed certificate
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			if _, err := w.Write([]byte(`{"message": "secure"}`)); err != nil {
				t.Errorf("Failed to write response: %v", err)
			}
		}))
		defer server.Close()

		// Test with insecure = false (should fail with self-signed cert)
		secureConfig := &Config{
			BaseURL:  server.URL + "/",
			Insecure: false,
		}
		secureClient := secureConfig.AnonymousHTTPClient()

		_, err := secureClient.Get(server.URL)
		if err == nil {
			t.Error("Expected TLS error with secure client, but request succeeded")
		}

		// Test with insecure = true (should succeed despite self-signed cert)
		insecureConfig := &Config{
			BaseURL:  server.URL + "/",
			Insecure: true,
		}
		insecureClient := insecureConfig.AnonymousHTTPClient()

		resp, err := insecureClient.Get(server.URL)
		if err != nil {
			t.Errorf("Expected insecure client to succeed, but got error: %v", err)
		}
		if resp != nil {
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				t.Errorf("Expected 200 status, got %d", resp.StatusCode)
			}
		}
	})

	t.Run("authenticated client uses insecure mode", func(t *testing.T) {
		// Create a TLS server with self-signed certificate
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check for Authorization header
			auth := r.Header.Get("Authorization")
			if auth != "Bearer test_token" {
				w.WriteHeader(401)
				return
			}
			w.WriteHeader(200)
			if _, err := w.Write([]byte(`{"message": "authenticated"}`)); err != nil {
				t.Errorf("Failed to write response: %v", err)
			}
		}))
		defer server.Close()

		config := &Config{
			Token:    "test_token",
			BaseURL:  server.URL + "/",
			Insecure: true,
		}

		client := config.AuthenticatedHTTPClient()
		resp, err := client.Get(server.URL)
		if err != nil {
			t.Errorf("Expected authenticated insecure client to succeed, but got error: %v", err)
		}
		if resp != nil {
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				t.Errorf("Expected 200 status, got %d", resp.StatusCode)
			}
		}
	})
}

// TestConfigRetryableErrorsBehavior tests that only configured status codes are retried
func TestConfigRetryableErrorsBehavior(t *testing.T) {
	testCases := []struct {
		name            string
		serverResponse  int
		retryableErrors map[int]bool
		maxRetries      int
		expectedRetries int
	}{
		{
			name:           "500 in retryable errors should retry",
			serverResponse: 500,
			retryableErrors: map[int]bool{
				500: true,
				502: true,
			},
			maxRetries:      2,
			expectedRetries: 3, // initial + 2 retries
		},
		{
			name:           "404 not in retryable errors should not retry",
			serverResponse: 404,
			retryableErrors: map[int]bool{
				500: true,
				502: true,
			},
			maxRetries:      2,
			expectedRetries: 1, // initial only
		},
		{
			name:           "429 in custom retryable errors should retry",
			serverResponse: 429,
			retryableErrors: map[int]bool{
				429: true,
				500: true,
			},
			maxRetries:      1,
			expectedRetries: 2, // initial + 1 retry
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestCount++
				w.WriteHeader(tc.serverResponse)
				if _, err := w.Write(fmt.Appendf(nil, `{"error": "status %d"}`, tc.serverResponse)); err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}))
			defer server.Close()

			config := &Config{
				BaseURL:         server.URL + "/",
				MaxRetries:      tc.maxRetries,
				RetryDelay:      10 * time.Millisecond, // Short delay for testing
				RetryableErrors: tc.retryableErrors,
			}

			client := config.AnonymousHTTPClient()
			resp, _ := client.Get(server.URL)
			if resp != nil {
				resp.Body.Close()
			}

			if requestCount != tc.expectedRetries {
				t.Errorf("Expected %d requests, got %d", tc.expectedRetries, requestCount)
			}
		})
	}
}

// TestConfigHTTPTransportIntegration tests the integration of all transport layers
func TestConfigHTTPTransportIntegration(t *testing.T) {
	config := &Config{
		BaseURL:    "https://enterprise.example.com/",
		ReadDelay:  50 * time.Millisecond,
		WriteDelay: 100 * time.Millisecond,
		MaxRetries: 1,
		RetryDelay: 25 * time.Millisecond,
		RetryableErrors: map[int]bool{
			500: true,
		},
		ParallelRequests: false,
	}

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			// First request fails
			w.WriteHeader(500)
		} else {
			// Second request succeeds
			w.WriteHeader(200)
		}
		if _, err := w.Write([]byte(`{"message": "response"}`)); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	config.BaseURL = server.URL + "/"
	client := config.AnonymousHTTPClient()

	startTime := time.Now()
	resp, err := client.Get(server.URL)
	duration := time.Since(startTime)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Should have made 2 requests (initial + 1 retry)
	if requestCount != 2 {
		t.Errorf("Expected 2 requests, got %d", requestCount)
	}

	// Should have taken at least retry_delay (25ms)
	if duration < 25*time.Millisecond {
		t.Errorf("Expected at least 25ms duration, got %v", duration)
	}

	// Final response should be successful
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200 status, got %d", resp.StatusCode)
	}
}

// TestTransportChaining verifies that all transports are properly chained
func TestTransportChaining(t *testing.T) {
	config := &Config{
		BaseURL:          "https://enterprise.example.com/",
		WriteDelay:       10 * time.Millisecond,
		ParallelRequests: false,
		MaxRetries:       1,
		RetryDelay:       5 * time.Millisecond,
		RetryableErrors:  map[int]bool{502: true},
	}

	client := config.AnonymousHTTPClient()

	// Verify transport chain by checking the types
	transport := client.Transport

	// Should have RetryTransport wrapping other transports
	if _, ok := transport.(*RetryTransport); !ok {
		t.Error("Expected outermost transport to be RetryTransport")
	}

	// The exact chain is: RetryTransport -> PreviewHeaderTransport -> RateLimitTransport -> EtagTransport -> HTTP Transport
	// This test verifies that the chain exists by checking that we can make successful requests

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		if _, err := w.Write([]byte(`{"test": "success"}`)); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	// The transport chain is properly established during client creation
	// We don't need to manually modify the chain for this test

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Errorf("Request through transport chain failed: %v", err)
	}
	if resp != nil {
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Errorf("Expected 200 status, got %d", resp.StatusCode)
		}
	}
}
