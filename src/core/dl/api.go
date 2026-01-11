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
        "net/http"
        "net/url"
        "regexp"
        "strings"

        "ashokshau/tgmusic/src/core/cache"
)

// ApiData provides a unified interface for fetching track and playlist information from various music platforms via an API gateway.
type ApiData struct {
        Query    string
        ApiUrl   string
        APIKey   string
        Patterns map[string]*regexp.Regexp
}

var apiPatterns = map[string]*regexp.Regexp{
        "yt_playlist": regexp.MustCompile(`(?i)^(?:https?://)?(?:www\.)?(?:youtube\.com|music\.youtube\.com)/(?:playlist|watch)\?.*\blist=([\w-]+)`),
        "yt_music":    regexp.MustCompile(`(?i)^(?:https?://)?music\.youtube\.com/(?:watch|playlist)\?.*v=([\w-]+)`),
}

// NewApiData creates and initializes a new ApiData instance with the provided query.
func NewApiData(query string) *ApiData {
        return &ApiData{
                Query:  strings.TrimSpace(query),
                ApiUrl: "https://yt-dl.officialhectormanuel.workers.dev",
                Patterns: apiPatterns,
        }
}

// IsValid checks if the query is a valid URL for any of the supported platforms.
// It returns true if the URL matches a known pattern, and false otherwise.
func (a *ApiData) IsValid() bool {
        if a.Query == "" || a.ApiUrl == "" {
                return false
        }

        for _, pattern := range a.Patterns {
                if pattern.MatchString(a.Query) {
                        return true
                }
        }
        // Also check if it's a standard youtube link which might not be in apiPatterns but handled by youtube.go
        if strings.Contains(a.Query, "youtube.com") || strings.Contains(a.Query, "youtu.be") {
                return true
        }

        return false
}

// GetTrack retrieves detailed information for a single track from the API.
// It returns a cache.TrackInfo object or an error if the request fails.
func (a *ApiData) GetTrack(ctx context.Context) (cache.TrackInfo, error) {
        fullURL := fmt.Sprintf("%s/?url=%s", a.ApiUrl, url.QueryEscape(a.Query))
        resp, err := http.Get(fullURL)
        if err != nil {
                return cache.TrackInfo{}, fmt.Errorf("the GetTrack request failed: %w", err)
        }
        defer resp.Body.Close()

        if resp.StatusCode != http.StatusOK {
                return cache.TrackInfo{}, fmt.Errorf("unexpected status code while fetching the track: %s", resp.Status)
        }

        var result struct {
                Status    bool   `json:"status"`
                Title     string `json:"title"`
                Thumbnail string `json:"thumbnail"`
                Audio     string `json:"audio"`
        }

        if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
                return cache.TrackInfo{}, fmt.Errorf("failed to decode the response: %w", err)
        }

        if !result.Status {
                return cache.TrackInfo{}, errors.New("API returned false status")
        }

        trackInfo := cache.TrackInfo{
                URL:      a.Query,
                CdnURL:   result.Audio,
                Name:     result.Title,
                Cover:    result.Thumbnail,
                Platform: "youtube",
        }

        return trackInfo, nil
}

// GetInfo retrieves metadata for a track or playlist from the API.
// It returns a PlatformTracks object or an error if the request fails.
func (a *ApiData) GetInfo(ctx context.Context) (cache.PlatformTracks, error) {
        track, err := a.GetTrack(ctx)
        if err != nil {
                return cache.PlatformTracks{}, err
        }
        return cache.PlatformTracks{
                Results: []cache.MusicTrack{
                        {
                                ID:       track.TC,
                                URL:      track.URL,
                                Name:     track.Name,
                                Duration: track.Duration,
                                Cover:    track.Cover,
                        },
                },
        }, nil
}

// Search queries the API for a track. The context can be used for timeouts or cancellations.
// If the query is a valid URL, it fetches the information directly.
// It returns a PlatformTracks object or an error if the search fails.
func (a *ApiData) Search(ctx context.Context) (cache.PlatformTracks, error) {
        // For now, since we only use the new API which takes a URL,
        // we assume if it's not a URL it might be a search query that youtube.go handles.
        // But to satisfy the interface, we return GetInfo if it's a valid URL.
        if a.IsValid() {
                return a.GetInfo(ctx)
        }
        return cache.PlatformTracks{}, errors.New("search is handled by youtube module")
}

// downloadTrack downloads a track using the API.
// It returns the file path of the downloaded track or an error if the download fails.
func (a *ApiData) downloadTrack(ctx context.Context, info cache.TrackInfo, video bool) (string, error) {
        downloader, err := NewDownload(ctx, info)
        if err != nil {
                return "", fmt.Errorf("failed to initialize the download: %w", err)
        }

        return downloader.Process()
}
