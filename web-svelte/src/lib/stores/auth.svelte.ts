import { browser } from '$app/environment';

interface User {
	id: string;
	username: string;
	email: string;
	full_name?: string;
	is_admin: boolean;
	role?: string;
}

interface AuthTokens {
	access_token: string;
	refresh_token: string;
	expires_at: number;
}

const ACCESS_TOKEN_KEY = 'access_token';
const REFRESH_TOKEN_KEY = 'refresh_token';
const USER_KEY = 'user';

class AuthStore {
	private _user = $state<User | null>(null);
	private _accessToken = $state<string | null>(null);
	private _refreshToken = $state<string | null>(null);
	private _initialized = false;

	init() {
		if (!browser || this._initialized) return;
		this._initialized = true;

		// Load from localStorage
		const accessToken = localStorage.getItem(ACCESS_TOKEN_KEY);
		const refreshToken = localStorage.getItem(REFRESH_TOKEN_KEY);
		const userJson = localStorage.getItem(USER_KEY);

		if (accessToken) {
			this._accessToken = accessToken;
		}
		if (refreshToken) {
			this._refreshToken = refreshToken;
		}
		if (userJson) {
			try {
				this._user = JSON.parse(userJson);
			} catch {
				localStorage.removeItem(USER_KEY);
			}
		}
	}

	get user() {
		return this._user;
	}

	get accessToken() {
		return this._accessToken;
	}

	get isAuthenticated() {
		return !!this._accessToken;
	}

	get isAdmin() {
		return this._user?.is_admin ?? false;
	}

	login(tokens: AuthTokens, user: User) {
		this._accessToken = tokens.access_token;
		this._refreshToken = tokens.refresh_token;
		this._user = user;

		if (browser) {
			localStorage.setItem(ACCESS_TOKEN_KEY, tokens.access_token);
			localStorage.setItem(REFRESH_TOKEN_KEY, tokens.refresh_token);
			localStorage.setItem(USER_KEY, JSON.stringify(user));
		}
	}

	logout() {
		this._accessToken = null;
		this._refreshToken = null;
		this._user = null;

		if (browser) {
			localStorage.removeItem(ACCESS_TOKEN_KEY);
			localStorage.removeItem(REFRESH_TOKEN_KEY);
			localStorage.removeItem(USER_KEY);
		}
	}

	updateUser(user: Partial<User>) {
		if (this._user) {
			this._user = { ...this._user, ...user };

			if (browser) {
				localStorage.setItem(USER_KEY, JSON.stringify(this._user));
			}
		}
	}

	updateTokens(tokens: AuthTokens) {
		this._accessToken = tokens.access_token;
		this._refreshToken = tokens.refresh_token;

		if (browser) {
			localStorage.setItem(ACCESS_TOKEN_KEY, tokens.access_token);
			localStorage.setItem(REFRESH_TOKEN_KEY, tokens.refresh_token);
		}
	}
}

export const authStore = new AuthStore();

