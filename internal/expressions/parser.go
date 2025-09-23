package expressions

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ExpressionChunkTypeType represents the type of an expression chunk
type ExpressionChunkTypeType int

const (
	TextChunk ExpressionChunkTypeType = iota
	ExpressionChunk
)

// ExpressionChunkType represents a piece of text or expression code
type ExpressionChunkType struct {
	Type    ExpressionChunkTypeType `json:"type"`
	Content string              `json:"content"`
	Start   int                 `json:"start"`
	End     int                 `json:"end"`
}

// ParsedExpression represents a parsed n8n expression
type ParsedExpression struct {
	OriginalText    string             `json:"originalText"`
	JavaScriptCode  string             `json:"javascriptCode"`
	Chunks          []ExpressionChunkType  `json:"chunks"`
	Variables       []string           `json:"variables"`
	HasExpressions  bool               `json:"hasExpressions"`
	IsExpression    bool               `json:"isExpression"`
	CacheTime       time.Time          `json:"cacheTime"`
}

// ExpressionParser handles parsing n8n expressions into executable JavaScript
type ExpressionParser struct {
	openBracketRegex  *regexp.Regexp
	closeBracketRegex *regexp.Regexp
	variableRegex     *regexp.Regexp
	cache             map[string]*ParsedExpression
	cacheMutex        sync.RWMutex
	maxCacheSize      int
}

// NewExpressionParser creates a new expression parser with n8n-compatible regex patterns
func NewExpressionParser() *ExpressionParser {
	return &ExpressionParser{
		// Matches {{ with optional escape character (exactly like n8n)
		openBracketRegex:  regexp.MustCompile(`(?P<escape>\\?)(?P<brackets>\{\{)`),
		// Matches }} with optional escape character (exactly like n8n)
		closeBracketRegex: regexp.MustCompile(`(?P<escape>\\?)(?P<brackets>\}\})`),
		// Matches variable references like $json, $node, $input etc.
		variableRegex:     regexp.MustCompile(`\$(?:json|input|node|parameter|workflow|execution|env|binary|vars)\b`),
		cache:             make(map[string]*ParsedExpression),
		maxCacheSize:      1000,
	}
}

// ParseExpression parses an n8n expression string into a ParsedExpression
func (p *ExpressionParser) ParseExpression(input string) (*ParsedExpression, error) {
	// Check cache first
	if cached := p.getFromCache(input); cached != nil {
		return cached, nil
	}

	originalInput := input

	// Detect if this is an n8n expression (starts with =)
	isExpression := strings.HasPrefix(input, "=")
	if isExpression {
		input = strings.TrimPrefix(input, "=")
	}

	// Split into chunks
	chunks, err := p.splitIntoChunks(input)
	if err != nil {
		return nil, fmt.Errorf("failed to split expression into chunks: %w", err)
	}

	// Build JavaScript code
	jsCode := p.buildJavaScriptCode(chunks, isExpression)

	// Extract variable references
	variables := p.extractVariables(input)

	// Determine if this contains expressions
	hasExpressions := len(chunks) > 1 || (len(chunks) == 1 && chunks[0].Type == ExpressionChunk)

	parsed := &ParsedExpression{
		OriginalText:   originalInput,
		JavaScriptCode: jsCode,
		Chunks:         chunks,
		Variables:      variables,
		HasExpressions: hasExpressions,
		IsExpression:   isExpression,
		CacheTime:      time.Now(),
	}

	// Cache the result
	p.cacheResult(originalInput, parsed)

	return parsed, nil
}

// splitIntoChunks splits the input into text and expression chunks
func (p *ExpressionParser) splitIntoChunks(input string) ([]ExpressionChunkType, error) {
	var chunks []ExpressionChunkType
	currentPos := 0

	for currentPos < len(input) {
		// Find next opening bracket
		openMatch := p.openBracketRegex.FindStringSubmatchIndex(input[currentPos:])

		if openMatch == nil {
			// No more expressions, add remaining text as text chunk
			if currentPos < len(input) {
				chunks = append(chunks, ExpressionChunkType{
					Type:    TextChunk,
					Content: input[currentPos:],
					Start:   currentPos,
					End:     len(input),
				})
			}
			break
		}

		// Adjust indices to be relative to full string
		openStart := currentPos + openMatch[0]
		openEnd := currentPos + openMatch[1]

		// Check if the opening bracket is escaped
		escapeGroup := openMatch[2] // escape group index
		if escapeGroup != -1 && input[currentPos+escapeGroup:currentPos+escapeGroup+1] == "\\" {
			// Escaped bracket, treat as text
			chunks = append(chunks, ExpressionChunkType{
				Type:    TextChunk,
				Content: input[currentPos:openEnd],
				Start:   currentPos,
				End:     openEnd,
			})
			currentPos = openEnd
			continue
		}

		// Add text before opening bracket as text chunk
		if openStart > currentPos {
			chunks = append(chunks, ExpressionChunkType{
				Type:    TextChunk,
				Content: input[currentPos:openStart],
				Start:   currentPos,
				End:     openStart,
			})
		}

		// Find matching closing bracket
		searchStart := openEnd
		braceCount := 1
		closePos := -1

		for searchStart < len(input) && braceCount > 0 {
			// Look for opening brackets
			nextOpen := p.openBracketRegex.FindStringSubmatchIndex(input[searchStart:])
			// Look for closing brackets
			nextClose := p.closeBracketRegex.FindStringSubmatchIndex(input[searchStart:])

			// Determine which comes first
			openPos := -1
			if nextOpen != nil {
				openPos = searchStart + nextOpen[0]
			}

			closeTestPos := -1
			if nextClose != nil {
				closeTestPos = searchStart + nextClose[0]
			}

			if closeTestPos != -1 && (openPos == -1 || closeTestPos < openPos) {
				// Closing bracket comes first
				escapeGroup := nextClose[2]
				if escapeGroup != -1 && input[searchStart+escapeGroup:searchStart+escapeGroup+1] == "\\" {
					// Escaped closing bracket, skip
					searchStart = searchStart + nextClose[1]
					continue
				}

				braceCount--
				if braceCount == 0 {
					closePos = searchStart + nextClose[1]
				}
				searchStart = searchStart + nextClose[1]
			} else if openPos != -1 {
				// Opening bracket comes first
				escapeGroup := nextOpen[2]
				if escapeGroup == -1 || input[searchStart+escapeGroup:searchStart+escapeGroup+1] != "\\" {
					// Not escaped, increment count
					braceCount++
				}
				searchStart = searchStart + nextOpen[1]
			} else {
				// No more brackets found
				break
			}
		}

		if closePos == -1 {
			// No matching closing bracket found, treat as text
			chunks = append(chunks, ExpressionChunkType{
				Type:    TextChunk,
				Content: input[openStart:],
				Start:   openStart,
				End:     len(input),
			})
			break
		}

		// Add expression chunk
		expressionContent := input[openEnd:closePos-2] // Exclude the closing }}
		chunks = append(chunks, ExpressionChunkType{
			Type:    ExpressionChunk,
			Content: expressionContent,
			Start:   openStart,
			End:     closePos,
		})

		currentPos = closePos
	}

	return chunks, nil
}

