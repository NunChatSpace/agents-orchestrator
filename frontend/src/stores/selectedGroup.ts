import { writable } from 'svelte/store';

export const selectedGroup = writable<string>('');
