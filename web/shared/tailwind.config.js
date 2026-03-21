window.tailwind = window.tailwind || {};
window.tailwind.config = {
  corePlugins: {
    preflight: false,
  },
  theme: {
    extend: {
      colors: {
        rf: {
          green: '#12b76a',
          greenDark: '#079455',
          bg: '#eef2f7',
          card: '#ffffff',
          border: 'rgba(15,23,42,0.08)',
          text: '#101828',
          muted: '#475467',
          subtle: '#667085',
          red: '#f04438',
          blue: '#2e90fa',
          orange: '#f79009',
          purple: '#7a5af8',
          cyan: '#06aed4',
          pink: '#ee46bc',
        },
      },
      boxShadow: {
        rf1: '0 10px 30px rgba(15,23,42,.06)',
        rf2: '0 24px 48px rgba(15,23,42,.1)',
        rf3: '0 32px 64px rgba(15,23,42,.14)',
      },
      borderRadius: {
        rf: '14px',
        'rf-lg': '18px',
        'rf-xl': '28px',
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
