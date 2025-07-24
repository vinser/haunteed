package geoip

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"
)

// LocationInfo stores geographic data and timezone.
type LocationInfo struct {
	Continent string    `json:"continent"`
	Country   string    `json:"country"`
	City      string    `json:"city"`
	Lat       float64   `json:"lat"`
	Lon       float64   `json:"lon"`
	Timezone  string    `json:"timezone"`
	IP        string    `json:"query"`
	TimeStamp time.Time `json:"-"`
}

// geoipClient is used to store configuration and cache.
type geoipClient struct {
	url         string
	httpTimeout time.Duration
	cacheTTL    time.Duration

	mu        sync.Mutex
	cache     *LocationInfo
	cacheTime time.Time
}

// Default client with default settings.
var Default = &geoipClient{
	url:         "http://ip-api.com/json/",
	httpTimeout: 5 * time.Second,
	cacheTTL:    1 * time.Hour,
}

// SetCacheTTL allows setting the cache lifetime.
func SetCacheTTL(d time.Duration) {
	Default.cacheTTL = d
}

// SetHTTPTimeout sets the timeout for HTTP requests.
func SetHTTPTimeout(d time.Duration) {
	Default.httpTimeout = d
}

// GetLocationInfo returns coordinates, location and error.
func GetLocationInfo() (*LocationInfo, error) {
	return Default.GetLocationInfo()
}

// GetLocationInfo If the cache is not expired, loads data with cache.
func (c *geoipClient) GetLocationInfo() (*LocationInfo, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Return cached data if it is not expired
	if c.cache != nil && time.Since(c.cacheTime) <= c.cacheTTL {
		return c.cache, nil
	}

	// New request
	client := http.Client{
		Timeout: c.httpTimeout,
	}
	resp, err := client.Get(c.url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("geoip: non-200 response from API")
	}

	info := &LocationInfo{}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	if info.Timezone == "" {
		return nil, errors.New("geoip: timezone not provided")
	}

	_, err = time.LoadLocation(info.Timezone) // Check if timezone is valid
	if err != nil {
		return nil, err
	}

	// Refresh cache
	c.cache = info
	c.cacheTime = time.Now()

	return info, nil
}
