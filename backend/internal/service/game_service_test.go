package service

import (
	"errors"
	"testing"

	"gamer-club/backend/internal/models"
)

// mockGameRepository implements repository.GameRepository
type mockGameRepository struct {
	games      map[int]*models.Game
	reviews    map[int][]*models.Review
	activeID   int
	saveCalled bool
}

func newMockGameRepository() *mockGameRepository {
	return &mockGameRepository{
		games:   make(map[int]*models.Game),
		reviews: make(map[int][]*models.Review),
	}
}

func (m *mockGameRepository) SaveGame(g *models.Game) error {
	m.saveCalled = true
	m.games[g.ID] = g
	return nil
}

func (m *mockGameRepository) GetGameByID(id int) (*models.Game, error) {
	g, ok := m.games[id]
	if !ok {
		return nil, nil
	}
	return g, nil
}

func (m *mockGameRepository) GetAllGames() ([]*models.Game, error) {
	var list []*models.Game
	for _, g := range m.games {
		list = append(list, g)
	}
	return list, nil
}

func (m *mockGameRepository) GetActiveGame() (*models.Game, error) {
	if m.activeID == 0 {
		return nil, nil
	}
	return m.games[m.activeID], nil
}

func (m *mockGameRepository) SetActiveGame(id int) error {
	m.activeID = id
	for _, g := range m.games {
		g.IsActive = (g.ID == id)
	}
	return nil
}

func (m *mockGameRepository) DeactivateActiveGame() error {
	m.activeID = 0
	for _, g := range m.games {
		g.IsActive = false
	}
	return nil
}

func (m *mockGameRepository) SaveReview(r *models.Review) error {
	m.reviews[r.GameID] = append(m.reviews[r.GameID], r)
	return nil
}

func (m *mockGameRepository) GetReviewsByGameID(gameID int) ([]*models.Review, error) {
	return m.reviews[gameID], nil
}

func (m *mockGameRepository) GetReviewByUserAndGame(userID, gameID int) (*models.Review, error) {
	for _, r := range m.reviews[gameID] {
		if r.UserID == userID {
			return r, nil
		}
	}
	return nil, nil
}

func (m *mockGameRepository) DeleteReview(userID, gameID int) error {
	var newList []*models.Review
	for _, r := range m.reviews[gameID] {
		if r.UserID != userID {
			newList = append(newList, r)
		}
	}
	m.reviews[gameID] = newList
	return nil
}

func (m *mockGameRepository) DeleteGame(id int) error {
	delete(m.games, id)
	delete(m.reviews, id)
	return nil
}

func (m *mockGameRepository) ToggleReaction(reviewID, userID int, emoji string) error {
	return nil
}

func (m *mockGameRepository) GetReactionsForGame(gameID, userID int) (map[int][]*models.EmojiReactionSummary, error) {
	return make(map[int][]*models.EmojiReactionSummary), nil
}

// mockIGDBClient implements service.IGDBClient
type mockIGDBClient struct {
	games map[int]*models.Game
}

func (m *mockIGDBClient) SearchGames(query string) ([]*models.Game, error) {
	return nil, nil
}

func (m *mockIGDBClient) GetGameDetails(id int) (*models.Game, error) {
	g, ok := m.games[id]
	if !ok {
		return nil, errors.New("game not found on IGDB")
	}
	return g, nil
}

