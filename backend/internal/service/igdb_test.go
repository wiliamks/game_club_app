package service

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIGDBClientMock(t *testing.T) {
	client := &igdbClient{useMock: true}

	// 1. Search mock games
	games, err := client.SearchGames("Zelda")
	if err != nil {
		t.Fatalf("expected no error searching mock, got %v", err)
	}
	if len(games) == 0 {
		t.Error("expected to find Zelda in mock games list")
	}

	// 2. Fetch specific mock game details
	game, err := client.GetGameDetails(1)
	if err != nil {
		t.Fatalf("expected no error getting details, got %v", err)
	}
	if game.Name != "The Legend of Zelda: Tears of the Kingdom" {
		t.Errorf("expected Zelda, got %s", game.Name)
	}

	// 3. Fallback for non-existent game ID
	fallbackGame, err := client.GetGameDetails(999)
	if err != nil {
		t.Fatalf("expected no error for unknown ID in mock, got %v", err)
	}
	if fallbackGame.ID != 999 {
		t.Errorf("expected ID 999, got %d", fallbackGame.ID)
	}
}

func TestIGDBClientHttp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/token" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"access_token": "mocked_access_token", "expires_in": 3600, "token_type": "bearer"}`))
			return
		}
		if r.URL.Path == "/v4/games" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id": 101, "name": "Final Fantasy VII", "summary": "An epic RPG", "cover": {"url": "//images.igdb.com/co1r3d.jpg"}, "first_release_date": 854582400}]`))
			return
		}
		if r.URL.Path == "/v4/game_time_to_beats" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"completely": 144000, "normally": 108000, "hastily": 72000, "game_id": 101}]`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := &igdbClient{
		clientID:     "dummy_client_id",
		clientSecret: "dummy_client_secret",
		httpClient:   server.Client(),
		tokenURL:     server.URL + "/oauth2/token",
		apiURL:       server.URL + "/v4",
		useMock:      false,
	}

	// 1. Test SearchGames
	games, err := client.SearchGames("Fantasy")
	if err != nil {
		t.Fatalf("SearchGames failed: %v", err)
	}
	if len(games) != 1 || games[0].Name != "Final Fantasy VII" {
		t.Errorf("expected Final Fantasy VII, got %v", games)
	}

	// 2. Test GetGameDetails
	game, err := client.GetGameDetails(101)
	if err != nil {
		t.Fatalf("GetGameDetails failed: %v", err)
	}
	if game.ID != 101 {
		t.Errorf("expected ID 101, got %d", game.ID)
	}
	if game.TimeToBeat != "30 hours" {
		t.Errorf("expected TimeToBeat '30 hours', got '%s'", game.TimeToBeat)
	}
}
