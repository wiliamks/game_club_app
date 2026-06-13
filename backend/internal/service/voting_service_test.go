package service

import (
	"encoding/json"
	"errors"
	"testing"

	"gamer-club/backend/internal/models"
)

// mockVotingRepository implements repository.VotingRepository
type mockVotingRepository struct {
	session     *models.VotingSession
	nominations []*models.Nomination
	votes       []*models.Vote
}

func newMockVotingRepository() *mockVotingRepository {
	return &mockVotingRepository{
		nominations: make([]*models.Nomination, 0),
		votes:       make([]*models.Vote, 0),
	}
}

func (m *mockVotingRepository) CreateSession(vs *models.VotingSession) error {
	vs.ID = 100
	m.session = vs
	return nil
}

func (m *mockVotingRepository) GetActiveSession() (*models.VotingSession, error) {
	return m.session, nil
}

func (m *mockVotingRepository) GetAllSessions() ([]*models.VotingSession, error) {
	if m.session == nil {
		return []*models.VotingSession{}, nil
	}
	return []*models.VotingSession{m.session}, nil
}

func (m *mockVotingRepository) GetSessionByID(id int) (*models.VotingSession, error) {
	if m.session != nil && m.session.ID == id {
		return m.session, nil
	}
	return nil, nil
}

func (m *mockVotingRepository) UpdateSessionPhase(id int, phase string) error {
	if m.session != nil && m.session.ID == id {
		m.session.Phase = phase
		return nil
	}
	return errors.New("session not found")
}

func (m *mockVotingRepository) DeleteSession(id int) error {
	m.session = nil
	m.nominations = nil
	m.votes = nil
	return nil
}

func (m *mockVotingRepository) CreateNomination(n *models.Nomination) error {
	n.ID = len(m.nominations) + 1
	m.nominations = append(m.nominations, n)
	return nil
}

func (m *mockVotingRepository) GetNominationsBySessionID(sessionID int) ([]*models.Nomination, error) {
	return m.nominations, nil
}

func (m *mockVotingRepository) GetUserNominations(sessionID, userID int) ([]*models.Nomination, error) {
	var list []*models.Nomination
	for _, n := range m.nominations {
		if n.UserID == userID {
			list = append(list, n)
		}
	}
	return list, nil
}

func (m *mockVotingRepository) SaveVote(v *models.Vote) error {
	// Upsert
	for idx, existVote := range m.votes {
		if existVote.UserID == v.UserID {
			m.votes[idx] = v
			return nil
		}
	}
	v.ID = len(m.votes) + 1
	m.votes = append(m.votes, v)
	return nil
}

func (m *mockVotingRepository) GetVotesBySessionID(sessionID int) ([]*models.Vote, error) {
	return m.votes, nil
}

func (m *mockVotingRepository) GetUserVote(sessionID, userID int) (*models.Vote, error) {
	for _, v := range m.votes {
		if v.UserID == userID {
			return v, nil
		}
	}
	return nil, nil
}

