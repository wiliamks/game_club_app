# Gamer Club 🎮️

Gamer Club is a modern, high-performance, full-stack web application designed for gaming communities to catalog games, submit detailed review scorecards, and run collaborative, ranked-choice preferential voting cycles.

The entire application is containerized, secure, fully internationalized (out-of-the-box support for English, Portuguese, Spanish, and Japanese), and responsive on all screen sizes, ranging from wide 4K desktop displays to compact mobile touchscreens.

---

## 🛠️ Tech Stack & Architecture

### Frontend
*   **React (v19) & TypeScript:** Component-driven design utilizing strict compile-time types.
*   **Tailwind CSS (v4):** Modern utility-first CSS styling, configured natively with class-based Light/Dark themes.
*   **Lucide React:** Sharp, modern vector iconography.
*   **Fault-Tolerant Context Architecture:**
    *   `AuthContext`: Handles profile state, stores secure JWT tokens, and manages a background refresh loop every 15 minutes.
    *   `ThemeContext`: Controls class-based Light/Dark persistent switches.
    *   `LanguageContext`: Light, zero-dependency token-interpolator traversing nested key dictionaries.

### Backend
*   **Golang (v1.25):** Clean, idiomatic architecture following the Single Responsibility Principle (SRP) per file.
*   **Native HTTP Router (Go 1.22+):** Fast, pattern-matched multiplexer (`http.NewServeMux`) without heavy third-party routing overhead.
*   **IGDB Client Wrapper:** Thread-safe client implementing the OAuth2 Client Credentials flow. Features local in-memory token caching, automatic background renewal, and a 30-game local mock fallback for unconfigured offline environments.

### Database
*   **SQLite (v3):** High-concurrency database connections configured with:
    *   `PRAGMA foreign_keys = ON` (strict relational cascade deletes).
    *   `PRAGMA journal_mode = WAL` (Write-Ahead Logging for high-frequency write operations).
*   **Self-Healing Bootstrapper:** Automated folder and parent-directory creation on start to ensure database stability during container cold-starts.

### DevOps & Orchestration
*   **Docker & Multi-Stage Builds:**
    *   `backend`: Compiled statically as a minimal Alpine runner (gcc, musl-dev toolchain used during compilation for CGO).
    *   `frontend`: Static assets built using Node.js Alpine and deployed into a lightweight Nginx Alpine container.
*   **Nginx Reverse Proxy:** Nginx acts as a high-performance Single Page Application (SPA) router and secure reverse proxy forwarding `/api` calls internally, eliminating CORS and domain mismatches.
*   **Docker Compose:** Harmonizes the multi-container graph and guarantees data persistence via Named Docker Volumes, fully compatible with unprivileged rootless ports.

---

## 📂 Directory Structure

```text
gamer-club/
├── .env                  # Environment configuration (ignored by git)
├── .gitignore            # Clean git exclusion rules
├── docker-compose.yml    # Runs frontend, backend, and volume orchestrations
├── backend/
│   ├── Dockerfile        # Multi-stage static CGO Alpine build
│   ├── main.go           # Dependency injection, CORS chain, & HTTP bootstrap
│   └── internal/
│       ├── database/     # SQLite WAL connection & schema migrations
│       ├── models/       # Shared JSON-annotated models and DTOs
│       ├── repository/   # Decoupled SQL data mappers (User, Game, Voting)
│       ├── service/      # Business logic (User, Games, Voting, IGDB)
│       │   └── *_test.go # Mock-driven unit tests (>=80% coverage)
│       ├── handler/      # REST API Controllers (JSON helpers)
│       └── middleware/   # JWT Auth context injection & Admin guards
└── frontend/
    ├── Dockerfile        # Node builder & Nginx static serving
    ├── nginx.conf        # Nginx SPA router and secure /api/ reverse proxy
    └── src/
        ├── assets/       # Static assets & brand logos
        ├── context/      # Auth, Theme, and Language context states
        ├── locales/      # i18n translation JSON dictionaries
        ├── components/   # Modular, responsive views and panels
        │   ├── Login.tsx     # Clean login screen
        │   ├── Sidebar.tsx   # Persistent desktop sidebar
        │   ├── GamesView.tsx # Games dashboard (scorecards & statistics)
        │   ├── VotingView.tsx# Preferential ranked-choice voting cycles
        │   ├── AccountView.tsx# Profile settings & custom avatar URLs
        │   └── AdminView.tsx # Admin controls (users, cycles, active game)
        ├── index.css     # Tailwind imports and custom base scrollbars
        └── main.tsx      # StrictReact mounting entrypoint
```

