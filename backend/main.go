package main

import (
	"log"
	"net/http"
	"os"

	"gamer-club/backend/internal/database"
	"gamer-club/backend/internal/handler"
	"gamer-club/backend/internal/middleware"
	"gamer-club/backend/internal/repository"
	"gamer-club/backend/internal/service"
)

func main() {
	log.Println("Starting Gamer Club backend...")

	// 1. Initialize SQLite Database
	db, err := database.InitDB()
	if err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer db.Close()

	// 2. Instantiate Repositories
	userRepo := repository.NewUserRepository(db)
	gameRepo := repository.NewGameRepository(db)
	votingRepo := repository.NewVotingRepository(db)

	// 3. Instantiate Services
	igdbClient := service.NewIGDBClient()
	userService := service.NewUserService(userRepo)
	gameService := service.NewGameService(gameRepo, igdbClient)
	votingService := service.NewVotingService(votingRepo, gameRepo, igdbClient)

	// 4. Instantiate Handlers
	authHandler := handler.NewAuthHandler(userService)
	gameHandler := handler.NewGameHandler(gameService, igdbClient)
	votingHandler := handler.NewVotingHandler(votingService)

	// 5. Initialize Router (using Go 1.22+ structured patterns)
	mux := http.NewServeMux()

	// --- Public / Auth Routes ---
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)

	// --- Authenticated Routes (wrapped manually with auth guards) ---
	mux.HandleFunc("GET /api/auth/me", middleware.AuthRequired(authHandler.Me))
	mux.HandleFunc("POST /api/auth/refresh", middleware.AuthRequired(authHandler.Refresh))
	mux.HandleFunc("PUT /api/account/update", middleware.AuthRequired(authHandler.UpdateAccount))

	// --- Games Dashboard Routes ---
	mux.HandleFunc("GET /api/games", middleware.AuthRequired(gameHandler.ListGames))
	mux.HandleFunc("GET /api/games/{id}", middleware.AuthRequired(gameHandler.GetGameDetails))
	mux.HandleFunc("POST /api/games/{id}/reviews", middleware.AuthRequired(gameHandler.SubmitReview))
	mux.HandleFunc("DELETE /api/games/{id}/reviews", middleware.AuthRequired(gameHandler.DeleteReview))
	mux.HandleFunc("GET /api/igdb/search", middleware.AuthRequired(gameHandler.SearchIGDB))

	// --- Voting / Ranking choice System ---
	mux.HandleFunc("GET /api/voting/session", middleware.AuthRequired(votingHandler.GetActiveSession))
	mux.HandleFunc("GET /api/voting/sessions", middleware.AuthRequired(votingHandler.ListSessions))
	mux.HandleFunc("GET /api/voting/nominations", middleware.AuthRequired(votingHandler.ListNominations))
	mux.HandleFunc("GET /api/voting/nominations/me", middleware.AuthRequired(votingHandler.ListMyNominations))
	mux.HandleFunc("POST /api/voting/nominations", middleware.AuthRequired(votingHandler.NominateGame))
	mux.HandleFunc("POST /api/voting/vote", middleware.AuthRequired(votingHandler.SubmitVote))
	mux.HandleFunc("GET /api/voting/vote/me", middleware.AuthRequired(votingHandler.GetMyVote))
	mux.HandleFunc("GET /api/voting/results", middleware.AuthRequired(votingHandler.GetResults))

	// --- Admin Controller Routes ---
	// User CRUD
	mux.HandleFunc("GET /api/admin/users", middleware.AuthRequired(middleware.AdminRequired(authHandler.ListUsers)))
	mux.HandleFunc("POST /api/admin/users", middleware.AuthRequired(middleware.AdminRequired(authHandler.CreateUser)))
	mux.HandleFunc("DELETE /api/admin/users/{id}", middleware.AuthRequired(middleware.AdminRequired(authHandler.DeleteUser)))

	// Active Game control
	mux.HandleFunc("POST /api/admin/active-game", middleware.AuthRequired(middleware.AdminRequired(gameHandler.SetActiveGame)))
	mux.HandleFunc("DELETE /api/admin/active-game", middleware.AuthRequired(middleware.AdminRequired(gameHandler.DeactivateActiveGame)))
	mux.HandleFunc("DELETE /api/admin/games/{id}", middleware.AuthRequired(middleware.AdminRequired(gameHandler.DeleteGame)))

	// Voting Event session control
	mux.HandleFunc("POST /api/admin/voting/session", middleware.AuthRequired(middleware.AdminRequired(votingHandler.CreateSession)))
	mux.HandleFunc("PUT /api/admin/voting/phase", middleware.AuthRequired(middleware.AdminRequired(votingHandler.UpdatePhase)))
	mux.HandleFunc("DELETE /api/admin/voting/session", middleware.AuthRequired(middleware.AdminRequired(votingHandler.CancelSession)))

	// 6. Chain Core Middlewares: Auth Context Injection + CORS + Logging
	handlerWithAuthCtx := middleware.AuthMiddleware(userService)(mux)
	finalHandler := enableCORS(handlerWithAuthCtx)

	// 7. Start Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server listening on port %s", port)
	if err := http.ListenAndServe(":" + port, finalHandler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow any origin during development or fetch from config
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
