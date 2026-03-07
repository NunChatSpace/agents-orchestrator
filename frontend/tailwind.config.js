/** @type {import('tailwindcss').Config} */
export default {
	content: ['./src/**/*.{html,js,svelte,ts}'],
	theme: {
		extend: {
			colors: {
				nx: {
					bg:      '#05050c',
					surface: '#0e0e20',
					s2:      '#131326',
					violet:  '#7c3aed',
					vb:      '#8b5cf6',
					lav:     '#a78bfa',
					soft:    '#c4b5fd'
				}
			},
			fontFamily: {
				sans:    ['Inter', 'system-ui', 'sans-serif'],
				display: ['Space Grotesk', 'system-ui', 'sans-serif']
			},
			boxShadow: {
				glow:      '0 0 20px rgba(124,58,237,0.35)',
				'glow-lg': '0 0 32px rgba(124,58,237,0.55)',
				card:      '0 0 28px rgba(124,58,237,0.07), 0 8px 24px rgba(0,0,0,0.25)'
			}
		}
	},
	plugins: []
};
