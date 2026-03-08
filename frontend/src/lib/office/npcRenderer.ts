import type { WorkerStatus } from '../../types/worker';

export interface OfficeNPC {
	workerId: string;
	name: string;
	groupName: string;
	status: WorkerStatus;
	tileX: number;
	tileY: number;
	activeJobCount: number;
}

export interface GroupZoneLabel {
	groupName: string;
	tileX: number;
	tileY: number;
}

type StatusVisual = {
	glow: string;
	body: string;
	badge: string;
	pulse: boolean;
	showAlert: boolean;
};

function visualForStatus(status: WorkerStatus): StatusVisual {
	if (status === 'idle') {
		return {
			glow: 'rgba(74,222,128,0.35)',
			body: '#4ade80',
			badge: '#4ade80',
			pulse: false,
			showAlert: false
		};
	}
	if (status === 'busy') {
		return {
			glow: 'rgba(251,146,60,0.45)',
			body: '#fb923c',
			badge: '#fb923c',
			pulse: true,
			showAlert: false
		};
	}
	if (status === 'pending_user') {
		return {
			glow: 'rgba(96,165,250,0.45)',
			body: '#60a5fa',
			badge: '#60a5fa',
			pulse: true,
			showAlert: true
		};
	}
	return {
		glow: 'rgba(107,114,128,0.2)',
		body: '#6b7280',
		badge: '#6b7280',
		pulse: false,
		showAlert: false
	};
}

function drawDesk(
	ctx: CanvasRenderingContext2D,
	screenX: number,
	screenY: number,
	tileSize: number,
	highlighted: boolean
) {
	const deskW = Math.floor(tileSize * 0.78);
	const deskH = Math.floor(tileSize * 0.34);
	const x = Math.round(screenX - deskW / 2);
	const y = Math.round(screenY + tileSize * 0.06);

	ctx.fillStyle = highlighted ? 'rgba(139,92,246,0.45)' : 'rgba(139,92,246,0.26)';
	ctx.fillRect(x, y, deskW, deskH);
	ctx.fillStyle = 'rgba(10,10,22,0.8)';
	ctx.fillRect(x + 2, y + 2, deskW - 4, deskH - 4);
}

function drawNpcSprite(
	ctx: CanvasRenderingContext2D,
	screenX: number,
	screenY: number,
	status: WorkerStatus,
	tick: number,
	highlighted: boolean
) {
	const visual = visualForStatus(status);
	const pulseScale = visual.pulse ? 1 + Math.sin(tick * 0.006) * 0.12 : 1;
	const glowRadius = highlighted ? 22 : 18;

	ctx.save();
	ctx.globalAlpha = 0.92;
	ctx.fillStyle = visual.glow;
	ctx.beginPath();
	ctx.arc(screenX, screenY - 10, glowRadius * pulseScale, 0, Math.PI * 2);
	ctx.fill();
	ctx.restore();

	const px = Math.round(screenX - 8);
	const py = Math.round(screenY - 20);

	ctx.fillStyle = '#f8fafc';
	ctx.fillRect(px + 5, py + 1, 6, 5);

	ctx.fillStyle = visual.body;
	ctx.fillRect(px + 3, py + 6, 10, 8);

	ctx.fillStyle = '#d1d5db';
	ctx.fillRect(px + 5, py + 14, 2, 4);
	ctx.fillRect(px + 9, py + 14, 2, 4);

	if (visual.showAlert) {
		ctx.fillStyle = '#e2e8f0';
		ctx.fillRect(px + 13, py + 1, 3, 7);
		ctx.fillRect(px + 13, py + 9, 3, 2);
	}

	ctx.fillStyle = visual.badge;
	ctx.beginPath();
	ctx.arc(screenX + 8, screenY - 22, 3.6, 0, Math.PI * 2);
	ctx.fill();
}

function drawLabels(
	ctx: CanvasRenderingContext2D,
	screenX: number,
	screenY: number,
	npc: OfficeNPC,
	highlighted: boolean
) {
	ctx.textAlign = 'center';
	ctx.font = '600 12px "Space Grotesk", sans-serif';
	ctx.fillStyle = highlighted ? '#d8ccff' : '#bba8ff';
	ctx.fillText(npc.name, screenX, screenY + 30);

	ctx.font = '500 10px Inter, sans-serif';
	ctx.fillStyle = 'rgba(196,181,253,0.66)';
	ctx.fillText(npc.groupName, screenX, screenY + 44);

	if (npc.activeJobCount > 0) {
		const badge = String(npc.activeJobCount);
		ctx.font = '700 9px Inter, sans-serif';
		const width = ctx.measureText(badge).width + 9;
		const bx = screenX + 14;
		const by = screenY - 30;
		ctx.fillStyle = 'rgba(124,58,237,0.88)';
		ctx.fillRect(bx - width / 2, by, width, 14);
		ctx.fillStyle = '#f5f3ff';
		ctx.fillText(badge, bx, by + 10);
	}
}

export function drawGroupLabels(
	ctx: CanvasRenderingContext2D,
	labels: GroupZoneLabel[],
	tileSize: number,
	cameraX: number,
	cameraY: number
) {
	ctx.textAlign = 'left';
	for (const label of labels) {
		const x = label.tileX * tileSize - cameraX;
		const y = label.tileY * tileSize - cameraY;
		ctx.fillStyle = 'rgba(167,139,250,0.12)';
		ctx.fillRect(x - 10, y - 22, 240, 22);
		ctx.strokeStyle = 'rgba(167,139,250,0.28)';
		ctx.strokeRect(x - 10, y - 22, 240, 22);
		ctx.font = '700 11px "Space Grotesk", sans-serif';
		ctx.fillStyle = 'rgba(196,181,253,0.85)';
		ctx.fillText(`NEXUS OFFICE — ${label.groupName}`, x, y - 8);
	}
}

export function drawNPCs(
	ctx: CanvasRenderingContext2D,
	npcs: OfficeNPC[],
	tileSize: number,
	cameraX: number,
	cameraY: number,
	tick: number,
	highlightedWorkerId: string | null
) {
	const sorted = [...npcs].sort((a, b) => a.tileY - b.tileY || a.tileX - b.tileX);
	for (const npc of sorted) {
		const worldX = npc.tileX * tileSize + tileSize / 2;
		const worldY = npc.tileY * tileSize + tileSize / 2;
		const screenX = worldX - cameraX;
		const screenY = worldY - cameraY;
		const highlighted = npc.workerId === highlightedWorkerId;
		drawDesk(ctx, screenX, screenY, tileSize, highlighted);
		drawNpcSprite(ctx, screenX, screenY, npc.status, tick, highlighted);
		drawLabels(ctx, screenX, screenY, npc, highlighted);
	}
}

export function hitTestNPC(
	worldX: number,
	worldY: number,
	npcs: OfficeNPC[],
	tileSize: number
): string | null {
	for (const npc of npcs) {
		const centerX = npc.tileX * tileSize + tileSize / 2;
		const centerY = npc.tileY * tileSize + tileSize / 2 - 10;
		const halfW = 14;
		const halfH = 18;
		if (
			worldX >= centerX - halfW &&
			worldX <= centerX + halfW &&
			worldY >= centerY - halfH &&
			worldY <= centerY + halfH
		) {
			return npc.workerId;
		}
	}
	return null;
}

