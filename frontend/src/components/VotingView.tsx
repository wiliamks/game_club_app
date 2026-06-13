import React, { useState, useEffect } from 'react';
import { useAuth } from '../context/AuthContext';
import { useLanguage } from '../context/LanguageContext';
import { 
  Search, 
  ArrowUp, 
  ArrowDown, 
  Award, 
  Check, 
  TrendingUp, 
  Hourglass,
  X,
  GripVertical
} from 'lucide-react';

interface Session {
  id: number;
  name: string;
  max_nominations: number;
  phase: 'nomination' | 'voting' | 'closed';
}

interface Nomination {
  id: number;
  session_id: number;
  user_id: number;
  game_id: number;
  name: string;
  cover_url: string;
  summary: string;
}

interface IGDBGame {
  id: number;
  name: string;
  summary: string;
  cover_url: string;
  time_to_beat: string;
  release_date: string | null;
}

interface CandidateResult {
  rank: number;
  game_id: number;
  name: string;
  cover_url: string;
  points: number;
}

export const VotingView: React.FC = () => {
  const { apiFetch } = useAuth();
  const { t } = useLanguage();

  const [session, setSession] = useState<Session | null>(null);
  const [nominations, setNominations] = useState<Nomination[]>([]);
  const [myNominations, setMyNominations] = useState<Nomination[]>([]);
  const [loading, setLoading] = useState(true);

  // Search states (Nomination Phase)
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState<IGDBGame[]>([]);
  const [searching, setSearching] = useState(false);
  const [isSearchModalOpen, setIsSearchModalOpen] = useState(false);
  const [currentPage, setCurrentPage] = useState(1);
  const pageSize = 7;

  // Voting preferences state (Voting Phase)
  const [preferences, setPreferences] = useState<Nomination[]>([]);
  const [votedSuccess, setVotedSuccess] = useState(false);
  const [submittingVote, setSubmittingVote] = useState(false);
  const [draggedIndex, setDraggedIndex] = useState<number | null>(null);

  // Results state (Closed Phase)
  const [results, setResults] = useState<CandidateResult[]>([]);
  const [sessions, setSessions] = useState<Session[]>([]);

  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  const loadSessionsList = async () => {
    try {
      setLoading(true);
      setError('');
      setSuccess('');
      
      let allSess = null;
      try {
        allSess = await apiFetch('/voting/sessions');
      } catch (e) {
        console.warn('New sessions API not available yet, falling back to active session:', e);
      }

      if (allSess && allSess.length > 0) {
        setSessions(allSess);
        await loadSessionDetails(allSess[0]);
      } else {
        // Fallback to active/latest session
        const sess = await apiFetch('/voting/session');
        if (sess) {
          setSession(sess);
          setSessions([sess]);
          await loadSessionDetails(sess);
        } else {
          setSession(null);
          setLoading(false);
        }
      }
    } catch (err: any) {
      setError(err.message || 'Failed to load voting cycles');
      setLoading(false);
    }
  };

  const loadSessionDetails = async (sess: Session) => {
    try {
      setLoading(true);
      setSession(sess);
      setVotedSuccess(false);

      // Load all nominations for this specific session ID
      const noms = await apiFetch(`/voting/nominations?session_id=${sess.id}`);
      setNominations(noms || []);

      if (sess.phase === 'nomination') {
        // Load my nominations
        const myNoms = await apiFetch(`/voting/nominations/me?session_id=${sess.id}`);
        setMyNominations(myNoms || []);
      } else if (sess.phase === 'voting') {
        // Initialize preferences with nominations list
        setPreferences(noms || []);
        
        // Check if already voted
        const existingVote = await apiFetch(`/voting/vote/me?session_id=${sess.id}`);
        if (existingVote) {
          setVotedSuccess(true);
          try {
            const prefIds: number[] = JSON.parse(existingVote.preference);
            const ordered = prefIds.map(id => noms.find((n: Nomination) => n.game_id === id)).filter(Boolean) as Nomination[];
            const remainder = noms.filter((n: Nomination) => !prefIds.includes(n.game_id));
            setPreferences([...ordered, ...remainder]);
          } catch (err) {
            console.error('Failed to parse existing vote preferences:', err);
          }
        }
      } else if (sess.phase === 'closed') {
        // Load Borda Count results
        const resultsData = await apiFetch(`/voting/results?session_id=${sess.id}`);
        setResults(resultsData || []);
      }
    } catch (err: any) {
      setError(err.message || 'Failed to load details for this voting cycle');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadSessionsList();
  }, []);

  // Handle live proxy search
  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!searchQuery.trim()) return;

    try {
      setSearching(true);
      setError('');
      setCurrentPage(1);
      const data = await apiFetch(`/igdb/search?q=${encodeURIComponent(searchQuery)}`);
      setSearchResults(data || []);
    } catch (err: any) {
      setError(err.message || 'IGDB search failed');
    } finally {
      setSearching(false);
    }
  };

  // Nominate game
  const handleNominate = async (game: IGDBGame) => {
    setError('');
    setSuccess('');

    try {
      await apiFetch('/voting/nominations', {
        method: 'POST',
        body: JSON.stringify({ game_id: game.id }),
      });

      setSuccess(`Successfully nominated "${game.name}"!`);
      // Reload nominations
      if (session) {
        loadSessionDetails(session);
      }
      // Remove from search results to prevent re-nominations
      setSearchResults((prev) => prev.filter((g) => g.id !== game.id));
    } catch (err: any) {
      setError(err.message || 'Failed to nominate game');
    }
  };

  // Preferential Vote ordering controls (Up/Down)
  const moveUp = (index: number) => {
    if (index === 0) return;
    const copy = [...preferences];
    const temp = copy[index - 1];
    copy[index - 1] = copy[index];
    copy[index] = temp;
    setPreferences(copy);
  };

  const moveDown = (index: number) => {
    if (index === preferences.length - 1) return;
    const copy = [...preferences];
    const temp = copy[index + 1];
    copy[index + 1] = copy[index];
    copy[index] = temp;
    setPreferences(copy);
  };

  // Native HTML5 Drag and Drop handlers for smooth list reordering
  const handleDragStart = (e: React.DragEvent, index: number) => {
    setDraggedIndex(index);
    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData('text/plain', index.toString());
  };

  const handleDragOver = (e: React.DragEvent, index: number) => {
    e.preventDefault();
    if (draggedIndex === null || draggedIndex === index) return;

    // Shift items dynamically as they are dragged over
    const copy = [...preferences];
    const itemToMove = copy[draggedIndex];
    copy.splice(draggedIndex, 1);
    copy.splice(index, 0, itemToMove);
    setDraggedIndex(index);
    setPreferences(copy);
  };

  const handleDragEnd = () => {
    setDraggedIndex(null);
  };

  // Submit ranked-choice vote
  const handleSubmitVote = async () => {
    setSubmittingVote(true);
    setError('');

    try {
      const prefIds = preferences.map((p) => p.game_id);
      await apiFetch('/voting/vote', {
        method: 'POST',
        body: JSON.stringify({ preference: prefIds }),
      });

      setVotedSuccess(true);
      setSuccess(t('voting.voteRecorded'));
    } catch (err: any) {
      setError(err.message || 'Failed to submit vote');
    } finally {
      setSubmittingVote(false);
    }
  };

  if (loading) {
    return (
      <div className="p-8 text-center text-slate-400 font-bold text-sm animate-pulse">
        {t('voting.syncing')}
      </div>
    );
  }

  if (!session) {
    return (
      <div className="p-8 text-center max-w-lg mx-auto py-16 space-y-4">
        {error && (
          <div className="bg-red-50 dark:bg-red-950/20 border-l-4 border-red-500 text-red-700 dark:text-red-400 p-4 rounded-r-xl text-sm font-semibold mb-4 text-left">
            {error}
          </div>
        )}
        <Hourglass className="w-16 h-16 mx-auto text-slate-300 dark:text-slate-700 animate-spin" />
        <h1 className="text-2xl font-black">{t('voting.noEvent')}</h1>
        <p className="text-slate-400 text-sm">{t('voting.nominationInstruction')}</p>
      </div>
    );
  }

  return (
    <div className="p-8 max-w-4xl mx-auto space-y-8 pb-16">
      
      {/* Voting Session History Selector */}
      {sessions.length > 1 && (
        <div className="flex items-center space-x-2 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl px-4 py-2 shadow-sm w-fit">
          <span className="text-xs font-bold text-slate-400">{t('voting.selectCycle')}</span>
          <select
            value={session.id}
            onChange={(e) => {
              const selected = sessions.find((s) => s.id === Number(e.target.value));
              if (selected) {
                loadSessionDetails(selected);
              }
            }}
            className="bg-transparent border-none text-xs font-extrabold focus:ring-0 cursor-pointer p-0 pl-1 outline-none dark:bg-slate-900"
          >
            {sessions.map((s) => (
              <option key={s.id} value={s.id}>
                {s.name} ({t('voting.' + s.phase)})
              </option>
            ))}
          </select>
        </div>
      )}

      {/* Session Title Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between border-b border-slate-100 dark:border-slate-800 pb-6 gap-4">
        <div>
          <h1 className="text-3xl font-black tracking-tight">{session.name}</h1>
          <p className="text-slate-500 mt-1 capitalize">
            {session.phase === 'nomination' && t('voting.nominationPhase')}
            {session.phase === 'voting' && t('voting.votingPhase')}
            {session.phase === 'closed' && t('voting.closedPhase')}
          </p>
        </div>

        {/* Phase Indicator Badge */}
        <span className={`px-4 py-1.5 rounded-full text-xs font-black capitalize tracking-wider ${
          session.phase === 'nomination' ? 'bg-amber-500/10 text-amber-500 border border-amber-500/20' :
          session.phase === 'voting' ? 'bg-indigo-500/10 text-indigo-500 border border-indigo-500/20' :
          'bg-emerald-500/10 text-emerald-500 border border-emerald-500/20'
        }`}>
          {t('voting.' + session.phase)}
        </span>
      </div>

      {/* Notifications */}
      {success && (
        <div className="bg-emerald-50 dark:bg-emerald-950/20 border-l-4 border-emerald-500 text-emerald-700 dark:text-emerald-400 p-4 rounded-r-xl text-sm font-semibold">
          {success}
        </div>
      )}

      {error && (
        <div className="bg-red-50 dark:bg-red-950/20 border-l-4 border-red-500 text-red-700 dark:text-red-400 p-4 rounded-r-xl text-sm font-semibold">
          {error}
        </div>
      )}

      {/* --- PHASE 1: NOMINATION PHASE --- */}
      {session.phase === 'nomination' && (
        <div className="grid grid-cols-1 lg:grid-cols-5 gap-8">
          
          {/* Nominate Search Trigger (Left, 3cols) */}
          <div className="lg:col-span-3 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl p-6 flex flex-col items-center justify-center text-center space-y-4 min-h-[350px]">
            <div className="w-16 h-16 rounded-full bg-indigo-50 dark:bg-indigo-950/30 text-indigo-600 dark:text-indigo-400 flex items-center justify-center">
              <Search className="w-8 h-8" />
            </div>
            <div className="space-y-1">
              <h3 className="text-lg font-bold">{t('voting.nominateTitle')}</h3>
              <p className="text-xs text-slate-500 max-w-sm">{t('voting.nominateDesc')}</p>
            </div>
            <button
              onClick={() => {
                setIsSearchModalOpen(true);
                setSearchQuery('');
                setSearchResults([]);
                setCurrentPage(1);
              }}
              disabled={myNominations.length >= session.max_nominations}
              className="px-6 py-2.5 bg-indigo-600 text-white rounded-xl font-bold text-sm shadow hover:bg-indigo-700 transition disabled:opacity-50"
            >
              {myNominations.length >= session.max_nominations ? t('voting.limitReached') : t('voting.openPortal')}
            </button>
          </div>

          {/* Current user's nominations (Right, 2cols) */}
          <div className="lg:col-span-2 space-y-6">
            <div className="bg-slate-100/50 dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl p-6">
              <h3 className="text-sm font-black text-slate-400 uppercase tracking-wider mb-4">
                {t('voting.nominationLimit', { count: myNominations.length, max: session.max_nominations })}
              </h3>
              
              <div className="space-y-3">
                {myNominations.map((nom) => (
                  <div key={nom.id} className="flex items-center space-x-3 bg-white dark:bg-slate-950 p-3 rounded-xl shadow-sm border border-slate-100 dark:border-slate-900">
                    <img
                      src={nom.cover_url || 'https://images.igdb.com/igdb/image/upload/t_cover_big/co1r3d.jpg'}
                      alt={nom.name}
                      className="w-8 h-11 rounded object-cover flex-shrink-0"
                    />
                    <span className="text-xs font-bold truncate flex-1">{nom.name}</span>
                  </div>
                ))}
                {myNominations.length === 0 && (
                  <p className="text-xs text-slate-400 italic text-center py-6">{t('voting.noNomsYet')}</p>
                )}
              </div>
            </div>

            {/* Total community candidates */}
            <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl p-6">
              <h3 className="text-sm font-bold mb-4">{t('voting.communityNoms')} ({nominations.length})</h3>
              <div className="grid grid-cols-2 gap-3 max-h-60 overflow-y-auto">
                {nominations.map((n) => (
                  <div key={n.id} className="flex items-center space-x-2 border border-slate-50 dark:border-slate-850 p-2 rounded-lg bg-slate-50/50 dark:bg-slate-950">
                    <img
                      src={n.cover_url || 'https://images.igdb.com/igdb/image/upload/t_cover_big/co1r3d.jpg'}
                      alt={n.name}
                      className="w-6 h-8 rounded object-cover flex-shrink-0"
                    />
                    <span className="text-[10px] font-bold truncate leading-tight">{n.name}</span>
                  </div>
                ))}
              </div>
            </div>
          </div>

        </div>
      )}

      {/* --- PHASE 2: PREFERENTIAL VOTING PHASE --- */}
      {session.phase === 'voting' && (
        <div className="max-w-2xl mx-auto bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl p-6 space-y-6">
          <div className="flex items-center space-x-2 pb-4 border-b border-slate-100 dark:border-slate-800">
            <TrendingUp className="w-6 h-6 text-indigo-600" />
            <h2 className="text-lg font-bold">{t('voting.rankPreferences')}</h2>
          </div>

          {votedSuccess ? (
            <div className="text-center py-12 space-y-4">
              <div className="w-16 h-16 bg-emerald-100 dark:bg-emerald-950/20 text-emerald-600 dark:text-emerald-400 rounded-full flex items-center justify-center mx-auto shadow-md">
                <Check className="w-8 h-8 stroke-[3]" />
              </div>
              <h3 className="text-xl font-bold">{t('voting.voteRecorded')}</h3>
              <p className="text-xs text-slate-400">You can adjust and re-submit your ranking choice list at any time until voting is closed.</p>
              <button
                onClick={() => setVotedSuccess(false)}
                className="px-6 py-2 bg-indigo-600 hover:bg-indigo-700 text-white rounded-xl font-bold text-xs shadow transition mt-2"
              >
                {t('voting.changeBallot')}
              </button>
            </div>
          ) : (
            <div className="space-y-6">
              <p className="text-xs text-slate-500 leading-relaxed">{t('voting.rankInstruction')}</p>
              
              <div className="space-y-2">
                {preferences.map((p, index) => (
                  <div 
                    key={p.id} 
                    draggable
                    onDragStart={(e) => handleDragStart(e, index)}
                    onDragOver={(e) => handleDragOver(e, index)}
                    onDragEnd={handleDragEnd}
                    className={`flex items-center justify-between p-3.5 bg-slate-50 dark:bg-slate-950 rounded-xl border border-slate-100 dark:border-slate-850 hover:border-indigo-200 dark:hover:border-indigo-950/50 shadow-sm transition cursor-grab active:cursor-grabbing ${
                      draggedIndex === index ? 'opacity-40 border-dashed border-indigo-400 dark:border-indigo-650 bg-indigo-50/10' : ''
                    }`}
                  >
                    <div className="flex items-center space-x-3 min-w-0">
                      <GripVertical className="w-4 h-4 text-slate-350 dark:text-slate-600 flex-shrink-0 cursor-grab" />
                      <span className="text-xs font-black text-indigo-500 w-5 text-right">{index + 1}.</span>
                      <img
                        src={p.cover_url || 'https://images.igdb.com/igdb/image/upload/t_cover_big/co1r3d.jpg'}
                        alt={p.name}
                        className="w-8 h-11 rounded object-cover flex-shrink-0"
                      />
                      <span className="text-sm font-bold truncate">{p.name}</span>
                    </div>

                    <div className="flex items-center space-x-1 flex-shrink-0">
                      <button
                        onClick={() => moveUp(index)}
                        disabled={index === 0}
                        className="p-1.5 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 disabled:opacity-30 text-slate-500"
                        title="Move Up"
                      >
                        <ArrowUp className="w-4 h-4" />
                      </button>
                      <button
                        onClick={() => moveDown(index)}
                        disabled={index === preferences.length - 1}
                        className="p-1.5 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 disabled:opacity-30 text-slate-500"
                        title="Move Down"
                      >
                        <ArrowDown className="w-4 h-4" />
                      </button>
                    </div>
                  </div>
                ))}
                {preferences.length === 0 && (
                  <p className="text-xs text-slate-400 italic text-center py-8">{t('voting.noNomsCycle')}</p>
                )}
              </div>

              {preferences.length > 0 && (
                <button
                  onClick={handleSubmitVote}
                  disabled={submittingVote}
                  className="w-full py-3 bg-indigo-600 hover:bg-indigo-700 text-white font-bold text-sm rounded-xl shadow-md transition disabled:opacity-50"
                >
                  {submittingVote ? '...' : t('voting.submitVote')}
                </button>
              )}
            </div>
          )}
        </div>
      )}

      {/* --- PHASE 3: CLOSED RESULTS PHASE --- */}
      {session.phase === 'closed' && (
        <div className="max-w-2xl mx-auto bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl p-6 space-y-6">
          <div className="flex items-center space-x-2 pb-4 border-b border-slate-100 dark:border-slate-800">
            <Award className="w-6 h-6 text-amber-500" />
            <h2 className="text-lg font-bold">{t('voting.resultsTitle')}</h2>
          </div>

          <div className="space-y-4">
            {results.map((r, index) => {
              const isWinner = index === 0;
              return (
                <div 
                  key={r.game_id} 
                  className={`flex items-center justify-between p-4 rounded-xl border transition shadow-sm ${
                    isWinner 
                      ? 'bg-amber-500/10 border-amber-500/20 ring-1 ring-amber-500/20' 
                      : 'bg-slate-50 dark:bg-slate-950 border-slate-100 dark:border-slate-850'
                  }`}
                >
                  <div className="flex items-center space-x-4 min-w-0">
                    <span className={`text-base font-black w-6 text-center ${
                      index === 0 ? 'text-amber-500 text-xl' :
                      index === 1 ? 'text-slate-400' :
                      index === 2 ? 'text-amber-700' : 'text-slate-500'
                    }`}>
                      #{r.rank}
                    </span>
                    <img
                      src={r.cover_url || 'https://images.igdb.com/igdb/image/upload/t_cover_big/co1r3d.jpg'}
                      alt={r.name}
                      className="w-10 h-14 rounded-lg object-cover shadow-sm flex-shrink-0"
                    />
                    <div className="min-w-0">
                      <h4 className="text-sm font-black truncate">{r.name}</h4>
                      {isWinner && (
                        <span className="inline-flex items-center text-[9px] font-black tracking-wider text-amber-600 bg-amber-500/15 px-2 py-0.5 rounded mt-1.5 uppercase">
                          {t('voting.winner')} 🏆
                        </span>
                      )}
                    </div>
                  </div>

                  <div className="text-right flex-shrink-0">
                    <span className="text-sm font-black text-indigo-600 dark:text-indigo-400">{r.points}</span>
                    <span className="text-[10px] text-slate-400 font-bold block mt-0.5">{t('voting.points')}</span>
                  </div>
                </div>
              );
            })}
            {results.length === 0 && (
              <p className="text-xs text-slate-400 italic text-center py-8">{t('voting.noVotesSubmitted')}</p>
            )}
          </div>
        </div>
      )}

      {/* --- POP-UP MODAL / WINDOW (80% window size) --- */}
      {isSearchModalOpen && (
        <div className="fixed inset-0 z-50 bg-slate-950/70 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="w-[80vw] h-[80vh] max-w-5xl bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 shadow-2xl flex flex-col overflow-hidden transition-colors">
            
            {/* Modal Header */}
            <div className="p-4 border-b border-slate-100 dark:border-slate-800 flex items-center justify-between">
              <div>
                <h2 className="text-lg font-bold flex items-center space-x-2">
                  <Search className="w-5 h-5 text-indigo-600" />
                  <span>{t('voting.nominateTitle')}</span>
                </h2>
                <p className="text-[10px] text-slate-400 font-bold uppercase mt-1">
                  {t('voting.nominationLimit', { count: myNominations.length, max: session.max_nominations })}
                </p>
              </div>
              <button 
                onClick={() => setIsSearchModalOpen(false)}
                className="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            {/* Modal Search Bar */}
            <form onSubmit={handleSearch} className="p-4 border-b border-slate-100 dark:border-slate-800 flex space-x-2 bg-slate-50 dark:bg-slate-950/40">
              <input
                type="text"
                required
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder={t('voting.searchPlaceholder')}
                className="flex-1 px-4 py-2.5 rounded-xl border border-slate-300 dark:border-slate-700 bg-transparent focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 outline-none text-sm dark:bg-slate-900"
              />
              <button
                type="submit"
                disabled={searching}
                className="px-6 py-2.5 bg-indigo-600 hover:bg-indigo-700 text-white rounded-xl font-bold text-sm shadow transition"
              >
                {searching ? '...' : t('voting.search')}
              </button>
            </form>

            {/* Modal Results Body (Scrollable, 5 items per page) */}
            <div className="flex-1 overflow-y-auto p-6 space-y-4">
              {searching ? (
                <div className="text-center py-12 text-slate-400 font-bold text-sm animate-pulse">{t('voting.searchingDb')}</div>
              ) : searchResults.length === 0 ? (
                <div className="text-center py-12 text-slate-400 font-bold text-sm">
                  {searchQuery ? t('voting.noGamesFound') : t('voting.searchPrompt')}
                </div>
              ) : (
                searchResults.slice((currentPage - 1) * pageSize, currentPage * pageSize).map((game) => {
                  const isAlreadyNominated = nominations.some((n) => n.game_id === game.id);
                  return (
                    <div key={game.id} className="p-4 bg-slate-50 dark:bg-slate-950/40 rounded-xl border border-slate-100 dark:border-slate-850 flex items-start space-x-4 shadow-sm">
                      <img
                        src={game.cover_url || 'https://images.igdb.com/igdb/image/upload/t_cover_big/co1r3d.jpg'}
                        alt={game.name}
                        className="w-12 h-16 rounded-lg object-cover flex-shrink-0 bg-slate-100 dark:bg-slate-800 border border-slate-200 dark:border-slate-800"
                      />
                      <div className="flex-1 min-w-0">
                        <h4 className="text-base font-bold truncate text-slate-900 dark:text-white">{game.name}</h4>
                        {game.release_date && (
                          <span className="text-[10px] text-slate-400 font-bold block mt-1 uppercase">
                            {t('games.released')}: {new Date(game.release_date).toLocaleDateString()}
                          </span>
                        )}
                        <p className="text-xs text-slate-500 dark:text-slate-400 line-clamp-2 mt-1.5 leading-relaxed">{game.summary || 'No summary available.'}</p>
                      </div>
                      <button
                        onClick={() => handleNominate(game)}
                        disabled={isAlreadyNominated || myNominations.length >= session.max_nominations}
                        className={`px-4 py-2 rounded-xl text-xs font-black shadow transition flex-shrink-0 ${
                          isAlreadyNominated
                            ? 'bg-slate-100 text-slate-400 dark:bg-slate-850 dark:text-slate-650 shadow-none'
                            : 'bg-indigo-600 hover:bg-indigo-700 text-white'
                        }`}
                      >
                        {isAlreadyNominated ? t('voting.alreadyNominated') : t('voting.nominateBtn')}
                      </button>
                    </div>
                  );
                })
              )}
            </div>

            {/* Modal Pagination Footer */}
            {!searching && searchResults.length > 0 && (
              <div className="p-4 border-t border-slate-150 dark:border-slate-800 flex items-center justify-between bg-slate-50 dark:bg-slate-950/40">
                <button
                  disabled={currentPage === 1}
                  onClick={() => setCurrentPage((p) => Math.max(p - 1, 1))}
                  className="px-4 py-2 border border-slate-200 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-800 text-xs font-bold rounded-xl disabled:opacity-40 transition"
                >
                  {t('voting.previous')}
                </button>
                <span className="text-xs text-slate-400 font-bold">
                  {t('voting.pageOf', { current: currentPage, total: Math.ceil(searchResults.length / pageSize) })}
                </span>
                <button
                  disabled={currentPage === Math.ceil(searchResults.length / pageSize)}
                  onClick={() => setCurrentPage((p) => Math.min(p + 1, Math.ceil(searchResults.length / pageSize)))}
                  className="px-4 py-2 border border-slate-200 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-800 text-xs font-bold rounded-xl disabled:opacity-40 transition"
                >
                  {t('voting.next')}
                </button>
              </div>
            )}

          </div>
        </div>
      )}

    </div>
  );
};