func TestVotingService(t *testing.T) {
	repo := newMockVotingRepository()
	gameRepo := newMockGameRepository()
	igdb := &mockIGDBClient{games: make(map[int]*models.Game)}

	s := NewVotingService(repo, gameRepo, igdb)

	// Inject popular games for nominations
	igdb.games[10] = &models.Game{ID: 10, Name: "Elden Ring"}
	igdb.games[20] = &models.Game{ID: 20, Name: "Zelda TotK"}
	igdb.games[30] = &models.Game{ID: 30, Name: "Cyberpunk 2077"}

	// Validation tests for StartSession
	_, err := s.StartSession("", 2)
	if err == nil {
		t.Error("expected error with empty session name")
	}
	_, err = s.StartSession("Test", 0)
	if err == nil {
		t.Error("expected error with 0 max nominations")
	}

	// 1. Start Session Test
	session, err := s.StartSession("Summer Vote", 2) // Max 2 nominations per user
	if err != nil {
		t.Fatalf("failed to start voting session: %v", err)
	}
	if session.Phase != "nomination" {
		t.Errorf("expected nomination phase, got %s", session.Phase)
	}

	// 2. Nominate Game Test
	err = s.NominateGame(1, 10) // User 1 nominates Elden Ring
	if err != nil {
		t.Fatalf("nomination 1 failed: %v", err)
	}

	err = s.NominateGame(1, 20) // User 1 nominates Zelda TotK
	if err != nil {
		t.Fatalf("nomination 2 failed: %v", err)
	}

	// Hit nomination limit (Max 2)
	err = s.NominateGame(1, 30)
	if err == nil {
		t.Error("expected error when user exceeds nomination limit, got nil")
	}

	// User 2 nominates Cyberpunk 2077
	err = s.NominateGame(2, 30)
	if err != nil {
		t.Fatalf("user 2 nomination failed: %v", err)
	}

	// Try nominating a duplicate game
	err = s.NominateGame(2, 10)
	if err == nil {
		t.Error("expected error when nominating already nominated game, got nil")
	}

	// 3. Submit Votes (fails since we are still in nomination phase)
	err = s.SubmitVote(1, []int{10, 20})
	if err == nil {
		t.Error("expected error submitting vote during nomination phase, got nil")
	}

	// Transition validations
	err = s.TransitionPhase(0, "invalid_phase")
	if err == nil {
		t.Error("expected error for invalid phase")
	}

	// Transition to voting phase
	err = s.TransitionPhase(0, "voting")
	if err != nil {
		t.Fatalf("failed to transition phase: %v", err)
	}

	// Now nominate fails since we are in voting phase
	err = s.NominateGame(3, 30)
	if err == nil {
		t.Error("expected error when nominating during voting phase, got nil")
	}

	// Submit Vote validations
	err = s.SubmitVote(1, []int{})
	if err == nil {
		t.Error("expected error with empty preferences list")
	}

	err = s.SubmitVote(1, []int{999}) // non-existent candidate
	if err == nil {
		t.Error("expected error with non-existent candidate ID")
	}

	// Submit Vote Success
	// User 1 ranks: 1st Zelda (20), 2nd Elden Ring (10). User 1 votes [20, 10].
	// Zelda (20) gets 2 points, Elden Ring (10) gets 1 point (len of vote preference is 2)
	err = s.SubmitVote(1, []int{20, 10})
	if err != nil {
		t.Fatalf("failed to submit vote user 1: %v", err)
	}

	// User 2 ranks: 1st Elden Ring (10), 2nd Cyberpunk (30). User 2 votes [10, 30].
	// Elden Ring (10) gets 2 points, Cyberpunk (30) gets 1 point
	err = s.SubmitVote(2, []int{10, 30})
	if err != nil {
		t.Fatalf("failed to submit vote user 2: %v", err)
	}

	// Transition to closed to calculate results
	err = s.TransitionPhase(0, "closed")
	if err != nil {
		t.Fatalf("failed to close session: %v", err)
	}

	// 4. Calculate Results Test
	// Total points expected:
	// Elden Ring (10): 1 point (from User 1) + 2 points (from User 2) = 3 points
	// Zelda TotK (20): 2 points (from User 1) = 2 points
	// Cyberpunk 2077 (30): 1 point (from User 2) = 1 point
	results, err := s.CalculateResults(0)
	if err != nil {
		t.Fatalf("failed to calculate results: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("expected 3 candidate results, got %d", len(results))
	}

	// Check Rank order
	if results[0].GameID != 10 || results[0].Points != 3 {
		t.Errorf("expected 1st place Elden Ring (ID 10) with 3 points, got ID %d with %d points", results[0].GameID, results[0].Points)
	}
	if results[1].GameID != 20 || results[1].Points != 2 {
		t.Errorf("expected 2nd place Zelda (ID 20) with 2 points, got ID %d with %d points", results[1].GameID, results[1].Points)
	}
	if results[2].GameID != 30 || results[2].Points != 1 {
		t.Errorf("expected 3rd place Cyberpunk (ID 30) with 1 point, got ID %d with %d points", results[2].GameID, results[2].Points)
	}

	// Verify User 1 can fetch their vote
	userVote, err := s.GetUserVote(0, 1)
	if err != nil {
		t.Fatalf("failed to fetch user 1 vote: %v", err)
	}
	var pref []int
	json.Unmarshal([]byte(userVote.Preference), &pref)
	if pref[0] != 20 {
		t.Errorf("expected user 1 first choice Zelda (20), got %d", pref[0])
	}

	// 5. GetActiveSession Test
	actSess, err := s.GetActiveSession()
	if err != nil {
		t.Fatalf("failed to get active session: %v", err)
	}
	if actSess == nil || actSess.Name != "Summer Vote" {
		t.Error("expected active session to be Summer Vote")
	}

	// ListSessions Test
	sessions, err := s.ListSessions()
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}
	if len(sessions) == 0 {
		t.Error("expected list of sessions to be non-empty")
	}

	// 6. GetNominations Test
	noms, err := s.GetNominations(0)
	if err != nil {
		t.Fatalf("failed to get nominations: %v", err)
	}
	if len(noms) != 3 {
		t.Errorf("expected 3 nominations, got %d", len(noms))
	}

	// 7. GetUserNominations Test
	userNoms, err := s.GetUserNominations(0, 1)
	if err != nil {
		t.Fatalf("failed to get user nominations: %v", err)
	}
	if len(userNoms) != 2 {
		t.Errorf("expected 2 user nominations, got %d", len(userNoms))
	}

	// 8. CancelSession Test
	err = s.CancelSession(0)
	if err != nil {
		t.Fatalf("failed to cancel session: %v", err)
	}
	actSess, _ = s.GetActiveSession()
	if actSess != nil {
		t.Error("expected session to be deleted/canceled")
	}
}
