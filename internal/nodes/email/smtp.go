package email

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime"
	"net/smtp"
	"path/filepath"
	"strings"
	"time"

	"github.com/neul-labs/m9m/internal/expressions"
	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
)

// SMTPNode sends emails via SMTP
type SMTPNode struct {
	*base.BaseNode
	evaluator *expressions.GojaExpressionEvaluator
}

// SMTPConfig holds SMTP server configuration
type SMTPConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	TLS      bool   `json:"tls"`
	Auth     string `json:"auth"` // plain, login, crammd5
}

// EmailMessage represents an email message
type EmailMessage struct {
	To          []string          `json:"to"`
	CC          []string          `json:"cc"`
	BCC         []string          `json:"bcc"`
	From        string            `json:"from"`
	ReplyTo     string            `json:"replyTo"`
	Subject     string            `json:"subject"`
	Body        string            `json:"body"`
	IsHTML      bool              `json:"isHTML"`
	Attachments []EmailAttachment `json:"attachments"`
	Headers     map[string]string `json:"headers"`
}

// EmailAttachment represents an email attachment
type EmailAttachment struct {
	Name     string `json:"name"`
	Content  []byte `json:"content"`
	MimeType string `json:"mimeType"`
}

// NewSMTPNode creates a new SMTP node
func NewSMTPNode() *SMTPNode {
	return &SMTPNode{
		BaseNode:  base.NewBaseNode(base.NodeDescription{Name: "Send Email", Description: "n8n-nodes-base.smtp", Category: "core"}),
		evaluator: expressions.NewGojaExpressionEvaluator(expressions.DefaultEvaluatorConfig()),
	}
}

// Execute sends emails via SMTP
func (n *SMTPNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	var results []model.DataItem

	// Get SMTP configuration
	smtpConfig, err := n.getSMTPConfig(nodeParams)
	if err != nil {
		return nil, n.CreateError("invalid SMTP configuration", map[string]interface{}{
			"error": err.Error(),
		})
	}

	for index, item := range inputData {
		// Create expression context
		context := &expressions.ExpressionContext{
			ActiveNodeName:      "Send Email",
			RunIndex:           0,
			ItemIndex:          index,
			Mode:               expressions.ModeManual,
			ConnectionInputData: []model.DataItem{item},
			AdditionalKeys: &expressions.AdditionalKeys{},
		}

		// Build email message
		message, err := n.buildEmailMessage(nodeParams, context)
		if err != nil {
			return nil, n.CreateError("failed to build email message", map[string]interface{}{
				"error": err.Error(),
			})
		}

		// Add attachments from binary data if available
		if item.Binary != nil && len(item.Binary) > 0 {
			for binaryKey, bd := range item.Binary {
				attachmentName := bd.FileName
				if attachmentName == "" {
					attachmentName, _ = nodeParams["attachmentName"].(string)
					if attachmentName == "" {
						attachmentName = binaryKey + ".bin"
					}
				}

				// Decode base64 binary data
				content, err := base64.StdEncoding.DecodeString(bd.Data)
				if err != nil {
					return nil, n.CreateError("failed to decode binary attachment", map[string]interface{}{
						"key":   binaryKey,
						"error": err.Error(),
					})
				}

				mimeType := bd.MimeType
				if mimeType == "" {
					mimeType = n.detectMimeType(attachmentName)
				}

				attachment := EmailAttachment{
					Name:     attachmentName,
					Content:  content,
					MimeType: mimeType,
				}
				message.Attachments = append(message.Attachments, attachment)
			}
		}

		// Send email
		messageID, err := n.sendEmail(smtpConfig, message)
		if err != nil {
			return nil, n.CreateError("failed to send email", map[string]interface{}{
				"to":    message.To,
				"error": err.Error(),
			})
		}

		// Create result item
		resultItem := model.DataItem{
			JSON: map[string]interface{}{
				"messageId":    messageID,
				"to":           message.To,
				"cc":           message.CC,
				"bcc":          message.BCC,
				"subject":      message.Subject,
				"sentAt":       time.Now().UTC().Format(time.RFC3339),
				"status":       "sent",
				"attachments":  len(message.Attachments),
				"smtpServer":   smtpConfig.Host,
			},
		}

		// Copy original data if requested
		keepOriginalData, _ := nodeParams["keepOriginalData"].(bool)
		if keepOriginalData {
			for k, v := range item.JSON {
				if k != "messageId" && k != "to" && k != "cc" && k != "bcc" &&
				   k != "subject" && k != "sentAt" && k != "status" && k != "attachments" && k != "smtpServer" {
					resultItem.JSON[k] = v
				}
			}
		}

		results = append(results, resultItem)
	}

	return results, nil
}

