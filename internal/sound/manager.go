// Package sound manages playback of audio samples with support for interrupting,
// resampling to a unified format, and avoiding overlapping playback of the same sample.
package sound

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"log"
	"path"
	"sync"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
	"github.com/vinser/haunteed/internal/embeddata"
)

// Sound names
const (
	SPLASH       = "haunteed_splash.wav"
	CHOMP        = "haunteed_chomp.wav"
	DEATH        = "haunteed_death.wav"
	EATFRUIT     = "haunteed_eatfruit.wav"
	EATGHOST     = "haunteed_eatghost.wav"
	EXTRAPAC     = "haunteed_extrapac.wav"
	INTERMISSION = "haunteed_intermission.wav"
)

const CommonSampleRate = 44100 // Common sample rate for normalization for all sounds

type Sound struct {
	Name   string
	Stream beep.StreamSeekCloser
	Format beep.Format
}

// Manager controls the loading and playback of audio samples.
type Manager struct {
	mu      sync.Mutex
	samples map[string]*beep.Buffer // Resampled and stored samples
	ctrl    map[string]*beep.Ctrl   // One active playback per sample
	mix     *beep.Mixer
	format  beep.Format
	muted   bool
	vol     *effects.Volume
}

// NewManager initializes the audio system and creates a new Manager.
func NewManager(sampleRate beep.SampleRate) (*Manager, error) {
	bufferSize := sampleRate.N(time.Second / 10) // 100 ms buffer
	if err := speaker.Init(sampleRate, bufferSize); err != nil {
		return nil, err
	}

	mgr := &Manager{
		samples: make(map[string]*beep.Buffer),
		ctrl:    make(map[string]*beep.Ctrl),
		mix:     &beep.Mixer{},
		format:  beep.Format{SampleRate: sampleRate, NumChannels: 1, Precision: 2},
	}

	speaker.Play(mgr.mix)

	return mgr, nil
}

func (mgr *Manager) LoadSamples() error {
	// Open the embedded ZIP archive
	soundsZip, err := embeddata.ReadSoundsZip()
	if err != nil {
		log.Fatalf("Failed to read embedded sounds.zip: %v", err)
	}
	reader, err := zip.NewReader(bytes.NewReader(soundsZip), int64(len(soundsZip)))
	if err != nil {
		log.Fatalf("Error reading embedded ZIP: %v", err)
	}

	// Iterate through the files in the ZIP archive
	for _, file := range reader.File {
		// Open the file
		rc, err := file.Open()
		if err != nil {
			log.Fatalf("Error reading files in the ZIP archive: %v", err)
		}
		defer rc.Close()

		// Check if the file matches one of the sound constants
		fileName := path.Base(file.Name)
		switch fileName {
		case SPLASH, CHOMP, DEATH, EATFRUIT, EATGHOST, EXTRAPAC, INTERMISSION:
			// Load the file into memory to enable seeking
			buf := new(bytes.Buffer)
			_, err = io.Copy(buf, rc)
			if err != nil {
				log.Fatalf("Error reading file into memory: %v", err)
			}
			err = mgr.LoadWAV(fileName, buf.Bytes())
			if err != nil {
				log.Fatalf("Error loading WAV sample %s: %v", fileName, err)
			}
		}
	}

	return nil
}

// LoadWAV loads and resamples a WAV sample into memory.
func (mgr *Manager) LoadWAV(name string, data []byte) error {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	stream, format, err := wav.Decode(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer stream.Close()

	// Resample to match manager format
	resampled := beep.Resample(3, format.SampleRate, mgr.format.SampleRate, stream)
	buf := beep.NewBuffer(mgr.format)
	buf.Append(resampled)

	mgr.samples[name] = buf
	return nil
}

// playInternal plays the sample by name, optionally looping it.
func (mgr *Manager) playInternal(name string, loop bool) error {
	if mgr == nil {
		return errors.New("sound manager is nil")
	}
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	if mgr.samples == nil {
		return errors.New("sound samples map is nil")
	}

	buf, ok := mgr.samples[name]
	if !ok {
		return errors.New("sample not loaded: " + name)
	}

	// Interrupt previous if exists
	if ctrl, exists := mgr.ctrl[name]; exists {
		ctrl.Paused = true
	}

	var stream beep.Streamer
	if loop {
		stream = beep.Loop(-1, buf.Streamer(0, buf.Len()))
	} else {
		stream = buf.Streamer(0, buf.Len())
	}
	ctrl := &beep.Ctrl{Streamer: stream, Paused: false}

	var streamWithMute beep.Streamer = ctrl
	if mgr.muted {
		streamWithMute = beep.Callback(func() {})
	}

	mgr.mix.Add(streamWithMute)
	mgr.ctrl[name] = ctrl
	return nil
}

// Play stops current playback of the sample (if any) and plays it from the start.
func (mgr *Manager) Play(name string) error {
	return mgr.playInternal(name, false)
}

// PlayLoop plays the sample in a continuous loop until stopped.
func (mgr *Manager) PlayLoop(name string) error {
	return mgr.playInternal(name, true)
}

// Stop halts playback of the sample if it's currently playing.
func (mgr *Manager) Stop(name string) {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	if ctrl, ok := mgr.ctrl[name]; ok {
		ctrl.Paused = true
		delete(mgr.ctrl, name)
	}
}

// StopAll halts playback of all currently playing samples.
func (mgr *Manager) StopAll() {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	for name, ctrl := range mgr.ctrl {
		ctrl.Paused = true
		delete(mgr.ctrl, name)
	}
}

// Mute disables all audio output.
func (mgr *Manager) Mute() {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	mgr.muted = true
}

// Unmute enables audio output.
func (mgr *Manager) Unmute() {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	mgr.muted = false
}

// Close stops the speaker and frees resources.
func (mgr *Manager) Close() {
	speaker.Clear()
}
