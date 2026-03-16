package core

import "context"

type TranslateRequest struct {
	Text           string            `json:"text"`
	SourceLanguage string            `json:"source_language"`
	TargetLanguage string            `json:"target_language"`
	Provider       string            `json:"provider"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

type TranslateResponse struct {
	Provider               string         `json:"provider"`
	SourceLanguage         string         `json:"source_language"`
	TargetLanguage         string         `json:"target_language"`
	Text                   string         `json:"text"`
	Translation            string         `json:"translation"`
	DetectedSourceLanguage string         `json:"detected_source_language,omitempty"`
	Metadata               map[string]any `json:"metadata,omitempty"`
}

type Translator interface {
	Name() string
	Translate(ctx context.Context, req TranslateRequest) (*TranslateResponse, error)
}
