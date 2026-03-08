import { writable } from 'svelte/store';
import type { Group } from '../lib/apis/groups';

export const allGroups = writable<Group[]>([]);
