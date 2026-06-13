package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"gamer-club/backend/internal/models"
)

// GameRepository defines the interface for games and reviews database operations
type GameRepository interface {
	SaveGame(g *models.Game) error
	GetGameByID(id int) (*models.Game, error)
	GetAllGames() ([]*models.Game, error)
	GetActiveGame() (*models.Game, error)
	SetActiveGame(id int) error
	DeactivateActiveGame() error
	SaveReview(r *models.Review) error
	GetReviewsByGameID(gameID int) ([]*models.Review, error)
	GetReviewByUserAndGame(userID, gameID int) (*models.Review, error)
	DeleteReview(userID, gameID int) error
	DeleteGame(id int) error
}

type gameRepository struct {
	db *sql.DB
}

// NewGameRepository creates a new SQLite implementation of GameRepository
func NewGameRepository(db *sql.DB) GameRepository {
	return &gameRepository{db: db}
}

func (r *gameRepository) SaveGame(g *models.Game) error {
	query := `INSERT INTO games (id, name, summary, cover_url, release_date, time_to_beat, last_active_date, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			summary = excluded.summary,
			cover_url = excluded.cover_url,
			release_date = excluded.release_date,
			time_to_beat = excluded.time_to_beat;`

	var relDate interface{}
	if g.ReleaseDate != nil {
		relDate = g.ReleaseDate.Format(time.RFC3339)
	}

	var lastActive interface{}
	if g.LastActiveDate != nil {
		lastActive = g.LastActiveDate.Format(time.RFC3339)
	}

	_, err := r.db.Exec(query, g.ID, g.Name, g.Summary, g.CoverURL, relDate, g.TimeToBeat, lastActive, g.IsActive)
	if err != nil {
		return fmt.Errorf("failed to save game: %w", err)
	}
	return nil
}

func (r *gameRepository) GetGameByID(id int) (*models.Game, error) {
	query := "SELECT id, name, summary, cover_url, release_date, time_to_beat, last_active_date, is_active FROM games WHERE id = ?"
	row := r.db.QueryRow(query, id)

	g := &models.Game{}
	var relDateStr, lastActiveStr sql.NullString
	err := row.Scan(&g.ID, &g.Name, &g.Summary, &g.CoverURL, &relDateStr, &g.TimeToBeat, &lastActiveStr, &g.IsActive)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan game: %w", err)
	}

	if relDateStr.Valid && relDateStr.String != "" {
		t, err := time.Parse(time.RFC3339, relDateStr.String)
		if err == nil {
			g.ReleaseDate = &t
		}
	}

	if lastActiveStr.Valid && lastActiveStr.String != "" {
		t, err := time.Parse(time.RFC3339, lastActiveStr.String)
		if err == nil {
			g.LastActiveDate = &t
		}
	}

	return g, nil
}

func (r *gameRepository) GetAllGames() ([]*models.Game, error) {
	query := "SELECT id, name, summary, cover_url, release_date, time_to_beat, last_active_date, is_active FROM games"
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query games: %w", err)
	}
	defer rows.Close()

	var games []*models.Game
	for rows.Next() {
		g := &models.Game{}
		var relDateStr, lastActiveStr sql.NullString
		err := rows.Scan(&g.ID, &g.Name, &g.Summary, &g.CoverURL, &relDateStr, &g.TimeToBeat, &lastActiveStr, &g.IsActive)
		if err != nil {
			return nil, fmt.Errorf("failed to scan game row: %w", err)
		}

		if relDateStr.Valid && relDateStr.String != "" {
			t, err := time.Parse(time.RFC3339, relDateStr.String)
			if err == nil {
				g.ReleaseDate = &t
			}
		}

		if lastActiveStr.Valid && lastActiveStr.String != "" {
			t, err := time.Parse(time.RFC3339, lastActiveStr.String)
			if err == nil {
				g.LastActiveDate = &t
			}
		}

		games = append(games, g)
	}
	return games, nil
}

func (r *gameRepository) GetActiveGame() (*models.Game, error) {
	query := "SELECT id, name, summary, cover_url, release_date, time_to_beat, last_active_date, is_active FROM games WHERE is_active = 1 LIMIT 1"
	row := r.db.QueryRow(query)

	g := &models.Game{}
	var relDateStr, lastActiveStr sql.NullString
	err := row.Scan(&g.ID, &g.Name, &g.Summary, &g.CoverURL, &relDateStr, &g.TimeToBeat, &lastActiveStr, &g.IsActive)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan active game: %w", err)
	}

	if relDateStr.Valid && relDateStr.String != "" {
		t, err := time.Parse(time.RFC3339, relDateStr.String)
		if err == nil {
			g.ReleaseDate = &t
		}
	}

	if lastActiveStr.Valid && lastActiveStr.String != "" {
		t, err := time.Parse(time.RFC3339, lastActiveStr.String)
		if err == nil {
			g.LastActiveDate = &t
		}
	}

	return g, nil
}

