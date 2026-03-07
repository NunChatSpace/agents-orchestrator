<script lang="ts">
	export let jobId: string = '';

	const backendBase = `${import.meta.env.VITE_BACKEND_URL ?? ''}/api/v1`;

	type LineType = 'add' | 'del' | 'hunk' | 'ctx' | 'meta';
	interface DiffLine { type: LineType; text: string; }
	interface FileDiff { id: string; path: string; op: 'write' | 'edit' | 'delete'; lines: DiffLine[]; }

	let fileDiffs: FileDiff[] = [];
	let loading = false;
	let error = '';
	let expanded: Record<string, boolean> = {};

	function toggle(id: string) {
		expanded[id] = !expanded[id];
		expanded = expanded;
	}

	function renderDiff(diff: string): DiffLine[] {
		return diff.split('\n').map((line): DiffLine => {
			if (line.startsWith('@@')) return { type: 'hunk', text: line };
			if (line.startsWith('+') && !line.startsWith('+++')) return { type: 'add', text: line.slice(1) };
			if (line.startsWith('-') && !line.startsWith('---')) return { type: 'del', text: line.slice(1) };
			if (
				line.startsWith('diff --git') || line.startsWith('index ') ||
				line.startsWith('---') || line.startsWith('+++') ||
				line.startsWith('new file') || line.startsWith('deleted file') ||
				line.startsWith('old mode') || line.startsWith('new mode')
			) return { type: 'meta', text: line };
			return { type: 'ctx', text: line.startsWith(' ') ? line.slice(1) : line };
		});
	}

	function splitPatch(raw: string): FileDiff[] {
		const result: FileDiff[] = [];
		let current = '';
		let currentPath = '';
		for (const line of raw.split('\n')) {
			if (line.startsWith('diff --git ')) {
				if (currentPath && current) {
					result.push(makeEntry(currentPath, current));
				}
				current = line + '\n';
				const m = line.match(/diff --git a\/.+ b\/(.+)/);
				currentPath = m ? m[1] : line;
			} else {
				current += line + '\n';
			}
		}
		if (currentPath && current.trim()) result.push(makeEntry(currentPath, current));
		return result;
	}

	function makeEntry(path: string, diff: string): FileDiff {
		const isNew = diff.includes('\nnew file mode') || diff.includes('new file mode\n');
		const isDel = diff.includes('\ndeleted file mode') || diff.includes('deleted file mode\n');
		return {
			id: path,
			path,
			op: isNew ? 'write' : isDel ? 'delete' : 'edit',
			lines: renderDiff(diff)
		};
	}

	async function fetchPatch(id: string) {
		if (!id) return;
		loading = true;
		error = '';
		fileDiffs = [];
		try {
			const res = await fetch(`${backendBase}/jobs/${id}/patch`, { credentials: 'include' });
			if (!res.ok) {
				if (res.status === 404) { error = 'No changes captured for this job yet.'; }
				else { error = `Failed to load patch (${res.status})`; }
				return;
			}
			const text = await res.text();
			fileDiffs = splitPatch(text);
			// Auto-expand all files
			expanded = Object.fromEntries(fileDiffs.map((f) => [f.id, true]));
		} catch {
			error = 'Failed to load patch.';
		} finally {
			loading = false;
		}
	}

	$: fetchPatch(jobId);
</script>

