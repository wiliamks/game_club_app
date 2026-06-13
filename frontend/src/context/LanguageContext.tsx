import React, { createContext, useContext, useState } from 'react';
import enUS from '../locales/en-US.json';
import ptBR from '../locales/pt-BR.json';
import esES from '../locales/es-ES.json';
import jaJP from '../locales/ja-JP.json';

const translations: Record<string, any> = {
  'en-US': enUS,
  'pt-BR': ptBR,
  'es-ES': esES,
  'ja-JP': jaJP,
};

type Language = 'en-US' | 'pt-BR' | 'es-ES' | 'ja-JP';

interface LanguageContextType {
  language: Language;
  setLanguage: (lang: Language) => void;
  t: (key: string, replacements?: Record<string, string | number>) => string;
}

const LanguageContext = createContext<LanguageContextType | undefined>(undefined);

export const LanguageProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [language, setLanguageState] = useState<Language>(() => {
    const saved = localStorage.getItem('language');
    if (saved && ['en-US', 'pt-BR', 'es-ES', 'ja-JP'].includes(saved)) {
      return saved as Language;
    }
    // Attempt browser language detection, default to en-US
    const browserLang = navigator.language;
    if (browserLang.startsWith('pt')) return 'pt-BR';
    if (browserLang.startsWith('es')) return 'es-ES';
    if (browserLang.startsWith('ja')) return 'ja-JP';
    return 'en-US';
  });

  const setLanguage = (lang: Language) => {
    setLanguageState(lang);
    localStorage.setItem('language', lang);
  };

  const t = (key: string, replacements?: Record<string, string | number>): string => {
    const keys = key.split('.');
    let value = translations[language];
    
    for (const k of keys) {
      if (value && value[k] !== undefined) {
        value = value[k];
      } else {
        // Fallback to English if translation is missing in chosen locale
        let fallback = translations['en-US'];
        for (const fk of keys) {
          if (fallback && fallback[fk] !== undefined) {
            fallback = fallback[fk];
          } else {
            return key;
          }
        }
        value = fallback;
        break;
      }
    }

    if (typeof value !== 'string') {
      return key;
    }

    if (replacements) {
      let result = value;
      Object.entries(replacements).forEach(([k, v]) => {
        result = result.replace(`{${k}}`, String(v));
      });
      return result;
    }

    return value;
  };

  return (
    <LanguageContext.Provider value={{ language, setLanguage, t }}>
      {children}
    </LanguageContext.Provider>
  );
};

export const useLanguage = () => {
  const context = useContext(LanguageContext);
  if (!context) {
    throw new Error('useLanguage must be used within a LanguageProvider');
  }
  return context;
};
