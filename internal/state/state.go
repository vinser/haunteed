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
	"sort"
	"time"

	"github.com/denisbrodbeck/machineid"
	"github.com/vinser/haunteed/internal/geoip"
)

// HighScore holds a single high score entry.

type HighScore struct {
	Nick  string `json:"nick"`
	Score int    `json:"score"`
}

// State holds persistent game data such as high scores.

type State struct {
	Version      string             `json:"version"`       // Version of the app when the state was last saved
	GameMode     string             `json:"game_mode"`     // Current game mode: easy, noisy or crazy
	NightOption  string             `json:"crazy_night"`   // Night option for crazy mode: never, always or real
	SpriteSize   string             `json:"sprite_size"`   // Sprite size: small, medium, large
	Mute         bool               `json:"mute"`          // Mute all sounds
	FloorSeeds   map[int]int64      `json:"floor_seeds"`   // Seed for each floor to reproduce the same sequence of mazes
	EasyScores   []HighScore        `json:"easy_scores"`   // Easy mode high score
	NoisyScores  []HighScore        `json:"noisy_scores"`  // Noisy mode high score
	CrazyScores  []HighScore        `json:"crazy_scores"`  // Crazy mode high score
	LocationInfo geoip.LocationInfo `json:"location_info"` // Location information
}

const (
	// Game modes
	ModeEasy    = "easy"
	ModeNoisy   = "noisy"
	ModeCrazy   = "crazy"
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

	maxHighScores = 5
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

// SetMute toggles the mute state.
func (s *State) SetMute(mute bool) {
	s.Mute = mute
}

// UpdateAndSave updates the state with new game results and persists it to a file.
func (s *State) UpdateAndSave(floor int, score int, seed int64, nick string) error {
	switch s.GameMode {
	case ModeEasy:
		s.EasyScores = updateHighScores(s.EasyScores, score, nick)
	case ModeNoisy:
		s.NoisyScores = updateHighScores(s.NoisyScores, score, nick)
	case ModeCrazy:
		s.CrazyScores = updateHighScores(s.CrazyScores, score, nick)
	}
	// Ensure the seed for the current floor is saved if it's new.
	s.FloorSeeds[floor] = seed
	return s.Save()
}

func updateHighScores(scores []HighScore, newScore int, newNick string) []HighScore {
	if newNick == "" {
		newNick = "nowhere man (aka rootless)"
	}
	// Add the new score
	scores = append(scores, HighScore{Nick: newNick, Score: newScore})

	// Sort scores in descending order
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})

	// Keep only the top N scores
	if len(scores) > maxHighScores {
		scores = scores[:maxHighScores]
	}

	return scores
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

func New(appVersion string) *State {
	seeds := make(map[int]int64)
	seeds[0] = time.Now().UnixNano()
	loc, err := geoip.GetLocationInfo()
	if err != nil {
		loc = fallbackLocation
	}
	s := &State{
		Version:      appVersion,
		GameMode:     ModeDefault,
		NightOption:  NightDefault,
		SpriteSize:   SpriteDefault,
		FloorSeeds:   seeds,
		LocationInfo: *loc,
	}
	return s
}

// Load reads the state from disk, decrypts and verifies it.
func Load(appVersion string) *State {
	s := &State{}

	path, err := getSavePath()
	if err != nil {
		return New(appVersion)
	}

	encrypted, err := os.ReadFile(path)
	if err != nil {
		return New(appVersion)
	}

	decrypted, err := decrypt(encrypted)
	if err != nil || len(decrypted) < 5 {
		return New(appVersion)
	}

	crcStored := binary.LittleEndian.Uint32(decrypted[:4])
	payload := decrypted[4:]
	if crc32.ChecksumIEEE(payload) != crcStored {
		return New(appVersion)
	}

	// Unmarshal into the current state struct
	if err = json.Unmarshal(payload, s); err != nil {
		return New(appVersion) // Corrupted JSON
	}

	return s
}

func Reset() error {
	path, err := getSavePath()
	if err != nil {
		return err
	}
	// os.Remove returns an error if the file does not exist, which is fine.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(path)
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

func (s *State) GetHighScores() []HighScore {
	var scores []HighScore
	switch s.GameMode {
	case ModeEasy:
		scores = s.EasyScores
	case ModeNoisy:
		scores = s.NoisyScores
	case ModeCrazy:
		scores = s.CrazyScores
	default:
		return scores
	}
	return scores
}

func (s *State) SetHighScore(score int, nick string) {
	max := func(a, b int) int {
		if a > b {
			return a
		}
		return b
	}
	switch s.GameMode {
	case ModeEasy:
		s.EasyScores[0].Score = max(s.EasyScores[0].Score, score)
		s.EasyScores[0].Nick = nick
	case ModeNoisy:
		s.NoisyScores[0].Score = max(s.NoisyScores[0].Score, score)
		s.NoisyScores[0].Nick = nick
	case ModeCrazy:
		s.CrazyScores[0].Score = max(s.CrazyScores[0].Score, score)
		s.CrazyScores[0].Nick = nick
	}
}
