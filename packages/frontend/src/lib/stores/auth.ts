/**
 * Auth Store
 *
 * Manages authentication state. Checks /api/auth/google/status for login status.
 */

import { writable, derived } from 'svelte/store';
import { api } from '../api/client';

interface AuthState {
	loggedIn: boolean;
	email: string | null;
	loading: boolean;
	checked: boolean;
}

function createAuthStore() {
	const { subscribe, set, update } = writable<AuthState>({
		loggedIn: false,
		email: null,
		loading: true,
		checked: false
	});

	return {
		subscribe,

		async check() {
			update((s) => ({ ...s, loading: true }));
			try {
				const status = await api.auth.status();
				update(() => ({
					loggedIn: status.loggedIn,
					email: status.email || null,
					loading: false,
					checked: true
				}));
				return status.loggedIn;
			} catch {
				update(() => ({
					loggedIn: false,
					email: null,
					loading: false,
					checked: true
				}));
				return false;
			}
		},

		async login() {
			const result = await api.calendar.getAuthUrl();
			window.location.href = result.url;
		},

		async logout() {
			try {
				await api.auth.logout();
			} catch {
				// Ignore errors — we're logging out
			}
			set({ loggedIn: false, email: null, loading: false, checked: true });
		}
	};
}

export const authStore = createAuthStore();

export const isLoggedIn = derived(authStore, ($store) => $store.loggedIn);
export const authChecked = derived(authStore, ($store) => $store.checked);
