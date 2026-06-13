package models

import "time"

// User represents a user account
type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Password  string `json:"-"` // Omit password from JSON
	Role      string `json:"role"` // "admin" or "user"
	AvatarURL string `json:"avatar_url"`
}

// UserUpdateDTO represents the payload for updating account settings
type UserUpdateDTO struct {
	Username  string `json:"username"`
	Password  string `json:"password,omitempty"`
	AvatarURL string `json:"avatar_url"`
}

// CreateUserDTO is used by admins to create users
type CreateUserDTO struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	Role      string `json:"role"`
	AvatarURL string `json:"avatar_url"`
}

// Game represents a game stored in our system
type Game struct {
	ID                   int        `json:"id"` // IGDB ID
	Name                 string     `json:"name"`
	Summary              string     `json:"summary"`
	CoverURL             string     `json:"cover_url"`
	ReleaseDate          *time.Time `json:"release_date"`
	TimeToBeat           string     `json:"time_to_beat"` // e.g. "15 hours" or calculated description
	LastActiveDate       *time.Time `json:"last_active_date"`
	IsActive             bool       `json:"is_active"`
	AverageScore         float64    `json:"average_score"`
	TimeToBeatNormal     string     `json:"time_to_beat_normal"`
	TimeToBeatHastily    string     `json:"time_to_beat_hastily"`
	TimeToBeatCompletely string     `json:"time_to_beat_completely"`
}

// Review represents a user review of a game
type Review struct {
	ID         int       `json:"id"`
	GameID     int       `json:"game_id"`
	UserID     int       `json:"user_id"`
	Username   string    `json:"username"`
	AvatarURL  string    `json:"avatar_url"`
	Title      string    `json:"title"`
	Gameplay   int       `json:"gameplay"`   // 0-5
	Art        int       `json:"art"`        // 0-5
	Story      int       `json:"story"`      // 0-5
	Soundtrack int       `json:"soundtrack"` // 0-5
	Fun        int       `json:"fun"`        // 0-5
	Comment    string    `json:"comment"`
	CreatedAt  time.Time `json:"created_at"`
}

// ReviewAverages represents the average scores of reviews for a game, excluding 0s
type ReviewAverages struct {
	Gameplay   float64 `json:"gameplay"`
	Art        float64 `json:"art"`
	Story      float64 `json:"story"`
	Soundtrack float64 `json:"soundtrack"`
	Fun        float64 `json:"fun"`
	Overall    float64 `json:"overall"`
}

// GameDetailsDTO combines a Game, its average scores, and its list of reviews
type GameDetailsDTO struct {
	Game     *Game           `json:"game"`
	Averages *ReviewAverages `json:"averages"`
	Reviews  []*Review       `json:"reviews"`
}

// VotingSession represents a ranked choice voting event
type VotingSession struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	MaxNominations int       `json:"max_nominations"`
	Phase          string    `json:"phase"` // "nomination", "voting", "closed"
	CreatedAt      time.Time `json:"created_at"`
}

// Nomination represents a user nominating a game from IGDB
type Nomination struct {
	ID        int    `json:"id"`
	SessionID int    `json:"session_id"`
	UserID    int    `json:"user_id"`
	GameID    int    `json:"game_id"` // IGDB ID
	Name      string `json:"name"`
	CoverURL  string `json:"cover_url"`
	Summary   string `json:"summary"`
}

// Vote represents a user's ranked list of nominations
type Vote struct {
	ID         int    `json:"id"`
	SessionID  int    `json:"session_id"`
	UserID     int    `json:"user_id"`
	Preference string `json:"preference"` // JSON array of GameIDs: "[123, 456, 789]"
}

// CandidateResult represents a candidate's standings in voting results
type CandidateResult struct {
	Rank     int    `json:"rank"`
	GameID   int    `json:"game_id"`
	Name     string `json:"name"`
	CoverURL string `json:"cover_url"`
	Points   int    `json:"points"`
}
