package main

import (
	"os"
	"strconv"
	"testing"
)

func TestInwxClient_GetRecordContent(t *testing.T) {
	// Skip test if credentials are not provided
	inwxUser := os.Getenv("TEST_INWX_USER")
	inwxPassword := os.Getenv("TEST_INWX_PASSWORD")
	recordIDStr := os.Getenv("TEST_INWX_RECORD_ID")

	if inwxUser == "" || inwxPassword == "" || recordIDStr == "" {
		t.Skip("Skipping test: TEST_INWX_USER, TEST_INWX_PASSWORD or TEST_INWX_RECORD_ID not set")
	}

	// Parse record ID
	recordID, err := strconv.Atoi(recordIDStr)
	if err != nil {
		t.Fatalf("Invalid TEST_INWX_RECORD_ID: %v", err)
	}

	c := NewInwxClient(&Config{
		InwxUser:     inwxUser,
		InwxPassword: inwxPassword,
		RecordID:     recordID,
		LogLevel:     "DEBUG",
		Record:       "lb",
	})

	content, err := c.GetRecordContent()
	if err != nil {
		t.Fatalf("GetRecordContent failed: %v", err)
	}

	t.Logf("Current record content: %s", content)
}

func TestInwxClient_UpdateDNS(t *testing.T) {
	// Skip test if credentials are not provided
	inwxUser := os.Getenv("TEST_INWX_USER")
	inwxPassword := os.Getenv("TEST_INWX_PASSWORD")
	recordIDStr := os.Getenv("TEST_INWX_RECORD_ID")
	testIP := os.Getenv("TEST_IP")

	if inwxUser == "" || inwxPassword == "" || recordIDStr == "" || testIP == "" {
		t.Skip("Skipping test: TEST_INWX_USER, TEST_INWX_PASSWORD, TEST_INWX_RECORD_ID or TEST_IP not set")
	}

	// Parse record ID
	recordID, err := strconv.Atoi(recordIDStr)
	if err != nil {
		t.Fatalf("Invalid TEST_INWX_RECORD_ID: %v", err)
	}

	c := NewInwxClient(&Config{
		InwxUser:     inwxUser,
		InwxPassword: inwxPassword,
		RecordID:     recordID,
		LogLevel:     "DEBUG",
		Record:       "lb",
	})

	if err := c.UpdateDNS(testIP); err != nil {
		t.Fatalf("UpdateDNS failed: %v", err)
	}

	t.Logf("DNS record updated successfully to %s", testIP)
}
