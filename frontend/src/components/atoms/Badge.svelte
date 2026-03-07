<script lang="ts">
	export let color: 'gray' | 'blue' | 'yellow' | 'green' | 'red' | 'purple' | 'orange' = 'gray';
</script>

<span class="badge badge-{color}">
	<slot />
</span>

<style>
	.badge {
		display: inline-flex;
		align-items: center;
		border-radius: 5px;
		padding: 2px 8px;
		font-size: 11px;
		font-weight: 500;
		letter-spacing: 0.02em;
		border: 1px solid transparent;
	}

	.badge-gray   { background: rgba(148,163,184,0.08); color: rgba(196,181,253,0.6); border-color: rgba(148,163,184,0.15); }
	.badge-blue   { background: rgba(79,70,229,0.12);   color: #a5b4fc;              border-color: rgba(79,70,229,0.25); }
	.badge-yellow { background: rgba(245,158,11,0.1);   color: #fcd34d;              border-color: rgba(245,158,11,0.2); }
	.badge-green  { background: rgba(52,211,153,0.1);   color: #6ee7b7;              border-color: rgba(52,211,153,0.2); }
	.badge-red    { background: rgba(239,68,68,0.1);    color: #fca5a5;              border-color: rgba(239,68,68,0.2); }
	.badge-purple { background: rgba(139,92,246,0.12);  color: #c4b5fd;              border-color: rgba(139,92,246,0.25); }
	.badge-orange { background: rgba(249,115,22,0.1);   color: #fdba74;              border-color: rgba(249,115,22,0.2); }
</style>