<div class="changes-wrap">
	<div class="changes-header">
		<span class="changes-title">Changes</span>
		{#if fileDiffs.length > 0}
			<span class="changes-count">{fileDiffs.length} file{fileDiffs.length !== 1 ? 's' : ''}</span>
		{/if}
		{#if jobId && fileDiffs.length > 0}
			<a class="download-btn" href="{backendBase}/jobs/{jobId}/patch" download target="_self">
				<svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2">
					<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
					<polyline points="7 10 12 15 17 10"/>
					<line x1="12" y1="15" x2="12" y2="3"/>
				</svg>
				.patch
			</a>
		{/if}
	</div>

	{#if loading}
		<p class="empty">Loading…</p>
	{:else if error}
		<p class="empty">{error}</p>
	{:else if fileDiffs.length === 0}
		<p class="empty">No changes yet.</p>
	{:else}
		{#each fileDiffs as entry}
			<div class="file-entry">
				<button class="file-row" on:click={() => toggle(entry.id)} aria-expanded={expanded[entry.id] ?? false}>
					<span class="op-badge badge-{entry.op}">
						{entry.op === 'write' ? '+' : entry.op === 'delete' ? '−' : '~'}
					</span>
					<span class="file-path">{entry.path}</span>
					<span class="caret">{expanded[entry.id] ? '▲' : '▼'}</span>
				</button>
				{#if expanded[entry.id]}
					<div class="diff-panel">
						{#each entry.lines as line}
							{#if line.type === 'add'}
								<div class="diff-line diff-add"><span class="diff-sign">+</span><span class="diff-text">{line.text}</span></div>
							{:else if line.type === 'del'}
								<div class="diff-line diff-del"><span class="diff-sign">-</span><span class="diff-text">{line.text}</span></div>
							{:else if line.type === 'hunk'}
								<div class="diff-hunk">{line.text}</div>
							{:else if line.type === 'ctx'}
								<div class="diff-line diff-ctx"><span class="diff-sign"> </span><span class="diff-text">{line.text}</span></div>
							{/if}
						{/each}
					</div>
				{/if}
			</div>
		{/each}
	{/if}
</div>

<style>
	.changes-wrap {
		display: flex;
		flex-direction: column;
		height: 100%;
		overflow-y: auto;
	}

	.changes-header {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 12px 16px 10px;
		border-bottom: 1px solid rgba(139,92,246,0.12);
		flex-shrink: 0;
		position: sticky;
		top: 0;
		background: rgba(10,10,25,0.95);
		z-index: 1;
	}

	.changes-title {
		font-size: 12px;
		font-weight: 600;
		color: rgba(196,181,253,0.7);
		letter-spacing: 0.04em;
		text-transform: uppercase;
	}

	.changes-count {
		font-size: 10.5px;
		font-weight: 700;
		padding: 1px 6px;
		border-radius: 10px;
		background: rgba(139,92,246,0.15);
		border: 1px solid rgba(139,92,246,0.25);
		color: rgba(167,139,250,0.85);
	}

	.download-btn {
		margin-left: auto;
		display: inline-flex;
		align-items: center;
		gap: 5px;
		padding: 4px 10px;
		border-radius: 6px;
		font-size: 11px;
		font-weight: 500;
		color: rgba(167,139,250,0.7);
		border: 1px solid rgba(139,92,246,0.2);
		text-decoration: none;
		transition: background 0.15s, color 0.15s, border-color 0.15s;
	}
	.download-btn:hover {
		background: rgba(139,92,246,0.1);
		color: #a78bfa;
		border-color: rgba(139,92,246,0.35);
	}

	.empty {
		font-size: 12px;
		color: rgba(196,181,253,0.25);
		text-align: center;
		padding-top: 40px;
		font-style: italic;
	}

	.file-entry { border-bottom: 1px solid rgba(139,92,246,0.07); }

	.file-row {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 7px 12px;
		background: none;
		border: none;
		cursor: pointer;
		width: 100%;
		text-align: left;
	}
	.file-row:hover { background: rgba(139,92,246,0.05); }
	.file-row:hover .file-path { color: rgba(196,181,253,0.95); }

	.op-badge {
		width: 16px;
		height: 16px;
		border-radius: 3px;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 11px;
		font-weight: 700;
		flex-shrink: 0;
	}
	.badge-write {
		background: rgba(34,197,94,0.15);
		border: 1px solid rgba(34,197,94,0.35);
		color: rgba(134,239,172,0.9);
	}
	.badge-edit {
		background: rgba(234,179,8,0.12);
		border: 1px solid rgba(234,179,8,0.3);
		color: rgba(253,224,71,0.85);
	}
	.badge-delete {
		background: rgba(239,68,68,0.1);
		border: 1px solid rgba(239,68,68,0.3);
		color: rgba(252,165,165,0.85);
	}

	.file-path {
		flex: 1;
		font-size: 11.5px;
		font-family: 'SFMono-Regular', Consolas, monospace;
		color: rgba(196,181,253,0.7);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		min-width: 0;
	}

	.caret { font-size: 9px; color: rgba(139,92,246,0.4); flex-shrink: 0; }

	.diff-panel {
		border-top: 1px solid rgba(139,92,246,0.1);
		font-family: 'SFMono-Regular', Consolas, monospace;
		font-size: 11.5px;
		overflow-x: auto;
	}
	.diff-line {
		display: flex;
		gap: 6px;
		padding: 0 10px;
		white-space: pre;
		line-height: 1.6;
	}
	.diff-add { background: rgba(34,197,94,0.08); border-left: 2px solid rgba(34,197,94,0.35); }
	.diff-del { background: rgba(239,68,68,0.1); border-left: 2px solid rgba(239,68,68,0.4); }
	.diff-ctx { border-left: 2px solid transparent; }
	.diff-sign { flex-shrink: 0; width: 10px; user-select: none; }
	.diff-add .diff-sign { color: rgba(134,239,172,0.8); }
	.diff-del .diff-sign { color: rgba(252,165,165,0.8); }
	.diff-ctx .diff-sign { color: transparent; }
	.diff-add .diff-text { color: rgba(134,239,172,0.9); }
	.diff-del .diff-text { color: rgba(252,165,165,0.85); }
	.diff-ctx .diff-text { color: rgba(196,181,253,0.5); }

	.diff-hunk {
		padding: 2px 10px;
		font-family: 'SFMono-Regular', Consolas, monospace;
		font-size: 10.5px;
		color: rgba(139,92,246,0.6);
		background: rgba(139,92,246,0.06);
		border-left: 2px solid rgba(139,92,246,0.2);
		white-space: pre;
	}
</style>
