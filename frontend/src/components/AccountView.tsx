import React, { useState } from 'react';
import { useAuth } from '../context/AuthContext';
import { useLanguage } from '../context/LanguageContext';
import { User, KeyRound, CheckCircle, Image } from 'lucide-react';

export const AccountView: React.FC = () => {
  const { user, apiFetch, updateUserContext } = useAuth();
  const { t } = useLanguage();

  const [username, setUsername] = useState(user?.username || '');
  const [password, setPassword] = useState('');
  const [avatarUrl, setAvatarUrl] = useState(user?.avatar_url || '');
  const [success, setSuccess] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSuccess('');
    setError('');
    setSubmitting(true);

    try {
      const data = await apiFetch('/account/update', {
        method: 'PUT',
        body: JSON.stringify({ username, password, avatar_url: avatarUrl }),
      });

      updateUserContext(data.user, data.token);
      setSuccess(t('account.success'));
      setPassword('');
    } catch (err: any) {
      setError(err.message || 'Failed to update account credentials');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="p-8 max-w-2xl mx-auto">
      <div className="mb-8">
        <h1 className="text-3xl font-black">{t('account.title')}</h1>
        <p className="text-slate-500 dark:text-slate-400 mt-2">{t('account.subtitle')}</p>
      </div>

      <div className="bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 shadow-sm p-6">
        <form onSubmit={handleSubmit} className="space-y-6">
          
          {success && (
            <div className="flex items-center space-x-2 bg-emerald-50 dark:bg-emerald-950/20 border-l-4 border-emerald-500 text-emerald-700 dark:text-emerald-400 p-4 rounded-r-xl">
              <CheckCircle className="w-5 h-5 flex-shrink-0" />
              <span className="text-sm font-semibold">{success}</span>
            </div>
          )}

          {error && (
            <div className="bg-red-50 dark:bg-red-950/20 border-l-4 border-red-500 text-red-700 dark:text-red-400 p-4 rounded-r-xl text-sm font-semibold">
              {error}
            </div>
          )}

          <div className="space-y-2">
            <label className="block text-sm font-bold text-slate-700 dark:text-slate-300">
              {t('account.newUsername')}
            </label>
            <div className="relative">
              <span className="absolute inset-y-0 left-0 flex items-center pl-3 text-slate-400">
                <User className="w-5 h-5" />
              </span>
              <input
                type="text"
                required
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="w-full pl-10 pr-4 py-2.5 rounded-xl border border-slate-300 dark:border-slate-700 bg-transparent focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 outline-none text-sm dark:bg-slate-950 transition"
                placeholder="Username"
              />
            </div>
          </div>

          <div className="space-y-2">
            <label className="block text-sm font-bold text-slate-700 dark:text-slate-300">
              {t('account.avatarUrl')}
            </label>
            <div className="relative">
              <span className="absolute inset-y-0 left-0 flex items-center pl-3 text-slate-400">
                <Image className="w-5 h-5" />
              </span>
              <input
                type="url"
                value={avatarUrl}
                onChange={(e) => setAvatarUrl(e.target.value)}
                className="w-full pl-10 pr-4 py-2.5 rounded-xl border border-slate-300 dark:border-slate-700 bg-transparent focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 outline-none text-sm dark:bg-slate-950 transition"
                placeholder="https://example.com/avatar.jpg"
              />
            </div>
          </div>

          <div className="space-y-2">
            <label className="block text-sm font-bold text-slate-700 dark:text-slate-300">
              {t('account.newPassword')}
            </label>
            <div className="relative">
              <span className="absolute inset-y-0 left-0 flex items-center pl-3 text-slate-400">
                <KeyRound className="w-5 h-5" />
              </span>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full pl-10 pr-4 py-2.5 rounded-xl border border-slate-300 dark:border-slate-700 bg-transparent focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 outline-none text-sm dark:bg-slate-950 transition"
                placeholder="••••••••"
              />
            </div>
          </div>

          <button
            type="submit"
            disabled={submitting}
            className="w-full flex justify-center py-2.5 px-4 border border-transparent rounded-xl shadow-md text-sm font-bold text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 transition disabled:opacity-50"
          >
            {submitting ? '...' : t('account.save')}
          </button>
        </form>
      </div>
    </div>
  );
};
