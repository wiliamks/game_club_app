package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"gamer-club/backend/internal/models"
)

// IGDBClient defines the interface for interacting with the IGDB API
type IGDBClient interface {
	SearchGames(query string) ([]*models.Game, error)
	GetGameDetails(id int) (*models.Game, error)
}

type igdbClient struct {
	clientID     string
	clientSecret string
	token        string
	expiresAt    time.Time
	mu           sync.RWMutex
	httpClient   *http.Client
	tokenURL     string
	apiURL       string
	useMock      bool
}

// NewIGDBClient creates a new IGDB Client
func NewIGDBClient() IGDBClient {
	clientID := "pga8hva80v3ur2khg3iagj7nv8al77"     //os.Getenv("IGDB_CLIENT_ID")
	clientSecret := "advlc9mu2r724uqft0lubhsl9f3jin" // os.Getenv("IGDB_CLIENT_SECRET")

	useMock := false
	if clientID == "" || clientSecret == "" || clientID == "your_igdb_client_id" || clientSecret == "your_igdb_client_secret" {
		log.Println("IGDB credentials not configured or set to placeholder. Falling back to local offline mock games service.")
		useMock = true
	}

	return &igdbClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		tokenURL:     "https://id.twitch.tv/oauth2/token",
		apiURL:       "https://api.igdb.com/v4",
		useMock:      useMock,
	}
}

// TokenResponse represents Twitch oauth token response
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

type igdbCover struct {
	URL string `json:"url"`
}

type igdbGame struct {
	ID               int        `json:"id"`
	Name             string     `json:"name"`
	Summary          string     `json:"summary"`
	Cover            *igdbCover `json:"cover"`
	FirstReleaseDate int64      `json:"first_release_date"`
}

// Helper to calculate deterministic time to beat based on game ID
func calculateTimeToBeat(id int) string {
	hours := (id % 45) + 10 // between 10 and 54 hours
	return fmt.Sprintf("%d hours", hours)
}

func (c *igdbClient) getAccessToken() (string, error) {
	c.mu.RLock()
	// If token exists and has at least 1 minute of validity remaining, use it
	if c.token != "" && time.Now().Add(1*time.Minute).Before(c.expiresAt) {
		token := c.token
		c.mu.RUnlock()
		return token, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if c.token != "" && time.Now().Add(1*time.Minute).Before(c.expiresAt) {
		return c.token, nil
	}

	// Request new token
	url := fmt.Sprintf("%s?client_id=%s&client_secret=%s&grant_type=client_credentials", c.tokenURL, c.clientID, c.clientSecret)
	resp, err := c.httpClient.Post(url, "application/json", nil)
	if err != nil {
		return "", fmt.Errorf("failed to contact Twitch OAuth server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("twitch OAuth error status %d: %s", resp.StatusCode, string(body))
	}

	var tr TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", fmt.Errorf("failed to decode twitch OAuth token response: %w", err)
	}

	c.token = tr.AccessToken
	c.expiresAt = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)

	log.Printf("Successfully acquired new IGDB/Twitch access token (expires in %d seconds)", tr.ExpiresIn)
	return c.token, nil
}

