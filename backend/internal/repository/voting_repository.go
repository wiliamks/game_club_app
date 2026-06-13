package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"gamer-club/backend/internal/models"
)

// VotingRepository defines the interface for database operations related to the Voting system
type VotingRepository interface {
	CreateSession(vs *models.VotingSession) error
	GetActiveSession() (*models.VotingSession, error)
	GetSessionByID(id int) (*models.VotingSession, error)
	GetAllSessions() ([]*models.VotingSession, error)
	UpdateSessionPhase(id int, phase string) error
	DeleteSession(id int) error
	CreateNomination(n *models.Nomination) error
	GetNominationsBySessionID(sessionID int) ([]*models.Nomination, error)
	GetUserNominations(sessionID, userID int) ([]*models.Nomination, error)
	SaveVote(v *models.Vote) error
	GetVotesBySessionID(sessionID int) ([]*models.Vote, error)
	GetUserVote(sessionID, userID int) (*models.Vote, error)
}

type votingRepository struct {
	db *sql.DB
}

// NewVotingRepository creates a new SQLite implementation of VotingRepository
func NewVotingRepository(db *sql.DB) VotingRepository {
	return &votingRepository{db: db}
}

func (r *votingRepository) CreateSession(vs *models.VotingSession) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := "INSERT INTO voting_sessions (name, max_nominations, phase, created_at) VALUES (?, ?, ?, ?)"
	createdAtStr := time.Now().UTC().Format(time.RFC3339)
	res, err := tx.Exec(query, vs.Name, vs.MaxNominations, vs.Phase, createdAtStr)
	if err != nil {
		return fmt.Errorf("failed to insert voting session: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	vs.ID = int(id)
	vs.CreatedAt = time.Now()

	return tx.Commit()
}

func (r *votingRepository) GetActiveSession() (*models.VotingSession, error) {
	// Active session is the one that is NOT closed, or the latest one if we want results.
	// Let's return the latest session overall.
	query := "SELECT id, name, max_nominations, phase, created_at FROM voting_sessions ORDER BY id DESC LIMIT 1"
	row := r.db.QueryRow(query)

	vs := &models.VotingSession{}
	var createdAtStr string
	err := row.Scan(&vs.ID, &vs.Name, &vs.MaxNominations, &vs.Phase, &createdAtStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan voting session: %w", err)
	}

	t, err := time.Parse(time.RFC3339, createdAtStr)
	if err == nil {
		vs.CreatedAt = t
	}

	return vs, nil
}

func (r *votingRepository) GetSessionByID(id int) (*models.VotingSession, error) {
	query := "SELECT id, name, max_nominations, phase, created_at FROM voting_sessions WHERE id = ?"
	row := r.db.QueryRow(query, id)

	vs := &models.VotingSession{}
	var createdAtStr string
	err := row.Scan(&vs.ID, &vs.Name, &vs.MaxNominations, &vs.Phase, &createdAtStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan session by ID: %w", err)
	}

	t, err := time.Parse(time.RFC3339, createdAtStr)
	if err == nil {
		vs.CreatedAt = t
	}

	return vs, nil
}

func (r *votingRepository) UpdateSessionPhase(id int, phase string) error {
	query := "UPDATE voting_sessions SET phase = ? WHERE id = ?"
	_, err := r.db.Exec(query, phase, id)
	if err != nil {
		return fmt.Errorf("failed to update session phase: %w", err)
	}
	return nil
}

func (r *votingRepository) DeleteSession(id int) error {
	query := "DELETE FROM voting_sessions WHERE id = ?"
	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

func (r *votingRepository) CreateNomination(n *models.Nomination) error {
	query := "INSERT INTO nominations (session_id, user_id, game_id, name, cover_url, summary) VALUES (?, ?, ?, ?, ?, ?)"
	res, err := r.db.Exec(query, n.SessionID, n.UserID, n.GameID, n.Name, n.CoverURL, n.Summary)
	if err != nil {
		return fmt.Errorf("failed to nominate game: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	n.ID = int(id)
	return nil
}

func (r *votingRepository) GetNominationsBySessionID(sessionID int) ([]*models.Nomination, error) {
	query := "SELECT id, session_id, user_id, game_id, name, cover_url, summary FROM nominations WHERE session_id = ?"
	rows, err := r.db.Query(query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query nominations: %w", err)
	}
	defer rows.Close()

	var nominations []*models.Nomination
	for rows.Next() {
		n := &models.Nomination{}
		err := rows.Scan(&n.ID, &n.SessionID, &n.UserID, &n.GameID, &n.Name, &n.CoverURL, &n.Summary)
		if err != nil {
			return nil, fmt.Errorf("failed to scan nomination: %w", err)
		}
		nominations = append(nominations, n)
	}
	return nominations, nil
}

func (r *votingRepository) GetUserNominations(sessionID, userID int) ([]*models.Nomination, error) {
	query := "SELECT id, session_id, user_id, game_id, name, cover_url, summary FROM nominations WHERE session_id = ? AND user_id = ?"
	rows, err := r.db.Query(query, sessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user nominations: %w", err)
	}
	defer rows.Close()

	var nominations []*models.Nomination
	for rows.Next() {
		n := &models.Nomination{}
		err := rows.Scan(&n.ID, &n.SessionID, &n.UserID, &n.GameID, &n.Name, &n.CoverURL, &n.Summary)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user nomination: %w", err)
		}
		nominations = append(nominations, n)
	}
	return nominations, nil
}

func (r *votingRepository) SaveVote(v *models.Vote) error {
	query := `INSERT INTO votes (session_id, user_id, preference)
		VALUES (?, ?, ?)
		ON CONFLICT(session_id, user_id) DO UPDATE SET
			preference = excluded.preference;`

	_, err := r.db.Exec(query, v.SessionID, v.UserID, v.Preference)
	if err != nil {
		return fmt.Errorf("failed to save vote: %w", err)
	}
	return nil
}

func (r *votingRepository) GetVotesBySessionID(sessionID int) ([]*models.Vote, error) {
	query := "SELECT id, session_id, user_id, preference FROM votes WHERE session_id = ?"
	rows, err := r.db.Query(query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query votes: %w", err)
	}
	defer rows.Close()

	var votes []*models.Vote
	for rows.Next() {
		v := &models.Vote{}
		err := rows.Scan(&v.ID, &v.SessionID, &v.UserID, &v.Preference)
		if err != nil {
			return nil, fmt.Errorf("failed to scan vote: %w", err)
		}
		votes = append(votes, v)
	}
	return votes, nil
}

func (r *votingRepository) GetUserVote(sessionID, userID int) (*models.Vote, error) {
	query := "SELECT id, session_id, user_id, preference FROM votes WHERE session_id = ? AND user_id = ?"
	row := r.db.QueryRow(query, sessionID, userID)

	v := &models.Vote{}
	err := row.Scan(&v.ID, &v.SessionID, &v.UserID, &v.Preference)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan user vote: %w", err)
	}
	return v, nil
}

func (r *votingRepository) GetAllSessions() ([]*models.VotingSession, error) {
	query := "SELECT id, name, max_nominations, phase, created_at FROM voting_sessions ORDER BY id DESC"
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all voting sessions: %w", err)
	}
	defer rows.Close()

	var list []*models.VotingSession
	for rows.Next() {
		vs := &models.VotingSession{}
		var createdAtStr string
		err := rows.Scan(&vs.ID, &vs.Name, &vs.MaxNominations, &vs.Phase, &createdAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to scan voting session: %w", err)
		}
		t, err := time.Parse(time.RFC3339, createdAtStr)
		if err == nil {
			vs.CreatedAt = t
		}
		list = append(list, vs)
	}
	return list, nil
}
