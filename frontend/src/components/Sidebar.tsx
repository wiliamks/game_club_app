import React from 'react';
import { useAuth } from '../context/AuthContext';
import { useLanguage } from '../context/LanguageContext';
import { useTheme } from '../context/ThemeContext';
import { 
  Gamepad2, 
  Vote, 
  User as UserIcon, 
  ShieldAlert, 
  LogOut, 
  Moon, 
  Sun,
  Languages
} from 'lucide-react';

interface SidebarProps {
  activeTab: string;
  setActiveTab: (tab: string) => void;
}

export const Sidebar: React.FC<SidebarProps> = ({ activeTab, setActiveTab }) => {
  const { user, logout } = useAuth();
  const { language, setLanguage, t } = useLanguage();
  const { theme, toggleTheme } = useTheme();

  const isAdmin = user?.role === 'admin';

  return (
    <div className="hidden md:flex w-64 bg-white dark:bg-slate-900 border-r border-slate-200 dark:border-slate-800 flex-col justify-between h-screen sticky top-0 transition-colors">
      
      {/* Upper Brand / Logo section */}
      <div className="p-6">
        <div className="flex items-center space-x-3 mb-8">
          <div className="w-10 h-10 rounded-xl bg-indigo-600 flex items-center justify-center text-white shadow-md shadow-indigo-500/20">
            <Gamepad2 className="w-6 h-6" />
          </div>
          <div>
            <h1 className="text-xl font-black tracking-tight leading-none bg-gradient-to-r from-indigo-500 to-purple-500 bg-clip-text text-transparent">
              Gamer Club
            </h1>
          </div>
        </div>

        {/* Navigation tabs */}
        <nav className="space-y-1">
          <button
            onClick={() => setActiveTab('games')}
            className={`w-full flex items-center space-x-3 px-4 py-3 rounded-xl text-sm font-bold transition-all ${
              activeTab === 'games'
                ? 'bg-indigo-50 dark:bg-indigo-950/40 text-indigo-600 dark:text-indigo-400'
                : 'text-slate-500 hover:bg-slate-50 dark:hover:bg-slate-800/40'
            }`}
          >
            <Gamepad2 className="w-5 h-5" />
            <span>{t('nav.games')}</span>
          </button>

          <button
            onClick={() => setActiveTab('voting')}
            className={`w-full flex items-center space-x-3 px-4 py-3 rounded-xl text-sm font-bold transition-all ${
              activeTab === 'voting'
                ? 'bg-indigo-50 dark:bg-indigo-950/40 text-indigo-600 dark:text-indigo-400'
                : 'text-slate-500 hover:bg-slate-50 dark:hover:bg-slate-800/40'
            }`}
          >
            <Vote className="w-5 h-5" />
            <span>{t('nav.voting')}</span>
          </button>

          <button
            onClick={() => setActiveTab('account')}
            className={`w-full flex items-center space-x-3 px-4 py-3 rounded-xl text-sm font-bold transition-all ${
              activeTab === 'account'
                ? 'bg-indigo-50 dark:bg-indigo-950/40 text-indigo-600 dark:text-indigo-400'
                : 'text-slate-500 hover:bg-slate-50 dark:hover:bg-slate-800/40'
            }`}
          >
            <UserIcon className="w-5 h-5" />
            <span>{t('nav.account')}</span>
          </button>

          {isAdmin && (
            <button
              onClick={() => setActiveTab('admin')}
              className={`w-full flex items-center space-x-3 px-4 py-3 rounded-xl text-sm font-bold transition-all ${
                activeTab === 'admin'
                  ? 'bg-red-50 dark:bg-red-950/20 text-red-600 dark:text-red-400'
                  : 'text-slate-500 hover:bg-slate-50 dark:hover:bg-slate-800/40'
              }`}
            >
              <ShieldAlert className="w-5 h-5" />
              <span>{t('nav.admin')}</span>
            </button>
          )}
        </nav>
      </div>

      {/* Footer controls & Profile section */}
      <div className="p-6 border-t border-slate-200 dark:border-slate-800 space-y-4">
        
        {/* Localization & Theme switches */}
        <div className="flex items-center justify-between">
          <div className="relative flex items-center bg-slate-100 dark:bg-slate-800 rounded-lg p-1.5 text-xs min-w-[50px] justify-center cursor-pointer hover:bg-slate-200 dark:hover:bg-slate-700 transition">
            {/* Visual Overlay - Displays ONLY the flag */}
            <div className="flex items-center space-x-1.5 text-xs font-semibold pointer-events-none select-none">
              <Languages className="w-3.5 h-3.5 text-slate-400" />
              <span>
                {language === 'en-US' && '🇬🇧'}
                {language === 'pt-BR' && '🇧🇷'}
                {language === 'es-ES' && '🇪🇸'}
                {language === 'ja-JP' && '🇯🇵'}
              </span>
            </div>

            {/* Invisible Native Select layered on top */}
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

          <button
            onClick={toggleTheme}
            className="p-1.5 rounded-lg bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 transition"
          >
            {theme === 'dark' ? <Moon className="w-4 h-4 text-indigo-400 fill-current" /> : <Sun className="w-4 h-4 text-yellow-500 fill-current" />}
          </button>
        </div>

        {/* Profile Card */}
        <div className="flex items-center justify-between bg-slate-50 dark:bg-slate-800/30 p-3 rounded-xl">
          <div className="truncate pr-2">
            <p className="text-sm font-bold truncate">{user?.username}</p>
            <p className="text-[10px] text-slate-400 capitalize">{user?.role}</p>
          </div>
          <button
            onClick={logout}
            className="p-1.5 rounded-lg bg-red-50 dark:bg-red-950/20 text-red-600 dark:text-red-400 hover:bg-red-100 dark:hover:bg-red-950/40 transition"
            title={t('nav.logout')}
          >
            <LogOut className="w-4 h-4" />
          </button>
        </div>
      </div>
    </div>
  );
};
