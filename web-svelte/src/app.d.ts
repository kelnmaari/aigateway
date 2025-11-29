// See https://svelte.dev/docs/kit/types#app.d.ts
// for information about these interfaces
declare global {
	namespace App {
		interface Error {
			message: string;
			code?: string;
		}
		interface Locals {
			user?: {
				id: string;
				username: string;
				email: string;
				is_admin: boolean;
			};
		}
		interface PageData {
			title?: string;
		}
		interface PageState {}
		interface Platform {}
	}
}

export {};

