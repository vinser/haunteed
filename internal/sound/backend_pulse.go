//go:build linux
// +build linux

package sound

import (
	"github.com/gopxl/beep/v2"
	"github.com/jfreymuth/pulse"
)

type pulseBackend struct {
	client *pulse.Client
	stream *pulse.PlaybackStream
	format beep.Format
	done   chan struct{}
}

// pulseControl wraps a beep.Streamer and allows immediate stop
// by setting the stopped flag.
type pulseControl struct {
	streamer beep.Streamer
	stopped  bool
}

func (pc *pulseControl) Stream(buf [][2]float64) (n int, ok bool) {
	if pc.stopped {
		for i := range buf {
			buf[i][0], buf[i][1] = 0, 0
		}
		return len(buf), true
	}
	return pc.streamer.Stream(buf)
}

func (pc *pulseControl) Err() error { return nil }

// beepToFloat32Func returns a func([]float32) (int, error) that pulls from a beep.Streamer
func beepToFloat32Func(ctrl *pulseControl, channels int) func([]float32) (int, error) {
	buf := make([][2]float64, 512)
	return func(out []float32) (int, error) {
		frames := len(out) / channels
		if frames > len(buf) {
			frames = len(buf)
		}
		n, ok := ctrl.Stream(buf[:frames])
		if !ok {
			return 0, pulse.EndOfData
		}
		idx := 0
		for i := 0; i < n; i++ {
			for ch := 0; ch < channels; ch++ {
				out[idx] = float32(buf[i][ch])
				idx++
			}
		}
		return idx, nil
	}
}

// initBackend initializes PulseAudio instead of beep/speaker.
func (mgr *Manager) initBackend(sampleRate beep.SampleRate, bufferSize int) error {
	client, err := pulse.NewClient()
	if err != nil {
		return err
	}

	channels := mgr.format.NumChannels
	ctrl := &pulseControl{streamer: mgr.vol}
	float32Func := beepToFloat32Func(ctrl, channels)
	stream, err := client.NewPlayback(
		pulse.Float32Reader(float32Func),
		pulse.PlaybackLatency(0.03), // ~30ms latency for low delay
	)
	if err != nil {
		client.Close()
		return err
	}

	stream.Start()

	mgr.backend = &pulseBackend{
		client: client,
		stream: stream,
		format: mgr.format,
		done:   make(chan struct{}),
	}

	// Save control for stopping
	mgr.pulseCtrl = ctrl

	return nil
}

// StopPulsePlayback stops playback immediately by setting stopped flag
func (mgr *Manager) StopPulsePlayback() {
	if mgr.pulseCtrl != nil {
		mgr.pulseCtrl.stopped = true
	}
}

// closeBackend cleans up PulseAudio.
func (mgr *Manager) closeBackend() {
	if pb, ok := mgr.backend.(*pulseBackend); ok {
		pb.stream.Close()
		pb.client.Close()
	}
}
