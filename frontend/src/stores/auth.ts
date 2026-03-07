import { writable, derived } from 'svelte/store';
import type { UserResponse } from '../lib/apis/auth';

export const user = writable<UserResponse | null>(null);
export const isSignedIn = derived(user, ($u) => $u !== null);
