package ocrglm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"neuralclaw/internal/config"
	"neuralclaw/internal/observability"
	"neuralclaw/pkg/types"

	"go.uber.org/zap"
)

// DefaultClient wraps the Zhipu MaaS endpoint directly.
type DefaultClient struct {
	client *http.Client
}

func NewClient() *DefaultClient {
	return &DefaultClient{
		client: &http.Client{Timeout: 300 * time.Second}, // OCR can take a while for large PDFs
	}
}

// ExtractText uses the GLM-OCR Zhipu cloud API (equivalent to their maas_client.py)
func (c *DefaultClient) ExtractText(ctx context.Context, input types.FileRef) (types.OCRResult, error) {
	cfg := config.GlobalConfig.Ingest.OCR

	if cfg.APIKey == "" {
		return types.OCRResult{}, fmt.Errorf("GLM-OCR api_key is empty. Cannot continue")
	}

	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "https://open.bigmodel.cn/api/paas/v4/layout_parsing"
	}

	modelName := cfg.Model
	if modelName == "" {
		modelName = "glm-ocr"
	}

	filePath := string(input)

	// 1. Read file bytes
	b, err := os.ReadFile(filePath)
	if err != nil {
		return types.OCRResult{}, fmt.Errorf("failed to read file '%s': %w", filePath, err)
	}

	// 2. Guess Mime and Encode to Data URI
	mime := sniffMime(b, filePath)
	b64Encoded := base64.StdEncoding.EncodeToString(b)
	dataUri := fmt.Sprintf("data:%s;base64,%s", mime, b64Encoded)

	observability.Logger.Info("Sending file to GLM-OCR MaaS API",
		zap.String("file", filePath),
		zap.Int("bytes", len(b)),
		zap.String("mime", mime))

	// 3. Construct payload mimicking `maas_client.py`
	payload := map[string]interface{}{
		"model": modelName,
		"file":  dataUri,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return types.OCRResult{}, err
	}

	// 4. Send HTTP Request
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return types.OCRResult{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	startTime := time.Now()
	resp, err := c.client.Do(req)
	if err != nil {
		return types.OCRResult{}, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyStr, _ := io.ReadAll(resp.Body)
		return types.OCRResult{}, fmt.Errorf("GLM-OCR API completely failed (Status %d): %s", resp.StatusCode, string(bodyStr))
	}

	// 5. Parse Response
	var apiResp struct {
		MdResults string `json:"md_results"`
		DataInfo  struct {
			PageCount int `json:"page_count"`
		} `json:"data_info"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return types.OCRResult{}, fmt.Errorf("failed to decode GLM-OCR API response: %w", err)
	}

	duration := time.Since(startTime)
	observability.Logger.Info("GLM-OCR response received",
		zap.Duration("latency", duration),
		zap.Int("pages", apiResp.DataInfo.PageCount),
		zap.Int("text_len", len(apiResp.MdResults)))

	return types.OCRResult{
		Text:  apiResp.MdResults,
		Pages: apiResp.DataInfo.PageCount,
	}, nil
}

func sniffMime(b []byte, fname string) string {
	if bytes.HasPrefix(b, []byte("%PDF-")) {
		return "application/pdf"
	}
	if bytes.HasPrefix(b, []byte("\x89PNG\r\n\x1a\n")) {
		return "image/png"
	}
	if bytes.HasPrefix(b, []byte("\xff\xd8\xff")) {
		return "image/jpeg"
	}

	// Fallback based on extension
	ext := strings.ToLower(filepath.Ext(fname))
	if ext == ".pdf" {
		return "application/pdf"
	}
	if ext == ".png" {
		return "image/png"
	}
	if ext == ".jpg" || ext == ".jpeg" {
		return "image/jpeg"
	}

	// Final fallback for maas_client.py imitation
	return "application/octet-stream"
}

// Ensure it implements OCREngine
var _ types.OCREngine = (*DefaultClient)(nil)
