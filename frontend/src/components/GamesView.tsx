import React, { useState, useEffect } from 'react';
import { useAuth } from '../context/AuthContext';
import { useLanguage } from '../context/LanguageContext';
import ALL_EMOJIS from '../emojis.json';
import { 
  Star, 
  Award, 
  MessageSquare, 
  ChevronRight, 
  Bookmark, 
  ArrowUpDown,
  Gamepad2,
  X,
  Search,
  ArrowUp,
  ArrowDown,
  Trash2,
  ChevronLeft,
  Smile
} from 'lucide-react';

interface Game {
  id: number;
  name: string;
  summary: string;
  cover_url: string;
  release_date: string | null;
  time_to_beat: string;
  last_active_date: string | null;
  is_active: boolean;
  average_score: number;
  time_to_beat_normal: string;
  time_to_beat_hastily: string;
  time_to_beat_completely: string;
}

interface EmojiReactionSummary {
  emoji: string;
  count: number;
  user_reacted: boolean;
  usernames?: string[];
}

interface Review {
  id: number;
  game_id: number;
  user_id: number;
  username: string;
  title: string;
  avatar_url: string;
  gameplay: number;
  art: number;
  story: number;
  soundtrack: number;
  fun: number;
  comment: string;
  created_at: string;
  reactions?: EmojiReactionSummary[];
}

interface ReviewAverages {
  gameplay: number;
  art: number;
  story: number;
  soundtrack: number;
  fun: number;
  overall: number;
}

interface GameDetails {
  game: Game;
  averages: ReviewAverages | null;
  reviews: Review[];
}

