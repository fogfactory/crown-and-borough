import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from 'react'

import {
  DEFAULT_LANGUAGE,
  LANGUAGE_STORAGE_KEY,
  type Language,
  type MessageKey,
  translate,
  type Translate,
} from './messages'

interface LanguageContextValue {
  language: Language
  setLanguage: (language: Language) => void
  t: Translate
}

const defaultContext: LanguageContextValue = {
  language: DEFAULT_LANGUAGE,
  setLanguage: () => undefined,
  t: (key, values) => translate(DEFAULT_LANGUAGE, key, values),
}

const LanguageContext = createContext<LanguageContextValue>(defaultContext)

interface LanguageProviderProps {
  children: ReactNode
  initialLanguage?: Language
}

function storedLanguage(): Language {
  if (typeof window === 'undefined') return DEFAULT_LANGUAGE
  const value = window.localStorage.getItem(LANGUAGE_STORAGE_KEY)
  return value === 'fr' || value === 'en' ? value : DEFAULT_LANGUAGE
}

export function LanguageProvider({ children, initialLanguage }: LanguageProviderProps) {
  const [language, setLanguage] = useState<Language>(initialLanguage ?? storedLanguage)
  const t = useCallback<Translate>(
    (key: MessageKey, values?: Record<string, string | number>) =>
      translate(language, key, values),
    [language],
  )

  useEffect(() => {
    window.localStorage.setItem(LANGUAGE_STORAGE_KEY, language)
    document.documentElement.lang = language
  }, [language])

  return (
    <LanguageContext.Provider value={{ language, setLanguage, t }}>
      {children}
    </LanguageContext.Provider>
  )
}

export function useLanguage(): LanguageContextValue {
  return useContext(LanguageContext)
}
