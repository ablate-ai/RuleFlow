window.tailwind = window.tailwind || {};
window.tailwind.config = {
  corePlugins: {
    preflight: false,
  },
  theme: {
    extend: {
      colors: {
        rf: {
          green: '#18c865',
          greenDark: '#15b25a',
          bg: '#f3f4f6',
          card: '#ffffff',
          border: '#e5e7eb',
          text: '#111827',
          muted: '#6b7280',
          subtle: '#9ca3af',
          red: '#ef4444',
          blue: '#3b82f6',
          orange: '#f97316',
          purple: '#8b5cf6',
          cyan: '#06b6d4',
          pink: '#ec4899',
        },
      },
      boxShadow: {
        rf1: '0 1px 3px rgba(0,0,0,.08), 0 1px 2px rgba(0,0,0,.04)',
        rf2: '0 4px 6px -1px rgba(0,0,0,.07), 0 2px 4px -1px rgba(0,0,0,.04)',
        rf3: '0 10px 15px -3px rgba(0,0,0,.1), 0 4px 6px -2px rgba(0,0,0,.05)',
      },
      borderRadius: {
        rf: '8px',
        'rf-lg': '12px',
        'rf-xl': '16px',
      },
      keyframes: {
        'slide-up-fade': {
          '0%': { transform: 'translateY(14px)', opacity: '0' },
          '100%': { transform: 'translateY(0)', opacity: '1' },
        },
        'slide-in-left': {
          '0%': { transform: 'translateX(-100%)', opacity: '0' },
          '100%': { transform: 'translateX(0)', opacity: '1' },
        },
        'fade-in': {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
      },
      animation: {
        'slide-up-fade': 'slide-up-fade .25s ease',
        'slide-in-left': 'slide-in-left .25s ease',
        'fade-in': 'fade-in .2s ease',
      },
    },
  },
};
