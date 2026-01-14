/** @type {import('tailwindcss').Config} */
export default {
	content: ['./src/**/*.{html,js,svelte,ts}'],
	theme: {
		extend: {
			colors: {
				trip: {
					conference: {
						bg: '#fed7aa',
						text: '#9a3412'
					},
					vacation: {
						bg: '#bfdbfe',
						text: '#1e40af'
					},
					business: {
						bg: '#fde68a',
						text: '#92400e'
					},
					family: {
						bg: '#bbf7d0',
						text: '#166534'
					},
					other: {
						bg: '#e9d5ff',
						text: '#6b21a8'
					}
				},
				item: {
					flight: '#3b82f6',
					hotel: '#8b5cf6',
					train: '#10b981',
					event: '#f97316',
					drive: '#6b7280'
				}
			}
		}
	},
	plugins: []
};