// getSMTPConfig extracts and validates SMTP configuration from node parameters
func (n *SMTPNode) getSMTPConfig(nodeParams map[string]interface{}) (*SMTPConfig, error) {
	config := &SMTPConfig{}

	// Host (required)
	host, ok := nodeParams["host"].(string)
	if !ok || host == "" {
		return nil, fmt.Errorf("host is required")
	}
	config.Host = host

	// Port (required)
	port, ok := nodeParams["port"]
	if !ok {
		return nil, fmt.Errorf("port is required")
	}
	switch p := port.(type) {
	case int:
		config.Port = p
	case float64:
		config.Port = int(p)
	case string:
		if p == "25" {
			config.Port = 25
		} else if p == "587" {
			config.Port = 587
		} else if p == "465" {
			config.Port = 465
		} else {
			return nil, fmt.Errorf("invalid port: %s", p)
		}
	default:
		return nil, fmt.Errorf("invalid port type: %T", port)
	}

	// Username (required for authentication)
	username, ok := nodeParams["username"].(string)
	if !ok || username == "" {
		return nil, fmt.Errorf("username is required")
	}
	config.Username = username

	// Password (required for authentication)
	password, ok := nodeParams["password"].(string)
	if !ok || password == "" {
		return nil, fmt.Errorf("password is required")
	}
	config.Password = password

	// TLS (optional, default true for ports 465, 587)
	config.TLS = true // Default to true
	if tls, ok := nodeParams["tls"].(bool); ok {
		config.TLS = tls
	} else if config.Port == 25 {
		config.TLS = false // Default to false for port 25
	}

	// Auth method (optional, default plain)
	config.Auth = "plain"
	if auth, ok := nodeParams["auth"].(string); ok {
		config.Auth = strings.ToLower(auth)
	}

	return config, nil
}

