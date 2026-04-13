package llamaparse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/cloudwego/eino/schema"
)

// Option is the option for LlamaParser.
type Option func(*LlamaParser)

// WithHTTPClient sets the http client.
func WithHTTPClient(client *http.Client) Option {
	return func(p *LlamaParser) {
		p.httpClient = client
	}
}

// WithBaseURL sets the base URL for LlamaParse API.
func WithBaseURL(url string) Option {
	return func(p *LlamaParser) {
		p.baseURL = url
	}
}

// WithCheckInterval sets the interval for polling job status.
func WithCheckInterval(interval time.Duration) Option {
	return func(p *LlamaParser) {
		p.checkInterval = interval
	}
}

// WithMaxTimeout sets the maximum timeout for a parsing job.
func WithMaxTimeout(timeout time.Duration) Option {
	return func(p *LlamaParser) {
		p.maxTimeout = timeout
	}
}

// ResultType defines the output format of LlamaParse.
type ResultType string

const (
	ResultTypeMarkdown ResultType = "markdown"
	ResultTypeText     ResultType = "text"
	ResultTypeJSON     ResultType = "json"
)

// LlamaParser implements the document.Parser interface for LlamaParse.
type LlamaParser struct {
	apiKey        string
	httpClient    *http.Client
	baseURL       string
	checkInterval time.Duration
	maxTimeout    time.Duration
}

// NewLlamaParser creates a new LlamaParser.
func NewLlamaParser(apiKey string, opts ...Option) *LlamaParser {
	p := &LlamaParser{
		apiKey:        apiKey,
		httpClient:    &http.Client{Timeout: 15 * time.Minute}, // LlamaParse can take time
		baseURL:       "https://api.cloud.llamaindex.ai/api/parsing",
		checkInterval: 2 * time.Second,
		maxTimeout:    10 * time.Minute,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// ParseOptions defines the options for a single Parse call.
type ParseOptions struct {
	Filename           string     `json:"filename"`
	Language           string     `json:"language"`
	ParsingInstruction string     `json:"parsing_instruction"`
	ResultType         ResultType `json:"result_type"`
}

// Parse parses the document using LlamaParse API.
// It supports passing *ParseOptions as an option.
func (p *LlamaParser) Parse(ctx context.Context, reader io.Reader, options ...any) ([]*schema.Document, error) {
	opts := &ParseOptions{
		Filename:   "document.pdf", // Default filename if not provided
		ResultType: ResultTypeMarkdown,
	}

	for _, opt := range options {
		if po, ok := opt.(*ParseOptions); ok {
			if po.Filename != "" {
				opts.Filename = po.Filename
			}
			if po.Language != "" {
				opts.Language = po.Language
			}
			if po.ParsingInstruction != "" {
				opts.ParsingInstruction = po.ParsingInstruction
			}
			if po.ResultType != "" {
				opts.ResultType = po.ResultType
			}
		}
	}

	return p.doParse(ctx, reader, opts)
}

func (p *LlamaParser) doParse(ctx context.Context, reader io.Reader, opts *ParseOptions) ([]*schema.Document, error) {
	// 1. Upload file
	jobID, err := p.upload(ctx, reader, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	// 2. Poll for completion
	err = p.poll(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to poll job status: %w", err)
	}

	// 3. Get result
	content, err := p.getResult(ctx, jobID, opts.ResultType)
	if err != nil {
		return nil, fmt.Errorf("failed to get result: %w", err)
	}

	return []*schema.Document{
		{
			Content: content,
			MetaData: map[string]any{
				"job_id":   jobID,
				"filename": opts.Filename,
			},
		},
	}, nil
}

func (p *LlamaParser) upload(ctx context.Context, reader io.Reader, opts *ParseOptions) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", opts.Filename)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, reader); err != nil {
		return "", err
	}

	// Add other fields if necessary
	if opts.Language != "" {
		_ = writer.WriteField("language", opts.Language)
	}
	if opts.ParsingInstruction != "" {
		_ = writer.WriteField("parsing_instruction", opts.ParsingInstruction)
	}

	if err := writer.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/upload", body)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.ID, nil
}

func (p *LlamaParser) poll(ctx context.Context, jobID string) error {
	start := time.Now()
	ticker := time.NewTicker(p.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Since(start) > p.maxTimeout {
				return fmt.Errorf("parsing job timed out after %v", p.maxTimeout)
			}

			status, err := p.checkStatus(ctx, jobID)
			if err != nil {
				return err
			}

			switch status {
			case "SUCCESS":
				return nil
			case "FAILED":
				return fmt.Errorf("parsing job failed")
			case "PENDING":
				// continue polling
			default:
				// unknown status, maybe log it?
			}
		}
	}
}

func (p *LlamaParser) checkStatus(ctx context.Context, jobID string) (string, error) {
	url := fmt.Sprintf("%s/job/%s", p.baseURL, jobID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("check status failed with status %d", resp.StatusCode)
	}

	var result struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Status, nil
}

func (p *LlamaParser) getResult(ctx context.Context, jobID string, resultType ResultType) (string, error) {
	if resultType == "" {
		resultType = ResultTypeMarkdown
	}
	url := fmt.Sprintf("%s/job/%s/result/%s", p.baseURL, jobID, resultType)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("get result failed with status %d", resp.StatusCode)
	}

	// For markdown and text, it returns the raw content
	// For JSON, it returns a JSON object.
	// But Eino Document.Content is a string.
	if resultType == ResultTypeJSON {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}

	// For markdown/text, LlamaParse might return a JSON like {"markdown": "..."} or raw content.
	// Based on common patterns, it's often a JSON response.
	// Let's check the search results again.
	// Result 4 says: interface MarkdownResult { markdown: string; }

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if val, ok := result[string(resultType)]; ok {
		return fmt.Sprint(val), nil
	}

	// Fallback: if not found by resultType key, maybe it's the whole body as string if it's not JSON?
	// But we already decoded it into a map.
	return "", fmt.Errorf("could not find %s in result", resultType)
}
