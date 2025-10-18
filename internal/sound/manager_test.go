package sound

import (
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gopxl/beep/v2"
)

var (
	soundMgr *Manager
	once     sync.Once
)

func TestMain(m *testing.M) {
	if os.Getenv("SKIP_AUDIO") == "1" { // For CI without sound on host
		os.Exit(0)
	}
	once.Do(func() {
		var err error
		soundMgr, err = NewManager(beep.SampleRate(44100))
		if err != nil {
			log.Fatalf("Failed to create sound manager: %v", err)
		}

		if err := soundMgr.LoadSamples(); err != nil {
			log.Fatalf("Failed to load samples: %v", err)
		}
	})

	code := m.Run()

	soundMgr.Close()
	os.Exit(code)
}
func TestSoundOutput(t *testing.T) {
	if err := soundMgr.Play(INTRO); err != nil {
		t.Fatalf("Failed to play INTRO: %v", err)
	}
	time.Sleep(24 * time.Second) // Adjust based on INTRO duration
}

// BenchmarkStepPlayback simulates gameplay step sounds with realistic key press delays.
func BenchmarkStepPlayback(b *testing.B) {
	// Simulate experienced player key press rate: ~100-200ms per press
	delay := 50 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := soundMgr.Play(STEP); err != nil {
			b.Fatalf("Play failed: %v", err)
		}
		time.Sleep(delay)
	}
}

// BenchmarkLoopPlayback simulates background or pause music with looping.
func BenchmarkLoopPlayback(b *testing.B) {
	// Use a longer sample for looping (e.g., INTRO or RESPAWNING)
	sampleName := INTRO // Assume this is a 1-5 second sample

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := soundMgr.PlayLoop(sampleName); err != nil {
			b.Fatalf("PlayLoop failed: %v", err)
		}
		// Simulate playback duration before potential stop/restart
		time.Sleep(5 * time.Second) // Adjust based on actual sample length
		soundMgr.StopListed(sampleName)
	}
}

// TestConcurrency remains for testing concurrent playback
func TestConcurrency(t *testing.T) {
	t.Parallel()
	numPlayers := 50
	done := make(chan bool)
	for i := 0; i < numPlayers; i++ {
		go func() {
			soundMgr.Play(STEP)
			done <- true
		}()
	}
	for i := 0; i < numPlayers; i++ {
		<-done
	}
	time.Sleep(1 * time.Second)
}