func TestGameService(t *testing.T) {
	repo := newMockGameRepository()
	igdb := &mockIGDBClient{games: make(map[int]*models.Game)}

	s := NewGameService(repo, igdb)

	// Inject a game on mock IGDB to test automatic import/cache
	igdbGame := &models.Game{
		ID:         777,
		Name:       "Chrono Cross",
		Summary:    "Parallel universes RPG",
		TimeToBeat: "35 hours",
	}
	igdb.games[777] = igdbGame

	// Test GetGameDetails for un-cached game (triggers IGDB fetch + save)
	details, err := s.GetGameDetails(777, 10)
	if err != nil {
		t.Fatalf("failed to get game details: %v", err)
	}
	if details.Game.Name != "Chrono Cross" {
		t.Errorf("expected game name Chrono Cross, got %s", details.Game.Name)
	}
	if !repo.saveCalled {
		t.Error("expected repository SaveGame to be called to cache IGDB details")
	}

	// Submit Reviews to verify critical Rule (0 in any category must be excluded from calculations)
	review1 := &models.Review{
		Gameplay:   5,
		Art:        0, // unrated/excluded
		Story:      4,
		Soundtrack: 5,
		Fun:        5,
		Comment:    "Amazing soundtrack",
	}
	review2 := &models.Review{
		Gameplay:   3,
		Art:        4,
		Story:      0, // unrated/excluded
		Soundtrack: 3,
		Fun:        4,
		Comment:    "Good game",
	}

	err = s.SubmitReview(10, "cloud", 777, review1)
	if err != nil {
		t.Fatalf("failed to submit review 1: %v", err)
	}
	err = s.SubmitReview(11, "tifa", 777, review2)
	if err != nil {
		t.Fatalf("failed to submit review 2: %v", err)
	}

	// Fetch game details again (with reviews now in local DB)
	details, err = s.GetGameDetails(777, 10)
	if err != nil {
		t.Fatalf("failed to get game details: %v", err)
	}

	if len(details.Reviews) != 2 {
		t.Errorf("expected 2 reviews, got %d", len(details.Reviews))
	}

	// Verify rating average calculations:
	// Gameplay: review1=5, review2=3 -> Avg = (5+3)/2 = 4.0
	// Art: review1=0 (excluded), review2=4 -> Avg = 4 / 1 = 4.0
	// Story: review1=4, review2=0 (excluded) -> Avg = 4 / 1 = 4.0
	// Soundtrack: review1=5, review2=3 -> Avg = (5+3)/2 = 4.0
	// Fun: review1=5, review2=4 -> Avg = (5+4)/2 = 4.5
	// Overall: average of the 5 non-zero categories averages -> (4.0 + 4.0 + 4.0 + 4.0 + 4.5) / 5 = 4.1

	if details.Averages.Gameplay != 4.0 {
		t.Errorf("expected gameplay avg 4.0, got %f", details.Averages.Gameplay)
	}
	if details.Averages.Art != 4.0 {
		t.Errorf("expected art avg 4.0, got %f", details.Averages.Art)
	}
	if details.Averages.Story != 4.0 {
		t.Errorf("expected story avg 4.0, got %f", details.Averages.Story)
	}
	if details.Averages.Soundtrack != 4.0 {
		t.Errorf("expected soundtrack avg 4.0, got %f", details.Averages.Soundtrack)
	}
	if details.Averages.Fun != 4.5 {
		t.Errorf("expected fun avg 4.5, got %f", details.Averages.Fun)
	}
	if details.Averages.Overall != 4.1 {
		t.Errorf("expected overall avg 4.1, got %f", details.Averages.Overall)
	}

	// Active Game Controller Tests
	err = s.SetActiveGame(777)
	if err != nil {
		t.Fatalf("failed to set active game: %v", err)
	}

	active, _ := repo.GetActiveGame()
	if active == nil || !active.IsActive {
		t.Error("expected game 777 to be active")
	}

	err = s.DeactivateActiveGame()
	if err != nil {
		t.Fatalf("failed to deactivate: %v", err)
	}

	active, _ = repo.GetActiveGame()
	if active != nil {
		t.Error("expected no active game after deactivation")
	}

	// ListGames Test
	allGames, err := s.ListGames()
	if err != nil {
		t.Fatalf("failed to list games: %v", err)
	}
	if len(allGames) == 0 {
		t.Error("expected list of games to be non-empty")
	}

	// DeleteReview Test
	err = s.DeleteReview(10, 777)
	if err != nil {
		t.Fatalf("failed to delete review: %v", err)
	}

	// DeleteGame Test
	err = s.DeleteGame(777)
	if err != nil {
		t.Fatalf("failed to delete game: %v", err)
	}
}
