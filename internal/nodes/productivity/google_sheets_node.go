package productivity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/dipankar/m9m/internal/model"
	"github.com/dipankar/m9m/internal/nodes/base"
)

// GoogleSheetsNode provides Google Sheets API integration
type GoogleSheetsNode struct {
	*base.BaseNode
	httpClient *http.Client
}

// NewGoogleSheetsNode creates a new Google Sheets node
func NewGoogleSheetsNode() *GoogleSheetsNode {
	return &GoogleSheetsNode{
		BaseNode: base.NewBaseNode(base.NodeDescription{
			Name:        "Google Sheets",
			Description: "Read, write, and manipulate Google Sheets data",
			Category:    "productivity",
		}),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Execute runs the node
func (n *GoogleSheetsNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	resource, _ := nodeParams["resource"].(string)
	if resource == "" {
		resource = "sheet"
	}

	operation, _ := nodeParams["operation"].(string)
	if operation == "" {
		operation = "read"
	}

	// Get access token from credentials
	accessToken := n.getAccessToken(nodeParams)
	if accessToken == "" {
		return nil, fmt.Errorf("no valid credentials provided")
	}

	var results []model.DataItem

	for _, item := range inputData {
		var result map[string]interface{}
		var err error

		switch resource {
		case "sheet":
			result, err = n.handleSheetOperation(operation, accessToken, nodeParams, item)
		case "spreadsheet":
			result, err = n.handleSpreadsheetOperation(operation, accessToken, nodeParams, item)
		default:
			return nil, fmt.Errorf("unsupported resource: %s", resource)
		}

		if err != nil {
			return nil, err
		}

		results = append(results, model.DataItem{JSON: result})
	}

	return results, nil
}

func (n *GoogleSheetsNode) getAccessToken(nodeParams map[string]interface{}) string {
	if creds, ok := nodeParams["credentials"].(map[string]interface{}); ok {
		if token, ok := creds["oauthToken"].(string); ok && token != "" {
			return token
		}
		if token, ok := creds["accessToken"].(string); ok && token != "" {
			return token
		}
	}
	return ""
}

func (n *GoogleSheetsNode) handleSheetOperation(operation, token string, nodeParams map[string]interface{}, item model.DataItem) (map[string]interface{}, error) {
	spreadsheetId, _ := nodeParams["spreadsheetId"].(string)
	if spreadsheetId == "" {
		if id, ok := item.JSON["spreadsheetId"].(string); ok {
			spreadsheetId = id
		}
	}
	if spreadsheetId == "" {
		return nil, fmt.Errorf("spreadsheetId is required")
	}

	sheetRange, _ := nodeParams["range"].(string)
	if sheetRange == "" {
		sheetRange = "A:Z"
	}

	switch operation {
	case "read":
		return n.readData(token, spreadsheetId, sheetRange, nodeParams)
	case "append":
		return n.appendData(token, spreadsheetId, sheetRange, nodeParams, item)
	case "update":
		return n.updateData(token, spreadsheetId, sheetRange, nodeParams, item)
	case "clear":
		return n.clearData(token, spreadsheetId, sheetRange)
	default:
		return nil, fmt.Errorf("unsupported sheet operation: %s", operation)
	}
}

func (n *GoogleSheetsNode) handleSpreadsheetOperation(operation, token string, nodeParams map[string]interface{}, item model.DataItem) (map[string]interface{}, error) {
	switch operation {
	case "create":
		title, _ := nodeParams["title"].(string)
		if title == "" {
			title = "New Spreadsheet"
		}
		return n.createSpreadsheet(token, title)
	case "get":
		spreadsheetId, _ := nodeParams["spreadsheetId"].(string)
		if spreadsheetId == "" {
			return nil, fmt.Errorf("spreadsheetId is required")
		}
		return n.getSpreadsheet(token, spreadsheetId)
	default:
		return nil, fmt.Errorf("unsupported spreadsheet operation: %s", operation)
	}
}

func (n *GoogleSheetsNode) readData(token, spreadsheetId, sheetRange string, nodeParams map[string]interface{}) (map[string]interface{}, error) {
	apiURL := fmt.Sprintf(
		"https://sheets.googleapis.com/v4/spreadsheets/%s/values/%s",
		spreadsheetId,
		url.QueryEscape(sheetRange),
	)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		return nil, fmt.Errorf("Google Sheets API error: %v", errorResp)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func (n *GoogleSheetsNode) appendData(token, spreadsheetId, sheetRange string, nodeParams map[string]interface{}, item model.DataItem) (map[string]interface{}, error) {
	values := nodeParams["values"]
	if values == nil {
		// Use item data as values
		values = [][]interface{}{{item.JSON}}
	}

	queryParams := url.Values{
		"valueInputOption": {"USER_ENTERED"},
		"insertDataOption": {"INSERT_ROWS"},
	}

	apiURL := fmt.Sprintf(
		"https://sheets.googleapis.com/v4/spreadsheets/%s/values/%s:append?%s",
		spreadsheetId,
		url.QueryEscape(sheetRange),
		queryParams.Encode(),
	)

	body := map[string]interface{}{
		"values": values,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		return nil, fmt.Errorf("Google Sheets API error: %v", errorResp)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func (n *GoogleSheetsNode) updateData(token, spreadsheetId, sheetRange string, nodeParams map[string]interface{}, item model.DataItem) (map[string]interface{}, error) {
	values := nodeParams["values"]
	if values == nil {
		values = [][]interface{}{{item.JSON}}
	}

	queryParams := url.Values{
		"valueInputOption": {"USER_ENTERED"},
	}

	apiURL := fmt.Sprintf(
		"https://sheets.googleapis.com/v4/spreadsheets/%s/values/%s?%s",
		spreadsheetId,
		url.QueryEscape(sheetRange),
		queryParams.Encode(),
	)

	body := map[string]interface{}{
		"values": values,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		return nil, fmt.Errorf("Google Sheets API error: %v", errorResp)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func (n *GoogleSheetsNode) clearData(token, spreadsheetId, sheetRange string) (map[string]interface{}, error) {
	apiURL := fmt.Sprintf(
		"https://sheets.googleapis.com/v4/spreadsheets/%s/values/%s:clear",
		spreadsheetId,
		url.QueryEscape(sheetRange),
	)

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader([]byte("{}")))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		return nil, fmt.Errorf("Google Sheets API error: %v", errorResp)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func (n *GoogleSheetsNode) createSpreadsheet(token, title string) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"properties": map[string]interface{}{
			"title": title,
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://sheets.googleapis.com/v4/spreadsheets", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		return nil, fmt.Errorf("Google Sheets API error: %v", errorResp)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func (n *GoogleSheetsNode) getSpreadsheet(token, spreadsheetId string) (map[string]interface{}, error) {
	apiURL := fmt.Sprintf("https://sheets.googleapis.com/v4/spreadsheets/%s", spreadsheetId)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		return nil, fmt.Errorf("Google Sheets API error: %v", errorResp)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}
