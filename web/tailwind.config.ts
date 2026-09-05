/** @type {import('tailwindcss').Config} */
import { type Config } from 'tailwindcss'

export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        bg: '#0a0a0f',
        bgAlt: '#0f0f15',
        surface: '#11111a',
        surfaceAlt: '#181820',
        border: '#232333',
        borderFocus: '#00d4ff',
        text: '#e0e0e8',
        textMuted: '#7a7a8a',
        textDim: '#5a5a6a',
        accent: '#00d4ff',
        accentHover: '#0099cc',
        green: '#4ade80',
        greenDim: '#22c55e',
        red: '#f87171',
        redDim: '#ef4444',
        yellow: '#fbbf24',
        yellowDim: '#eab308',
        blue: '#60a5fa',
        blueDim: '#3b82f6',
        purple: '#a855f7',
        orange: '#fb923c',
        cyan: '#06b6d4',
      },
      fontFamily: {
        mono: ['Fira Code', 'Monaco', 'Consolas', 'monospace'],
        sans: ['Inter', 'system-ui', 'sans-serif'],
      },
      boxShadow: {
        neon: '0 0 12px rgba(0, 212, 255, 0.3), 0 0 24px rgba(0, 212, 255, 0.1)',
        neonGreen: '0 0 8px rgba(74, 222, 128, 0.3), 0 0 16px rgba(74, 222, 128, 0.1)',
        neonRed: '0 0 8px rgba(248, 113, 113, 0.3), 0 0 16px rgba(248, 113, 113, 0.1)',
        neonYellow: '0 0 8px rgba(251, 191, 36, 0.3), 0 0 16px rgba(251, 191, 36, 0.1)',
      },
      keyframes: {
        'pulse-slow': {
          '0%, 100%': { opacity: '1' },
          '50%': { opacity: '0.5' },
        },
        'bounce-subtle': {
          '0%, 100%': { transform: 'translateY(0)' },
          '50%': { transform: 'translateY(-4px)' },
        },
        'spin-slow': {
          '0%': { transform: 'rotate(0deg)' },
          '100%': { transform: 'rotate(360deg)' },
        },
      },
      animation: {
        'pulse-slow': 'pulse-slow 2s ease-in-out infinite',
        'bounce-subtle': 'bounce-subtle 2s ease-in-out infinite',
        'spin-slow': 'spin-slow 8s linear infinite',
      },
    },
  },
  plugins: [],
} satisfies Config
