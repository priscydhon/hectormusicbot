/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/priscydhon/hectormusicbot
 */

package dl

import (
        "context"
        "encoding/json"
        "errors"
        "fmt"
        "io"
        "log"
        "net/http"
        "net/url"
        "strings"

        "ashokshau/tgmusic/src/config"
)

// CustomYTAPIResponse is the structure for the custom YouTube API response
type CustomYTAPIResponse struct {
        Status    bool              `json:"status"`
        Creator   string            `json:"creator"`
        Title     string            `json:"title"`
        Thumbnail string            `json:"thumbnail"`
        Audio     string            `json:"audio"`
        Videos    map[string]string `json:"videos"`
}

// FetchFromCustomYTAPI calls the custom YouTube downloader API
func FetchFromCustomYTAPI(ctx context.Context, videoURL string) (*CustomYTAPIResponse, error) {
        apiURL := strings.TrimRight(config.Conf.ApiUrl, "/")
        if apiURL == "" || apiURL == "https://tgmusic.fallenapi.fun" {
                return nil, errors.New("custom API not configured")
        }

        fullURL := fmt.Sprintf("%s?url=%s", apiURL, url.QueryEscape(videoURL))

        req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
        if err != nil {
                return nil, fmt.Errorf("failed to create request: %w", err)
        }

        req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

        resp, err := http.DefaultClient.Do(req)
        if err != nil {
                return nil, fmt.Errorf("API request failed: %w", err)
        }
        defer func(Body io.ReadCloser) {
                _ = Body.Close()
        }(resp.Body)

        if resp.StatusCode != http.StatusOK {
                return nil, fmt.Errorf("API returned status code: %d", resp.StatusCode)
        }

        var apiResp CustomYTAPIResponse
        if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
                return nil, fmt.Errorf("failed to decode API response: %w", err)
        }

        if !apiResp.Status || apiResp.Audio == "" {
                return nil, errors.New("API response invalid or no audio available")
        }

        return &apiResp, nil
}

// DownloadWithCustomAPI downloads using the custom YouTube API
func (y *YouTubeData) DownloadWithCustomAPI(ctx context.Context, videoID string, video bool) (string, error) {
        videoURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)

        apiResp, err := FetchFromCustomYTAPI(ctx, videoURL)
        if err != nil {
                return "", fmt.Errorf("custom API fetch failed: %w", err)
        }

        downloadURL := apiResp.Audio
        if video && len(apiResp.Videos) > 0 {
                // Try to get best video quality
                if url, ok := apiResp.Videos["720"]; ok {
                        downloadURL = url
                } else if url, ok := apiResp.Videos["480"]; ok {
                        downloadURL = url
                } else {
                        // Get first available video
                        for _, url := range apiResp.Videos {
                                downloadURL = url
                                break
                        }
                }
        }

        log.Printf("Downloading from custom API: %s", downloadURL)
        filePath, err := DownloadFile(ctx, downloadURL, "", false)
        if err != nil {
                return "", fmt.Errorf("download failed: %w", err)
        }

        return filePath, nil
}

// IsCustomAPIConfigured checks if custom YouTube API is configured
func IsCustomAPIConfigured() bool {
        apiURL := strings.TrimRight(config.Conf.ApiUrl, "/")
        return apiURL != "" && apiURL != "https://tgmusic.fallenapi.fun"
}