---

## 🔄 Core Application Workflow

### 1. Authentication & Account Settings
*   **Admin-Provisioned Accounts:** Accounts are created exclusively by administrators. A default master administrator is seeded on first boot:
    *   **Username:** `admin`
    *   **Password:** `admin`
*   **Profile Scorecard**: Users can update their Username, Password, and paste a custom **Profile Picture URL (Avatar)** in the **Account** tab, instantly synchronizing their header card and reviews list.

### 2. Pinned Games & Review Scorecards
*   **Two-Column Dashboard:**
    *   *Column 1 (List)*: Displays the currently promoted active game pinned permanently on top, followed by a historical list sorted chronologically by last active date. Includes local search filtering and ascending/descending sorting toggles (by name, release date, or average rating).
    *   *Column 2 (Details)*: Displays complete game metadata (IGDB cover, summary, localized Playstyle Durations, and aggregated ratings).
*   **Playstyle Durations**: Displays three distinct real-world completion times fetched from the IGDB `/game_time_to_beats` endpoint:
    1.  **Main Story** (hastily)
    2.  **Main + Side Quests** (normally)
    3.  **Completionist** (completely)
*   **Exclusionary Average Grading**: Community members can submit or update a review scorecard grading **Gameplay, Art, Story, Soundtrack, and Fun** (from 1 to 5 stars).
    *   *Critical Rule*: A score of `0` denotes an unrated category and is **completely excluded** from average calculation variables.
*   **Community Matrix**: Clicking on any average score box opens a sticky detailed tabular grid with users as columns, categories as rows, and scores shown as unvarnished integers (with frozen categories columns for horizontal scrolls).

### 3. Ranked-Choice Preferential Voting Cycles
*   **Nomination Phase**: Users can search the global IGDB database inside an 80% width/height pop-up modal, browse candidate metadata (including first release dates), and nominate games up to a session limit configured by the administrator.
*   **Preferential Voting Phase**: Nominees are listed inside a drag-and-drop preferential ballot card. Users can drag or click Up/Down buttons to rank candidates in order of preference.
*   **Closed Phase (Results)**: The system automatically computes standings on-the-fly using the **Borda Count** algorithm. Ties are resolved non-deterministically (randomly) while preserving strict mathematical transitivity. All user-nominated games of a closed session are automatically imported into the main games repository, enabling community reviews and active pinning.

---

## 🚀 Setup & Launch Instructions

### Prerequisites
Ensure you have **Podman** (or **Docker**) and **Docker Compose** installed on your host system. Since the configuration is optimized with Named Volumes and unprivileged ports, it is fully compatible with **Rootless Docker** out of the box.

### Quick Start
1.  **Clone and Enter:**
    ```bash
    git clone https://github.com/wiliamks/game_club_app.git
    cd game_club_app
    ```
2.  **Configure environment:** Create a `.env` file at the project root (or leave empty to default to offline fallback mock mode):
    ```ini
    PORT=8080
    JWT_SECRET=use_a_strong_random_secret_here
    DB_PATH=/app/data/gamer_club.db
    
    # Twitch developer portal credentials (optional)
    IGDB_CLIENT_ID=your_client_id
    IGDB_CLIENT_SECRET=your_client_secret
    ```
3.  **Launch the container graph:**
    ```bash
    docker compose up --build -d
    ```
    *(If using Podman, run `podman-compose up --build -d` instead).*
4.  **Open the application:**
    *   **Frontend Web Panel:** Open `http://localhost:3000` in your browser.
    *   **Login credentials:** Use `admin` / `admin` to log in as the default administrator.
