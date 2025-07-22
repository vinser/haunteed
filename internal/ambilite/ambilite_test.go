package ambilite

import (
	"math"
	"testing"
	"time"
)

func TestAmbilite(t *testing.T) {
	tests := []struct {
		name     string
		time     time.Time
		lat, lon float64
		tz       string
		wantMin  float64
		wantMax  float64
	}{
		{
			name: "Midnight in SF",
			time: time.Date(2022, 1, 1, 0, 0, 0, 0, mustTZ("America/Los_Angeles")),
			lat:  34.03, lon: -118.15, tz: "America/Los_Angeles",
			wantMin: 0.0, wantMax: 0.0,
		},
		{
			name: "Dawn in SF",
			time: time.Date(2022, 1, 1, 6, 41, 0, 0, mustTZ("America/Los_Angeles")), // ~6:30-6:58 PST
			lat:  34.03, lon: -118.15, tz: "America/Los_Angeles",
			wantMin: 0.33, wantMax: 0.34,
		},
		{
			name: "Noon in SF",
			time: time.Date(2022, 1, 1, 12, 0, 0, 0, mustTZ("America/Los_Angeles")),
			lat:  34.03, lon: -118.15, tz: "America/Los_Angeles",
			wantMin: 1.0, wantMax: 1.0,
		},
		{
			name: "Dusk in SF",
			time: time.Date(2022, 1, 1, 17, 21, 0, 0, mustTZ("America/Los_Angeles")), // ~16:54-17:22 PST Jan 1
			lat:  34.03, lon: -118.15, tz: "America/Los_Angeles",
			wantMin: 0.02, wantMax: 0.05,
		},
		{
			name: "Night after dusk",
			time: time.Date(2022, 1, 1, 19, 0, 0, 0, mustTZ("America/Los_Angeles")),
			lat:  34.03, lon: -118.15, tz: "America/Los_Angeles",
			wantMin: 0.0, wantMax: 0.0,
		},
		{
			name: "Polar Day",
			time: time.Date(2022, 6, 21, 12, 0, 0, 0, time.UTC),
			lat:  90.0, lon: 0.0, tz: "UTC",
			wantMin: 1.0, wantMax: 1.0,
		},
		{
			name: "Polar Night",
			time: time.Date(2022, 12, 21, 12, 0, 0, 0, time.UTC),
			lat:  90.0, lon: 0.0, tz: "UTC",
			wantMin: 0.0, wantMax: 0.0,
		},
		{
			name: "Invalid TimeZone",
			time: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
			lat:  34.03, lon: -118.15, tz: "Invalid/TimeZone",
			wantMin: 0.0, wantMax: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Intensity(tt.time, tt.lat, tt.lon, tt.tz)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("Ambilite() = %f; want in [%f, %f]\n", got, tt.wantMin, tt.wantMax)
			}

		})
	}
}

func TestSolarAltitude(t *testing.T) {
	tests := []struct {
		name     string
		time     time.Time
		lat, lon float64
		tz       string
		want     float64
		delta    float64
	}{
		{
			name: "Midnight in SF",
			time: time.Date(2022, 1, 1, 0, 0, 0, 0, mustTZ("America/Los_Angeles")),
			lat:  34.03, lon: -118.15, tz: "America/Los_Angeles",
			want: -79, delta: 0.1,
		},
		{
			name: "Dawn in SF",
			time: time.Date(2022, 1, 1, 6, 41, 0, 0, mustTZ("America/Los_Angeles")), // ~6:30-6:58 PST
			lat:  34.03, lon: -118.15, tz: "America/Los_Angeles",
			want: -4, delta: 0.1,
		},
		{
			name: "Noon in SF",
			time: time.Date(2022, 1, 1, 12, 0, 0, 0, mustTZ("America/Los_Angeles")),
			lat:  34.03, lon: -118.15, tz: "America/Los_Angeles",
			want: 33.0, delta: 0.1,
		},
		{
			name: "Dusk in SF",
			time: time.Date(2022, 1, 1, 17, 22, 0, 0, mustTZ("America/Los_Angeles")), // ~16:54-17:22 PST Jan 1
			lat:  34.03, lon: -118.15, tz: "America/Los_Angeles",
			want: -6.0, delta: 0.1,
		},
		{
			name: "Night after dusk",
			time: time.Date(2022, 1, 1, 18, 24, 0, 0, mustTZ("America/Los_Angeles")),
			lat:  34.03, lon: -118.15, tz: "America/Los_Angeles",
			want: -18.0, delta: 0.1,
		},
		{
			name: "Polar Day",
			time: time.Date(2022, 6, 21, 12, 0, 0, 0, time.UTC),
			lat:  90.0, lon: 0.0, tz: "UTC",
			want: 23.0, delta: 0.5,
		},
		{
			name: "Polar Night",
			time: time.Date(2022, 12, 21, 12, 0, 0, 0, time.UTC),
			lat:  90.0, lon: 0.0, tz: "UTC",
			want: -23.0, delta: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := solarAltitude(tt.time.UTC(), tt.lat, tt.lon)
			if got < tt.want-tt.delta || got > tt.want+tt.delta {
				t.Errorf("solarAltitude() = %f; want %f degrees", got, tt.want)
			}
		})
	}
}

// interpolate returns a value between min and max based on t's position between start and end
func TestInterpolate(t *testing.T) {
	tests := []struct {
		name     string
		start    time.Time
		end      time.Time
		current  time.Time
		expected float64
	}{
		{"start after end", time.Now().Add(1 * time.Hour), time.Now(), time.Now(), 1.0},
		{"current before start", time.Now(), time.Now().Add(1 * time.Hour), time.Now().Add(-1 * time.Hour), 0.0},
		{"current at start", time.Now(), time.Now().Add(1 * time.Hour), time.Now(), 0.0},
		{"current between start and end", time.Now(), time.Now().Add(1 * time.Hour), time.Now().Add(30 * time.Minute), 0.5},
		{"current at end", time.Now(), time.Now().Add(1 * time.Hour), time.Now().Add(1 * time.Hour), 1.0},
		{"current after end", time.Now(), time.Now().Add(1 * time.Hour), time.Now().Add(2 * time.Hour), 1.0},
	}

	delta := 1e-9
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := interpolate(tt.start, tt.end, tt.current)
			if math.Abs(actual-tt.expected) > delta {
				t.Errorf("interpolate() = %v, want %v", actual, tt.expected)
			}
		})
	}
}

// mustTZ is a helper to panic on location load error (used for constants)
func mustTZ(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}
