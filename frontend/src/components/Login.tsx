import React, { useState } from 'react';
import { useAuth } from '../context/AuthContext';
import { useLanguage } from '../context/LanguageContext';
import { useTheme } from '../context/ThemeContext';
import { Gamepad2, Sun, Moon, Languages } from 'lucide-react';

export const Login: React.FC = () => {
  const { login } = useAuth();
  const { language, setLanguage, t } = useLanguage();
  const { theme, toggleTheme } = useTheme();

  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSubmitting(true);

    try {
      await login(username, password);
    } catch (err: any) {
      setError(err.message || t('login.invalid'));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="min-h-screen flex flex-col justify-center py-12 sm:px-6 lg:px-8 bg-slate-50 dark:bg-slate-950 text-slate-900 dark:text-slate-50 transition-colors">
      
      {/* Header controls */}
      <div className="absolute top-4 right-4 flex items-center space-x-4">
        {/* Language selector */}
        <div className="relative flex items-center bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-lg p-1.5 shadow-sm min-w-[50px] justify-center cursor-pointer hover:bg-slate-100 dark:hover:bg-slate-800 transition">
          {/* Visual Overlay - Displays ONLY the flag */}
          <div className="flex items-center space-x-1.5 text-sm font-semibold pointer-events-none select-none">
            <Languages className="w-4 h-4 text-slate-400 dark:text-slate-500" />
            <span>
              {language === 'en-US' && '🇬🇧'}
              {language === 'pt-BR' && '🇧🇷'}
              {language === 'es-ES' && '🇪🇸'}
              {language === 'ja-JP' && '🇯🇵'}
            </span>
          </div>

          {/* Invisible Native Select layered directly on top */}
          <select
            value={language}
            onChange={(e) => setLanguage(e.target.value as any)}
            className="absolute inset-0 w-full h-full opacity-0 cursor-pointer"
          >
            <option value="en-US">🇬🇧 English</option>
            <option value="pt-BR">🇧🇷 Português</option>
            <option value="es-ES">🇪🇸 Español</option>
            <option value="ja-JP">🇯🇵 日本語</option>
          </select>
        </div>

        {/* Theme toggle */}
        <button
          onClick={toggleTheme}
          className="p-2 rounded-lg bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-sm hover:bg-slate-100 dark:hover:bg-slate-800 transition"
        >
          {theme === 'dark' ? <Moon className="w-4 h-4 text-indigo-400 fill-current" /> : <Sun className="w-4 h-4 text-yellow-500 fill-current" />}
        </button>
      </div>

      <div className="sm:mx-auto sm:w-full sm:max-w-md">
        <div className="flex justify-center">
          <div className="w-16 h-16 rounded-2xl bg-indigo-600 flex items-center justify-center text-white shadow-lg shadow-indigo-500/20">
            <Gamepad2 className="w-10 h-10 animate-bounce" />
          </div>
        </div>
        <h2 className="mt-6 text-center text-3xl font-extrabold tracking-tight">
          {t('login.title')}
        </h2>
        <p className="mt-2 text-center text-sm text-slate-500 dark:text-slate-400">
          {t('login.subtitle')}
        </p>
      </div>

      <div className="mt-8 sm:mx-auto sm:w-full sm:max-w-md">
        <div className="bg-white dark:bg-slate-900 py-8 px-4 border border-slate-200 dark:border-slate-850 sm:rounded-2xl sm:px-10 shadow-xl dark:shadow-slate-950/50">
          <form className="space-y-6" onSubmit={handleSubmit}>
            
            {error && (
              <div className="bg-red-550 border-l-4 border-red-550 bg-red-50 dark:bg-red-950/20 text-red-700 dark:text-red-400 p-3 rounded-r-lg text-sm font-medium">
                {error}
              </div>
            )}

            <div>
              <label className="block text-sm font-semibold mb-1">
                {t('login.username')}
              </label>
              <input
                type="text"
                required
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="w-full rounded-xl border border-slate-300 dark:border-slate-700 bg-transparent px-4 py-2.5 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 outline-none text-sm dark:bg-slate-950 transition"
              />
            </div>

            <div>
              <label className="block text-sm font-semibold mb-1">
                {t('login.password')}
              </label>
              <input
                type="password"
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full rounded-xl border border-slate-300 dark:border-slate-700 bg-transparent px-4 py-2.5 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 outline-none text-sm dark:bg-slate-950 transition"
              />
            </div>

            <div>
              <button
                type="submit"
                disabled={submitting}
                className="w-full flex justify-center py-2.5 px-4 border border-transparent rounded-xl shadow-md text-sm font-bold text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 transition disabled:opacity-50"
              >
                {submitting ? '...' : t('login.button')}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
};