// buildJavaScriptCode builds executable JavaScript from the parsed chunks
func (p *ExpressionParser) buildJavaScriptCode(chunks []ExpressionChunkType, isExpression bool) string {
	if len(chunks) == 0 {
		return "''"
	}

	// If this is a pure expression (={{ ... }}), return the expression directly
	if isExpression && len(chunks) == 1 && chunks[0].Type == ExpressionChunk {
		return p.preprocessExpression(chunks[0].Content)
	}

	// If this is mixed content or text, build a template literal
	var parts []string

	for _, chunk := range chunks {
		switch chunk.Type {
		case TextChunk:
			// Escape text for JavaScript string literal
			escaped := p.escapeJavaScriptString(chunk.Content)
			parts = append(parts, fmt.Sprintf("'%s'", escaped))
		case ExpressionChunk:
			// Preprocess expression to handle reserved keywords
			processedContent := p.preprocessExpression(chunk.Content)
			// Add expression as-is, wrapped in parentheses for safety
			parts = append(parts, fmt.Sprintf("(%s)", processedContent))
		}
	}

	// If only one part, return it directly
	if len(parts) == 1 {
		return parts[0]
	}

	// Multiple parts, concatenate with +
	return strings.Join(parts, " + ")
}

// escapeJavaScriptString escapes a string for use in JavaScript
func (p *ExpressionParser) escapeJavaScriptString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

// extractVariables extracts variable references from the expression
func (p *ExpressionParser) extractVariables(input string) []string {
	matches := p.variableRegex.FindAllString(input, -1)

	// Remove duplicates
	seen := make(map[string]bool)
	var unique []string
	for _, match := range matches {
		if !seen[match] {
			seen[match] = true
			unique = append(unique, match)
		}
	}

	return unique
}

// getFromCache retrieves a cached parsed expression
func (p *ExpressionParser) getFromCache(input string) *ParsedExpression {
	p.cacheMutex.RLock()
	defer p.cacheMutex.RUnlock()

	if cached, exists := p.cache[input]; exists {
		// Check if cache entry is not too old (1 hour)
		if time.Since(cached.CacheTime) < time.Hour {
			return cached
		}
		// Remove expired entry
		delete(p.cache, input)
	}

	return nil
}

// cacheResult caches a parsed expression result
func (p *ExpressionParser) cacheResult(input string, parsed *ParsedExpression) {
	p.cacheMutex.Lock()
	defer p.cacheMutex.Unlock()

	// Check cache size and evict oldest if necessary
	if len(p.cache) >= p.maxCacheSize {
		p.evictOldest()
	}

	p.cache[input] = parsed
}

// evictOldest removes the oldest cache entry
func (p *ExpressionParser) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range p.cache {
		if oldestKey == "" || entry.CacheTime.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.CacheTime
		}
	}

	if oldestKey != "" {
		delete(p.cache, oldestKey)
	}
}

// GetCacheStats returns statistics about the parser cache
func (p *ExpressionParser) GetCacheStats() map[string]interface{} {
	p.cacheMutex.RLock()
	defer p.cacheMutex.RUnlock()

	return map[string]interface{}{
		"size":        len(p.cache),
		"maxSize":     p.maxCacheSize,
		"utilization": float64(len(p.cache)) / float64(p.maxCacheSize),
	}
}

// ClearCache clears the parser cache
func (p *ExpressionParser) ClearCache() {
	p.cacheMutex.Lock()
	defer p.cacheMutex.Unlock()

	p.cache = make(map[string]*ParsedExpression)
}

// preprocessExpression handles reserved keywords and other transformations
func (p *ExpressionParser) preprocessExpression(content string) string {
	// Use regex to find function calls with reserved keywords
	reservedKeywordPattern := regexp.MustCompile(`\bif\s*\(`)

	// Replace 'if(' with 'this["if"](' to use bracket notation
	processed := reservedKeywordPattern.ReplaceAllString(content, `this["if"](`)

	return processed
}