// buildEmailMessage constructs an email message from node parameters
func (n *SMTPNode) buildEmailMessage(nodeParams map[string]interface{}, context *expressions.ExpressionContext) (*EmailMessage, error) {
	message := &EmailMessage{
		Headers: make(map[string]string),
	}

	// To (required)
	toExpr, ok := nodeParams["to"].(string)
	if !ok {
		return nil, fmt.Errorf("to parameter is required")
	}
	toResult, err := n.evaluator.EvaluateExpression(toExpr, context)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate 'to' field: %w", err)
	}
	message.To = n.parseEmailList(toResult)

	// CC (optional)
	if ccExpr, ok := nodeParams["cc"].(string); ok && ccExpr != "" {
		ccResult, err := n.evaluator.EvaluateExpression(ccExpr, context)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate 'cc' field: %w", err)
		}
		message.CC = n.parseEmailList(ccResult)
	}

	// BCC (optional)
	if bccExpr, ok := nodeParams["bcc"].(string); ok && bccExpr != "" {
		bccResult, err := n.evaluator.EvaluateExpression(bccExpr, context)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate 'bcc' field: %w", err)
		}
		message.BCC = n.parseEmailList(bccResult)
	}

	// From (optional, defaults to username)
	if fromExpr, ok := nodeParams["from"].(string); ok && fromExpr != "" {
		fromResult, err := n.evaluator.EvaluateExpression(fromExpr, context)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate 'from' field: %w", err)
		}
		message.From = fmt.Sprintf("%v", fromResult)
	}

	// Reply-To (optional)
	if replyToExpr, ok := nodeParams["replyTo"].(string); ok && replyToExpr != "" {
		replyToResult, err := n.evaluator.EvaluateExpression(replyToExpr, context)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate 'replyTo' field: %w", err)
		}
		message.ReplyTo = fmt.Sprintf("%v", replyToResult)
	}

	// Subject (required)
	subjectExpr, ok := nodeParams["subject"].(string)
	if !ok {
		return nil, fmt.Errorf("subject parameter is required")
	}
	subjectResult, err := n.evaluator.EvaluateExpression(subjectExpr, context)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate subject: %w", err)
	}
	message.Subject = fmt.Sprintf("%v", subjectResult)

	// Body (required)
	bodyExpr, ok := nodeParams["body"].(string)
	if !ok {
		return nil, fmt.Errorf("body parameter is required")
	}
	bodyResult, err := n.evaluator.EvaluateExpression(bodyExpr, context)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate body: %w", err)
	}
	message.Body = fmt.Sprintf("%v", bodyResult)

	// HTML flag (optional, default false)
	if isHTML, ok := nodeParams["isHTML"].(bool); ok {
		message.IsHTML = isHTML
	}

	// Custom headers (optional)
	if headersParam, ok := nodeParams["headers"]; ok {
		if headersMap, ok := headersParam.(map[string]interface{}); ok {
			for k, v := range headersMap {
				message.Headers[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	return message, nil
}

// parseEmailList parses various email list formats
func (n *SMTPNode) parseEmailList(value interface{}) []string {
	switch v := value.(type) {
	case string:
		// Split by comma, semicolon, or whitespace
		emails := strings.FieldsFunc(v, func(c rune) bool {
			return c == ',' || c == ';' || c == ' ' || c == '\t' || c == '\n'
		})
		var result []string
		for _, email := range emails {
			email = strings.TrimSpace(email)
			if email != "" {
				result = append(result, email)
			}
		}
		return result
	case []interface{}:
		var result []string
		for _, item := range v {
			if str := fmt.Sprintf("%v", item); str != "" {
				result = append(result, strings.TrimSpace(str))
			}
		}
		return result
	default:
		return []string{fmt.Sprintf("%v", value)}
	}
}

// sendEmail sends the email using SMTP
func (n *SMTPNode) sendEmail(config *SMTPConfig, message *EmailMessage) (string, error) {
	// Create SMTP client
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)

	var client *smtp.Client
	var err error

	if config.TLS && config.Port == 465 {
		// SSL/TLS connection (SMTPS)
		tlsConfig := &tls.Config{ServerName: config.Host}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return "", fmt.Errorf("failed to establish TLS connection: %w", err)
		}
		client, err = smtp.NewClient(conn, config.Host)
		if err != nil {
			return "", fmt.Errorf("failed to create SMTP client: %w", err)
		}
	} else {
		// Plain connection (potentially with STARTTLS)
		client, err = smtp.Dial(addr)
		if err != nil {
			return "", fmt.Errorf("failed to connect to SMTP server: %w", err)
		}

		// Start TLS if required and supported
		if config.TLS {
			tlsConfig := &tls.Config{ServerName: config.Host}
			if err = client.StartTLS(tlsConfig); err != nil {
				client.Close()
				return "", fmt.Errorf("failed to start TLS: %w", err)
			}
		}
	}
	defer client.Close()

	// Authenticate
	auth := n.createAuth(config)
	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return "", fmt.Errorf("authentication failed: %w", err)
		}
	}

	// Set sender
	from := message.From
	if from == "" {
		from = config.Username
	}
	if err = client.Mail(from); err != nil {
		return "", fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	allRecipients := append(append(message.To, message.CC...), message.BCC...)
	for _, recipient := range allRecipients {
		if err = client.Rcpt(recipient); err != nil {
			return "", fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	// Send message
	writer, err := client.Data()
	if err != nil {
		return "", fmt.Errorf("failed to start data transmission: %w", err)
	}

	// Generate message ID
	messageID := fmt.Sprintf("<%d.%s@%s>", time.Now().UnixNano(), generateRandomString(8), config.Host)

	// Write message
	err = n.writeMessage(writer, message, messageID, from)
	if err != nil {
		writer.Close()
		return "", fmt.Errorf("failed to write message: %w", err)
	}

	if err = writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close data transmission: %w", err)
	}

	return messageID, nil
}

// createAuth creates the appropriate authentication mechanism
func (n *SMTPNode) createAuth(config *SMTPConfig) smtp.Auth {
	switch config.Auth {
	case "plain":
		return smtp.PlainAuth("", config.Username, config.Password, config.Host)
	case "login":
		return &LoginAuth{username: config.Username, password: config.Password}
	case "crammd5":
		return smtp.CRAMMD5Auth(config.Username, config.Password)
	default:
		return smtp.PlainAuth("", config.Username, config.Password, config.Host)
	}
}

// writeMessage writes the complete email message
func (n *SMTPNode) writeMessage(writer interface{ Write([]byte) (int, error) }, message *EmailMessage, messageID, from string) error {
	// Headers
	fmt.Fprintf(writer, "Message-ID: %s\r\n", messageID)
	fmt.Fprintf(writer, "From: %s\r\n", from)
	fmt.Fprintf(writer, "To: %s\r\n", strings.Join(message.To, ", "))
	if len(message.CC) > 0 {
		fmt.Fprintf(writer, "CC: %s\r\n", strings.Join(message.CC, ", "))
	}
	if message.ReplyTo != "" {
		fmt.Fprintf(writer, "Reply-To: %s\r\n", message.ReplyTo)
	}
	fmt.Fprintf(writer, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", message.Subject))
	fmt.Fprintf(writer, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))

	// Custom headers
	for k, v := range message.Headers {
		fmt.Fprintf(writer, "%s: %s\r\n", k, v)
	}

	// MIME headers
	boundary := generateRandomString(16)
	if len(message.Attachments) > 0 {
		fmt.Fprintf(writer, "MIME-Version: 1.0\r\n")
		fmt.Fprintf(writer, "Content-Type: multipart/mixed; boundary=%s\r\n", boundary)
		fmt.Fprintf(writer, "\r\n")

		// Body part
		fmt.Fprintf(writer, "--%s\r\n", boundary)
		if message.IsHTML {
			fmt.Fprintf(writer, "Content-Type: text/html; charset=utf-8\r\n")
		} else {
			fmt.Fprintf(writer, "Content-Type: text/plain; charset=utf-8\r\n")
		}
		fmt.Fprintf(writer, "\r\n")
		fmt.Fprintf(writer, "%s\r\n", message.Body)

		// Attachments
		for _, attachment := range message.Attachments {
			fmt.Fprintf(writer, "--%s\r\n", boundary)
			fmt.Fprintf(writer, "Content-Type: %s\r\n", attachment.MimeType)
			fmt.Fprintf(writer, "Content-Disposition: attachment; filename=\"%s\"\r\n", attachment.Name)
			fmt.Fprintf(writer, "Content-Transfer-Encoding: base64\r\n")
			fmt.Fprintf(writer, "\r\n")

			// Write base64 encoded attachment
			encoded := n.encodeBase64(attachment.Content)
			for i := 0; i < len(encoded); i += 76 {
				end := i + 76
				if end > len(encoded) {
					end = len(encoded)
				}
				fmt.Fprintf(writer, "%s\r\n", encoded[i:end])
			}
		}

		fmt.Fprintf(writer, "--%s--\r\n", boundary)
	} else {
		// Simple message without attachments
		if message.IsHTML {
			fmt.Fprintf(writer, "Content-Type: text/html; charset=utf-8\r\n")
		} else {
			fmt.Fprintf(writer, "Content-Type: text/plain; charset=utf-8\r\n")
		}
		fmt.Fprintf(writer, "\r\n")
		fmt.Fprintf(writer, "%s\r\n", message.Body)
	}

	return nil
}

// detectMimeType detects MIME type based on file extension
func (n *SMTPNode) detectMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	mimeTypes := map[string]string{
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".txt":  "text/plain",
		".csv":  "text/csv",
		".json": "application/json",
		".xml":  "application/xml",
		".zip":  "application/zip",
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
	}

	if mimeType, exists := mimeTypes[ext]; exists {
		return mimeType
	}
	return "application/octet-stream"
}