export const GamesView: React.FC = () => {
  const { user, apiFetch } = useAuth();
  const { t } = useLanguage();

  const [games, setGames] = useState<Game[]>([]);
  const [selectedGameId, setSelectedGameId] = useState<number | null>(null);
  const [details, setDetails] = useState<GameDetails | null>(null);
  const [loadingList, setLoadingList] = useState(true);
  const [loadingDetails, setLoadingDetails] = useState(false);
  const [sortBy, setSortBy] = useState<'last_active' | 'name' | 'release_date' | 'score'>('last_active');
  const [searchQueryList, setSearchQueryList] = useState('');
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc');

  // Review Form state
  const [reviewTitle, setReviewTitle] = useState('');
  const [gameplay, setGameplay] = useState(0);
  const [art, setArt] = useState(0);
  const [story, setStory] = useState(0);
  const [soundtrack, setSoundtrack] = useState(0);
  const [fun, setFun] = useState(0);
  const [comment, setComment] = useState('');
  const [reviewError, setReviewError] = useState('');
  const [reviewSuccess, setReviewSuccess] = useState('');
  const [inspectedReview, setInspectedReview] = useState<Review | null>(null);
  const [isMatrixModalOpen, setIsMatrixModalOpen] = useState(false);
  const [activePickerId, setActivePickerId] = useState<number | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [recentEmojis, setRecentEmojis] = useState<string[]>(() => {
    try {
      const saved = localStorage.getItem('recent_emojis');
      return saved ? JSON.parse(saved) : [];
    } catch {
      return [];
    }
  });

  // Fetch games list
  const loadGames = async () => {
    try {
      setLoadingList(true);
      const data = await apiFetch('/games');
      const gamesList = data || [];
      setGames(gamesList);
      
      // Filter list for auto-selection to match what's visible on screen
      const visibleGames = gamesList.filter(
        (g: Game) => g.is_active || g.last_active_date !== null
      );

      // Auto-select active game first, or fallback to first visible game
      if (visibleGames.length > 0 && selectedGameId === null) {
        const active = visibleGames.find((g: Game) => g.is_active);
        if (active) {
          setSelectedGameId(active.id);
        } else {
          // Sort by last active desc
          const sorted = [...visibleGames].sort((a, b) => {
            const dateA = a.last_active_date ? new Date(a.last_active_date).getTime() : 0;
            const dateB = b.last_active_date ? new Date(b.last_active_date).getTime() : 0;
            return dateB - dateA;
          });
          setSelectedGameId(sorted[0].id);
        }
      } else if (visibleGames.length === 0) {
        setSelectedGameId(null);
      }
    } catch (err) {
      console.error('Failed to load games list:', err);
    } finally {
      setLoadingList(false);
    }
  };

  useEffect(() => {
    loadGames();
  }, []);

  useEffect(() => {
    const handleGlobalClick = () => {
      setActivePickerId(null);
    };
    document.addEventListener('click', handleGlobalClick);
    return () => {
      document.removeEventListener('click', handleGlobalClick);
    };
  }, []);

  // Fetch selected game details & reviews
  useEffect(() => {
    if (selectedGameId === null) return;

    const loadDetails = async () => {
      try {
        setLoadingDetails(true);
        setReviewError('');
        setReviewSuccess('');
        
        const data = await apiFetch(`/games/${selectedGameId}`);
        setDetails(data);

        // Pre-populate user's existing review if any exists
        const userReview = data.reviews?.find((r: Review) => r.user_id === user?.id);
        if (userReview) {
          setReviewTitle(userReview.title || '');
          setGameplay(userReview.gameplay);
          setArt(userReview.art);
          setStory(userReview.story);
          setSoundtrack(userReview.soundtrack);
          setFun(userReview.fun);
          setComment(userReview.comment || '');
        } else {
          // Reset form
          setReviewTitle('');
          setGameplay(0);
          setArt(0);
          setStory(0);
          setSoundtrack(0);
          setFun(0);
          setComment('');
        }
      } catch (err) {
        console.error('Failed to load game details:', err);
      } finally {
        setLoadingDetails(false);
      }
    };

    loadDetails();
  }, [selectedGameId]);

  // Handle review submission
  const handleReviewSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (selectedGameId === null) return;
    setReviewError('');
    setReviewSuccess('');

    try {
      await apiFetch(`/games/${selectedGameId}/reviews`, {
        method: 'POST',
        body: JSON.stringify({
          title: reviewTitle,
          avatar_url: user?.avatar_url || '',
          gameplay,
          art,
          story,
          soundtrack,
          fun,
          comment,
        }),
      });

      setReviewSuccess(t('games.reviewSaved'));
      // Reload details to recalculate averages
      const updatedDetails = await apiFetch(`/games/${selectedGameId}`);
      setDetails(updatedDetails);
    } catch (err: any) {
      setReviewError(err.message || 'Failed to submit review');
    }
  };

  const handleDeleteReview = async () => {
    if (!inspectedReview || selectedGameId === null) return;
    if (!window.confirm(t('games.confirmDeleteReview'))) return;

    try {
      let url = `/games/${selectedGameId}/reviews`;
      if (user?.role === 'admin' && inspectedReview.user_id !== user.id) {
        url += `?user_id=${inspectedReview.user_id}`;
      }

      await apiFetch(url, { method: 'DELETE' });
      setReviewSuccess(t('games.reviewDeleted'));
      setInspectedReview(null);
      // Reload
      const updatedDetails = await apiFetch(`/games/${selectedGameId}`);
      setDetails(updatedDetails);
    } catch (err: any) {
      setReviewError(err.message || 'Failed to delete review');
    }
  };

  const updateRecentEmoji = (emoji: string) => {
    setRecentEmojis((prev) => {
      const updated = [emoji, ...prev.filter((e) => e !== emoji)].slice(0, 10);
      localStorage.setItem('recent_emojis', JSON.stringify(updated));
      return updated;
    });
  };

  const handleReactionToggle = async (reviewId: number, emoji: string) => {
    if (selectedGameId === null) return;
    try {
      await apiFetch(`/reviews/${reviewId}/react`, {
        method: 'POST',
        body: JSON.stringify({ emoji }),
      });
      updateRecentEmoji(emoji);
      // Re-fetch to sync
      const updatedDetails = await apiFetch(`/games/${selectedGameId}`);
      setDetails(updatedDetails);
    } catch (err: any) {
      console.error('Failed to toggle reaction:', err);
    }
  };


  const handleDeleteGame = async () => {
    if (selectedGameId === null || !details) return;
    if (!window.confirm(t('admin.confirmDeleteGame', { name: details.game.name }))) return;

    try {
      await apiFetch(`/admin/games/${selectedGameId}`, { method: 'DELETE' });
      setReviewSuccess(t('admin.gameDeleted'));
      setSelectedGameId(null);
      setDetails(null);
      loadGames();
    } catch (err: any) {
      setReviewError(err.message || 'Failed to delete game');
    }
  };

  // Sorting & Pinning list calculation
  const getSortedGamesList = () => {
    // Filter games: only show active game or games that were active at some point
    let filtered = games.filter(
      (g) => g.is_active || g.last_active_date !== null
    );

    // Apply name search filtering locally in the browser
    if (searchQueryList.trim() !== '') {
      const q = searchQueryList.toLowerCase();
      filtered = filtered.filter((g) => g.name.toLowerCase().includes(q));
    }

    const activeGame = filtered.find((g) => g.is_active);
    const restGames = filtered.filter((g) => !g.is_active);

    // Sort rest games based on sortBy criteria and sortOrder direction
    restGames.sort((a, b) => {
      let comparison = 0;
      if (sortBy === 'name') {
        comparison = a.name.localeCompare(b.name);
      } else if (sortBy === 'release_date') {
        const dA = a.release_date ? new Date(a.release_date).getTime() : 0;
        const dB = b.release_date ? new Date(b.release_date).getTime() : 0;
        comparison = dA - dB; // default asc (older first)
      } else if (sortBy === 'score') {
        comparison = a.average_score - b.average_score; // default asc (lower score first)
      } else {
        // last_active
        const dA = a.last_active_date ? new Date(a.last_active_date).getTime() : 0;
        const dB = b.last_active_date ? new Date(b.last_active_date).getTime() : 0;
        comparison = dA - dB; // default asc (older last active first)
      }

      // Handle directions:
      // name: default is asc (A-Z). If sortOrder is "desc", reverse.
      // release_date, score, last_active: default is desc (highest/newest first). If sortOrder is "asc", reverse.
      const defaultDesc = sortBy !== 'name';
      if (sortOrder === 'asc') {
        return defaultDesc ? comparison : -comparison;
      } else {
        return defaultDesc ? -comparison : comparison;
      }
    });

    return activeGame ? [activeGame, ...restGames] : restGames;
  };

  const sortedGames = getSortedGamesList();

  // Helper to render interactive rating stars (0-5 stars)
  const RatingInput: React.FC<{
    label: string;
    value: number;
    onChange: (val: number) => void;
  }> = ({ label, value, onChange }) => (
    <div className="flex items-center justify-between">
      <span className="text-sm font-semibold">{label}</span>
      <div className="flex items-center space-x-1">
        {[1, 2, 3, 4, 5].map((star) => (
          <button
            key={star}
            type="button"
            onClick={() => onChange(value === star ? 0 : star)} // Toggle score
            className={`p-1 transition-transform active:scale-90 ${
              star <= value ? 'text-amber-400' : 'text-slate-200 dark:text-slate-800'
            }`}
          >
            <Star className="w-5 h-5 fill-current" />
          </button>
        ))}
      </div>
    </div>
  );

  const AverageBadge: React.FC<{ label: string; score: number }> = ({ label, score }) => (
    <div 
      onClick={() => setIsMatrixModalOpen(true)}
      className="bg-slate-50 dark:bg-slate-900 border border-slate-100 dark:border-slate-800 rounded-xl p-3 text-center cursor-pointer hover:border-indigo-500 dark:hover:border-indigo-500 active:scale-95 transition-all"
    >
      <span className="text-[11px] font-bold text-slate-400 uppercase">{label}</span>
      <div className="flex items-center justify-center space-x-1 mt-1 text-indigo-600 dark:text-indigo-400 font-extrabold text-lg">
        {score > 0 ? (
          <>
            <Star className="w-4 h-4 fill-current text-amber-400" />
            <span>{score.toFixed(1)}</span>
          </>
        ) : (
          <span className="text-slate-400 text-xs font-normal">{t('games.unrated')}</span>
        )}
      </div>
    </div>
  );

  return (
    <div className="flex flex-col lg:flex-row h-[calc(100vh-64px)] md:h-screen overflow-hidden">
      
      {/* Column 1: Games List (remains visible) */}
      <div className={`w-full lg:w-96 border-r border-slate-200 dark:border-slate-800 flex flex-col h-full bg-slate-50/50 dark:bg-slate-950/20 transition-colors ${selectedGameId !== null ? 'hidden lg:flex' : 'flex'}`}>
        
        {/* List Header and Sorting Controls */}
        <div className="p-4 border-b border-slate-200 dark:border-slate-800 space-y-3">
          <div className="flex items-center justify-between">
            <h2 className="text-xl font-black">{t('nav.games')}</h2>
            <button
              onClick={loadGames}
              className="text-xs font-bold text-indigo-600 hover:underline dark:text-indigo-400"
            >
              {t('games.refresh')}
            </button>
          </div>

          {/* Local Game Name Search Bar */}
          <div className="relative flex items-center">
            <input
              type="text"
              value={searchQueryList}
              onChange={(e) => setSearchQueryList(e.target.value)}
              placeholder={t('games.searchGamesPlaceholder')}
              className="w-full pl-9 pr-4 py-1.5 rounded-xl border border-slate-200 dark:border-slate-850 bg-white dark:bg-slate-900 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 outline-none text-xs transition"
            />
            <span className="absolute left-3 text-slate-400">
              <Search className="w-3.5 h-3.5" />
            </span>
            {searchQueryList && (
              <button
                onClick={() => setSearchQueryList('')}
                className="absolute right-3 text-slate-400 hover:text-slate-650 text-xs"
              >
                <X className="w-3 h-3" />
              </button>
            )}
          </div>

          {/* Sorting controls with direction toggle */}
          <div className="flex items-center justify-between space-x-2">
            <div className="flex-1 flex items-center justify-between bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl px-3 py-1.5 shadow-sm text-xs">
              <div className="flex items-center space-x-1 text-slate-400 font-bold">
                <ArrowUpDown className="w-3.5 h-3.5" />
                <span>{t('games.sortBy')}:</span>
              </div>
              <select
                value={sortBy}
                onChange={(e) => setSortBy(e.target.value as any)}
                className="bg-transparent border-none text-xs font-bold p-0 pl-1 focus:ring-0 cursor-pointer"
              >
                <option value="last_active">{t('games.sortActive')}</option>
                <option value="name">{t('games.sortName')}</option>
                <option value="release_date">{t('games.sortRelease')}</option>
                <option value="score">{t('games.averageScore')}</option>
              </select>
            </div>

            <button
              onClick={() => setSortOrder(p => p === 'asc' ? 'desc' : 'asc')}
              className="p-2 border border-slate-200 dark:border-slate-800 rounded-xl bg-white dark:bg-slate-900 hover:bg-slate-50 dark:hover:bg-slate-800 shadow-sm transition active:scale-90"
              title={sortOrder === 'desc' ? 'Sort Descending' : 'Sort Ascending'}
            >
              {sortOrder === 'desc' ? (
                <ArrowDown className="w-4 h-4 text-indigo-500" />
              ) : (
                <ArrowUp className="w-4 h-4 text-indigo-500" />
              )}
            </button>
          </div>
        </div>

        {/* Scrollable list */}
        <div className="flex-1 overflow-y-auto divide-y divide-slate-100 dark:divide-slate-850">
          {loadingList ? (
            <div className="p-8 text-center text-slate-400 animate-pulse font-bold text-sm">{t('games.loadingGames')}</div>
          ) : sortedGames.length === 0 ? (
            <div className="p-8 text-center text-slate-400 font-bold text-sm">{t('games.noGames')}</div>
          ) : (
            sortedGames.map((g) => (
              <button
                key={g.id}
                onClick={() => setSelectedGameId(g.id)}
                className={`w-full text-left p-4 flex items-start space-x-3 transition-colors ${
                  selectedGameId === g.id
                    ? 'bg-indigo-50/70 dark:bg-indigo-950/20 border-r-2 border-indigo-500'
                    : 'hover:bg-slate-100 dark:hover:bg-slate-900/40'
                }`}
              >
                <img
                  src={g.cover_url || 'https://images.igdb.com/igdb/image/upload/t_cover_big/co1r3d.jpg'}
                  alt={g.name}
                  className="w-12 h-16 rounded-lg object-cover shadow-sm bg-slate-100 dark:bg-slate-900 border border-slate-200 dark:border-slate-800"
                />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center space-x-1.5">
                    {g.is_active && (
                      <span className="bg-amber-500 text-white text-[9px] font-black tracking-wider px-1.5 py-0.5 rounded flex items-center space-x-0.5">
                        <Bookmark className="w-2.5 h-2.5 fill-current" />
                        <span>{t('games.pin')}</span>
                      </span>
                    )}
                  </div>
                  <h3 className="text-sm font-black text-slate-900 dark:text-white truncate mt-1">{g.name}</h3>
                  {g.average_score > 0 && (
                    <div className="flex items-center text-[10px] font-black text-amber-500 bg-amber-500/10 px-1.5 py-0.5 rounded w-fit mt-1">
                      <Star className="w-3 h-3 fill-current mr-0.5" />
                      <span>{g.average_score.toFixed(1)}</span>
                    </div>
                  )}
                  {g.release_date && (
                    <p className="text-[11px] text-slate-400 dark:text-slate-500 truncate mt-1">
                      {new Date(g.release_date).toLocaleDateString()}
                    </p>
                  )}
                </div>
                <ChevronRight className="w-4 h-4 text-slate-300 self-center" />
              </button>
            ))
          )}
        </div>
      </div>

      {/* Column 2: Selected Game Details */}
      <div className={`flex-1 overflow-y-auto h-full bg-white dark:bg-slate-900 transition-colors ${selectedGameId === null ? 'hidden lg:block' : 'block'}`}>
        {loadingDetails ? (
          <div className="flex items-center justify-center h-full text-slate-400 font-bold text-sm animate-pulse">
            {t('games.loadingDetails')}
          </div>
        ) : !details ? (
          <div className="flex flex-col items-center justify-center h-full text-slate-400 p-8 text-center space-y-2">
            <Gamepad2 className="w-12 h-12 text-slate-300" />
            <h3 className="font-bold text-sm">{t('games.noGameSelected')}</h3>
            <p className="text-xs">{t('games.clickGameSidebar')}</p>
          </div>
        ) : (
          <div className="p-4 md:p-8 space-y-8">
            
            {/* Mobile Back Button */}
            <button
              onClick={() => setSelectedGameId(null)}
              className="lg:hidden flex items-center space-x-1 text-xs font-black text-indigo-600 dark:text-indigo-400 hover:underline mb-2 transition active:scale-95"
            >
              <ChevronLeft className="w-4 h-4 stroke-[3]" />
              <span>Back to Games List</span>
            </button>

            {/* Game metadata block */}
            <div className="flex flex-col md:flex-row items-start md:space-x-6 space-y-4 md:space-y-0 pb-6 border-b border-slate-100 dark:border-slate-800 relative">
              {user?.role === 'admin' && (
                <button
                  onClick={handleDeleteGame}
                  className="absolute top-0 right-0 p-2 text-red-500 hover:text-red-700 hover:bg-red-50 dark:hover:bg-red-950/20 rounded-xl transition active:scale-90"
                  title="Delete Game"
                >
                  <Trash2 className="w-5 h-5" />
                </button>
              )}
              <img
                src={details.game.cover_url || 'https://images.igdb.com/igdb/image/upload/t_cover_big/co1r3d.jpg'}
                alt={details.game.name}
                className="w-32 h-44 rounded-2xl object-cover shadow-lg border border-slate-200 dark:border-slate-800 bg-slate-100 dark:bg-slate-800"
              />
              <div className="flex-1 space-y-3">
                <div className="flex items-center space-x-2">
                  {details.game.is_active && (
                    <span className="bg-amber-500 text-white text-[9px] font-black px-2 py-0.5 rounded">
                      ★ {t('games.active')}
                    </span>
                  )}
                </div>
                <h1 className="text-3xl font-black tracking-tight leading-tight">{details.game.name}</h1>
                <p className="text-sm text-slate-600 dark:text-slate-400 leading-relaxed max-w-2xl">{details.game.summary}</p>
                
                {/* Metadata Row: Release Date + Averages on the same line */}
                <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-slate-400 font-bold pt-1">
                  {details.game.release_date && (
                    <span>
                      {t('games.released')}: <strong className="text-slate-600 dark:text-slate-300">{new Date(details.game.release_date).toLocaleDateString()}</strong>
                    </span>
                  )}

                  {(details.game.time_to_beat_hastily || details.game.time_to_beat_normal || details.game.time_to_beat_completely) && (
                    <div className="flex items-center space-x-1 text-slate-500 dark:text-slate-400 flex-wrap">
                      {details.game.release_date && <span className="text-slate-300 dark:text-slate-700 mr-2">•</span>}
                      {(() => {
                        const formatHours = (val: string) => {
                          if (!val) return '';
                          const num = val.replace(' hours', '').trim();
                          return t('games.hours', { count: num });
                        };

                        return (
                          <>
                            <span className="flex items-center">
                              <span className="mr-1.5">🕒</span>
                              {t('games.timeToBeat')}: <strong className="text-indigo-600 dark:text-indigo-400 ml-1">{formatHours(details.game.time_to_beat_hastily || details.game.time_to_beat)}</strong>
                            </span>
                            {details.game.time_to_beat_normal && (
                              <span className="flex items-center">
                                <span className="mx-1.5 text-slate-300 dark:text-slate-700">|</span>
                                {t('games.mainSides')}: <strong className="text-indigo-600 dark:text-indigo-400 ml-1">{formatHours(details.game.time_to_beat_normal)}</strong>
                              </span>
                            )}
                            {details.game.time_to_beat_completely && (
                              <span className="flex items-center">
                                <span className="mx-1.5 text-slate-300 dark:text-slate-700">|</span>
                                {t('games.completionist')}: <strong className="text-indigo-600 dark:text-indigo-400 ml-1">{formatHours(details.game.time_to_beat_completely)}</strong>
                              </span>
                            )}
                          </>
                        );
                      })()}
                    </div>
                  )}
                </div>
              </div>
            </div>

            {/* Averages block */}
            <div className="space-y-3">
              <h2 className="text-lg font-bold flex items-center space-x-1.5">
                <Award className="w-5 h-5 text-indigo-600" />
                <span>{t('games.averageScore')}</span>
              </h2>
              
              <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-6 gap-3">
                <div 
                  onClick={() => setIsMatrixModalOpen(true)}
                  className="bg-indigo-500/10 dark:bg-indigo-950/30 border-2 border-indigo-500/20 rounded-xl p-3 text-center flex flex-col justify-center cursor-pointer hover:border-indigo-500 active:scale-95 transition-all"
                >
                  <span className="text-[10px] font-black text-indigo-600 dark:text-indigo-400 uppercase tracking-wider">{t('games.overall')}</span>
                  <div className="flex items-center justify-center space-x-1 mt-1 text-indigo-600 dark:text-indigo-400 font-black text-2xl">
                    {details.averages && details.averages.overall > 0 ? (
                      <>
                        <Star className="w-5 h-5 fill-current text-amber-400" />
                        <span>{details.averages.overall.toFixed(1)}</span>
                      </>
                    ) : (
                      <span className="text-slate-400 text-xs font-normal">{t('games.unrated')}</span>
                    )}
                  </div>
                </div>

                <AverageBadge label={t('games.gameplay')} score={details.averages?.gameplay || 0} />
                <AverageBadge label={t('games.art')} score={details.averages?.art || 0} />
                <AverageBadge label={t('games.story')} score={details.averages?.story || 0} />
                <AverageBadge label={t('games.soundtrack')} score={details.averages?.soundtrack || 0} />
                <AverageBadge label={t('games.fun')} score={details.averages?.fun || 0} />
              </div>
            </div>

            <div className="grid grid-cols-1 xl:grid-cols-3 gap-8">
              
              {/* Left hand: Submit/Update Review form */}
              <div className="xl:col-span-1 bg-slate-50 dark:bg-slate-900 border border-slate-100 dark:border-slate-800 rounded-2xl p-6 h-fit space-y-4">
                <h3 className="text-base font-bold pb-2 border-b border-slate-200 dark:border-slate-800">
                  {details.reviews?.some((r) => r.user_id === user?.id)
                    ? t('games.updateReview')
                    : t('games.writeReview')}
                </h3>

                {reviewSuccess && (
                  <div className="bg-emerald-50 dark:bg-emerald-950/20 border-l-4 border-emerald-500 text-emerald-700 dark:text-emerald-400 p-3 rounded text-xs font-semibold">
                    {reviewSuccess}
                  </div>
                )}

                {reviewError && (
                  <div className="bg-red-50 dark:bg-red-950/20 border-l-4 border-red-500 text-red-700 dark:text-red-400 p-3 rounded text-xs font-semibold">
                    {reviewError}
                  </div>
                )}

                <form onSubmit={handleReviewSubmit} className="space-y-4">
                  <div className="space-y-1">
                    <input
                      type="text"
                      value={reviewTitle}
                      onChange={(e) => setReviewTitle(e.target.value)}
                      placeholder={t('games.reviewTitlePlaceholder')}
                      className="w-full px-3 py-2 rounded-xl border border-slate-300 dark:border-slate-700 bg-transparent text-xs focus:ring-1 focus:ring-indigo-500 outline-none dark:bg-slate-950"
                    />
                  </div>

                  <div className="space-y-2.5">
                    <RatingInput label={t('games.gameplay')} value={gameplay} onChange={setGameplay} />
                    <RatingInput label={t('games.art')} value={art} onChange={setArt} />
                    <RatingInput label={t('games.story')} value={story} onChange={setStory} />
                    <RatingInput label={t('games.soundtrack')} value={soundtrack} onChange={setSoundtrack} />
                    <RatingInput label={t('games.fun')} value={fun} onChange={setFun} />
                  </div>

                  <div className="space-y-1">
                    <textarea
                      value={comment}
                      onChange={(e) => setComment(e.target.value)}
                      placeholder={t('games.commentPlaceholder')}
                      className="w-full h-24 p-3 rounded-xl border border-slate-300 dark:border-slate-700 bg-transparent text-xs focus:ring-1 focus:ring-indigo-500 outline-none dark:bg-slate-950"
                    />
                  </div>

                  <button
                    type="submit"
                    className="w-full py-2 bg-indigo-600 text-white font-bold text-xs rounded-xl hover:bg-indigo-700 transition shadow"
                  >
                    {t('games.submit')}
                  </button>
                </form>
              </div>

              {/* Right hand: Individual Reviews List */}
              <div className="xl:col-span-2 space-y-4">
                <h3 className="text-base font-bold flex items-center space-x-1.5 pb-2 border-b border-slate-100 dark:border-slate-800">
                  <MessageSquare className="w-5 h-5 text-indigo-600" />
                  <span>{t('games.reviews')} ({details.reviews?.length || 0})</span>
                </h3>

                {(!details.reviews || details.reviews.length === 0) ? (
                  <p className="text-xs text-slate-400 italic py-6">{t('games.noReviews')}</p>
                ) : (
                  (() => {
                    const leftReviews = details.reviews.filter((_, idx) => idx % 2 === 0);
                    const rightReviews = details.reviews.filter((_, idx) => idx % 2 !== 0);

                    const renderReviewCard = (r: Review) => {
                      const getReviewAverage = (rev: Review) => {
                        let sum = 0;
                        let count = 0;
                        if (rev.gameplay > 0) { sum += rev.gameplay; count++; }
                        if (rev.art > 0) { sum += rev.art; count++; }
                        if (rev.story > 0) { sum += rev.story; count++; }
                        if (rev.soundtrack > 0) { sum += rev.soundtrack; count++; }
                        if (rev.fun > 0) { sum += rev.fun; count++; }
                        return count > 0 ? (sum / count).toFixed(1) : 'unrated';
                      };

                      return (
                        <div 
                          key={r.id} 
                          onClick={() => setInspectedReview(r)}
                          className="border border-slate-100 dark:border-slate-800 rounded-2xl p-5 space-y-3 shadow-sm hover:shadow-md hover:border-indigo-400 dark:hover:border-indigo-400 transition cursor-pointer active:scale-[0.98] transform flex flex-col justify-between group"
                        >
                          <div className="flex items-start justify-between space-x-3">
                            <div className="flex items-center space-x-3 min-w-0">
                              {r.avatar_url ? (
                                <img 
                                  src={r.avatar_url} 
                                  alt={r.username} 
                                  className="w-10 h-10 rounded-full object-cover border border-slate-200 dark:border-slate-800 flex-shrink-0"
                                />
                              ) : (
                                <div className="w-10 h-10 rounded-full bg-indigo-50 dark:bg-indigo-950/40 text-indigo-600 dark:text-indigo-400 flex items-center justify-center font-black text-sm flex-shrink-0">
                                  {r.username[0].toUpperCase()}
                                </div>
                              )}
                              <div className="min-w-0">
                                <span className="text-sm font-black truncate block" title={r.title || `${r.username}'s review`}>
                                  {r.title ? r.title : `${r.username}'s review`}
                                </span>
                                <span className="text-[10px] text-slate-400 font-semibold block">
                                  {t('games.by')} {r.username}
                                </span>
                              </div>
                            </div>
                            <div className="bg-indigo-500/15 dark:bg-indigo-950/40 text-indigo-600 dark:text-indigo-400 text-xs font-black px-2.5 py-1 rounded-lg flex items-center space-x-1 flex-shrink-0">
                              <Star className="w-3.5 h-3.5 fill-current text-amber-400" />
                              <span>{getReviewAverage(r)}</span>
                            </div>
                          </div>
                          {r.comment && (
                            <p className="text-xs text-slate-500 dark:text-slate-400 line-clamp-10 leading-relaxed italic">{r.comment}</p>
                          )}
                          <div className="flex items-center justify-between mt-3 group/picker h-6">
                            <span className="text-[10px] text-slate-400 font-semibold">{new Date(r.created_at).toLocaleDateString()}</span>
                            
                            <div className="relative flex items-center" onClick={(e) => e.stopPropagation()}>
                              
                              {/* Emojis Shortcut Row (Hidden by default, slides out horizontally to the left on hover - hides if search picker is active) */}
                              {activePickerId !== r.id && (
                                <div className="hidden group-hover/picker:flex items-center space-x-1.5 mr-1.5 bg-white dark:bg-slate-900 px-1.5 py-0.5 rounded-lg border border-slate-150 dark:border-slate-800 transition-all duration-200">
                                  {(() => {
                                    const defaultQuick = ['👍', '❤️', '😂', '😮', '😢'];
                                    const recentToShow = recentEmojis.filter((e) => !defaultQuick.includes(e)).slice(0, 2);
                                    const quickRow = [...defaultQuick, ...recentToShow];
                                    
                                    return quickRow.map((emoji) => (
                                      <button
                                        key={emoji}
                                        onClick={() => handleReactionToggle(r.id, emoji)}
                                        className="p-1 hover:bg-slate-100 dark:hover:bg-slate-800 rounded text-sm transition active:scale-90"
                                      >
                                        {emoji}
                                      </button>
                                    ));
                                  })()}
                                </div>
                              )}

                              {/* Smile Trigger Button */}
                              <button
                                onClick={() => {
                                  setActivePickerId(activePickerId === r.id ? null : r.id);
                                  setSearchQuery('');
                                }}
                                className="opacity-0 group-hover:opacity-100 p-0.5 rounded text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 transition focus:opacity-100"
                                title="Search all emojis"
                              >
                                <Smile className="w-3.5 h-3.5" />
                              </button>
                              {/* Full Search Popover */}
                              {activePickerId === r.id && (
                                <div 
                                  className="absolute right-full mr-1.5 -bottom-1.5 z-40 p-2 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl shadow-xl transition-all w-48"
                                >
                                  <div className="flex items-center space-x-1.5 border border-slate-150 dark:border-slate-800 rounded-lg px-2 py-1 mb-2 bg-slate-50 dark:bg-slate-950/40">
                                    <Search className="w-3 h-3 text-slate-400" />
                                    <input
                                      type="text"
                                      placeholder="Search..."
                                      value={searchQuery}
                                      onChange={(e) => setSearchQuery(e.target.value)}
                                      className="w-full bg-transparent border-none text-[10px] focus:ring-0 focus:outline-none text-slate-700 dark:text-slate-300"
                                      autoFocus
                                    />
                                  </div>
                                  <div className="grid grid-cols-6 gap-1 max-h-24 overflow-y-auto">
                                    {(() => {
                                      const defaultQuick = ['👍', '❤️', '😂', '😮', '😢'];
                                      const recentList = recentEmojis.filter((e) => !defaultQuick.includes(e));
                                      const orderedEmojis = [
                                        ...ALL_EMOJIS.filter((x) => defaultQuick.includes(x.char)),
                                        ...ALL_EMOJIS.filter((x) => recentList.includes(x.char)),
                                        ...ALL_EMOJIS.filter((x) => !defaultQuick.includes(x.char) && !recentList.includes(x.char)),
                                      ];
                                      return orderedEmojis.filter((x) =>
                                        searchQuery === '' || x.name.toLowerCase().includes(searchQuery.toLowerCase())
                                      ).map((x) => (
                                        <button
                                          key={x.char}
                                          onClick={() => {
                                            handleReactionToggle(r.id, x.char);
                                            setActivePickerId(null);
                                          }}
                                          className="hover:bg-slate-100 dark:hover:bg-slate-800 rounded text-sm transition"
                                        >
                                          {x.char}
                                        </button>
                                      ));
                                    })()}
                                  </div>
                                </div>
                              )}
                            </div>
                          </div>
                          
                          {/* Emoji Reactions Row (Only visible if reactions exist and count > 0) */}
                          {r.reactions && r.reactions.some((rx) => rx.count > 0) && (
                            <div className="flex flex-wrap gap-1.5 pt-2 border-t border-slate-50 dark:border-slate-800/40" onClick={(e) => e.stopPropagation()}>
                              {r.reactions.filter((rx) => rx.count > 0).map((rx) => (
                                <button
                                  key={rx.emoji}
                                  onClick={() => handleReactionToggle(r.id, rx.emoji)}
                                  className={`flex items-center space-x-1 px-2 py-0.5 rounded-full text-sm font-semibold border transition active:scale-95 ${
                                    rx.user_reacted
                                      ? 'bg-indigo-50 dark:bg-indigo-950/40 border-indigo-300 dark:border-indigo-800 text-indigo-600 dark:text-indigo-400 font-bold'
                                      : 'bg-slate-50 hover:bg-slate-100 dark:bg-slate-950/20 dark:hover:bg-slate-950/40 border-slate-150 dark:border-slate-800 text-slate-500 dark:text-slate-400'
                                  }`}
                                  title={rx.usernames && rx.usernames.length > 0 ? rx.usernames.join(', ') : ''}
                                >
                                  <span>{rx.emoji}</span>
                                  <span>{rx.count}</span>
                                </button>
                              ))}
                            </div>
                          )}
                        </div>
                      );
                    };

                    return (
                      <div className="flex flex-col md:flex-row space-y-4 md:space-y-0 md:space-x-4 items-start">
                        <div className="flex-1 space-y-4 w-full">
                          {leftReviews.map(renderReviewCard)}
                        </div>
                        <div className="flex-1 space-y-4 w-full">
                          {rightReviews.map(renderReviewCard)}
                        </div>
                      </div>
                    );
                  })()
                )}
              </div>

            </div>

          </div>
        )}
      </div>

      {/* --- POPUP 1: INDIVIDUAL REVIEW CARD DETAILS MODAL --- */}
      {inspectedReview && (
        <div className="fixed inset-0 z-50 bg-slate-950/70 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="w-full max-w-lg bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 shadow-2xl p-6 space-y-6 transition-colors">
            
            {/* Modal Header */}
            <div className="flex items-center justify-between pb-4 border-b border-slate-100 dark:border-slate-800">
              <div className="flex items-center space-x-3">
                {inspectedReview.avatar_url ? (
                  <img 
                    src={inspectedReview.avatar_url} 
                    alt={inspectedReview.username} 
                    className="w-12 h-12 rounded-full object-cover border border-slate-200 dark:border-slate-800 flex-shrink-0"
                  />
                ) : (
                  <div className="w-12 h-12 rounded-full bg-indigo-50 dark:bg-indigo-950/40 text-indigo-600 dark:text-indigo-400 flex items-center justify-center font-black text-lg flex-shrink-0">
                    {inspectedReview.username[0].toUpperCase()}
                  </div>
                )}
                <div>
                  <h2 className="text-lg font-black text-slate-900 dark:text-white">
                    {inspectedReview.title ? inspectedReview.title : `${inspectedReview.username}'s review`}
                  </h2>
                  <span className="text-xs text-slate-400">{t('games.by')} {inspectedReview.username} • {new Date(inspectedReview.created_at).toLocaleDateString()}</span>
                </div>
              </div>
              <button 
                onClick={() => setInspectedReview(null)}
                className="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            {/* Scores List */}
            <div className="space-y-3">
              {[
                { label: t('games.gameplay'), score: inspectedReview.gameplay },
                { label: t('games.art'), score: inspectedReview.art },
                { label: t('games.story'), score: inspectedReview.story },
                { label: t('games.soundtrack'), score: inspectedReview.soundtrack },
                { label: t('games.fun'), score: inspectedReview.fun },
              ].map(({ label, score }) => (
                <div key={label} className="flex items-center justify-between text-sm">
                  <span className="font-semibold text-slate-600 dark:text-slate-400">{label}</span>
                  <div className="flex items-center space-x-1">
                    {score > 0 ? (
                      <>
                        {[1, 2, 3, 4, 5].map((s) => (
                          <Star 
                            key={s} 
                            className={`w-4 h-4 ${s <= score ? 'text-amber-400 fill-current' : 'text-slate-200 dark:text-slate-800'}`} 
                          />
                        ))}
                        <span className="text-xs font-bold ml-2 text-slate-700 dark:text-slate-300">({score}/5)</span>
                      </>
                    ) : (
                      <span className="text-xs font-normal text-slate-400 italic">{t('games.unrated')}</span>
                    )}
                  </div>
                </div>
              ))}
            </div>

            {/* Full Comment */}
            {inspectedReview.comment && (
              <div className="pt-4 border-t border-slate-100 dark:border-slate-800 space-y-1.5">
                <h4 className="text-xs font-bold text-slate-400 uppercase tracking-wider">{t('games.comment')}</h4>
                <p className="text-xs text-slate-700 dark:text-slate-300 leading-relaxed whitespace-pre-wrap bg-slate-50 dark:bg-slate-950/40 p-3.5 rounded-xl border border-slate-100 dark:border-slate-850">
                  {inspectedReview.comment}
                </p>
              </div>
            )}

            {/* Delete Review Button (Visible if Author or Admin) */}
            {(user?.id === inspectedReview.user_id || user?.role === 'admin') && (
              <div className="pt-4 border-t border-slate-100 dark:border-slate-800 flex justify-end">
                <button
                  onClick={handleDeleteReview}
                  className="px-4 py-2 bg-red-50 hover:bg-red-100 dark:bg-red-950/20 dark:hover:bg-red-950/40 text-red-600 dark:text-red-400 text-xs font-bold rounded-xl transition active:scale-95"
                >
                  {t('games.deleteReview')}
                </button>
              </div>
            )}

          </div>
        </div>
      )}

      {/* --- POPUP 2: COMMUNITY COMPARISON MATRIX TABLE MODAL --- */}
      {isMatrixModalOpen && details && (
        <div className="fixed inset-0 z-50 bg-slate-950/70 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="w-[80vw] max-w-5xl bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 shadow-2xl p-6 space-y-6 transition-colors flex flex-col max-h-[85vh]">
            
            {/* Header */}
            <div className="flex items-center justify-between pb-4 border-b border-slate-100 dark:border-slate-800 flex-shrink-0">
              <div>
                <h2 className="text-lg font-black text-slate-900 dark:text-white">{t('games.communityMatrix')}</h2>
                <p className="text-xs text-slate-400 mt-1">{t('games.matrixDesc', { name: details.game.name })}</p>
              </div>
              <button 
                onClick={() => setIsMatrixModalOpen(false)}
                className="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            {/* Scrollable Table Container */}
            <div className="flex-1 overflow-x-auto overflow-y-auto border border-slate-150 dark:border-slate-850 rounded-xl bg-slate-50 dark:bg-slate-950/20">
              {(!details.reviews || details.reviews.length === 0) ? (
                <div className="text-center py-12 text-slate-400 font-bold text-sm">
                  {t('games.noReviewsSubmitted')}
                </div>
              ) : (
                <table className="min-w-full divide-y divide-slate-150 dark:divide-slate-800 text-left border-collapse">
                  <thead>
                    <tr className="bg-slate-100 dark:bg-slate-950">
                      <th className="px-4 py-3 text-xs font-bold text-slate-400 uppercase tracking-wider sticky left-0 bg-slate-100 dark:bg-slate-950 z-10 border-r border-slate-200 dark:border-slate-800 min-w-[150px]">
                        {t('games.category')}
                      </th>
                      {details.reviews.map((r) => (
                        <th key={r.id} className="px-4 py-3 text-xs font-black text-slate-700 dark:text-slate-300 border-r border-slate-200 dark:border-slate-800 text-center min-w-[100px] truncate max-w-[150px]" title={r.username}>
                          {r.username}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-150 dark:divide-slate-800 bg-white dark:bg-slate-900 text-sm">
                    
                    {/* Row 1: Gameplay */}
                    <tr className="hover:bg-slate-50 dark:hover:bg-slate-800/10">
                      <td className="px-4 py-3 font-bold text-slate-600 dark:text-slate-400 sticky left-0 bg-white dark:bg-slate-900 border-r border-slate-200 dark:border-slate-800">
                        {t('games.gameplay')}
                      </td>
                      {details.reviews.map((r) => (
                        <td key={r.id} className="px-4 py-3 text-center font-black text-indigo-600 dark:text-indigo-400 border-r border-slate-200 dark:border-slate-800">
                          {r.gameplay > 0 ? r.gameplay : '-'}
                        </td>
                      ))}
                    </tr>

                    {/* Row 2: Art */}
                    <tr className="hover:bg-slate-50 dark:hover:bg-slate-800/10">
                      <td className="px-4 py-3 font-bold text-slate-600 dark:text-slate-400 sticky left-0 bg-white dark:bg-slate-900 border-r border-slate-200 dark:border-slate-800">
                        {t('games.art')}
                      </td>
                      {details.reviews.map((r) => (
                        <td key={r.id} className="px-4 py-3 text-center font-black text-indigo-600 dark:text-indigo-400 border-r border-slate-200 dark:border-slate-800">
                          {r.art > 0 ? r.art : '-'}
                        </td>
                      ))}
                    </tr>

                    {/* Row 3: Story */}
                    <tr className="hover:bg-slate-50 dark:hover:bg-slate-800/10">
                      <td className="px-4 py-3 font-bold text-slate-600 dark:text-slate-400 sticky left-0 bg-white dark:bg-slate-900 border-r border-slate-200 dark:border-slate-800">
                        {t('games.story')}
                      </td>
                      {details.reviews.map((r) => (
                        <td key={r.id} className="px-4 py-3 text-center font-black text-indigo-600 dark:text-indigo-400 border-r border-slate-200 dark:border-slate-800">
                          {r.story > 0 ? r.story : '-'}
                        </td>
                      ))}
                    </tr>

                    {/* Row 4: Soundtrack */}
                    <tr className="hover:bg-slate-50 dark:hover:bg-slate-800/10">
                      <td className="px-4 py-3 font-bold text-slate-600 dark:text-slate-400 sticky left-0 bg-white dark:bg-slate-900 border-r border-slate-200 dark:border-slate-800">
                        {t('games.soundtrack')}
                      </td>
                      {details.reviews.map((r) => (
                        <td key={r.id} className="px-4 py-3 text-center font-black text-indigo-600 dark:text-indigo-400 border-r border-slate-200 dark:border-slate-800">
                          {r.soundtrack > 0 ? r.soundtrack : '-'}
                        </td>
                      ))}
                    </tr>

                    {/* Row 5: Fun */}
                    <tr className="hover:bg-slate-50 dark:hover:bg-slate-800/10">
                      <td className="px-4 py-3 font-bold text-slate-600 dark:text-slate-400 sticky left-0 bg-white dark:bg-slate-900 border-r border-slate-200 dark:border-slate-800">
                        {t('games.fun')}
                      </td>
                      {details.reviews.map((r) => (
                        <td key={r.id} className="px-4 py-3 text-center font-black text-indigo-600 dark:text-indigo-400 border-r border-slate-200 dark:border-slate-800">
                          {r.fun > 0 ? r.fun : '-'}
                        </td>
                      ))}
                    </tr>

                  </tbody>
                </table>
              )}
            </div>

          </div>
        </div>
      )}

    </div>
  );
};
