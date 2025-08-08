// Package sound manages playback of audio samples with support for interrupting,
// resampling to a unified format, and avoiding overlapping playback of the same sample.
package sound

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"log"
	"math/rand"
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
	INTRO        = "intro.wav"
	CHOMP_CRUMB  = "chomp_crumb.wav"
	FUSE_OFF     = "fuse_off.wav"
	DEATH        = "death.wav"
	EATGHOST     = "eatghost.wav"
	INTERMISSION = "intermission.wav"
)

const CommonSampleRate = 44100 // Common sample rate for normalization for all sounds

// Manager controls the loading and playback of audio samples.
type Manager struct {
	mu         sync.Mutex
	samples    map[string]*beep.Buffer
	ctrl       map[string]*beep.Ctrl
	mix        *beep.Mixer
	format     beep.Format
	muted      bool
	vol        *effects.Volume    // master volume
	sampleVols map[string]float64 // per-sample volume in dB
}

// NewManager initializes the audio system and creates a new Manager.
func NewManager(sampleRate beep.SampleRate) (*Manager, error) {
	bufferSize := sampleRate.N(time.Second / 10)
	if err := speaker.Init(sampleRate, bufferSize); err != nil {
		return nil, err
	}

	mgr := &Manager{
		samples:    make(map[string]*beep.Buffer),
		ctrl:       make(map[string]*beep.Ctrl),
		mix:        &beep.Mixer{},
		format:     beep.Format{SampleRate: sampleRate, NumChannels: 1, Precision: 2},
		sampleVols: make(map[string]float64),
	}
	mgr.vol = &effects.Volume{
		Streamer: mgr.mix,
		Base:     2,
		Volume:   0, // 0 dB
		Silent:   false,
	}
	speaker.Play(mgr.vol)
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
		fileExt := path.Ext(fileName)
		if fileExt == ".wav" {
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

func (mgr *Manager) SetMasterVolume(db float64) {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	if mgr.vol != nil {
		mgr.vol.Volume = db
	}
}

func (mgr *Manager) SetVolume(name string, db float64) {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	mgr.sampleVols[name] = db
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
		ctrl.Streamer = nil // Drain the current streamer
	}

	var stream beep.Streamer
	if loop {
		stream = beep.Loop(-1, buf.Streamer(0, buf.Len()))
	} else {
		stream = buf.Streamer(0, buf.Len())
	}

	// Wrap with per-sample volume
	vol := &effects.Volume{
		Streamer: stream,
		Base:     2,
		Volume:   mgr.sampleVols[name], // default 0 if not set
		Silent:   false,
	}

	ctrl := &beep.Ctrl{Streamer: vol, Paused: false}

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

// PlayWithVolume plays the sample with specified volume in dB.
func (mgr *Manager) PlayWithVolume(name string, db float64) error {
	mgr.SetVolume(name, db)
	return mgr.playInternal(name, false)
}

// PlayLoop plays the sample in a continuous loop until stopped.
func (mgr *Manager) PlayLoop(name string) error {
	return mgr.playInternal(name, true)
}

// PlayLoopWithVolume plays the sample in a continuous loop with specified volume.
func (mgr *Manager) PlayLoopWithVolume(name string, db float64) error {
	mgr.SetVolume(name, db)
	return mgr.playInternal(name, true)
}

// MakeSequence combines the given samples into a single sequence sample and adds it to the manager.
// The sequence can then be played/looped/stopped by its name like any other sample.
func (mgr *Manager) MakeSequence(seqName string, sampleNames ...string) error {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	// Check all samples exist and collect their streamers
	var streamers []beep.Streamer
	for _, name := range sampleNames {
		buf, ok := mgr.samples[name]
		if !ok {
			return errors.New("sample not loaded: " + name)
		}
		streamers = append(streamers, buf.Streamer(0, buf.Len()))
	}

	// Combine into a sequence streamer
	seqStreamer := beep.Seq(streamers...)

	// Buffer the sequence so it can be replayed
	seqBuf := beep.NewBuffer(mgr.format)
	seqBuf.Append(seqStreamer)

	// Store the sequence as a new sample
	mgr.samples[seqName] = seqBuf
	return nil
}

// PlayRandom plays a random sample from the given list of names.
func (mgr *Manager) PlayRandom(sampleNames ...string) error {
	return mgr.PlayRandomWithVolume(0, sampleNames...)
}

// PlayRandomWithVolume plays a random sample from the given list of names with specified volume.
func (mgr *Manager) PlayRandomWithVolume(volume float64, sampleNames ...string) error {
	if len(sampleNames) == 0 {
		return errors.New("no samples provided for random playback")
	}
	// Select a random sample name
	randomIndex := rand.Intn(len(sampleNames))
	randomSample := sampleNames[randomIndex]
	return mgr.PlayWithVolume(randomSample, volume)
}

// StopListed stops playback of the specified samples by name.
// If a sample is not currently playing, it is ignored.
func (mgr *Manager) StopListed(names ...string) {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	for _, name := range names {
		if ctrl, ok := mgr.ctrl[name]; ok {
			ctrl.Paused = true
			delete(mgr.ctrl, name)
		}
	}
}

// StopAll halts playback of all currently playing samples.
func (mgr *Manager) StopAll() {
	for name := range mgr.ctrl {
		mgr.StopListed(name)
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
