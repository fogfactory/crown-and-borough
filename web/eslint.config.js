import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'
import eslintConfigPrettier from 'eslint-config-prettier'
import { globalIgnores } from 'eslint/config'

// 6/8-digit hex always match; 3-digit hex only when it contains a letter a-f,
// so issue references such as '#196' are not mistaken for colors. Matched
// anywhere in the string, with a non-word boundary after it.
const HEX_COLOR =
  '/#(?:[0-9a-fA-F]{6}(?:[0-9a-fA-F]{2})?|(?=[0-9]*[a-fA-F])[0-9a-fA-F]{3})\\b/'
const HEX_COLOR_MESSAGE =
  'Hex color literals are forbidden here: use a Tailwind theme token from src/index.css, or a named constant from src/lib/map-palette.ts for SVG/inline colors.'

export default tseslint.config([
  globalIgnores(['dist']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      js.configs.recommended,
      tseslint.configs.recommended,
      reactHooks.configs.flat.recommended,
      reactRefresh.configs.vite,
      eslintConfigPrettier,
    ],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
    rules: {
      'react-refresh/only-export-components': 'off',
      // eslint-plugin-react-hooks v7 adds React Compiler rules. The app does not
      // use the compiler yet, so only the classic hook rules are enforced.
      'react-hooks/preserve-manual-memoization': 'off',
      'react-hooks/refs': 'off',
      'react-hooks/set-state-in-effect': 'off',
    },
  },
  {
    // Raw colors live in the theme tokens (src/index.css) or in the SVG
    // palette module; tests may assert on literal values.
    files: ['src/**/*.{ts,tsx}'],
    ignores: [
      'src/lib/map-palette.ts',
      'src/**/*.test.{ts,tsx}',
      'src/fixtures/**',
      'src/test/**',
    ],
    rules: {
      'no-restricted-syntax': [
        'error',
        {
          selector: `Literal[value=${HEX_COLOR}]`,
          message: HEX_COLOR_MESSAGE,
        },
        {
          selector: `TemplateElement[value.raw=${HEX_COLOR}]`,
          message: HEX_COLOR_MESSAGE,
        },
        {
          selector: `JSXText[value=${HEX_COLOR}]`,
          message: HEX_COLOR_MESSAGE,
        },
      ],
    },
  },
])