// encodeBase64 encodes binary data to base64
func (n *SMTPNode) encodeBase64(data []byte) string {
	const base64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

	var result strings.Builder
	for i := 0; i < len(data); i += 3 {
		group := 0
		groupSize := 0

		for j := 0; j < 3 && i+j < len(data); j++ {
			group = (group << 8) | int(data[i+j])
			groupSize++
		}

		group <<= (3 - groupSize) * 8

		for j := 0; j < 4; j++ {
			if j < groupSize+1 {
				result.WriteByte(base64Chars[(group>>(18-j*6))&0x3F])
			} else {
				result.WriteByte('=')
			}
		}
	}

	return result.String()
}

// ValidateParameters validates the node parameters
func (n *SMTPNode) ValidateParameters(params map[string]interface{}) error {
	// Check required parameters
	required := []string{"host", "port", "username", "password", "to", "subject", "body"}
	for _, param := range required {
		if _, ok := params[param]; !ok {
			return fmt.Errorf("%s parameter is required", param)
		}
	}

	// Validate port
	if port, ok := params["port"]; ok {
		switch p := port.(type) {
		case int:
			if p < 1 || p > 65535 {
				return fmt.Errorf("invalid port: %d", p)
			}
		case float64:
			if p < 1 || p > 65535 {
				return fmt.Errorf("invalid port: %f", p)
			}
		case string:
			if p != "25" && p != "587" && p != "465" {
				return fmt.Errorf("invalid port: %s", p)
			}
		default:
			return fmt.Errorf("port must be a number")
		}
	}

	return nil
}

// LoginAuth implements LOGIN authentication
type LoginAuth struct {
	username, password string
}

func (a *LoginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte{}, nil
}

func (a *LoginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		default:
			return nil, fmt.Errorf("unknown challenge: %s", fromServer)
		}
	}
	return nil, nil
}

// generateRandomString generates a random string for boundaries and message IDs
func generateRandomString(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[time.Now().UnixNano()%int64(len(chars))]
	}
	return string(result)
}