func (r *gameRepository) SetActiveGame(id int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Deactivate all games
	_, err = tx.Exec("UPDATE games SET is_active = 0")
	if err != nil {
		return fmt.Errorf("failed to deactivate games in transaction: %w", err)
	}

	// Set game as active and update its last_active_date
	nowStr := time.Now().UTC().Format(time.RFC3339)
	_, err = tx.Exec("UPDATE games SET is_active = 1, last_active_date = ? WHERE id = ?", nowStr, id)
	if err != nil {
		return fmt.Errorf("failed to activate game: %w", err)
	}

	return tx.Commit()
}

func (r *gameRepository) DeactivateActiveGame() error {
	_, err := r.db.Exec("UPDATE games SET is_active = 0")
	if err != nil {
		return fmt.Errorf("failed to deactivate active game: %w", err)
	}
	return nil
}

func (r *gameRepository) SaveReview(review *models.Review) error {
	// Defense-in-depth: explicitly clear any existing review from the same user to ensure zero duplicate records
	_, _ = r.db.Exec("DELETE FROM reviews WHERE game_id = ? AND user_id = ?", review.GameID, review.UserID)

	query := `INSERT INTO reviews (game_id, user_id, username, title, avatar_url, gameplay, art, story, soundtrack, fun, comment, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(game_id, user_id) DO UPDATE SET
			username = excluded.username,
			title = excluded.title,
			avatar_url = excluded.avatar_url,
			gameplay = excluded.gameplay,
			art = excluded.art,
			story = excluded.story,
			soundtrack = excluded.soundtrack,
			fun = excluded.fun,
			comment = excluded.comment,
			created_at = excluded.created_at;`

	createdAtStr := time.Now().UTC().Format(time.RFC3339)

	_, err := r.db.Exec(query, review.GameID, review.UserID, review.Username, review.Title, review.AvatarURL,
		review.Gameplay, review.Art, review.Story, review.Soundtrack, review.Fun,
		review.Comment, createdAtStr)

	if err != nil {
		return fmt.Errorf("failed to save review: %w", err)
	}
	return nil
}

func (r *gameRepository) GetReviewsByGameID(gameID int) ([]*models.Review, error) {
	query := "SELECT id, game_id, user_id, username, title, avatar_url, gameplay, art, story, soundtrack, fun, comment, created_at FROM reviews WHERE game_id = ? ORDER BY created_at DESC"
	rows, err := r.db.Query(query, gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to query reviews: %w", err)
	}
	defer rows.Close()

	var reviews []*models.Review
	for rows.Next() {
		re := &models.Review{}
		var createdAtStr string
		err := rows.Scan(&re.ID, &re.GameID, &re.UserID, &re.Username, &re.Title, &re.AvatarURL, &re.Gameplay, &re.Art, &re.Story, &re.Soundtrack, &re.Fun, &re.Comment, &createdAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to scan review: %w", err)
		}

		t, err := time.Parse(time.RFC3339, createdAtStr)
		if err == nil {
			re.CreatedAt = t
		} else {
			// Fallback parse
			t, err = time.Parse("2006-01-02 15:04:05", createdAtStr)
			if err == nil {
				re.CreatedAt = t
			}
		}

		reviews = append(reviews, re)
	}
	return reviews, nil
}

func (r *gameRepository) GetReviewByUserAndGame(userID, gameID int) (*models.Review, error) {
	query := "SELECT id, game_id, user_id, username, title, avatar_url, gameplay, art, story, soundtrack, fun, comment, created_at FROM reviews WHERE user_id = ? AND game_id = ?"
	row := r.db.QueryRow(query, userID, gameID)

	re := &models.Review{}
	var createdAtStr string
	err := row.Scan(&re.ID, &re.GameID, &re.UserID, &re.Username, &re.Title, &re.AvatarURL, &re.Gameplay, &re.Art, &re.Story, &re.Soundtrack, &re.Fun, &re.Comment, &createdAtStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan user review: %w", err)
	}

	t, err := time.Parse(time.RFC3339, createdAtStr)
	if err == nil {
		re.CreatedAt = t
	}

	return re, nil
}

func (r *gameRepository) DeleteReview(userID, gameID int) error {
	query := "DELETE FROM reviews WHERE user_id = ? AND game_id = ?"
	_, err := r.db.Exec(query, userID, gameID)
	if err != nil {
		return fmt.Errorf("failed to delete review: %w", err)
	}
	return nil
}

func (r *gameRepository) DeleteGame(id int) error {
	query := "DELETE FROM games WHERE id = ?"
	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete game: %w", err)
	}
	return nil
}