func (c *igdbClient) SearchGames(query string) ([]*models.Game, error) {
	if c.useMock {
		return c.getMockGames(query), nil
	}

	token, err := c.getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate with IGDB: %w", err)
	}

	// Construct IGDB Query
	// Fields: name, summary, cover.url, first_release_date
	bodyQuery := fmt.Sprintf(`search "%s"; fields name, summary, cover.url, first_release_date; limit 30;`, strings.ReplaceAll(query, `"`, `\"`))

	req, err := http.NewRequest("POST", c.apiURL+"/games", bytes.NewBufferString(bodyQuery))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Client-ID", c.clientID)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed making request to IGDB: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("igdb search status %d: %s", resp.StatusCode, string(body))
	}

	var rawGames []igdbGame
	if err := json.NewDecoder(resp.Body).Decode(&rawGames); err != nil {
		return nil, err
	}

	var games []*models.Game
	for _, rg := range rawGames {
		cover := ""
		if rg.Cover != nil {
			cover = rg.Cover.URL
			if strings.HasPrefix(cover, "//") {
				cover = "https:" + cover
			}
			// Replace default thumb with higher resolution image if possible (t_cover_big or t_720p)
			cover = strings.Replace(cover, "t_thumb", "t_cover_big", 1)
		}

		var relDate *time.Time
		if rg.FirstReleaseDate > 0 {
			t := time.Unix(rg.FirstReleaseDate, 0).UTC()
			relDate = &t
		}

		games = append(games, &models.Game{
			ID:          rg.ID,
			Name:        rg.Name,
			Summary:     rg.Summary,
			CoverURL:    cover,
			ReleaseDate: relDate,
			TimeToBeat:  calculateTimeToBeat(rg.ID),
		})
	}

	return games, nil
}

func (c *igdbClient) GetGameDetails(id int) (*models.Game, error) {
	if c.useMock {
		games := c.getMockGames("")
		for _, g := range games {
			if g.ID == id {
				return g, nil
			}
		}
		// If not found in default mock games, return a generated details struct
		t := time.Now().AddDate(-5, 0, 0)
		return &models.Game{
			ID:          id,
			Name:        fmt.Sprintf("Mock Game #%d", id),
			Summary:     "This is a mocked fallback game description since IGDB credentials are not active or the game was not found in the static list.",
			CoverURL:    "https://images.igdb.com/igdb/image/upload/t_cover_big/co1r3d.jpg",
			ReleaseDate: &t,
			TimeToBeat:  calculateTimeToBeat(id),
		}, nil
	}

	token, err := c.getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate with IGDB: %w", err)
	}

	bodyQuery := fmt.Sprintf(`fields name, summary, cover.url, first_release_date; where id = %d;`, id)

	req, err := http.NewRequest("POST", c.apiURL+"/games", bytes.NewBufferString(bodyQuery))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Client-ID", c.clientID)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed making request to IGDB: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("igdb details status %d: %s", resp.StatusCode, string(body))
	}

	var rawGames []igdbGame
	if err := json.NewDecoder(resp.Body).Decode(&rawGames); err != nil {
		return nil, err
	}

	if len(rawGames) == 0 {
		return nil, fmt.Errorf("game not found on IGDB with ID %d", id)
	}

	rg := rawGames[0]
	cover := ""
	if rg.Cover != nil {
		cover = rg.Cover.URL
		if strings.HasPrefix(cover, "//") {
			cover = "https:" + cover
		}
		cover = strings.Replace(cover, "t_thumb", "t_cover_big", 1)
	}

	var relDate *time.Time
	if rg.FirstReleaseDate > 0 {
		t := time.Unix(rg.FirstReleaseDate, 0).UTC()
		relDate = &t
	}

	// 2. Fetch Time to Beat details from `/game_time_to_beats`
	timeToBeat := calculateTimeToBeat(rg.ID) // Default fallback

	timeToBeatBody := fmt.Sprintf(`fields completely, normally, hastily, game_id; where game_id = %d;`, rg.ID)
	reqTTB, err := http.NewRequest("POST", c.apiURL+"/game_time_to_beats", bytes.NewBufferString(timeToBeatBody))
	if err == nil {
		reqTTB.Header.Set("Client-ID", c.clientID)
		reqTTB.Header.Set("Authorization", "Bearer "+token)
		reqTTB.Header.Set("Accept", "application/json")

		respTTB, err := c.httpClient.Do(reqTTB)
		if err == nil && respTTB.StatusCode == http.StatusOK {
			defer respTTB.Body.Close()
			var rawTTB []struct {
				Completely int64 `json:"completely"`
				Normally   int64 `json:"normally"`
				Hastily    int64 `json:"hastily"`
			}
			if err := json.NewDecoder(respTTB.Body).Decode(&rawTTB); err == nil && len(rawTTB) > 0 {
				ttb := rawTTB[0]
				var seconds int64
				if ttb.Normally > 0 {
					seconds = ttb.Normally
				} else if ttb.Hastily > 0 {
					seconds = ttb.Hastily
				} else if ttb.Completely > 0 {
					seconds = ttb.Completely
				}

				if seconds > 0 {
					hours := seconds / 3600
					if hours > 0 {
						timeToBeat = fmt.Sprintf("%d hours", hours)
					}
				}
			}
		}
	}

	return &models.Game{
		ID:          rg.ID,
		Name:        rg.Name,
		Summary:     rg.Summary,
		CoverURL:    cover,
		ReleaseDate: relDate,
		TimeToBeat:  timeToBeat,
	}, nil
}

