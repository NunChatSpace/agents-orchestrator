<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { getMe } from '../lib/apis/auth';
	import { user } from '../stores/auth';
	import { connectWS, disconnectWS } from '../stores/ws';
	import { get } from 'svelte/store';

	onMount(() => {
		const path = get(page).url.pathname;
		if (path === '/login') return;
		let cancelled = false;

		void (async () => {
			try {
				const u = await getMe();
				if (cancelled) return;
				user.set(u);
				connectWS();
			} catch {
				if (!cancelled) {
					goto('/login');
				}
			}
		})();

		return () => {
			cancelled = true;
			disconnectWS();
		};
	});
</script>

<slot />
