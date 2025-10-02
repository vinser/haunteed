//go:build !linux
// +build !linux

package sound

import (
	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/speaker"
)

type pulseControl struct{}

// initBackend initializes the default beep speaker backend.
func (mgr *Manager) initBackend(sampleRate beep.SampleRate, bufferSize int) error {
	if err := speaker.Init(sampleRate, bufferSize); err != nil {
		return err
	}
	speaker.Play(mgr.vol)
	return nil
}

// closeBackend shuts down the speaker backend.
func (mgr *Manager) closeBackend() {
	speaker.Clear()
}