// getMockGames returns a static list of highly popular games to use when credentials are not configured
func (c *igdbClient) getMockGames(query string) []*models.Game {
	d := func(y, m, d int) *time.Time {
		t := time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
		return &t
	}

	mocks := []*models.Game{
		{
			ID:          1,
			Name:        "The Legend of Zelda: Tears of the Kingdom",
			Summary:     "The Legend of Zelda: Tears of the Kingdom is the sequel to Breath of the Wild. Link’s adventure expands to include the massive floating islands in the skies above Hyrule.",
			CoverURL:    "https://images.igdb.com/igdb/image/upload/t_cover_big/co68v4.jpg",
			ReleaseDate: d(2023, 5, 12),
			TimeToBeat:  "55 hours",
		},
		{
			ID:          2,
			Name:        "Elden Ring",
			Summary:     "Elden Ring is a legendary fantasy action-RPG set within a world created by Hidetaka Miyazaki and George R.R. Martin. Rise, Tarnished, and be guided by grace to brandish the power of the Elden Ring.",
			CoverURL:    "https://images.igdb.com/igdb/image/upload/t_cover_big/co4kbj.jpg",
			ReleaseDate: d(2022, 2, 25),
			TimeToBeat:  "60 hours",
		},
		{
			ID:          3,
			Name:        "Cyberpunk 2077",
			Summary:     "An open-world action-adventure RPG set in Night City, a megalopolis obsessed with power, glamour and body modification. Play as V, a mercenary outlaw searching for a unique cybernetic implant.",
			CoverURL:    "https://images.igdb.com/igdb/image/upload/t_cover_big/co2m37.jpg",
			ReleaseDate: d(2020, 12, 10),
			TimeToBeat:  "35 hours",
		},
		{
			ID:          4,
			Name:        "Chrono Trigger",
			Summary:     "When a newly-developed teleportation device malfunctions, Crono must travel through time to rescue his companions from paste, present, and apocalyptic future perils in this classic square RPG.",
			CoverURL:    "https://images.igdb.com/igdb/image/upload/t_cover_big/co3v90.jpg",
			ReleaseDate: d(1995, 3, 11),
			TimeToBeat:  "25 hours",
		},
		{
			ID:          5,
			Name:        "Red Dead Redemption 2",
			Summary:     "America, 1899. Arthur Morgan and the Van der Linde gang are outlaws on the run. With federal agents on their heels, they must rob, steal, and fight across the rugged heartland of America.",
			CoverURL:    "https://images.igdb.com/igdb/image/upload/t_cover_big/co1r1h.jpg",
			ReleaseDate: d(2018, 10, 26),
			TimeToBeat:  "50 hours",
		},
		{
			ID:          6,
			Name:        "Super Mario Odyssey",
			Summary:     "Join Mario on a massive, globe-trotting 3D adventure! Use his incredible new cap-based abilities to collect Power Moons and rescue Princess Peach from Bowser.",
			CoverURL:    "https://images.igdb.com/igdb/image/upload/t_cover_big/co1mab.jpg",
			ReleaseDate: d(2017, 10, 27),
			TimeToBeat:  "15 hours",
		},
		{
			ID:          7,
			Name:        "Halo: Combat Evolved",
			Summary:     "A science fiction first-person shooter where the player assumes the role of the Master Chief, a cybernetically enhanced supersoldier, battling the alien Covenant forces on a mysterious ringworld.",
			CoverURL:    "https://images.igdb.com/igdb/image/upload/t_cover_big/co3fkv.jpg",
			ReleaseDate: d(2001, 11, 15),
			TimeToBeat:  "10 hours",
		},
		{
			ID:          8,
			Name:        "Final Fantasy VII",
			Summary:     "A post-industrial sci-fi fantasy RPG following Cloud Strife, a mercenary joining an eco-terrorist group to stop the megacorporation Shinra from draining the planet's life blood.",
			CoverURL:    "https://images.igdb.com/igdb/image/upload/t_cover_big/co1r3d.jpg",
			ReleaseDate: d(1997, 1, 31),
			TimeToBeat:  "40 hours",
		},
		{
			ID:          9,
			Name:        "Grand Theft Auto V",
			Summary:     "Three very different criminals plot their own chances of survival and success in Los Santos: a street hustler, a retired bank robber, and a terrifying psychopathic redneck.",
			CoverURL:    "https://images.igdb.com/igdb/image/upload/t_cover_big/co2lbd.jpg",
			ReleaseDate: d(2013, 9, 17),
			TimeToBeat:  "32 hours",
		},
		{
			ID:          10,
			Name:        "Half-Life 2",
			Summary:     "By taking the suspense, challenge and visceral charge of the original, and adding startling new realism and responsiveness, Half-Life 2 opens the door to a world where the player's presence affects everything.",
			CoverURL:    "https://images.igdb.com/igdb/image/upload/t_cover_big/co2fhd.jpg",
			ReleaseDate: d(2004, 11, 16),
			TimeToBeat:  "13 hours",
		},
		{
			ID:          11,
			Name:        "The Witcher 3: Wild Hunt",
			Summary:     "Become Geralt of Rivia, a professional monster slayer hired to find a child of prophecy in a vast open world rich with merchant cities, dangerous mountain passes, and forgotten caverns.",
			CoverURL:    "https://images.igdb.com/igdb/image/upload/t_cover_big/co1r7f.jpg",
			ReleaseDate: d(2015, 5, 19),
			TimeToBeat:  "52 hours",
		},
		{
			ID:          12,
			Name:        "Minecraft",
			Summary:     "A sandbox game where players can build, mine, craft, and explore infinitely generated block-based 3D terrains. Includes survival, creative, and multiplayer game modes.",
			CoverURL:    "https://images.igdb.com/igdb/image/upload/t_cover_big/co1p81.jpg",
			ReleaseDate: d(2011, 11, 18),
			TimeToBeat:  "80 hours",
		},
		{
			ID:          13,
			Name:        "Portal 2",
			Summary:     "A mind-bending first-person puzzle-platformer. Players must navigate tests in Aperture Science Laboratories using a hand-held portal device while overcoming the sentient AI, GLaDOS.",
			CoverURL:    "https://images.igdb.com/igdb/image/upload/t_cover_big/co1xs4.jpg",
			ReleaseDate: d(2011, 4, 18),
			TimeToBeat:  "9 hours",
		},
		{
			ID:          14,
			Name:        "Dark Souls III",
			Summary:     "As fires fade and the world falls into ruin, journey into a universe filled with more colossal enemies and environments. Immersive gameplay and high-difficulty action RPG.",
			CoverURL:    "https://images.igdb.com/igdb/image/upload/t_cover_big/co1xgh.jpg",
			ReleaseDate: d(2016, 3, 24),
			TimeToBeat:  "32 hours",
		},
		{
			ID:          15,
			Name:        "Resident Evil 4",
			Summary:     "Special Agent Leon S. Kennedy is sent on a mission to rescue the U.S. President's kidnapped daughter from a secluded European village controlled by a dangerous parasite cult.",
			CoverURL:    "https://images.igdb.com/igdb/image/upload/t_cover_big/co5z8f.jpg",
			ReleaseDate: d(2005, 1, 11),
			TimeToBeat:  "15 hours",
		},
		{
			ID:          16,
			Name:        "God of War",
			Summary:     "Kratos, having left his vengeance against the gods of Olympus behind, now lives as a man in the realm of Norse Gods and monsters. He must fight to survive and teach his son to do the same.",
			CoverURL:    "https://images.igdb.com/igdb/image/upload/t_cover_big/co1v9m.jpg",
			ReleaseDate: d(2018, 4, 20),
			TimeToBeat:  "21 hours",
		},
		{
			ID:          17,
			Name:        "The Last of Us Part I",
			Summary:     "In a devastated civilization, where infected and hardened survivors run rampant, Joel, a weary protagonist, is hired to smuggle 14-year-old Ellie out of a military quarantine zone.",
			CoverURL:    "https://images.igdb.com/igdb/image/upload/t_cover_big/co5v7f.jpg",
			ReleaseDate: d(2013, 6, 14),
			TimeToBeat:  "15 hours",
		},
		{
			ID:          18,
			Name:        "Mass Effect 2",
			Summary:     "Recruit an elite team of specialized operatives from across the galaxy and lead them on a suicide mission to stop a mysterious alien threat that is abducting human colonies.",
			CoverURL:    "https://images.igdb.com/igdb/image/upload/t_cover_big/co2947.jpg",
			ReleaseDate: d(2010, 1, 26),
			TimeToBeat:  "35 hours",
		},
		{
			ID:          19,
			Name:        "The Elder Scrolls V: Skyrim",
			Summary:     "The ancient dragon Alduin has returned to destroy the world. Rise as the Dragonborn, use shout powers, and explore an infinite snowy open-world empire.",
			CoverURL:    "https://images.igdb.com/igdb/image/upload/t_cover_big/co1v8f.jpg",
			ReleaseDate: d(2011, 11, 11),
			TimeToBeat:  "75 hours",
		},
		{
			ID:          20,
			Name:        "Doom (2016)",
			Summary:     "A modern reboot of the legendary first-person shooter. Play as the Doom Slayer, battling hordes of demonic forces across Union Aerospace research facilities on Mars and Hell.",
			CoverURL:    "https://images.igdb.com/igdb/image/upload/t_cover_big/co1v4s.jpg",
			ReleaseDate: d(2016, 5, 13),
			TimeToBeat:  "12 hours",
		},
		{
			ID:          21,
			Name:        "Hades",
			Summary:     "A rogue-like dungeon crawler following Zagreus, the prince of the Underworld, as he hacks and slashes his way out of the realms of his father, Hades, aided by Olympus.",
			CoverURL:    "https://images.igdb.com/igdb/image/upload/t_cover_big/co28gq.jpg",
			ReleaseDate: d(2020, 9, 17),
			TimeToBeat:  "22 hours",
		},
		{
			ID:          22,
			Name:        "Stardew Valley",
			Summary:     "You've inherited your grandfather's old farm plot in Stardew Valley. Armed with hand-me-down tools and a few coins, you set out to begin your new life in the countryside.",
			CoverURL:    "https://images.igdb.com/igdb/image/upload/t_cover_big/co1v7f.jpg",
			ReleaseDate: d(2016, 2, 26),
			TimeToBeat:  "50 hours",
		},
		{
			ID:          23,
			Name:        "Persona 5",
			Summary:     "Transferring to a school in Tokyo, a high schooler forms the Phantom Thieves of Hearts, a group of rebel students using persona alter egos to reform corrupt corrupt hearts.",
			CoverURL:    "https://images.igdb.com/igdb/image/upload/t_cover_big/co1um5.jpg",
			ReleaseDate: d(2016, 9, 15),
			TimeToBeat:  "97 hours",
		},
		{
			ID:          24,
			Name:        "Chrono Cross",
			Summary:     "The sequel to Chrono Trigger. Follow Serge, a young boy traveling between two parallel dimensions to discover the history behind his own childhood death.",
			CoverURL:    "https://images.igdb.com/igdb/image/upload/t_cover_big/co1r3d.jpg",
			ReleaseDate: d(1999, 11, 18),
			TimeToBeat:  "38 hours",
		},
		{
			ID:          25,
			Name:        "Metroid Prime",
			Summary:     "Samus Aran explores the mysterious Tallon IV planet. Mind-bending 3D sci-fi adventure, scanning environments and unlocking visors and morph ball abilities.",
			CoverURL:    "https://images.igdb.com/igdb/image/upload/t_cover_big/co1pba.jpg",
			ReleaseDate: d(2002, 11, 18),
			TimeToBeat:  "14 hours",
		},
		{
			ID:          26,
			Name:        "Pokemon Red/Blue",
			Summary:     "The classic Game Boy monster capturing RPG. Journey across the Kanto region, collect badges, catch 151 Pokemon, and defeat the Elite Four.",
			CoverURL:    "https://images.igdb.com/igdb/image/upload/t_cover_big/co1qff.jpg",
			ReleaseDate: d(1996, 2, 27),
			TimeToBeat:  "26 hours",
		},
		{
			ID:          27,
			Name:        "Sonic the Hedgehog",
			Summary:     "Run at lightning speeds as Sonic the Hedgehog! Blast through loop-de-loops, gather golden rings, and stop the evil Dr. Robotnik from taking over South Island.",
			CoverURL:    "https://images.igdb.com/igdb/image/upload/t_cover_big/co1v9x.jpg",
			ReleaseDate: d(1991, 6, 23),
			TimeToBeat:  "2 hours",
		},
		{
			ID:          28,
			Name:        "World of Warcraft",
			Summary:     "The massive multiplayer online RPG (MMORPG) set in the universe of Azeroth. Join the Alliance or the Horde and quest across continents.",
			CoverURL:    "https://images.igdb.com/igdb/image/upload/t_cover_big/co1r8c.jpg",
			ReleaseDate: d(2004, 11, 23),
			TimeToBeat:  "120 hours",
		},
		{
			ID:          29,
			Name:        "Tetris",
			Summary:     "The legendary Soviet puzzle game. Arrange falling tetromino blocks of different shapes into complete horizontal rows to clear them and score points.",
			CoverURL:    "https://images.igdb.com/igdb/image/upload/t_cover_big/co20m3.jpg",
			ReleaseDate: d(1984, 6, 6),
			TimeToBeat:  "5 hours",
		},
		{
			ID:          30,
			Name:        "Metal Gear Solid",
			Summary:     "Solid Snake must infiltrate a nuclear weapons facility in Alaska to neutralize the rogue special forces unit, FOXHOUND, in this tactical stealth-action masterpiece.",
			CoverURL:    "https://images.igdb.com/igdb/image/upload/t_cover_big/co1u9x.jpg",
			ReleaseDate: d(1998, 9, 3),
			TimeToBeat:  "12 hours",
		},
	}

	// Extract search query tokens (words) for flexible conjunctive (AND) matching
	words := strings.Fields(strings.ToLower(query))
	if len(words) == 0 {
		return mocks
	}

	var results []*models.Game
	for _, m := range mocks {
		nameLower := strings.ToLower(m.Name)
		summaryLower := strings.ToLower(m.Summary)
		match := true

		// All search words must be present in either Name or Summary
		for _, w := range words {
			if !strings.Contains(nameLower, w) && !strings.Contains(summaryLower, w) {
				match = false
				break
			}
		}
		if match {
			results = append(results, m)
		}
	}
	return results
}
