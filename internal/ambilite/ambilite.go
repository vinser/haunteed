package ambilite

import (
	"math"
	"time"

	"github.com/soniakeys/meeus/v3/julian"
	"github.com/soniakeys/meeus/v3/sidereal"
	"github.com/soniakeys/meeus/v3/solar"
)

// Intensity returns ambient light intensity [0.0, 1.0] for given local time, lat/lon, and timezone.
// Implements handling of polar day/night based on civil twilight (-6°).
func Intensity(now time.Time, lat, lon float64, tz string) float64 {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return 0.0
	}
	localNow := now.In(loc)
	date := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, loc)

	// Find dawn, sunrise, sunset, and dusk
	dawn, dawnOk := findSolarEventOptional(date, lat, lon, -6, false)
	sunrise, _ := findSolarEventOptional(date, lat, lon, 0, false)
	sunset, _ := findSolarEventOptional(date, lat, lon, 0, true)
	dusk, duskOk := findSolarEventOptional(date, lat, lon, -6, true)

	// Detect polar day/night:
	if !dawnOk || !duskOk || dawn.After(dusk) {
		// Find max altitude at local noon
		noon := time.Date(date.Year(), date.Month(), date.Day(), 12, 0, 0, 0, loc)
		maxAlt := solarAltitude(noon.UTC(), lat, lon)
		if maxAlt > -6 {
			// Polar day or civil twilight day — bright
			return 1.0
		}
		// Polar night or no civil twilight — dark
		return 0.0
	}

	// Normal day/night cycle with dawn < dusk
	switch {
	case localNow.Before(dawn):
		return 0.0 // Night
	case localNow.Before(sunrise):
		return interpolate(dawn, sunrise, localNow) // Dawn
	case localNow.Before(sunset):
		return 1.0 // Day
	case localNow.Before(dusk):
		return 1.0 - interpolate(sunset, dusk, localNow) // Dusk
	default:
		return 0.0
	}
}

// findSolarEventOptional returns event time and bool if found, or false if no event found.
func findSolarEventOptional(date time.Time, lat, lon float64, targetAlt float64, afterNoon bool) (time.Time, bool) {
	loc := date.Location()
	start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, loc)
	end := start.Add(24 * time.Hour)
	noon := time.Date(date.Year(), date.Month(), date.Day(), 12, 0, 0, 0, loc)

	// Check altitudes at kmidnight and noon key points
	midnightAlt := solarAltitude(start.UTC(), lat, lon)
	noonAlt := solarAltitude(noon.UTC(), lat, lon)
	// Check for sign change around target altitude in either half of the day
	crosses := (midnightAlt-targetAlt)*(noonAlt-targetAlt) <= 0

	if !crosses {
		return time.Time{}, false // No event on this day
	}

	// Binary search between 0:00 and 24:00 to find when solar altitude crosses targetAlt
	const epsilon = time.Minute
	for end.Sub(start) > epsilon {
		mid := start.Add(end.Sub(start) / 2)
		alt := solarAltitude(mid.UTC(), lat, lon)

		if (alt > targetAlt) == afterNoon {
			start = mid
		} else {
			end = mid
		}
	}

	return start.Round(time.Minute), true
}

// solarAltitude returns solar altitude in degrees for UTC time t, lat and lon.
func solarAltitude(t time.Time, lat, lon float64) float64 {
	jd := julian.TimeToJD(t)
	θ := sidereal.Apparent(jd).Rad()
	θ += lon * math.Pi / 180
	ra, dec := solar.ApparentEquatorial(jd)
	H := θ - ra.Rad()
	H = math.Mod(H+2*math.Pi, 2*math.Pi)
	φ := lat * math.Pi / 180
	δ := dec.Rad()
	sinAlt := math.Sin(φ)*math.Sin(δ) + math.Cos(φ)*math.Cos(δ)*math.Cos(H)
	return math.Asin(sinAlt) * 180 / math.Pi
}

// interpolate linear from 0 to 1 between start and end times
func interpolate(start, end, current time.Time) float64 {
	if !end.After(start) {
		return 1.0
	}
	total := end.Sub(start).Seconds()
	elapsed := current.Sub(start).Seconds()
	return max(0.0, min(1.0, elapsed/total))
}
