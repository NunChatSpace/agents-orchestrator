import { GET, POST } from './client';

export interface UserResponse {
	user_id: string;
	username: string;
}

export const login = (username: string, password: string) =>
	POST<UserResponse>('/auth/login', { username, password });

export const logout = () => POST<null>('/auth/logout');

export const getMe = () => GET<UserResponse>('/auth/me');
