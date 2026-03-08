import type { MapConfig } from './mapConfig';
import { PlayerController, type VirtualDirection } from './playerController';
import { findNearestDeskInRange } from './proximityDetector';
import { drawTiles } from './tileRenderer';
import { drawGroupLabels, drawNPCs, hitTestNPC, type GroupZoneLabel, type OfficeNPC } from './npcRenderer';

type Facing = 'up' | 'down' | 'left' | 'right';

export interface PlayerState {
	x: number;
	y: number;
	facing: Facing;
	moving: boolean;
}

export interface OfficeEngineCallbacks {
	onNearestWorkerChange?: (workerId: string | null) => void;
	onInteract?: (workerId: string) => void;
}

function clamp(value: number, min: number, max: number): number {
	return Math.max(min, Math.min(max, value));
}

export class OfficeEngine {
	private readonly canvas: HTMLCanvasElement;
	private readonly ctx: CanvasRenderingContext2D;
	private readonly map: MapConfig;
	private readonly callbacks: OfficeEngineCallbacks;
	private readonly controller = new PlayerController();
	private readonly moveSpeed: number;
	private readonly playerHalfSize = 8;

	private workers: OfficeNPC[] = [];
	private groupLabels: GroupZoneLabel[] = [];
	private blockedTiles = new Set<string>();
	private nearestWorkerId: string | null = null;

	private frameId: number | null = null;
	private running = false;
	private lastTime = 0;
	private viewportWidth = 0;
	private viewportHeight = 0;

	private cameraX = 0;
	private cameraY = 0;

	private player: PlayerState;

	constructor(
		canvas: HTMLCanvasElement,
		mapConfig: MapConfig,
		callbacks: OfficeEngineCallbacks = {}
	) {
		const ctx = canvas.getContext('2d');
		if (!ctx) throw new Error('Canvas 2D context is not available');

		this.canvas = canvas;
		this.ctx = ctx;
		this.map = mapConfig;
		this.callbacks = callbacks;
		this.moveSpeed = this.map.tileSize * 4;
		this.player = {
			x: this.map.tileSize * 4 + this.map.tileSize / 2,
			y: this.map.tileSize * 4 + this.map.tileSize / 2,
			facing: 'down',
			moving: false
		};
	}

	start() {
		if (this.running) return;
		this.running = true;
		this.controller.attach();
		this.lastTime = performance.now();
		this.frameId = requestAnimationFrame(this.loop);
	}

	stop() {
		if (!this.running) return;
		this.running = false;
		if (this.frameId !== null) {
			cancelAnimationFrame(this.frameId);
			this.frameId = null;
		}
		this.controller.detach();
	}

	destroy() {
		this.stop();
	}

	resize(width: number, height: number) {
		const safeWidth = Math.max(1, Math.floor(width));
		const safeHeight = Math.max(1, Math.floor(height));
		const dpr = window.devicePixelRatio || 1;

		this.viewportWidth = safeWidth;
		this.viewportHeight = safeHeight;

		this.canvas.width = Math.floor(safeWidth * dpr);
		this.canvas.height = Math.floor(safeHeight * dpr);
		this.canvas.style.width = `${safeWidth}px`;
		this.canvas.style.height = `${safeHeight}px`;
		this.ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
	}

	setWorkers(workers: OfficeNPC[], labels: GroupZoneLabel[]) {
		this.workers = workers;
		this.groupLabels = labels;
		this.rebuildBlockedTiles();
		this.recalculateNearest();
	}

	setVirtualDirection(direction: VirtualDirection, active: boolean) {
		this.controller.setVirtualDirection(direction, active);
	}

	interactWithNearest() {
		if (!this.nearestWorkerId) return;
		this.callbacks.onInteract?.(this.nearestWorkerId);
	}

	handleCanvasClick(screenX: number, screenY: number) {
		const worldX = screenX + this.cameraX;
		const worldY = screenY + this.cameraY;
		const workerId = hitTestNPC(worldX, worldY, this.workers, this.map.tileSize);
		if (workerId) {
			this.callbacks.onInteract?.(workerId);
		}
	}

	getNearestWorkerId(): string | null {
		return this.nearestWorkerId;
	}

	private readonly loop = (timestamp: number) => {
		if (!this.running) return;
		const deltaMs = Math.min(40, timestamp - this.lastTime);
		this.lastTime = timestamp;
		this.update(deltaMs / 1000);
		this.render(timestamp);
		this.frameId = requestAnimationFrame(this.loop);
	};

