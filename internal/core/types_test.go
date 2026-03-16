package core

import (
	"encoding/json"
	"testing"
)

func TestTranslateResponseJSONFieldNames(t *testing.T) {
	resp := TranslateResponse{
		Provider:               "google",
		SourceLanguage:         "en",
		TargetLanguage:         "zh",
		Text:                   "How are you?",
		Translation:            "你好吗？",
		DetectedSourceLanguage: "en",
		Metadata: map[string]any{
			"model": "general/nmt",
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	for _, key := range []string{
		"provider",
		"source_language",
		"target_language",
		"text",
		"translation",
		"detected_source_language",
		"metadata",
	} {
		if _, ok := got[key]; !ok {
			t.Fatalf("expected JSON key %q in %s", key, string(data))
		}
	}
}
