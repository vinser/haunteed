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

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
	"github.com/gopxl/beep/v2/wav"
	"github.com/vinser/haunteed/internal/embeddata"
)

// Sound names
const (
	// SFX
	STEP        = "step.wav"        // -- Step on the floor
	STEP_BUMP   = "step_bump.wav"   // Bump a wall
	STEP_CREAKY = "step_creaky.wav" // Step on creaky floor
	PICK_CRUMB  = "pick_crumb.wav"  // Pick up a breadcrumb (you are on the right way to the floor exit)
	EAT_PELLET  = "eat_pellet.wav"  // Eat power pellet
	KILL_GHOST  = "kill_ghost.wav"  // Kill a ghost
	WALL_BREAK  = "wall_break.wav"  // Wall crambling
	FUSE_ARC    = "fuse_arc.wav"    // Fuse arc
	FUSE_TOGGLE = "fuse_toggle.wav" // Fuse toggle
	LOSE_LIFE   = "lose_life.wav"   // Lose a life and start respawning
	GAME_OVER   = "game_over.wav"   // Game over
	HIGH_SCORE  = "high_score.wav"  // New high score
	QUIT        = "quit.wav"        // Quit game
	// Transitions
	TRANSITION_UP   = "transition_up.wav"   // Up stairs
	TRANSITION_DOWN = "transition_down.wav" // Down stairs
	// Melody
	INTRO      = "intro.wav" // Splash screen background music
	RESPAWNING = "respawning.wav"
	PAUSE_GAME = "pause_game.wav" // Pause game background music
	// UI
	UI_CLICK  = "ui_click.wav"  // Ok
	UI_SAVE   = "ui_save.wav"   // Ok
	UI_CANCEL = "ui_cancel.wav" // Ok
)

const CommonSampleRate = 44100 // Common sample rate for normalization for all sounds

// Manager controls the loading and playback of audio samples.
type Manager struct {
	mu         sync.Mutex
	samples    map[string]*beep.Buffer
	ctrl       map[string]*beep.Ctrl
	mix        *beep.Mixer
	format     beep.Format
	vol        *effects.Volume    // master volume
	sampleVols map[string]float64 // per-sample volume in dB

	backend   any           // backend-specific data
	pulseCtrl *pulseControl // PulseAudio control for immediate stop
}

// NewManager initializes the audio system and creates a new Manager.
func NewManager(sampleRate beep.SampleRate) (*Manager, error) {
	bufferSize := sampleRate.N(time.Second / 10)

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
		Volume:   0,
		Silent:   false,
	}

	if err := mgr.initBackend(sampleRate, bufferSize); err != nil {
		mgr.vol.Silent = true
		return mgr, err
	}

	return mgr, nil
}

func (mgr *Manager) Close() {
	if mgr == nil {
		return
	}
	mgr.closeBackend()
}

func (mgr *Manager) LoadSamples() error {
	if mgr == nil {
		return errors.New("sound manager is nil")
	}
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
	if mgr == nil {
		return errors.New("sound manager is nil")
	}
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
	if mgr == nil {
		return
	}
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	if mgr.vol != nil {
		mgr.vol.Volume = db
	}
}

func (mgr *Manager) SetVolume(name string, db float64) {
	if mgr == nil {
		return
	}
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	mgr.sampleVols[name] = db
}

// playInternal plays the sample by name, optionally looping it.
func (mgr *Manager) playInternal(name string, loop bool, onEnd func()) error {
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
		stream, _ = beep.Loop2(buf.Streamer(0, buf.Len()))
	} else {
		stream = buf.Streamer(0, buf.Len())
	}

	// If a callback is provided for a non-looping sound, sequence it.
	if !loop && onEnd != nil {
		stream = beep.Seq(stream, beep.Callback(onEnd))
	}

	// Wrap with per-sample volume
	vol := &effects.Volume{
		Streamer: stream,
		Base:     2,
		Volume:   mgr.sampleVols[name], // default 0 if not set
		Silent:   false,
	}

	ctrl := &beep.Ctrl{Streamer: vol, Paused: false}

	mgr.mix.Add(ctrl)
	mgr.ctrl[name] = ctrl
	return nil
}

// Play stops current playback of the sample (if any) and plays it from the start.
func (mgr *Manager) Play(name string) error {
	return mgr.playInternal(name, false, nil)
}

// PlayWithVolume plays the sample with specified volume in dB.
func (mgr *Manager) PlayWithVolume(name string, db float64) error {
	mgr.SetVolume(name, db)
	return mgr.playInternal(name, false, nil)
}

// PlayWithCallback plays a sample and executes a callback function when it finishes.
// The callback will not be executed if the sound is stopped manually or if it's a looping sound.
func (mgr *Manager) PlayWithCallback(name string, onEnd func()) error {
	return mgr.playInternal(name, false, onEnd)
}

// PlayLoop plays the sample in a continuous loop until stopped.
func (mgr *Manager) PlayLoop(name string) error {
	return mgr.playInternal(name, true, nil)
}

// PlayLoopWithVolume plays the sample in a continuous loop with specified volume.
func (mgr *Manager) PlayLoopWithVolume(name string, db float64) error {
	mgr.SetVolume(name, db)
	return mgr.playInternal(name, true, nil)
}

// MakeSequence combines the given samples into a single sequence sample and adds it to the manager.
// The sequence can then be played/looped/stopped by its name like any other sample.
func (mgr *Manager) MakeSequence(seqName string, sampleNames ...string) error {
	if mgr == nil {
		return errors.New("Ssound manager is nil")
	}
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
	if mgr == nil || mgr.ctrl == nil {
		return
	}
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	for _, name := range names {
		if ctrl, ok := mgr.ctrl[name]; ok {
			// Setting the streamer to nil signals the mixer to remove this sound
			// on the next processing tick. This is the correct way to stop and
			// clean up a sound, preventing it from leaking in the mixer.
			ctrl.Streamer = nil
			delete(mgr.ctrl, name)
		}
	}
}

// StopAll halts playback of all currently playing samples.
func (mgr *Manager) StopAll() {
	if mgr == nil {
		return
	}
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	for _, ctrl := range mgr.ctrl {
		ctrl.Streamer = nil
	}
	mgr.ctrl = make(map[string]*beep.Ctrl)
}

// Mute disables all audio output.
func (mgr *Manager) Mute() {
	if mgr == nil {
		return
	}
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	mgr.vol.Silent = true
}

// Unmute enables audio output.
func (mgr *Manager) Unmute() {
	if mgr == nil {
		return
	}
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	mgr.vol.Silent = false
}

// Initialize creates and loads a sound manager.
// It returns the manager and a boolean indicating if initialization failed (and thus should be muted).
func Initialize() (*Manager, bool) {
	soundMgr, err := NewManager(CommonSampleRate)
	if err != nil {
		return nil, true // Muted due to init error
	}
	if err := soundMgr.LoadSamples(); err != nil {
		return nil, true // Muted due to load error
	}
	return soundMgr, false
}

// Close stops the speaker and frees resources.
// func (mgr *Manager) Close() {
// 	speaker.Clear()
// }
