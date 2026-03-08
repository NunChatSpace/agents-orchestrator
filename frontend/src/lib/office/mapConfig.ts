import type { Worker } from '../../types/worker';

export type TileType = 'floor' | 'wall' | 'desk' | 'void';

export interface MapConfig {
	width: number;
	height: number;
	tileSize: number;
	tiles: TileType[][];
}

export interface DeskConfig {
	workerId: string;
	tileX: number;
	tileY: number;
	labelOffset?: { x: number; y: number };
}

export interface DeskPosition {
	tileX: number;
	tileY: number;
	labelOffset?: { x: number; y: number };
}

const MAP_WIDTH = 72;
const MAP_HEIGHT = 44;
const TILE_SIZE = 32;

function buildTiles(width: number, height: number): TileType[][] {
	const tiles: TileType[][] = Array.from({ length: height }, () =>
		Array.from({ length: width }, () => 'floor' as TileType)
	);

	for (let y = 0; y < height; y++) {
		for (let x = 0; x < width; x++) {
			if (x === 0 || y === 0 || x === width - 1 || y === height - 1) {
				tiles[y][x] = 'wall';
			}
		}
	}

	for (let x = 2; x < width - 2; x++) {
		if (x >= 30 && x <= 40) continue;
		tiles[14][x] = 'wall';
		tiles[28][x] = 'wall';
	}

	return tiles;
}

export const MAP_CONFIG: MapConfig = {
	width: MAP_WIDTH,
	height: MAP_HEIGHT,
	tileSize: TILE_SIZE,
	tiles: buildTiles(MAP_WIDTH, MAP_HEIGHT)
};

// workerId accepts real worker_id or worker name for local fallback mapping.
export const DESK_CONFIGS: DeskConfig[] = [
	{ workerId: 'fi-backend1', tileX: 9, tileY: 8 },
	{ workerId: 'fi-backend2', tileX: 17, tileY: 8 },
	{ workerId: 'fi-frontend1', tileX: 9, tileY: 22 },
	{ workerId: 'fi-frontend2', tileX: 17, tileY: 22 },
	{ workerId: 'ib-kha', tileX: 9, tileY: 36 }
];

type GroupFallback = {
	startX: number;
	startY: number;
	columns: number;
	colGap: number;
	rowGap: number;
};

const GROUP_FALLBACKS: Record<string, GroupFallback> = {
	'fi-backend': { startX: 9, startY: 8, columns: 4, colGap: 8, rowGap: 4 },
	'fi-frontend': { startX: 9, startY: 22, columns: 4, colGap: 8, rowGap: 4 },
	'ib-kha': { startX: 9, startY: 36, columns: 3, colGap: 8, rowGap: 4 }
};

function fallbackForGroup(groupName: string): GroupFallback {
	const existing = GROUP_FALLBACKS[groupName];
	if (existing) return existing;

	const hashed = Array.from(groupName).reduce((acc, char) => acc + char.charCodeAt(0), 0);
	const zone = hashed % 3;
	if (zone === 0) return { startX: 36, startY: 8, columns: 4, colGap: 7, rowGap: 4 };
	if (zone === 1) return { startX: 36, startY: 22, columns: 4, colGap: 7, rowGap: 4 };
	return { startX: 36, startY: 36, columns: 4, colGap: 7, rowGap: 4 };
}

function configuredDesk(worker: Worker): DeskConfig | null {
	const desk = DESK_CONFIGS.find(
		(config) => config.workerId === worker.worker_id || config.workerId === worker.name
	);
	return desk ?? null;
}

function clampTileX(x: number): number {
	return Math.max(1, Math.min(MAP_CONFIG.width - 2, x));
}

function clampTileY(y: number): number {
	return Math.max(1, Math.min(MAP_CONFIG.height - 2, y));
}

export function resolveDeskPosition(worker: Worker, indexInGroup: number): DeskPosition {
	if (worker.map_x > 0 || worker.map_y > 0) {
		return {
			tileX: clampTileX(worker.map_x),
			tileY: clampTileY(worker.map_y)
		};
	}

	const config = configuredDesk(worker);
	if (config) {
		return {
			tileX: clampTileX(config.tileX),
			tileY: clampTileY(config.tileY),
			labelOffset: config.labelOffset
		};
	}

	const fallback = fallbackForGroup(worker.group_name);
	const col = indexInGroup % fallback.columns;
	const row = Math.floor(indexInGroup / fallback.columns);

	return {
		tileX: clampTileX(fallback.startX + col * fallback.colGap),
		tileY: clampTileY(fallback.startY + row * fallback.rowGap)
	};
}

