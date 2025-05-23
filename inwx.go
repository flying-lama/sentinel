package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// InwxClient handles communication with the INWX API
type InwxClient struct {
	client  *http.Client
	cookies []*http.Cookie
	config  *Config
}

// InwxRequest represents a request to the INWX API
type InwxRequest struct {
	Method string      `json:"method"`
	Params interface{} `json:"params"`
}

// InwxLoginParams represents login parameters for INWX
type InwxLoginParams struct {
	User     string `json:"user"`
	Password string `json:"pass"`
}

// InwxUpdateParams represents parameters for updating a DNS record
type InwxUpdateParams struct {
	RecordID int    `json:"id"`
	Content  string `json:"content"`
}

// InwxInfoParams represents parameters for getting record info
type InwxInfoParams struct {
	RecordID int `json:"id"`
}

// InwxResponse represents a generic INWX API response
type InwxResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"msg"`
	ResData interface{} `json:"resData"`
}

// InwxRecordInfo represents DNS record information
type InwxRecordInfo struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
}

// NewInwxClient creates a new INWX API client
func NewInwxClient(config *Config) *InwxClient {
	return &InwxClient{
		client: &http.Client{},
		config: config,
	}
}

// Login authenticates with the INWX API
func (i *InwxClient) Login() error {
	loginReq := InwxRequest{
		Method: "account.login",
		Params: InwxLoginParams{
			User:     i.config.InwxUser,
			Password: i.config.InwxPassword,
		},
	}

	loginData, err := json.Marshal(loginReq)
	if err != nil {
		return fmt.Errorf("error marshaling login request: %v", err)
	}

	resp, err := i.client.Post(
		"https://api.domrobot.com/jsonrpc/",
		"application/json",
		bytes.NewBuffer(loginData),
	)
	if err != nil {
		return fmt.Errorf("login request failed: %v", err)
	}
	defer resp.Body.Close()

	// Get cookies for session
	i.cookies = resp.Cookies()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading login response: %v", err)
	}

	var loginResp InwxResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return fmt.Errorf("error parsing login response: %v", err)
	}

	if loginResp.Code != 1000 {
		return fmt.Errorf("login failed: %s (code %d)", loginResp.Message, loginResp.Code)
	}

	return nil
}

// GetRecordContent gets the current content of a DNS record
func (i *InwxClient) GetRecordContent() (string, error) {
	if i.cookies == nil {
		if err := i.Login(); err != nil {
			return "", fmt.Errorf("login failed: %v", err)
		}
	}

	infoReq := InwxRequest{
		Method: "nameserver.info",
		Params: InwxInfoParams{
			RecordID: i.config.RecordID,
		},
	}

	infoData, err := json.Marshal(infoReq)
	if err != nil {
		return "", fmt.Errorf("error marshaling info request: %v", err)
	}

	// Create a new request with cookies
	req, err := http.NewRequest(
		"POST",
		"https://api.domrobot.com/jsonrpc/",
		bytes.NewBuffer(infoData),
	)
	if err != nil {
		return "", fmt.Errorf("error creating info request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add cookies to request
	for _, cookie := range i.cookies {
		req.AddCookie(cookie)
	}

	// Send info request
	infoResp, err := i.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("info request failed: %v", err)
	}
	defer infoResp.Body.Close()

	// Read response
	infoBody, err := io.ReadAll(infoResp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading info response: %v", err)
	}

	var respData struct {
		Code    int    `json:"code"`
		Message string `json:"msg"`
		ResData struct {
			Record InwxRecordInfo `json:"record"`
		} `json:"resData"`
	}

	if err := json.Unmarshal(infoBody, &respData); err != nil {
		return "", fmt.Errorf("error parsing info response: %v", err)
	}

	if respData.Code != 1000 {
		// Session might have expired, try to login again
		if respData.Code == 1500 {
			i.cookies = nil
			return i.GetRecordContent()
		}
		return "", fmt.Errorf("info request failed: %s (code %d)", respData.Message, respData.Code)
	}

	return respData.ResData.Record.Content, nil
}

// UpdateDNS updates the DNS record with a new IP
func (i *InwxClient) UpdateDNS(newIP string) error {
	if i.cookies == nil {
		if err := i.Login(); err != nil {
			return fmt.Errorf("login failed: %v", err)
		}
	}

	updateReq := InwxRequest{
		Method: "nameserver.updateRecord",
		Params: InwxUpdateParams{
			RecordID: i.config.RecordID,
			Content:  newIP,
		},
	}

	updateData, err := json.Marshal(updateReq)
	if err != nil {
		return fmt.Errorf("error marshaling update request: %v", err)
	}

	// Create a new request with cookies
	req, err := http.NewRequest(
		"POST",
		"https://api.domrobot.com/jsonrpc/",
		bytes.NewBuffer(updateData),
	)
	if err != nil {
		return fmt.Errorf("error creating update request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add cookies to request
	for _, cookie := range i.cookies {
		req.AddCookie(cookie)
	}

	// Send update request
	updateResp, err := i.client.Do(req)
	if err != nil {
		return fmt.Errorf("update request failed: %v", err)
	}
	defer updateResp.Body.Close()

	// Read response
	updateBody, err := io.ReadAll(updateResp.Body)
	if err != nil {
		return fmt.Errorf("error reading update response: %v", err)
	}

	var respData InwxResponse
	if err := json.Unmarshal(updateBody, &respData); err != nil {
		return fmt.Errorf("error parsing update response: %v", err)
	}

	if respData.Code != 1000 {
		// Session might have expired, try to login again
		if respData.Code == 1500 {
			i.cookies = nil
			return i.UpdateDNS(newIP)
		}
		return fmt.Errorf("update failed: %s (code %d)", respData.Message, respData.Code)
	}

	log.Printf("DNS record updated successfully to %s", newIP)
	return nil
}