	private update(deltaSeconds: number) {
		const vector = this.controller.getVector();
		this.player.moving = vector.x !== 0 || vector.y !== 0;

		if (vector.x !== 0) {
			this.player.facing = vector.x > 0 ? 'right' : 'left';
		} else if (vector.y !== 0) {
			this.player.facing = vector.y > 0 ? 'down' : 'up';
		}

		const moveAmount = this.moveSpeed * deltaSeconds;
		if (vector.x !== 0) {
			this.tryMoveTo(this.player.x + vector.x * moveAmount, this.player.y);
		}
		if (vector.y !== 0) {
			this.tryMoveTo(this.player.x, this.player.y + vector.y * moveAmount);
		}

		const targetX = this.player.x - this.viewportWidth / 2;
		const targetY = this.player.y - this.viewportHeight / 2;
		const maxCameraX = Math.max(0, this.map.width * this.map.tileSize - this.viewportWidth);
		const maxCameraY = Math.max(0, this.map.height * this.map.tileSize - this.viewportHeight);
		this.cameraX = clamp(this.cameraX + (targetX - this.cameraX) * 0.12, 0, maxCameraX);
		this.cameraY = clamp(this.cameraY + (targetY - this.cameraY) * 0.12, 0, maxCameraY);

		this.recalculateNearest();
	}

	private render(timestamp: number) {
		if (this.viewportWidth <= 0 || this.viewportHeight <= 0) return;

		this.ctx.clearRect(0, 0, this.viewportWidth, this.viewportHeight);
		drawTiles(
			this.ctx,
			this.map,
			this.cameraX,
			this.cameraY,
			this.viewportWidth,
			this.viewportHeight
		);
		drawGroupLabels(this.ctx, this.groupLabels, this.map.tileSize, this.cameraX, this.cameraY);
		drawNPCs(
			this.ctx,
			this.workers,
			this.map.tileSize,
			this.cameraX,
			this.cameraY,
			timestamp,
			this.nearestWorkerId
		);
		this.drawPlayer();
	}

	private drawPlayer() {
		const x = this.player.x - this.cameraX;
		const y = this.player.y - this.cameraY;
		const px = Math.round(x - 8);
		const py = Math.round(y - 18);

		this.ctx.fillStyle = 'rgba(124,58,237,0.4)';
		this.ctx.beginPath();
		this.ctx.arc(x, y - 8, 18, 0, Math.PI * 2);
		this.ctx.fill();

		this.ctx.fillStyle = '#f8fafc';
		this.ctx.fillRect(px + 5, py + 1, 6, 5);
		this.ctx.fillStyle = '#8b5cf6';
		this.ctx.fillRect(px + 3, py + 6, 10, 8);
		this.ctx.fillStyle = '#ddd6fe';
		this.ctx.fillRect(px + 5, py + 14, 2, 4);
		this.ctx.fillRect(px + 9, py + 14, 2, 4);

		this.ctx.textAlign = 'center';
		this.ctx.font = '700 12px "Space Grotesk", sans-serif';
		this.ctx.fillStyle = '#e9ddff';
		this.ctx.fillText('You', x, py - 4);
	}

	private tryMoveTo(nextX: number, nextY: number) {
		if (this.isBlocked(nextX, nextY)) return;
		this.player.x = nextX;
		this.player.y = nextY;
	}

	private isBlocked(pixelX: number, pixelY: number): boolean {
		const minX = pixelX - this.playerHalfSize;
		const maxX = pixelX + this.playerHalfSize;
		const minY = pixelY - this.playerHalfSize;
		const maxY = pixelY + this.playerHalfSize;

		const corners = [
			[minX, minY],
			[maxX, minY],
			[minX, maxY],
			[maxX, maxY]
		] as const;

		for (const [x, y] of corners) {
			const tileX = Math.floor(x / this.map.tileSize);
			const tileY = Math.floor(y / this.map.tileSize);
			if (tileX < 0 || tileY < 0 || tileX >= this.map.width || tileY >= this.map.height) {
				return true;
			}
			const tile = this.map.tiles[tileY]?.[tileX];
			if (tile === 'wall' || tile === 'void') return true;
			if (this.blockedTiles.has(`${tileX},${tileY}`)) return true;
		}

		return false;
	}

	private rebuildBlockedTiles() {
		this.blockedTiles = new Set<string>();
		for (const worker of this.workers) {
			this.blockedTiles.add(`${worker.tileX},${worker.tileY}`);
		}
	}

	private recalculateNearest() {
		const nearest = findNearestDeskInRange(
			this.player.x,
			this.player.y,
			this.map.tileSize,
			this.workers.map((worker) => ({
				workerId: worker.workerId,
				tileX: worker.tileX,
				tileY: worker.tileY
			})),
			2
		);

		const nearestId = nearest?.workerId ?? null;
		if (nearestId !== this.nearestWorkerId) {
			this.nearestWorkerId = nearestId;
			this.callbacks.onNearestWorkerChange?.(nearestId);
		}
	}
}

