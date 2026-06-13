package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"time"

	"gamer-club/backend/internal/models"
	"gamer-club/backend/internal/repository"
)

// VotingService defines the interface for ranked choice voting business logic
type VotingService interface {
	StartSession(name string, maxNominations int) (*models.VotingSession, error)
	GetActiveSession() (*models.VotingSession, error)
	ListSessions() ([]*models.VotingSession, error)
	GetNominations(sessionID int) ([]*models.Nomination, error)
	GetUserNominations(sessionID, userID int) ([]*models.Nomination, error)
	NominateGame(userID int, gameID int) error
	SubmitVote(userID int, preference []int) error
	GetUserVote(sessionID, userID int) (*models.Vote, error)
	TransitionPhase(sessionID int, phase string) error
	CancelSession(sessionID int) error
	CalculateResults(sessionID int) ([]*models.CandidateResult, error)
}

type votingService struct {
	repo       repository.VotingRepository
	gameRepo   repository.GameRepository
	igdbClient IGDBClient
}

// NewVotingService creates a new VotingService implementation
func NewVotingService(repo repository.VotingRepository, gameRepo repository.GameRepository, igdbClient IGDBClient) VotingService {
	return &votingService{
		repo:       repo,
		gameRepo:   gameRepo,
		igdbClient: igdbClient,
	}
}

func (s *votingService) StartSession(name string, maxNominations int) (*models.VotingSession, error) {
	if name == "" {
		return nil, errors.New("session name cannot be empty")
	}
	if maxNominations <= 0 {
		return nil, errors.New("max nominations must be greater than 0")
	}

	vs := &models.VotingSession{
		Name:           name,
		MaxNominations: maxNominations,
		Phase:          "nomination",
	}

	if err := s.repo.CreateSession(vs); err != nil {
		return nil, err
	}

	return vs, nil
}

func (s *votingService) GetActiveSession() (*models.VotingSession, error) {
	return s.repo.GetActiveSession()
}

func (s *votingService) ListSessions() ([]*models.VotingSession, error) {
	return s.repo.GetAllSessions()
}

func (s *votingService) GetNominations(sessionID int) ([]*models.Nomination, error) {
	var id int
	if sessionID > 0 {
		id = sessionID
	} else {
		session, err := s.repo.GetActiveSession()
		if err != nil {
			return nil, err
		}
		if session == nil {
			return []*models.Nomination{}, nil
		}
		id = session.ID
	}
	return s.repo.GetNominationsBySessionID(id)
}

func (s *votingService) GetUserNominations(sessionID, userID int) ([]*models.Nomination, error) {
	var id int
	if sessionID > 0 {
		id = sessionID
	} else {
		session, err := s.repo.GetActiveSession()
		if err != nil {
			return nil, err
		}
		if session == nil {
			return []*models.Nomination{}, nil
		}
		id = session.ID
	}
	return s.repo.GetUserNominations(id, userID)
}

func (s *votingService) NominateGame(userID int, gameID int) error {
	session, err := s.repo.GetActiveSession()
	if err != nil {
		return err
	}
	if session == nil {
		return errors.New("no active voting event is currently running")
	}
	if session.Phase != "nomination" {
		return errors.New("cannot nominate games outside of the Nomination Phase")
	}

	// Check if user has reached max nominations limit
	userNominations, err := s.repo.GetUserNominations(session.ID, userID)
	if err != nil {
		return err
	}
	if len(userNominations) >= session.MaxNominations {
		return fmt.Errorf("nomination limit reached: you can nominate at most %d games", session.MaxNominations)
	}

	// Check if this game is already nominated in this session (avoid duplicate entries)
	allNominations, err := s.repo.GetNominationsBySessionID(session.ID)
	if err != nil {
		return err
	}
	for _, nom := range allNominations {
		if nom.GameID == gameID {
			return errors.New("this game has already been nominated for this session")
		}
	}

	// Fetch game details from IGDB (offline mockup will trigger if unconfigured)
	game, err := s.igdbClient.GetGameDetails(gameID)
	if err != nil {
		return fmt.Errorf("failed to fetch game details from IGDB: %w", err)
	}

	nom := &models.Nomination{
		SessionID: session.ID,
		UserID:    userID,
		GameID:    gameID,
		Name:      game.Name,
		CoverURL:  game.CoverURL,
		Summary:   game.Summary,
	}

	return s.repo.CreateNomination(nom)
}

func (s *votingService) SubmitVote(userID int, preference []int) error {
	session, err := s.repo.GetActiveSession()
	if err != nil {
		return err
	}
	if session == nil {
		return errors.New("no active voting session")
	}
	if session.Phase != "voting" {
		return errors.New("cannot submit votes: voting is not currently open")
	}

	if len(preference) == 0 {
		return errors.New("preferential list cannot be empty")
	}

	// Validate that all preference IDs correspond to actual nominations in this session
	nominations, err := s.repo.GetNominationsBySessionID(session.ID)
	if err != nil {
		return err
	}

	nomMap := make(map[int]bool)
	for _, n := range nominations {
		nomMap[n.GameID] = true
	}

	seen := make(map[int]bool)
	for _, id := range preference {
		if !nomMap[id] {
			return fmt.Errorf("game ID %d is not a nominated candidate in this session", id)
		}
		if seen[id] {
			return errors.New("duplicate game IDs found in preferences")
		}
		seen[id] = true
	}

	prefJSON, err := json.Marshal(preference)
	if err != nil {
		return fmt.Errorf("failed to marshal preferences: %w", err)
	}

	vote := &models.Vote{
		SessionID:  session.ID,
		UserID:     userID,
		Preference: string(prefJSON),
	}

	return s.repo.SaveVote(vote)
}

