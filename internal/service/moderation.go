package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/pnj-anonymous-bot/internal/config"
	"github.com/pnj-anonymous-bot/internal/logger"
	"go.uber.org/zap"
)

type SightengineResponse struct {
	Status string `json:"status"`
	Nudity struct {
		SexualActivity float64 `json:"sexual_activity"`
		SexualDisplay  float64 `json:"sexual_display"`
		Erotica        float64 `json:"erotica"`
		Suggestive     float64 `json:"suggestive"`
	} `json:"nudity"`
	Weapon  float64 `json:"weapon"`
	Alcohol float64 `json:"alcohol"`
	Drugs   float64 `json:"drugs"`
	Request struct {
		ID string `json:"id"`
	} `json:"request"`
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type ModerationService struct {
	apiUser   string
	apiSecret string
	enabled   bool
	client    *http.Client
}

func NewModerationService(cfg *config.Config) *ModerationService {
	return &ModerationService{
		apiUser:   cfg.SightengineAPIUser,
		apiSecret: cfg.SightengineAPISecret,
		enabled:   cfg.SightengineAPIUser != "" && cfg.SightengineAPISecret != "",
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (s *ModerationService) IsSafe(ctx context.Context, imageURL string) (bool, string, error) {
	if !s.enabled {
		return true, "", nil
	}

	parsedURL, err := url.Parse(imageURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
		return true, "", fmt.Errorf("invalid image URL: %s", imageURL)
	}

	u, err := url.Parse("https://api.sightengine.com/1.0/check.json")
	if err != nil {
		return true, "", fmt.Errorf("failed to parse sightengine API URL: %w", err)
	}
	q := u.Query()
	q.Set("models", "nudity-2.0,wad")
	q.Set("api_user", s.apiUser)
	q.Set("api_secret", s.apiSecret)
	q.Set("url", imageURL)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return true, "", fmt.Errorf("failed to create moderation request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return true, "", fmt.Errorf("sightengine request failed: %w", err)
	}
	defer resp.Body.Close()

	var data SightengineResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return true, "", fmt.Errorf("failed to decode sightengine response: %w", err)
	}

	if data.Status == "failure" {
		logger.Warn("Sightengine API reported failure",
			zap.Int("code", data.Error.Code),
			zap.String("message", data.Error.Message),
			zap.String("request_id", data.Request.ID),
		)
		return true, "", nil
	}

	if data.Nudity.SexualActivity > 0.5 || data.Nudity.SexualDisplay > 0.5 || data.Nudity.Erotica > 0.8 {
		return false, "Konten mengandung unsur seksual atau ketelanjangan (NSFW).", nil
	}

	if data.Weapon > 0.8 {
		return false, "Konten mengandung unsur senjata (WAD).", nil
	}

	if data.Drugs > 0.8 {
		return false, "Konten mengandung unsur narkoba (WAD).", nil
	}

	return true, "", nil
}

func (s *ModerationService) IsEnabled() bool {
	return s.enabled
}
