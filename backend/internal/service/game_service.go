package service

import (
	"errors"
	"fmt"
	"time"

	"gamer-club/backend/internal/models"
	"gamer-club/backend/internal/repository"
)

// GameService defines the business logic interface for games and reviews
type GameService interface {
	ListGames() ([]*models.Game, error)
	GetGameDetails(gameID int) (*models.GameDetailsDTO, error)
	SubmitReview(userID int, username string, gameID int, r *models.Review) error
	SetActiveGame(gameID int) error
	DeactivateActiveGame() error
	DeleteReview(userID, gameID int) error
	DeleteGame(id int) error
}

type gameService struct {
	repo       repository.GameRepository
	igdbClient IGDBClient
}

// NewGameService creates a new GameService implementation
func NewGameService(repo repository.GameRepository, igdbClient IGDBClient) GameService {
	return &gameService{
		repo:       repo,
		igdbClient: igdbClient,
	}
}

func (s *gameService) ListGames() ([]*models.Game, error) {
	games, err := s.repo.GetAllGames()
	if err != nil {
		return nil, err
	}

	for _, g := range games {
		reviews, err := s.repo.GetReviewsByGameID(g.ID)
		if err == nil && len(reviews) > 0 {
			avg := s.calculateAverages(reviews)
			g.AverageScore = avg.Overall
		}
	}

	return games, nil
}

func (s *gameService) GetGameDetails(gameID int) (*models.GameDetailsDTO, error) {
	// 1. Check if game exists in local DB
	g, err := s.repo.GetGameByID(gameID)
	if err != nil {
		return nil, err
	}

	// If game does not exist locally, fetch from IGDB and insert into local DB
	if g == nil {
		g, err = s.igdbClient.GetGameDetails(gameID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch game details from IGDB: %w", err)
		}

		if err := s.repo.SaveGame(g); err != nil {
			return nil, fmt.Errorf("failed to save imported game details locally: %w", err)
		}
	}

	// 2. Fetch reviews
	reviews, err := s.repo.GetReviewsByGameID(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve reviews: %w", err)
	}

	// 3. Calculate averages, excluding 0-scores per category
	averages := s.calculateAverages(reviews)

	return &models.GameDetailsDTO{
		Game:     g,
		Averages: averages,
		Reviews:  reviews,
	}, nil
}

func (s *gameService) SubmitReview(userID int, username string, gameID int, r *models.Review) error {
	if r.Gameplay < 0 || r.Gameplay > 5 ||
		r.Art < 0 || r.Art > 5 ||
		r.Story < 0 || r.Story > 5 ||
		r.Soundtrack < 0 || r.Soundtrack > 5 ||
		r.Fun < 0 || r.Fun > 5 {
		return errors.New("review ratings must be between 0 and 5")
	}

	// Ensure the game exists in our local DB before saving the review (since reviews reference games)
	g, err := s.repo.GetGameByID(gameID)
	if err != nil {
		return err
	}

	if g == nil {
		g, err = s.igdbClient.GetGameDetails(gameID)
		if err != nil {
			return fmt.Errorf("failed to import game details for review: %w", err)
		}
		if err := s.repo.SaveGame(g); err != nil {
			return fmt.Errorf("failed to save imported game for review: %w", err)
		}
	}

	r.GameID = gameID
	r.UserID = userID
	r.Username = username
	r.CreatedAt = time.Now()

	return s.repo.SaveReview(r)
}

func (s *gameService) SetActiveGame(gameID int) error {
	g, err := s.repo.GetGameByID(gameID)
	if err != nil {
		return err
	}

	// Ensure the game exists locally
	if g == nil {
		g, err = s.igdbClient.GetGameDetails(gameID)
		if err != nil {
			return fmt.Errorf("failed to import game details: %w", err)
		}
		if err := s.repo.SaveGame(g); err != nil {
			return fmt.Errorf("failed to save imported game: %w", err)
		}
	}

	return s.repo.SetActiveGame(gameID)
}

func (s *gameService) DeactivateActiveGame() error {
	return s.repo.DeactivateActiveGame()
}

func (s *gameService) calculateAverages(reviews []*models.Review) *models.ReviewAverages {
	var gameplaySum, gameplayCount int
	var artSum, artCount int
	var storySum, storyCount int
	var soundtrackSum, soundtrackCount int
	var funSum, funCount int

	for _, r := range reviews {
		if r.Gameplay > 0 {
			gameplaySum += r.Gameplay
			gameplayCount++
		}
		if r.Art > 0 {
			artSum += r.Art
			artCount++
		}
		if r.Story > 0 {
			storySum += r.Story
			storyCount++
		}
		if r.Soundtrack > 0 {
			soundtrackSum += r.Soundtrack
			soundtrackCount++
		}
		if r.Fun > 0 {
			funSum += r.Fun
			funCount++
		}
	}

	averages := &models.ReviewAverages{}
	var nonZeroCategories int
	var categoryAveragesSum float64

	if gameplayCount > 0 {
		averages.Gameplay = float64(gameplaySum) / float64(gameplayCount)
		categoryAveragesSum += averages.Gameplay
		nonZeroCategories++
	}
	if artCount > 0 {
		averages.Art = float64(artSum) / float64(artCount)
		categoryAveragesSum += averages.Art
		nonZeroCategories++
	}
	if storyCount > 0 {
		averages.Story = float64(storySum) / float64(storyCount)
		categoryAveragesSum += averages.Story
		nonZeroCategories++
	}
	if soundtrackCount > 0 {
		averages.Soundtrack = float64(soundtrackSum) / float64(soundtrackCount)
		categoryAveragesSum += averages.Soundtrack
		nonZeroCategories++
	}
	if funCount > 0 {
		averages.Fun = float64(funSum) / float64(funCount)
		categoryAveragesSum += averages.Fun
		nonZeroCategories++
	}

	if nonZeroCategories > 0 {
		averages.Overall = categoryAveragesSum / float64(nonZeroCategories)
	}

	return averages
}

func (s *gameService) DeleteReview(userID, gameID int) error {
	return s.repo.DeleteReview(userID, gameID)
}

func (s *gameService) DeleteGame(id int) error {
	return s.repo.DeleteGame(id)
}