func (s *votingService) GetUserVote(sessionID, userID int) (*models.Vote, error) {
	var id int
	if sessionID > 0 {
		id = sessionID
	} else {
		session, err := s.repo.GetActiveSession()
		if err != nil {
			return nil, err
		}
		if session == nil {
			return nil, nil
		}
		id = session.ID
	}
	return s.repo.GetUserVote(id, userID)
}

func (s *votingService) TransitionPhase(sessionID int, phase string) error {
	if phase != "nomination" && phase != "voting" && phase != "closed" {
		return errors.New("invalid phase: must be 'nomination', 'voting', or 'closed'")
	}

	var id int
	if sessionID > 0 {
		id = sessionID
	} else {
		session, err := s.repo.GetActiveSession()
		if err != nil {
			return err
		}
		if session == nil {
			return errors.New("no active voting session found to transition")
		}
		id = session.ID
	}

	err := s.repo.UpdateSessionPhase(id, phase)
	if err != nil {
		return err
	}

	// Copy nominated games to games table when closing a voting session
	if phase == "closed" {
		noms, err := s.repo.GetNominationsBySessionID(id)
		if err == nil {
			for _, n := range noms {
				gExists, err := s.gameRepo.GetGameByID(n.GameID)
				if err == nil && gExists == nil {
					// Fetch FULL details from IGDB (including real ReleaseDate and real TimeToBeat!)
					realGame, err := s.igdbClient.GetGameDetails(n.GameID)
					if err == nil && realGame != nil {
						_ = s.gameRepo.SaveGame(realGame)
					} else {
						// Fallback if offline/unconfigured
						newGame := &models.Game{
							ID:          n.GameID,
							Name:        n.Name,
							Summary:     n.Summary,
							CoverURL:    n.CoverURL,
							TimeToBeat:  calculateTimeToBeat(n.GameID),
							IsActive:    false,
						}
						_ = s.gameRepo.SaveGame(newGame)
					}
				}
			}
		}
	}

	return nil
}

func (s *votingService) CancelSession(sessionID int) error {
	var id int
	if sessionID > 0 {
		id = sessionID
	} else {
		session, err := s.repo.GetActiveSession()
		if err != nil {
			return err
		}
		if session == nil {
			return nil
		}
		id = session.ID
	}

	return s.repo.DeleteSession(id)
}

func (s *votingService) CalculateResults(sessionID int) ([]*models.CandidateResult, error) {
	var id int
	if sessionID > 0 {
		id = sessionID
	} else {
		session, err := s.repo.GetActiveSession()
		if err != nil {
			return nil, err
		}
		if session == nil {
			return []*models.CandidateResult{}, nil
		}
		id = session.ID
	}

	// 1. Load nominations to form the candidate pool
	noms, err := s.repo.GetNominationsBySessionID(id)
	if err != nil {
		return nil, err
	}

	if len(noms) == 0 {
		return []*models.CandidateResult{}, nil
	}

	nomMap := make(map[int]*models.Nomination)
	pointsMap := make(map[int]int)
	for _, n := range noms {
		nomMap[n.GameID] = n
		pointsMap[n.GameID] = 0 // Initialize with 0 points
	}

	// 2. Load votes and calculate Borda Count points
	votes, err := s.repo.GetVotesBySessionID(id)
	if err != nil {
		return nil, err
	}

	for _, v := range votes {
		var pref []int
		if err := json.Unmarshal([]byte(v.Preference), &pref); err != nil {
			continue // Skip malformed votes
		}

		// Standard Borda count based on rank index
		// First choice (index 0) gets most points: len(pref) points.
		// Last choice gets 1 point.
		// If a user has ranked fewer games, it degrades correctly.
		prefLen := len(pref)
		for i, gameID := range pref {
			if _, exists := pointsMap[gameID]; exists {
				pointsMap[gameID] += (prefLen - i)
			}
		}
	}

	// 3. Compile candidates
	var results []*models.CandidateResult
	for gameID, pts := range pointsMap {
		nom := nomMap[gameID]
		results = append(results, &models.CandidateResult{
			GameID:   gameID,
			Name:     nom.Name,
			CoverURL: nom.CoverURL,
			Points:   pts,
		})
	}

	// 4. Resolve tie-breakers non-deterministically (preserves transitivity by pre-assigning random values)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	tieBreaker := make(map[int]float64)
	for _, r := range results {
		tieBreaker[r.GameID] = rng.Float64()
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Points != results[j].Points {
			return results[i].Points > results[j].Points // Points descending
		}
		// Tie-breaker
		return tieBreaker[results[i].GameID] > tieBreaker[results[j].GameID]
	})

	// 5. Assign Rank numbers
	for i, r := range results {
		r.Rank = i + 1
	}

	return results, nil
}
