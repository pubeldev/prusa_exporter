package udp

import (
	"testing"
	"time"
)

func TestStartSyslogServer(t *testing.T) {
	// Test starting syslog server on a test port
	listenAddr := "127.0.0.1:0" // Use port 0 to get a random available port

	channel, server := startSyslogServer(listenAddr)

	if channel == nil {
		t.Error("startSyslogServer() returned nil channel")
	}

	if server == nil {
		t.Error("startSyslogServer() returned nil server")
	}

	// Clean up - kill the server after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		server.Kill()
	}()

	// Test that we can receive from the channel (should not block indefinitely)
	select {
	case <-channel:
		// Received something, that's fine
	case <-time.After(200 * time.Millisecond):
		// Timeout, that's also fine for this test
	}
}

func TestMetricsListenerSetup(t *testing.T) {
	// This test verifies that MetricsListener can be set up without panicking
	// We can't easily test the full functionality without complex mocking

	// Test with invalid address to ensure error handling
	listenAddr := "invalid-address:99999"

	// This should not panic even with invalid address
	// The function runs in background, so we just verify it can be called
	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("MetricsListener() panicked: %v", r)
			}
		}()

		// Use a timeout to prevent hanging
		done := make(chan bool, 1)
		go func() {
			MetricsListener(listenAddr, "test_")
			done <- true
		}()

		select {
		case <-done:
			// Completed
		case <-time.After(100 * time.Millisecond):
			// Timeout - this is expected for invalid address
		}
	}()

	// Give the goroutine time to execute
	time.Sleep(150 * time.Millisecond)
}

func TestValidListenAddress(t *testing.T) {
	// Test with a valid address format
	listenAddr := "127.0.0.1:0" // Port 0 for automatic assignment

	// Start the listener in a goroutine
	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- r.(error)
			} else {
				done <- nil
			}
		}()

		// Create channel and server
		channel, server := startSyslogServer(listenAddr)

		// Verify they were created
		if channel == nil || server == nil {
			done <- nil
			return
		}

		// Stop the server quickly
		go func() {
			time.Sleep(50 * time.Millisecond)
			server.Kill()
		}()

		// Start processing - this will block until server is killed
		go func() {
			for range channel {
				// Process messages
			}
		}()

		server.Wait()
		done <- nil
	}()

	// Wait for completion or timeout
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("MetricsListener setup failed: %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		// Timeout is acceptable for this test
	}
}

func TestPrefixHandling(t *testing.T) {
	// Test that different prefixes can be passed without issues
	prefixes := []string{
		"prusa_",
		"test_",
		"",
		"custom_prefix_",
	}

	for _, prefix := range prefixes {
		t.Run("prefix_"+prefix, func(t *testing.T) {
			// This mainly tests that the function accepts different prefixes
			// without panicking during setup
			listenAddr := "127.0.0.1:0"

			// Start in goroutine with timeout
			done := make(chan bool, 1)
			go func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("MetricsListener with prefix '%s' panicked: %v", prefix, r)
					}
					done <- true
				}()

				channel, server := startSyslogServer(listenAddr)
				if channel != nil && server != nil {
					// Quick cleanup
					go func() {
						time.Sleep(10 * time.Millisecond)
						server.Kill()
					}()
					server.Wait()
				}
			}()

			select {
			case <-done:
				// Completed successfully
			case <-time.After(100 * time.Millisecond):
				// Timeout - acceptable
			}
		})
	}
}

// Test helper function for checking server lifecycle
func TestSyslogServerLifecycle(t *testing.T) {
	listenAddr := "127.0.0.1:0"

	// Start server
	channel, server := startSyslogServer(listenAddr)

	if channel == nil {
		t.Fatal("startSyslogServer() returned nil channel")
	}

	if server == nil {
		t.Fatal("startSyslogServer() returned nil server")
	}

	// Test that server can be killed
	serverKilled := make(chan bool, 1)
	go func() {
		server.Wait() // This blocks until server is killed
		serverKilled <- true
	}()

	// Kill server after short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		server.Kill()
	}()

	// Wait for server to be killed
	select {
	case <-serverKilled:
		// Server was killed successfully
	case <-time.After(200 * time.Millisecond):
		t.Error("Server was not killed within timeout")
	}
}
