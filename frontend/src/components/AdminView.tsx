import React, { useState, useEffect } from 'react';
import { useAuth } from '../context/AuthContext';
import { useLanguage } from '../context/LanguageContext';
import { 
  Users, 
  Gamepad, 
  Vote, 
  UserPlus, 
  Trash2, 
  Activity, 
  Play, 
  ArrowRight, 
  XCircle, 
  AlertTriangle 
} from 'lucide-react';

interface LocalUser {
  id: number;
  username: string;
  role: string;
}

interface LocalGame {
  id: number;
  name: string;
  is_active: boolean;
}

interface LocalSession {
  id: number;
  name: string;
  max_nominations: number;
  phase: string;
}

export const AdminView: React.FC = () => {
  const { user, apiFetch } = useAuth();
  const { t } = useLanguage();

  const [users, setUsers] = useState<LocalUser[]>([]);
  const [games, setGames] = useState<LocalGame[]>([]);
  const [session, setSession] = useState<LocalSession | null>(null);
  const [sessions, setSessions] = useState<LocalSession[]>([]);
  const [isGameModalOpen, setIsGameModalOpen] = useState(false);
  const [gameSearchQuery, setGameSearchQuery] = useState('');
  const [gameCurrentPage, setGameCurrentPage] = useState(1);
  const gamePageSize = 7;

  // Form states
  const [newUsername, setNewUsername] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [newRole, setNewRole] = useState<'admin' | 'user'>('user');

  const [votingSessionName, setVotingSessionName] = useState('');
  const [maxNominations, setMaxNominations] = useState(3);

  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  const isAdmin = user?.role === 'admin';

  const fetchData = async () => {
    if (!isAdmin) return;
    try {
      const usersData = await apiFetch('/admin/users');
      setUsers(usersData || []);

      const gamesData = await apiFetch('/games');
      setGames(gamesData || []);

      const allSess = await apiFetch('/voting/sessions');
      setSessions(allSess || []);

      if (allSess && allSess.length > 0) {
        setSession((prev) => {
          if (prev) {
            const refreshed = allSess.find((s: LocalSession) => s.id === prev.id);
            return refreshed || allSess[0];
          }
          return allSess[0];
        });
      } else {
        setSession(null);
      }
    } catch (err: any) {
      console.error('Failed to load admin data:', err);
    }
  };

  useEffect(() => {
    fetchData();
  }, [isAdmin]);

  if (!isAdmin) {
    return (
      <div className="p-8 text-center">
        <AlertTriangle className="w-16 h-16 mx-auto text-red-500 mb-4 animate-pulse" />
        <h1 className="text-2xl font-bold text-red-600">{t('admin.accessForbidden')}</h1>
        <p className="text-slate-500 dark:text-slate-400 mt-2">{t('admin.adminOnlyPage')}</p>
      </div>
    );
  }

  // --- USER CONTROLS ---
  const handleCreateUser = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccess('');

    try {
      const newUser = await apiFetch('/admin/users', {
        method: 'POST',
        body: JSON.stringify({
          username: newUsername,
          password: newPassword,
          role: newRole,
        }),
      });

      setUsers((prev) => [...prev, newUser]);
      setSuccess(`User ${newUsername} created successfully.`);
      setNewUsername('');
      setNewPassword('');
    } catch (err: any) {
      setError(err.message || 'Failed to create user');
    }
  };

  const handleDeleteUser = async (id: number) => {
    if (id === 1) return; // Prevent deleting primary admin
    if (!window.confirm(t('admin.confirmDelete'))) return;

    setError('');
    setSuccess('');

    try {
      await apiFetch(`/admin/users/${id}`, { method: 'DELETE' });
      setUsers((prev) => prev.filter((u) => u.id !== id));
      setSuccess(t('admin.userDeleted'));
    } catch (err: any) {
      setError(err.message || 'Failed to delete user');
    }
  };

  // --- ACTIVE GAME CONTROLS ---
  const handleSetActiveGame = async (gameId: number) => {
    setError('');
    setSuccess('');

    try {
      await apiFetch('/admin/active-game', {
        method: 'POST',
        body: JSON.stringify({ game_id: gameId }),
      });
      setSuccess(t('admin.gameConfigured'));
      setIsGameModalOpen(false);
      fetchData(); // reload
    } catch (err: any) {
      setError(err.message || 'Failed to configure active game');
    }
  };

  const handleDeactivateActiveGame = async () => {
    setError('');
    setSuccess('');

    try {
      await apiFetch('/admin/active-game', { method: 'DELETE' });
      setSuccess(t('admin.gameDeactivated'));
      fetchData(); // reload
    } catch (err: any) {
      setError(err.message || 'Failed to deactivate game');
    }
  };

  // --- VOTING SESSION CONTROLS ---
  const handleStartVoting = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccess('');

    try {
      const newSess = await apiFetch('/admin/voting/session', {
        method: 'POST',
        body: JSON.stringify({
          name: votingSessionName,
          max_nominations: maxNominations,
        }),
      });

      setSuccess(t('admin.sessionInitiated', { name: votingSessionName }));
      setVotingSessionName('');
      
      const allSess = await apiFetch('/voting/sessions');
      setSessions(allSess || []);
      setSession(newSess);
    } catch (err: any) {
      setError(err.message || 'Failed to start voting session');
    }
  };

  const handleAdvancePhase = async () => {
    if (!session) return;
    setError('');
    setSuccess('');

    let nextPhase = 'nomination';
    if (session.phase === 'nomination') nextPhase = 'voting';
    else if (session.phase === 'voting') nextPhase = 'closed';

    try {
      await apiFetch('/admin/voting/phase', {
        method: 'PUT',
        body: JSON.stringify({ 
          session_id: session.id,
          phase: nextPhase 
        }),
      });

      setSuccess(t('admin.cycleAdvanced', { phase: nextPhase.toUpperCase() }));
      fetchData();
    } catch (err: any) {
      setError(err.message || 'Failed to transition voting phase');
    }
  };

  const handleCancelVoting = async () => {
    if (!session) return;
    if (!window.confirm(t('admin.confirmCancelSession', { name: session.name }))) return;
    setError('');
    setSuccess('');

    try {
      await apiFetch(`/admin/voting/session?session_id=${session.id}`, { 
        method: 'DELETE' 
      });
      setSuccess(t('admin.sessionCanceled'));
      
      const remainingSess = sessions.filter((s) => s.id !== session.id);
      setSessions(remainingSess);
      if (remainingSess.length > 0) {
        setSession(remainingSess[0]);
      } else {
        setSession(null);
      }
    } catch (err: any) {
      setError(err.message || 'Failed to cancel voting session');
    }
  };

  return (
    <div className="p-8 max-w-5xl mx-auto space-y-8 pb-16">
      
      {/* Upper header */}
      <div>
        <h1 className="text-3xl font-black">{t('admin.title')}</h1>
        <p className="text-slate-500 mt-1">{t('admin.adminSubtitle')}</p>
      </div>

      {/* Banner messages */}
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

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        
        {/* Panel 1: User Management */}
        <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl p-6 space-y-6">
          <div className="flex items-center space-x-2 pb-4 border-b border-slate-100 dark:border-slate-800">
            <Users className="w-6 h-6 text-indigo-600" />
            <h2 className="text-lg font-bold">{t('admin.userManagement')}</h2>
          </div>

          {/* User registration form */}
          <form onSubmit={handleCreateUser} className="space-y-4">
            <h3 className="text-sm font-semibold text-slate-400 uppercase tracking-wider">{t('admin.createUser')}</h3>
            <div className="grid grid-cols-2 gap-4">
              <input
                type="text"
                required
                placeholder="Username"
                value={newUsername}
                onChange={(e) => setNewUsername(e.target.value)}
                className="w-full px-4 py-2 border border-slate-200 dark:border-slate-700 rounded-xl bg-transparent focus:ring-1 focus:ring-indigo-500 outline-none text-sm dark:bg-slate-950"
              />
              <input
                type="password"
                required
                placeholder="Password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                className="w-full px-4 py-2 border border-slate-200 dark:border-slate-700 rounded-xl bg-transparent focus:ring-1 focus:ring-indigo-500 outline-none text-sm dark:bg-slate-950"
              />
            </div>
            <div className="flex items-center space-x-4">
              <button
                type="button"
                onClick={() => setNewRole(p => p === 'admin' ? 'user' : 'admin')}
                className="flex-1 flex items-center justify-between border border-slate-200 dark:border-slate-700 rounded-xl px-4 py-2 text-sm font-semibold hover:bg-slate-50 dark:hover:bg-slate-800 transition active:scale-95"
              >
                <span className="text-xs text-slate-400 font-bold">{t('admin.role')}:</span>
                <span className={`text-[10px] px-2.5 py-0.5 rounded-full font-black uppercase tracking-wider ${
                  newRole === 'admin' 
                    ? 'bg-red-500/15 text-red-500 border border-red-500/20' 
                    : 'bg-slate-100 dark:bg-slate-800 text-slate-500 dark:text-slate-400'
                }`}>
                  {newRole === 'admin' ? 'Admin' : 'User'}
                </span>
              </button>
              <button
                type="submit"
                className="px-6 py-2.5 bg-indigo-600 text-white rounded-xl font-bold text-sm shadow hover:bg-indigo-700 transition flex items-center space-x-1.5"
              >
                <UserPlus className="w-4 h-4" />
                <span>{t('admin.createUser')}</span>
              </button>
            </div>
          </form>

          {/* Users List */}
          <div className="overflow-hidden border border-slate-100 dark:border-slate-800 rounded-xl">
            <div className="divide-y divide-slate-100 dark:divide-slate-800">
              {users.map((u) => (
                <div key={u.id} className="flex items-center justify-between p-3.5 hover:bg-slate-50 dark:hover:bg-slate-800/20">
                  <div>
                    <p className="text-sm font-bold">{u.username}</p>
                    <p className="text-[10px] text-slate-400 uppercase font-semibold">{u.role}</p>
                  </div>
                  {u.id !== 1 && (
                    <button
                      onClick={() => handleDeleteUser(u.id)}
                      className="p-1.5 rounded-lg text-slate-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-950/20 transition"
                      title={t('admin.delete')}
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  )}
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Panel 2: Active Game & Voting cycle Controllers */}
        <div className="space-y-8">
          
          {/* Section A: Active Game */}
          <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl p-6 space-y-4">
            <div className="flex items-center space-x-2 pb-4 border-b border-slate-100 dark:border-slate-800">
              <Gamepad className="w-6 h-6 text-indigo-600" />
              <h2 className="text-lg font-bold">{t('admin.activeGameController')}</h2>
            </div>

            <div className="space-y-4">
              {/* Display current active game status if any is active */}
              {games.some(g => g.is_active) ? (
                <div className="bg-indigo-50 dark:bg-indigo-950/20 p-4 rounded-xl border border-indigo-100 dark:border-indigo-950 text-sm">
                  <span className="text-xs font-bold text-indigo-500 block uppercase mb-1">Active Game:</span>
                  <span className="font-extrabold text-slate-800 dark:text-slate-200">
                    {games.find(g => g.is_active)?.name}
                  </span>
                </div>
              ) : (
                <p className="text-xs text-slate-400">{t('admin.noActiveGame')}</p>
              )}

              {/* Button to open active game selector modal */}
              <button
                onClick={() => {
                  setIsGameModalOpen(true);
                  setGameSearchQuery('');
                  setGameCurrentPage(1);
                }}
                className="w-full py-2.5 bg-indigo-600 hover:bg-indigo-700 text-white font-bold text-sm rounded-xl shadow transition"
              >
                {t('admin.activeGameSelectModal')}
              </button>

              <button
                onClick={handleDeactivateActiveGame}
                disabled={!games.some(g => g.is_active)}
                className="w-full py-2 bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-300 font-bold text-xs rounded-xl transition disabled:opacity-50"
              >
                {t('admin.activeGameDeactivate')}
              </button>
            </div>
          </div>

          {/* Section B: Voting controller */}
          <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl p-6 space-y-6">
            <div className="flex items-center space-x-2 pb-4 border-b border-slate-100 dark:border-slate-800">
              <Vote className="w-6 h-6 text-indigo-600" />
              <h2 className="text-lg font-bold">{t('admin.votingController')}</h2>
            </div>

            {/* Create new voting session form (Always visible) */}
            <form onSubmit={handleStartVoting} className="space-y-4">
              <h3 className="text-xs font-semibold text-slate-400 uppercase tracking-wider">{t('admin.startVoting')}</h3>
              <div className="space-y-3">
                <input
                  type="text"
                  required
                  placeholder={t('admin.sessionName')}
                  value={votingSessionName}
                  onChange={(e) => setVotingSessionName(e.target.value)}
                  className="w-full px-4 py-2 border border-slate-200 dark:border-slate-700 rounded-xl bg-transparent focus:ring-1 focus:ring-indigo-500 outline-none text-sm dark:bg-slate-950"
                />
                <div className="flex items-center space-x-3">
                  <div className="flex-1 flex items-center space-x-2 border border-slate-200 dark:border-slate-700 rounded-xl px-4 py-2 text-sm">
                    <span className="text-xs text-slate-400 whitespace-nowrap">{t('admin.maxNominations')}:</span>
                    <input
                      type="number"
                      min={1}
                      max={10}
                      required
                      value={maxNominations}
                      onChange={(e) => setMaxNominations(Number(e.target.value))}
                      className="bg-transparent border-none text-sm font-semibold focus:ring-0 p-0 w-full"
                    />
                  </div>
                  <button
                    type="submit"
                    className="px-6 py-2.5 bg-indigo-600 text-white rounded-xl font-bold text-sm shadow hover:bg-indigo-700 transition flex items-center space-x-1.5"
                  >
                    <Play className="w-4 h-4" />
                    <span>{t('admin.start')}</span>
                  </button>
                </div>
              </div>
            </form>

            {/* Manage Existing Sessions dropdown (Shown if sessions exist) */}
            {sessions.length > 0 && (
              <div className="space-y-4 pt-6 border-t border-slate-100 dark:border-slate-800">
                <h3 className="text-xs font-semibold text-slate-400 uppercase tracking-wider">{t('admin.manageSessions')}</h3>
                <div className="flex items-center space-x-2 bg-slate-50 dark:bg-slate-950/40 border border-slate-200 dark:border-slate-800 rounded-xl px-4 py-2 text-xs">
                  <span className="text-xs font-bold text-slate-400">{t('admin.selectEvent')}</span>
                  <select
                    value={session?.id || ''}
                    onChange={(e) => {
                      const selected = sessions.find((s) => s.id === Number(e.target.value));
                      if (selected) {
                        setSession(selected);
                      }
                    }}
                    className="bg-transparent border-none text-xs font-extrabold focus:ring-0 cursor-pointer p-0 pl-1 outline-none dark:bg-slate-900 flex-1"
                  >
                    {sessions.map((s) => (
                      <option key={s.id} value={s.id}>
                        {s.name} ({t('voting.' + s.phase)})
                      </option>
                    ))}
                  </select>
                </div>

                {session && (
                  <div className="space-y-4 pt-2">
                    <div className="flex items-center justify-between bg-indigo-50 dark:bg-indigo-950/20 p-4 rounded-xl border border-indigo-100 dark:border-indigo-950">
                      <div>
                        <h3 className="text-sm font-bold text-indigo-950 dark:text-indigo-200">{session.name}</h3>
                        <p className="text-[11px] text-indigo-400 mt-1 capitalize">{t('admin.currentPhase')}: <strong className="font-extrabold">{t('voting.' + session.phase)}</strong></p>
                      </div>
                      <div className="flex items-center space-x-1 text-[11px] bg-indigo-100 dark:bg-indigo-950 px-2.5 py-1 rounded-md text-indigo-600 dark:text-indigo-400 font-bold">
                        <Activity className="w-3.5 h-3.5 animate-pulse" />
                        <span>{t('admin.status')}</span>
                      </div>
                    </div>

                    {session.phase !== 'closed' ? (
                      <button
                        onClick={handleAdvancePhase}
                        className="w-full flex items-center justify-center space-x-2 py-3 bg-indigo-600 hover:bg-indigo-700 text-white font-bold text-sm rounded-xl shadow-md shadow-indigo-650/10 transition"
                      >
                        <span>{t('admin.nextPhase', { phase: session.phase === 'nomination' ? 'VOTING' : 'CLOSED' })}</span>
                        <ArrowRight className="w-4 h-4" />
                      </button>
                    ) : (
                      <div className="p-3 bg-slate-50 dark:bg-slate-800/40 rounded-xl text-center text-xs font-semibold text-slate-500">
                        🏆 Voting Cycle Complete. Results are published publically.
                      </div>
                    )}

                    <button
                      onClick={handleCancelVoting}
                      className="w-full flex items-center justify-center space-x-2 py-2.5 bg-red-50 dark:bg-red-950/20 hover:bg-red-100 text-red-600 dark:text-red-400 font-bold text-xs rounded-xl transition border border-red-100 dark:border-red-950"
                    >
                      <XCircle className="w-4 h-4" />
                      <span>{t('admin.cancelDeleteSession')}</span>
                    </button>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* --- ACTIVE GAME SELECTOR POPUP MODAL --- */}
      {isGameModalOpen && (
        <div className="fixed inset-0 z-50 bg-slate-950/70 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="w-[80vw] h-[80vh] max-w-5xl bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 shadow-2xl flex flex-col overflow-hidden transition-colors">
            
            {/* Modal Header */}
            <div className="p-4 border-b border-slate-100 dark:border-slate-800 flex items-center justify-between flex-shrink-0">
              <div>
                <h2 className="text-lg font-bold flex items-center space-x-2">
                  <Gamepad className="w-5 h-5 text-indigo-600" />
                  <span>{t('admin.activeGameSelectModal')}</span>
                </h2>
                <p className="text-[10px] text-slate-400 font-bold uppercase mt-1">
                  {t('admin.activeGameSelectDesc')}
                </p>
              </div>
              <button 
                onClick={() => setIsGameModalOpen(false)}
                className="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition"
              >
                <XCircle className="w-5 h-5" />
              </button>
            </div>

            {/* Modal Search Bar */}
            <div className="p-4 border-b border-slate-100 dark:border-slate-800 flex bg-slate-50 dark:bg-slate-950/40 flex-shrink-0">
              <input
                type="text"
                value={gameSearchQuery}
                onChange={(e) => {
                  setGameSearchQuery(e.target.value);
                  setGameCurrentPage(1);
                }}
                placeholder={t('admin.searchLocalGames')}
                className="flex-1 px-4 py-2.5 rounded-xl border border-slate-300 dark:border-slate-700 bg-transparent focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 outline-none text-sm dark:bg-slate-900"
              />
            </div>

            {/* Modal Results Body (Scrollable, 7 items per page) */}
            <div className="flex-1 overflow-y-auto p-6 space-y-4">
              {(() => {
                const filteredGames = games.filter((g) =>
                  g.name.toLowerCase().includes(gameSearchQuery.toLowerCase())
                );
                const totalPages = Math.ceil(filteredGames.length / gamePageSize);
                const paginatedGames = filteredGames.slice(
                  (gameCurrentPage - 1) * gamePageSize,
                  gameCurrentPage * gamePageSize
                );

                if (filteredGames.length === 0) {
                  return (
                    <div className="text-center py-12 text-slate-400 font-bold text-sm">
                      {t('admin.noLocalGamesFound')}
                    </div>
                  );
                }

                return (
                  <>
                    <div className="space-y-4">
                      {paginatedGames.map((g) => (
                        <div key={g.id} className="p-4 bg-slate-50 dark:bg-slate-950/40 rounded-xl border border-slate-100 dark:border-slate-850 flex items-start justify-between shadow-sm">
                          <div className="flex items-center space-x-3 min-w-0">
                            <span className="text-sm font-bold text-slate-800 dark:text-slate-200 truncate">{g.name}</span>
                            {g.is_active && (
                              <span className="bg-amber-500 text-white text-[9px] font-black px-2 py-0.5 rounded tracking-wider uppercase flex-shrink-0">
                                ★ Active
                              </span>
                            )}
                          </div>
                          <button
                            onClick={() => handleSetActiveGame(g.id)}
                            disabled={g.is_active}
                            className={`px-4 py-2 rounded-xl text-xs font-black shadow transition flex-shrink-0 ${
                              g.is_active
                                ? 'bg-slate-100 text-slate-400 dark:bg-slate-850 dark:text-slate-650 shadow-none'
                                : 'bg-indigo-600 hover:bg-indigo-700 text-white'
                            }`}
                          >
                            {g.is_active ? t('admin.currentlyActive') : t('admin.activeGameSet')}
                          </button>
                        </div>
                      ))}
                    </div>

                    {/* Modal Pagination Footer */}
                    {totalPages > 1 && (
                      <div className="mt-6 border-t border-slate-150 dark:border-slate-800 pt-4 flex items-center justify-between">
                        <button
                          disabled={gameCurrentPage === 1}
                          onClick={() => setGameCurrentPage((p) => Math.max(p - 1, 1))}
                          className="px-4 py-2 border border-slate-200 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-800 text-xs font-bold rounded-xl disabled:opacity-40 transition"
                        >
                          {t('voting.previous')}
                        </button>
                        <span className="text-xs text-slate-400 font-bold">
                          {t('voting.pageOf', { current: gameCurrentPage, total: totalPages })}
                        </span>
                        <button
                          disabled={gameCurrentPage === totalPages}
                          onClick={() => setGameCurrentPage((p) => Math.min(p + 1, totalPages))}
                          className="px-4 py-2 border border-slate-200 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-800 text-xs font-bold rounded-xl disabled:opacity-40 transition"
                        >
                          {t('voting.next')}
                        </button>
                      </div>
                    )}
                  </>
                );
              })()}
            </div>

          </div>
        </div>
      )}

    </div>
  );
};
