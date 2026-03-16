package google

import (
	"context"
	"fmt"

	translate "cloud.google.com/go/translate/apiv3"
	"cloud.google.com/go/translate/apiv3/translatepb"
	gax "github.com/googleapis/gax-go/v2"
	"google.golang.org/api/option"

	"qiao/internal/config"
	"qiao/internal/core"
)

type translationClient interface {
	TranslateText(context.Context, *translatepb.TranslateTextRequest, ...gax.CallOption) (*translatepb.TranslateTextResponse, error)
	Close() error
}

type clientFactory func(context.Context, ...option.ClientOption) (translationClient, error)

type Provider struct {
	projectID       string
	location        string
	credentialsFile string
	client          translationClient
	clientFactory   clientFactory
}

func New(cfg config.Config) (core.Translator, error) {
	providerConfig, _ := cfg.ProviderConfig("google")

	projectID := providerConfig["project_id"]
	if projectID == "" {
		return nil, fmt.Errorf(`google provider requires "project_id"`)
	}

	location := providerConfig["location"]
	if location == "" {
		location = "global"
	}

	return &Provider{
		projectID:       projectID,
		location:        location,
		credentialsFile: providerConfig["credentials_file"],
		clientFactory:   newTranslationClient,
	}, nil
}

func (p *Provider) Name() string {
	return "google"
}

func (p *Provider) Translate(ctx context.Context, req core.TranslateRequest) (*core.TranslateResponse, error) {
	client, err := p.getClient(ctx)
	if err != nil {
		return nil, err
	}

	googleReq := &translatepb.TranslateTextRequest{
		Parent:             fmt.Sprintf("projects/%s/locations/%s", p.projectID, p.location),
		Contents:           []string{req.Text},
		MimeType:           "text/plain",
		TargetLanguageCode: req.TargetLanguage,
	}
	if req.SourceLanguage != "" && req.SourceLanguage != "auto" {
		googleReq.SourceLanguageCode = req.SourceLanguage
	}

	googleResp, err := client.TranslateText(ctx, googleReq)
	if err != nil {
		return nil, err
	}

	resp := &core.TranslateResponse{
		Provider:       p.Name(),
		SourceLanguage: req.SourceLanguage,
		TargetLanguage: req.TargetLanguage,
		Text:           req.Text,
	}
	if len(googleResp.GetTranslations()) == 0 {
		return resp, nil
	}

	first := googleResp.GetTranslations()[0]
	resp.Translation = first.GetTranslatedText()
	resp.DetectedSourceLanguage = first.GetDetectedLanguageCode()
	if model := first.GetModel(); model != "" {
		resp.Metadata = map[string]any{
			"model": model,
		}
	}

	return resp, nil
}

func (p *Provider) getClient(ctx context.Context) (translationClient, error) {
	if p.client != nil {
		return p.client, nil
	}

	options := []option.ClientOption{}
	if p.credentialsFile != "" {
		options = append(options, option.WithCredentialsFile(p.credentialsFile))
	}

	client, err := p.clientFactory(ctx, options...)
	if err != nil {
		return nil, err
	}

	p.client = client

	return client, nil
}

type cloudTranslationClient struct {
	client *translate.TranslationClient
}

func newTranslationClient(ctx context.Context, opts ...option.ClientOption) (translationClient, error) {
	client, err := translate.NewTranslationClient(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return &cloudTranslationClient{client: client}, nil
}

func (c *cloudTranslationClient) TranslateText(ctx context.Context, req *translatepb.TranslateTextRequest, opts ...gax.CallOption) (*translatepb.TranslateTextResponse, error) {
	return c.client.TranslateText(ctx, req, opts...)
}

func (c *cloudTranslationClient) Close() error {
	return c.client.Close()
}
