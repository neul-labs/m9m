package email

import (
	"testing"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// === SMTPNode creation and Description ===

func TestSMTPNode_NewSMTPNode(t *testing.T) {
	node := NewSMTPNode()
	require.NotNil(t, node)
	require.NotNil(t, node.BaseNode)
	require.NotNil(t, node.evaluator)
}

func TestSMTPNode_Description(t *testing.T) {
	node := NewSMTPNode()
	desc := node.Description()

	assert.Equal(t, "Send Email", desc.Name)
	assert.Equal(t, "n8n-nodes-base.smtp", desc.Description)
	assert.Equal(t, "core", desc.Category)
}

func TestSMTPNode_ImplementsNodeExecutor(t *testing.T) {
	var _ base.NodeExecutor = (*SMTPNode)(nil)
}

// === ValidateParameters ===

func TestSMTPNode_ValidateParameters(t *testing.T) {
	node := NewSMTPNode()

	tests := []struct {
		name        string
		params      map[string]interface{}
		expectErr   bool
		errContains string
	}{
		{
			name: "all required params present",
			params: map[string]interface{}{
				"host":     "smtp.example.com",
				"port":     587,
				"username": "user",
				"password": "pass",
				"to":       "test@example.com",
				"subject":  "Hello",
				"body":     "World",
			},
			expectErr: false,
		},
		{
			name: "missing host",
			params: map[string]interface{}{
				"port":     587,
				"username": "user",
				"password": "pass",
				"to":       "test@example.com",
				"subject":  "Hello",
				"body":     "World",
			},
			expectErr:   true,
			errContains: "host parameter is required",
		},
		{
			name: "missing port",
			params: map[string]interface{}{
				"host":     "smtp.example.com",
				"username": "user",
				"password": "pass",
				"to":       "test@example.com",
				"subject":  "Hello",
				"body":     "World",
			},
			expectErr:   true,
			errContains: "port parameter is required",
		},
		{
			name: "missing username",
			params: map[string]interface{}{
				"host":     "smtp.example.com",
				"port":     587,
				"password": "pass",
				"to":       "test@example.com",
				"subject":  "Hello",
				"body":     "World",
			},
			expectErr:   true,
			errContains: "username parameter is required",
		},
		{
			name: "missing password",
			params: map[string]interface{}{
				"host":     "smtp.example.com",
				"port":     587,
				"username": "user",
				"to":       "test@example.com",
				"subject":  "Hello",
				"body":     "World",
			},
			expectErr:   true,
			errContains: "password parameter is required",
		},
		{
			name: "missing to",
			params: map[string]interface{}{
				"host":     "smtp.example.com",
				"port":     587,
				"username": "user",
				"password": "pass",
				"subject":  "Hello",
				"body":     "World",
			},
			expectErr:   true,
			errContains: "to parameter is required",
		},
		{
			name: "missing subject",
			params: map[string]interface{}{
				"host":     "smtp.example.com",
				"port":     587,
				"username": "user",
				"password": "pass",
				"to":       "test@example.com",
				"body":     "World",
			},
			expectErr:   true,
			errContains: "subject parameter is required",
		},
		{
			name: "missing body",
			params: map[string]interface{}{
				"host":     "smtp.example.com",
				"port":     587,
				"username": "user",
				"password": "pass",
				"to":       "test@example.com",
				"subject":  "Hello",
			},
			expectErr:   true,
			errContains: "body parameter is required",
		},
		{
			name: "port as valid int",
			params: map[string]interface{}{
				"host":     "smtp.example.com",
				"port":     25,
				"username": "user",
				"password": "pass",
				"to":       "test@example.com",
				"subject":  "Hello",
				"body":     "World",
			},
			expectErr: false,
		},
		{
			name: "port as valid float64",
			params: map[string]interface{}{
				"host":     "smtp.example.com",
				"port":     float64(587),
				"username": "user",
				"password": "pass",
				"to":       "test@example.com",
				"subject":  "Hello",
				"body":     "World",
			},
			expectErr: false,
		},
		{
			name: "port as valid string 587",
			params: map[string]interface{}{
				"host":     "smtp.example.com",
				"port":     "587",
				"username": "user",
				"password": "pass",
				"to":       "test@example.com",
				"subject":  "Hello",
				"body":     "World",
			},
			expectErr: false,
		},
		{
			name: "port as valid string 25",
			params: map[string]interface{}{
				"host":     "smtp.example.com",
				"port":     "25",
				"username": "user",
				"password": "pass",
				"to":       "test@example.com",
				"subject":  "Hello",
				"body":     "World",
			},
			expectErr: false,
		},
		{
			name: "port as valid string 465",
			params: map[string]interface{}{
				"host":     "smtp.example.com",
				"port":     "465",
				"username": "user",
				"password": "pass",
				"to":       "test@example.com",
				"subject":  "Hello",
				"body":     "World",
			},
			expectErr: false,
		},
		{
			name: "port as invalid string",
			params: map[string]interface{}{
				"host":     "smtp.example.com",
				"port":     "9999",
				"username": "user",
				"password": "pass",
				"to":       "test@example.com",
				"subject":  "Hello",
				"body":     "World",
			},
			expectErr:   true,
			errContains: "invalid port",
		},
		{
			name: "port out of range int",
			params: map[string]interface{}{
				"host":     "smtp.example.com",
				"port":     99999,
				"username": "user",
				"password": "pass",
				"to":       "test@example.com",
				"subject":  "Hello",
				"body":     "World",
			},
			expectErr:   true,
			errContains: "invalid port",
		},
		{
			name: "port negative int",
			params: map[string]interface{}{
				"host":     "smtp.example.com",
				"port":     -1,
				"username": "user",
				"password": "pass",
				"to":       "test@example.com",
				"subject":  "Hello",
				"body":     "World",
			},
			expectErr:   true,
			errContains: "invalid port",
		},
		{
			name: "port out of range float64",
			params: map[string]interface{}{
				"host":     "smtp.example.com",
				"port":     float64(70000),
				"username": "user",
				"password": "pass",
				"to":       "test@example.com",
				"subject":  "Hello",
				"body":     "World",
			},
			expectErr:   true,
			errContains: "invalid port",
		},
		{
			name: "port as invalid type (bool)",
			params: map[string]interface{}{
				"host":     "smtp.example.com",
				"port":     true,
				"username": "user",
				"password": "pass",
				"to":       "test@example.com",
				"subject":  "Hello",
				"body":     "World",
			},
			expectErr:   true,
			errContains: "port must be a number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := node.ValidateParameters(tt.params)
			if tt.expectErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// === getSMTPConfig ===

func TestSMTPNode_getSMTPConfig(t *testing.T) {
	node := NewSMTPNode()

	tests := []struct {
		name        string
		params      map[string]interface{}
		expectErr   bool
		errContains string
		validate    func(t *testing.T, config *SMTPConfig)
	}{
		{
			name: "valid config with defaults",
			params: map[string]interface{}{
				"host":     "smtp.gmail.com",
				"port":     587,
				"username": "user@gmail.com",
				"password": "secret",
			},
			expectErr: false,
			validate: func(t *testing.T, config *SMTPConfig) {
				assert.Equal(t, "smtp.gmail.com", config.Host)
				assert.Equal(t, 587, config.Port)
				assert.Equal(t, "user@gmail.com", config.Username)
				assert.Equal(t, "secret", config.Password)
				assert.True(t, config.TLS, "TLS should be true by default")
				assert.Equal(t, "plain", config.Auth)
			},
		},
		{
			name: "TLS explicitly disabled",
			params: map[string]interface{}{
				"host":     "smtp.local",
				"port":     25,
				"username": "user",
				"password": "pass",
				"tls":      false,
			},
			expectErr: false,
			validate: func(t *testing.T, config *SMTPConfig) {
				assert.False(t, config.TLS)
			},
		},
		{
			name: "TLS explicitly enabled",
			params: map[string]interface{}{
				"host":     "smtp.local",
				"port":     465,
				"username": "user",
				"password": "pass",
				"tls":      true,
			},
			expectErr: false,
			validate: func(t *testing.T, config *SMTPConfig) {
				assert.True(t, config.TLS)
			},
		},
		{
			name: "custom auth method login",
			params: map[string]interface{}{
				"host":     "smtp.local",
				"port":     587,
				"username": "user",
				"password": "pass",
				"auth":     "LOGIN",
			},
			expectErr: false,
			validate: func(t *testing.T, config *SMTPConfig) {
				assert.Equal(t, "login", config.Auth)
			},
		},
		{
			name: "custom auth method crammd5",
			params: map[string]interface{}{
				"host":     "smtp.local",
				"port":     587,
				"username": "user",
				"password": "pass",
				"auth":     "CRAMMD5",
			},
			expectErr: false,
			validate: func(t *testing.T, config *SMTPConfig) {
				assert.Equal(t, "crammd5", config.Auth)
			},
		},
		{
			name: "port as float64 (from JSON)",
			params: map[string]interface{}{
				"host":     "smtp.local",
				"port":     float64(465),
				"username": "user",
				"password": "pass",
			},
			expectErr: false,
			validate: func(t *testing.T, config *SMTPConfig) {
				assert.Equal(t, 465, config.Port)
			},
		},
		{
			name: "port as string 587",
			params: map[string]interface{}{
				"host":     "smtp.local",
				"port":     "587",
				"username": "user",
				"password": "pass",
			},
			expectErr: false,
			validate: func(t *testing.T, config *SMTPConfig) {
				assert.Equal(t, 587, config.Port)
			},
		},
		{
			name: "port as invalid string",
			params: map[string]interface{}{
				"host":     "smtp.local",
				"port":     "8080",
				"username": "user",
				"password": "pass",
			},
			expectErr:   true,
			errContains: "invalid port",
		},
		{
			name: "missing host",
			params: map[string]interface{}{
				"port":     587,
				"username": "user",
				"password": "pass",
			},
			expectErr:   true,
			errContains: "host is required",
		},
		{
			name: "empty host",
			params: map[string]interface{}{
				"host":     "",
				"port":     587,
				"username": "user",
				"password": "pass",
			},
			expectErr:   true,
			errContains: "host is required",
		},
		{
			name: "missing port",
			params: map[string]interface{}{
				"host":     "smtp.local",
				"username": "user",
				"password": "pass",
			},
			expectErr:   true,
			errContains: "port is required",
		},
		{
			name: "missing username",
			params: map[string]interface{}{
				"host":     "smtp.local",
				"port":     587,
				"password": "pass",
			},
			expectErr:   true,
			errContains: "username is required",
		},
		{
			name: "missing password",
			params: map[string]interface{}{
				"host":     "smtp.local",
				"port":     587,
				"username": "user",
			},
			expectErr:   true,
			errContains: "password is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := node.getSMTPConfig(tt.params)
			if tt.expectErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, config)
				if tt.validate != nil {
					tt.validate(t, config)
				}
			}
		})
	}
}

// === parseEmailList ===

func TestSMTPNode_parseEmailList(t *testing.T) {
	node := NewSMTPNode()

	tests := []struct {
		name     string
		input    interface{}
		expected []string
	}{
		{
			name:     "single email string",
			input:    "user@example.com",
			expected: []string{"user@example.com"},
		},
		{
			name:     "comma separated emails",
			input:    "a@example.com, b@example.com, c@example.com",
			expected: []string{"a@example.com", "b@example.com", "c@example.com"},
		},
		{
			name:     "semicolon separated emails",
			input:    "a@example.com;b@example.com;c@example.com",
			expected: []string{"a@example.com", "b@example.com", "c@example.com"},
		},
		{
			name:     "space separated emails",
			input:    "a@example.com b@example.com",
			expected: []string{"a@example.com", "b@example.com"},
		},
		{
			name:     "mixed separators",
			input:    "a@example.com, b@example.com; c@example.com",
			expected: []string{"a@example.com", "b@example.com", "c@example.com"},
		},
		{
			name:     "emails with extra whitespace",
			input:    "  a@example.com ,  b@example.com  ",
			expected: []string{"a@example.com", "b@example.com"},
		},
		{
			name:     "slice of interfaces",
			input:    []interface{}{"a@example.com", "b@example.com"},
			expected: []string{"a@example.com", "b@example.com"},
		},
		{
			name:     "non-string value",
			input:    42,
			expected: []string{"42"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := node.parseEmailList(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// === detectMimeType ===

func TestSMTPNode_detectMimeType(t *testing.T) {
	node := NewSMTPNode()

	tests := []struct {
		filename string
		expected string
	}{
		{"document.pdf", "application/pdf"},
		{"file.doc", "application/msword"},
		{"file.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"data.xls", "application/vnd.ms-excel"},
		{"data.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{"readme.txt", "text/plain"},
		{"data.csv", "text/csv"},
		{"config.json", "application/json"},
		{"data.xml", "application/xml"},
		{"archive.zip", "application/zip"},
		{"image.png", "image/png"},
		{"photo.jpg", "image/jpeg"},
		{"photo.jpeg", "image/jpeg"},
		{"animation.gif", "image/gif"},
		{"unknown.xyz", "application/octet-stream"},
		{"noextension", "application/octet-stream"},
		{"IMAGE.PNG", "image/png"}, // case insensitive due to strings.ToLower
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := node.detectMimeType(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// === encodeBase64 ===

func TestSMTPNode_encodeBase64(t *testing.T) {
	node := NewSMTPNode()

	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "empty input",
			input:    []byte{},
			expected: "",
		},
		{
			name:     "single byte",
			input:    []byte{0x4D}, // 'M'
			expected: "TQ==",
		},
		{
			name:     "two bytes",
			input:    []byte{0x4D, 0x61}, // "Ma"
			expected: "TWE=",
		},
		{
			name:     "three bytes (exact block)",
			input:    []byte{0x4D, 0x61, 0x6E}, // "Man"
			expected: "TWFu",
		},
		{
			name:     "simple text",
			input:    []byte("Hello"),
			expected: "SGVsbG8=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := node.encodeBase64(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// === createAuth ===

func TestSMTPNode_createAuth(t *testing.T) {
	node := NewSMTPNode()

	tests := []struct {
		name     string
		authType string
	}{
		{"plain auth", "plain"},
		{"login auth", "login"},
		{"crammd5 auth", "crammd5"},
		{"default falls back to plain", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &SMTPConfig{
				Host:     "smtp.example.com",
				Port:     587,
				Username: "user",
				Password: "pass",
				Auth:     tt.authType,
			}
			auth := node.createAuth(config)
			assert.NotNil(t, auth)
		})
	}
}

// === LoginAuth ===

func TestLoginAuth_Start(t *testing.T) {
	auth := &LoginAuth{username: "user", password: "pass"}
	method, resp, err := auth.Start(nil)
	require.NoError(t, err)
	assert.Equal(t, "LOGIN", method)
	assert.Equal(t, []byte{}, resp)
}

func TestLoginAuth_Next(t *testing.T) {
	auth := &LoginAuth{username: "testuser", password: "testpass"}

	tests := []struct {
		name       string
		fromServer []byte
		more       bool
		expected   []byte
		expectErr  bool
	}{
		{
			name:       "username challenge",
			fromServer: []byte("Username:"),
			more:       true,
			expected:   []byte("testuser"),
			expectErr:  false,
		},
		{
			name:       "password challenge",
			fromServer: []byte("Password:"),
			more:       true,
			expected:   []byte("testpass"),
			expectErr:  false,
		},
		{
			name:       "unknown challenge",
			fromServer: []byte("Unknown:"),
			more:       true,
			expected:   nil,
			expectErr:  true,
		},
		{
			name:       "no more challenges",
			fromServer: []byte("anything"),
			more:       false,
			expected:   nil,
			expectErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := auth.Next(tt.fromServer, tt.more)
			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

// === generateRandomString ===

func TestGenerateRandomString(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"zero length", 0},
		{"length 1", 1},
		{"length 8", 8},
		{"length 16", 16},
		{"length 32", 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateRandomString(tt.length)
			assert.Len(t, result, tt.length)

			// Check all characters are alphanumeric
			for _, c := range result {
				assert.True(t,
					(c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9'),
					"character %c should be alphanumeric", c,
				)
			}
		})
	}
}

// === Execute: empty input data ===

func TestSMTPNode_Execute_EmptyInput(t *testing.T) {
	node := NewSMTPNode()

	params := map[string]interface{}{
		"host":     "smtp.example.com",
		"port":     587,
		"username": "user",
		"password": "pass",
		"to":       "test@example.com",
		"subject":  "Hello",
		"body":     "World",
	}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.Nil(t, result)
}

// === Execute: invalid SMTP config ===

func TestSMTPNode_Execute_InvalidSMTPConfig(t *testing.T) {
	node := NewSMTPNode()

	tests := []struct {
		name   string
		params map[string]interface{}
	}{
		{
			name: "missing host",
			params: map[string]interface{}{
				"port":     587,
				"username": "user",
				"password": "pass",
				"to":       "test@example.com",
				"subject":  "Hello",
				"body":     "World",
			},
		},
		{
			name: "missing port",
			params: map[string]interface{}{
				"host":     "smtp.example.com",
				"username": "user",
				"password": "pass",
				"to":       "test@example.com",
				"subject":  "Hello",
				"body":     "World",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputData := []model.DataItem{
				{JSON: map[string]interface{}{"key": "value"}},
			}
			_, err := node.Execute(inputData, tt.params)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid SMTP configuration")
		})
	}
}

// === Execute: missing required email fields ===

func TestSMTPNode_Execute_MissingEmailFields(t *testing.T) {
	node := NewSMTPNode()

	tests := []struct {
		name        string
		params      map[string]interface{}
		errContains string
	}{
		{
			name: "missing to field",
			params: map[string]interface{}{
				"host":     "smtp.example.com",
				"port":     587,
				"username": "user",
				"password": "pass",
				"subject":  "Hello",
				"body":     "World",
			},
			errContains: "failed to build email message",
		},
		{
			name: "missing subject field",
			params: map[string]interface{}{
				"host":     "smtp.example.com",
				"port":     587,
				"username": "user",
				"password": "pass",
				"to":       "test@example.com",
				"body":     "World",
			},
			errContains: "failed to build email message",
		},
		{
			name: "missing body field",
			params: map[string]interface{}{
				"host":     "smtp.example.com",
				"port":     587,
				"username": "user",
				"password": "pass",
				"to":       "test@example.com",
				"subject":  "Hello",
			},
			errContains: "failed to build email message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputData := []model.DataItem{
				{JSON: map[string]interface{}{}},
			}
			_, err := node.Execute(inputData, tt.params)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContains)
		})
	}
}

// === SMTPConfig struct ===

func TestSMTPConfig_Struct(t *testing.T) {
	config := SMTPConfig{
		Host:     "smtp.test.com",
		Port:     465,
		Username: "testuser",
		Password: "testpass",
		TLS:      true,
		Auth:     "login",
	}

	assert.Equal(t, "smtp.test.com", config.Host)
	assert.Equal(t, 465, config.Port)
	assert.Equal(t, "testuser", config.Username)
	assert.Equal(t, "testpass", config.Password)
	assert.True(t, config.TLS)
	assert.Equal(t, "login", config.Auth)
}

// === EmailMessage struct ===

func TestEmailMessage_Struct(t *testing.T) {
	msg := EmailMessage{
		To:      []string{"a@test.com", "b@test.com"},
		CC:      []string{"c@test.com"},
		BCC:     []string{"d@test.com"},
		From:    "sender@test.com",
		ReplyTo: "reply@test.com",
		Subject: "Test Subject",
		Body:    "<h1>Hello</h1>",
		IsHTML:  true,
		Headers: map[string]string{"X-Custom": "value"},
	}

	assert.Len(t, msg.To, 2)
	assert.Len(t, msg.CC, 1)
	assert.Len(t, msg.BCC, 1)
	assert.Equal(t, "sender@test.com", msg.From)
	assert.Equal(t, "reply@test.com", msg.ReplyTo)
	assert.Equal(t, "Test Subject", msg.Subject)
	assert.True(t, msg.IsHTML)
	assert.Equal(t, "value", msg.Headers["X-Custom"])
}

// === EmailAttachment struct ===

func TestEmailAttachment_Struct(t *testing.T) {
	attachment := EmailAttachment{
		Name:     "file.pdf",
		Content:  []byte("test-content"),
		MimeType: "application/pdf",
	}

	assert.Equal(t, "file.pdf", attachment.Name)
	assert.Equal(t, []byte("test-content"), attachment.Content)
	assert.Equal(t, "application/pdf", attachment.MimeType)
}
