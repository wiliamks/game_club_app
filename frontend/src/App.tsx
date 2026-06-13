import React, { useState } from 'react';
import { AuthProvider, useAuth } from './context/AuthContext';
import { ThemeProvider, useTheme } from './context/ThemeContext';
import { LanguageProvider, useLanguage } from './context/LanguageContext';
import { Login } from './components/Login';
import { Sidebar } from './components/Sidebar';
import { GamesView } from './components/GamesView';
import { VotingView } from './components/VotingView';
import { AccountView } from './components/AccountView';
import { AdminView } from './components/AdminView';
import { 
  Gamepad2, 
  Vote, 
  User as UserIcon, 
  ShieldAlert, 
  LogOut, 
  Sun, 
  Moon 
} from 'lucide-react';

const AppContent: React.FC = () => {
  const { user, loading, logout } = useAuth();
  const { language, setLanguage, t } = useLanguage();
  const { theme, toggleTheme } = useTheme();
  const [activeTab, setActiveTab] = useState<string>('games');

  if (loading) {
    return (
      <div className="min-h-screen bg-slate-50 dark:bg-slate-950 flex flex-col items-center justify-center space-y-4">
        <div className="w-12 h-12 rounded-xl bg-indigo-600 flex items-center justify-center text-white animate-spin">
          <Gamepad2 className="w-6 h-6" />
        </div>
        <span className="text-xs font-bold text-slate-400 dark:text-slate-500 animate-pulse uppercase tracking-wider">
          {t('nav.initializing')}
        </span>
      </div>
    );
  }

  if (!user) {
    return <Login />;
  }

  return (
    <div className="min-h-screen flex flex-col md:flex-row bg-slate-50 dark:bg-slate-950 text-slate-900 dark:text-slate-50 transition-colors">
      
      {/* Mobile Top Header (Only on small screens) */}
      <div className="md:hidden flex items-center justify-between p-4 bg-white dark:bg-slate-900 border-b border-slate-200 dark:border-slate-800 sticky top-0 z-45 transition-colors">
        <div className="flex items-center space-x-2">
          <div className="w-8 h-8 rounded-lg bg-indigo-600 flex items-center justify-center text-white shadow-md">
            <Gamepad2 className="w-5 h-5" />
          </div>
          <span className="font-black text-sm">Gamer Club</span>
        </div>
        
        {/* Compact Quick Actions inside Header */}
        <div className="flex items-center space-x-3">
          <div className="relative flex items-center bg-slate-100 dark:bg-slate-800 rounded-lg p-1 text-xs justify-center cursor-pointer min-w-[35px] h-8">
            <span className="text-xs font-bold select-none pointer-events-none">
              {language === 'en-US' && '🇬🇧'}
              {language === 'pt-BR' && '🇧🇷'}
              {language === 'es-ES' && '🇪🇸'}
              {language === 'ja-JP' && '🇯🇵'}
            </span>
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
          <button onClick={toggleTheme} className="p-1.5 h-8 w-8 flex items-center justify-center rounded-lg bg-slate-100 dark:bg-slate-800 text-slate-500 hover:text-slate-700 dark:hover:text-slate-200 transition">
            {theme === 'dark' ? <Moon className="w-4 h-4 text-indigo-400 fill-current" /> : <Sun className="w-4 h-4 text-yellow-500 fill-current" />}
          </button>
          <button onClick={logout} className="p-1.5 h-8 w-8 flex items-center justify-center rounded-lg bg-red-50 dark:bg-red-950/20 text-red-600 dark:text-red-400 hover:bg-red-100 dark:hover:bg-red-950/40 transition">
            <LogOut className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Persistent Sidebar (Visible only on tablets & desktops) */}
      <Sidebar activeTab={activeTab} setActiveTab={setActiveTab} />

      {/* Main Tab Screen Area */}
      <main className={`flex-1 h-screen pb-16 md:pb-0 ${activeTab === 'games' ? 'overflow-hidden' : 'overflow-y-auto'}`}>
        {activeTab === 'games' && <GamesView />}
        {activeTab === 'voting' && <VotingView />}
        {activeTab === 'account' && <AccountView />}
        {activeTab === 'admin' && <AdminView />}
      </main>

      {/* Sticky Mobile Bottom Navigation Bar (Visible only on small screens) */}
      <nav className="md:hidden fixed bottom-0 left-0 right-0 h-16 bg-white dark:bg-slate-900 border-t border-slate-200 dark:border-slate-800 flex items-center justify-around z-40 transition-colors">
        <button
          onClick={() => setActiveTab('games')}
          className={`flex flex-col items-center justify-center flex-1 py-1.5 transition-colors ${
            activeTab === 'games' ? 'text-indigo-600 dark:text-indigo-400' : 'text-slate-400'
          }`}
        >
          <Gamepad2 className="w-5 h-5" />
          <span className="text-[10px] font-bold mt-1">{t('nav.games')}</span>
        </button>
        <button
          onClick={() => setActiveTab('voting')}
          className={`flex flex-col items-center justify-center flex-1 py-1.5 transition-colors ${
            activeTab === 'voting' ? 'text-indigo-600 dark:text-indigo-400' : 'text-slate-400'
          }`}
        >
          <Vote className="w-5 h-5" />
          <span className="text-[10px] font-bold mt-1">{t('nav.voting')}</span>
        </button>
        <button
          onClick={() => setActiveTab('account')}
          className={`flex flex-col items-center justify-center flex-1 py-1.5 transition-colors ${
            activeTab === 'account' ? 'text-indigo-600 dark:text-indigo-400' : 'text-slate-400'
          }`}
        >
          <UserIcon className="w-5 h-5" />
          <span className="text-[10px] font-bold mt-1">{t('nav.account')}</span>
        </button>
        {user?.role === 'admin' && (
          <button
            onClick={() => setActiveTab('admin')}
            className={`flex flex-col items-center justify-center flex-1 py-1.5 transition-colors ${
              activeTab === 'admin' ? 'text-red-600 dark:text-red-400' : 'text-slate-400'
            }`}
          >
            <ShieldAlert className="w-5 h-5" />
            <span className="text-[10px] font-bold mt-1">{t('nav.admin')}</span>
          </button>
        )}
      </nav>

    </div>
  );
};

function App() {
  return (
    <LanguageProvider>
      <ThemeProvider>
        <AuthProvider>
          <AppContent />
        </AuthProvider>
      </ThemeProvider>
    </LanguageProvider>
  );
}

export default App;
