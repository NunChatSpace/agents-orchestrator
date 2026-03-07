<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { getMe } from '../lib/apis/auth';
	import { user } from '../stores/auth';
	import { connectWS, disconnectWS } from '../stores/ws';
	import { get } from 'svelte/store';

	onMount(async () => {
		const path = get(page).url.pathname;
		if (path === '/login') return;
		try {
			const u = await getMe();
			user.set(u);
			connectWS();
		} catch {
			goto('/login');
		}
		return () => disconnectWS();
	});
</script>

<slot />
