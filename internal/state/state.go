package state

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"hash/crc32"
	"os"
	"path/filepath"
	"time"

	"github.com/denisbrodbeck/machineid"
	"github.com/vinser/haunteed/internal/geoip"
	"github.com/vinser/haunteed/internal/sound"
)

// State holds persistent game data such as high scores.
type State struct {
	GameMode     string             `json:"game_mode"`     // Current game mode: easy, noisy or crazy
	NightOption  string             `json:"crazy_night"`   // Night option for crazy mode: never, always or real
	SpriteSize   string             `json:"sprite_size"`   // Sprite size: small, medium, large
	Mute         bool               `json:"mute"`          // Mute all sounds
	FloorSeeds   map[int]int64      `json:"floor_seeds"`   // Seed for each floor to reproduce the same sequence of mazes
	EasyScore    int                `json:"easy_score"`    // Easy mode high score
	NoisyScore   int                `json:"noisy_score"`   // Noisy mode high score
	CrazyScore   int                `json:"crazy_score"`   // Crazy mode high score
	TestScore    int                `json:"test_score"`    // Test mode high score
	LocationInfo geoip.LocationInfo `json:"location_info"` // Location information
	// Sounds       map[string]sound.Sound `json:"-"`             // Map of loaded sounds
	SoundManager *sound.Manager `json:"-"` //
}

const (
	// Game modes
	ModeEasy    = "easy"
	ModeNoisy   = "noisy"
	ModeCrazy   = "crazy"
	ModeTest    = "test"
	ModeDefault = ModeEasy

	// Night options
	NightNever   = "never"
	NightAlways  = "always"
	NightReal    = "real"
	NightDefault = NightReal

	// Sprite sizes
	SpriteSmall   = "small"
	SpriteMedium  = "medium"
	SpriteLarge   = "large"
	SpriteDefault = SpriteMedium
)

var encryptionKey = generateKey()

// generateKey creates a 32-byte AES key from system-specific data.
func generateKey() []byte {
	appID, err := machineid.ProtectedID("haunteed")
	if err != nil {
		appID = "default-haunteed-id" // Fallback if machine ID fails
	}
	sum := sha256.Sum256([]byte(appID))
	return sum[:]
}

// SetMute toggles the mute state and applies it to the sound manager.
func (s *State) SetMute(mute bool) {
	s.Mute = mute
	if s.SoundManager == nil {
		return
	}
	if s.Mute {
		s.SoundManager.Mute()
	} else {
		s.SoundManager.Unmute()
	}
}

// UpdateAndSave updates the state with new game results and persists it to a file.
func (s *State) UpdateAndSave(floor, score int, seed int64) error {
	switch s.GameMode {
	case ModeEasy:
		if score > s.EasyScore {
			s.EasyScore = score
		}
	case ModeNoisy:
		if score > s.NoisyScore {
			s.NoisyScore = score
		}
	case ModeCrazy:
		if score > s.CrazyScore {
			s.CrazyScore = score
		}
	case ModeTest:
		if score > s.TestScore {
			s.TestScore = score
		}
	}
	// Ensure the seed for the current floor is saved if it's new.
	s.FloorSeeds[floor] = seed
	return s.Save()
}

// Save persists the current state to an encrypted file with an integrity check.
func (s *State) Save() error {
	path, err := getSavePath()
	if err != nil {
		return err
	}

	// Serialize to JSON
	raw, err := json.Marshal(s)
	if err != nil {
		return err
	}

	// Prepend CRC32 checksum
	crc := crc32.ChecksumIEEE(raw)
	data := make([]byte, 4+len(raw))
	binary.LittleEndian.PutUint32(data[:4], crc)
	copy(data[4:], raw)

	// Encrypt
	encrypted, err := encrypt(data)
	if err != nil {
		return err
	}

	// Save to file
	return os.WriteFile(path, encrypted, 0644)
}

var fallbackLocation = &geoip.LocationInfo{
	Continent: "Europe",
	Country:   "The Netherlands",
	City:      "Amsterdam",
	Lat:       52.3728,
	Lon:       4.88805,
	Timezone:  "Europe/Amsterdam",
	IP:        "193.0.11.51",
	TimeStamp: time.Now(),
}

func New() *State {
	seeds := make(map[int]int64)
	seeds[0] = time.Now().UnixNano()
	loc, err := geoip.GetLocationInfo()
	if err != nil {
		loc = fallbackLocation
	}
	soundMgr, soundInitFailed := initializeSound()
	s := &State{
		GameMode:     ModeDefault,
		NightOption:  NightDefault,
		SpriteSize:   SpriteDefault,
		FloorSeeds:   seeds,
		LocationInfo: *loc,
		SoundManager: soundMgr,
	}
	s.SetMute(soundInitFailed)
	return s
}

// Load reads the state from disk, decrypts and verifies it.
func Load() *State {
	s := &State{}

	path, err := getSavePath()
	if err != nil {
		return New()
	}

	encrypted, err := os.ReadFile(path)
	if err != nil {
		return New()
	}

	decrypted, err := decrypt(encrypted)
	if err != nil || len(decrypted) < 5 {
		return New()
	}

	crcStored := binary.LittleEndian.Uint32(decrypted[:4])
	payload := decrypted[4:]
	if crc32.ChecksumIEEE(payload) != crcStored {
		return New()
	}

	// Unmarshal into the current state struct
	if err = json.Unmarshal(payload, s); err != nil {
		return New() // Corrupted JSON
	}

	// Initialize sound manager. If it fails, the game is forced into mute mode,
	// but we still respect the user's saved preference for the next session.
	soundMgr, soundInitFailed := initializeSound()
	s.SoundManager = soundMgr
	if soundInitFailed {
		s.Mute = true
	}
	// Apply the loaded mute state to the newly created sound manager.
	s.SetMute(s.Mute)
	return s
}

// initializeSound creates and loads a sound manager.
// It returns the manager and a boolean indicating if initialization failed (and thus should be muted).
func initializeSound() (*sound.Manager, bool) {
	soundMgr, err := sound.NewManager(sound.CommonSampleRate)
	if err != nil {
		return nil, true // Muted due to init error
	}
	if err := soundMgr.LoadSamples(); err != nil {
		return nil, true // Muted due to load error
	}
	return soundMgr, false
}

// ======================
// 🔐 AES Encryption
// ======================

func encrypt(plain []byte) ([]byte, error) {
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plain, nil), nil
}

func decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}
	return gcm.Open(nil, ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():], nil)
}

// getSavePath returns the path to the save file inside the user config directory.
func getSavePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	saveDir := filepath.Join(configDir, "haunteed")
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(saveDir, "state.dat"), nil
}

func (s *State) GetHighScore() int {
	switch s.GameMode {
	case ModeEasy:
		return s.EasyScore
	case ModeNoisy:
		return s.NoisyScore
	case ModeCrazy:
		return s.CrazyScore
	case ModeTest:
		return s.TestScore
	default:
		return 0
	}
}

func (s *State) SetHighScore(score int) {
	max := func(a, b int) int {
		if a > b {
			return a
		}
		return b
	}
	switch s.GameMode {
	case ModeEasy:
		s.EasyScore = max(s.EasyScore, score)
	case ModeNoisy:
		s.NoisyScore = max(s.NoisyScore, score)
	case ModeCrazy:
		s.CrazyScore = max(s.CrazyScore, score)
	case ModeTest:
		s.TestScore = max(s.TestScore, score)
	default:
		s.EasyScore = max(s.EasyScore, score)
	}
}
