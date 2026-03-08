import { GET, POST } from './client';

export interface Group {
	group_id: string;
	name: string;
}

export const listGroups = () => GET<Group[]>('/groups');

export const createGroup = (name: string) => POST<Group>('/groups', { name });
