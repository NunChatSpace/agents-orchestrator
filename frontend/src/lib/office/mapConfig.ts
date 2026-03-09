import type { Worker, WorkerStatus } from '../../types/worker';

export type OfficeAccent = 'cyan' | 'magenta' | 'purple' | 'yellow' | 'green' | 'orange' | 'blue' | 'dim';

export interface DeskLayout {
	startX: number;
	startY: number;
	columns: number;
	rows: number;
	colGap: number;
	rowGap: number;
	overflowDirection: 'left' | 'right';
}

export interface GroupZoneConfig {
	groupName: string;
	label: string;
	labelMapX: number;
	labelMapY: number;
	accent: OfficeAccent;
	layout: DeskLayout;
}

export interface DeskConfig {
	workerId: string;
	mapX: number;
	mapY: number;
}

export interface DeskPosition {
	mapX: number;
	mapY: number;
	worldX: number;
	worldZ: number;
	accent: OfficeAccent;
}

export interface OfficeWorkerPlacement {
	workerId: string;
	name: string;
	groupName: string;
	status: WorkerStatus;
	mapX: number;
	mapY: number;
	worldX: number;
	worldZ: number;
	accent: OfficeAccent;
}

export interface GroupZoneLabel {
	groupName: string;
	label: string;
	accent: OfficeAccent;
	worldX: number;
	worldZ: number;
}

export interface MapConfig {
	grid: {
		minMapX: number;
		maxMapX: number;
		minMapY: number;
		maxMapY: number;
		worldOriginX: number;
		worldOriginZ: number;
		stepX: number;
		stepZ: number;
	};
	floorBounds: {
		minX: number;
		maxX: number;
		minZ: number;
		maxZ: number;
	};
	playerSpawn: {
		x: number;
		z: number;
	};
	cameraOffset: {
		x: number;
		y: number;
		z: number;
	};
	lookOffset: {
		x: number;
		y: number;
		z: number;
	};
	interactionRadius: number;
	deskCollider: {
		halfWidth: number;
		halfDepth: number;
	};
}

const GRID_MIN_X = 1;
const GRID_MAX_X = 24;
const GRID_MIN_Y = 1;
const GRID_MAX_Y = 14;

const STEP_X = 1.05;
const STEP_Z = 0.95;
const WORLD_ORIGIN_X = -10.2;
const WORLD_ORIGIN_Z = -6.15;

const KNOWN_ZONES: GroupZoneConfig[] = [
	{
		groupName: 'fi-backend',
		label: 'FI-BACKEND',
		labelMapX: 4,
		labelMapY: 1,
		accent: 'cyan',
		layout: {
			startX: 2,
			startY: 3,
			columns: 2,
			rows: 5,
			colGap: 3,
			rowGap: 2,
			overflowDirection: 'right'
		}
	},
	{
		groupName: 'fi-frontend',
		label: 'FI-FRONTEND',
		labelMapX: 13,
		labelMapY: 1,
		accent: 'magenta',
		layout: {
			startX: 11,
			startY: 3,
			columns: 2,
			rows: 5,
			colGap: 3,
			rowGap: 2,
			overflowDirection: 'right'
		}
	},
	{
		groupName: 'ib-kha',
		label: 'IB-KHA',
		labelMapX: 9,
		labelMapY: 6,
		accent: 'purple',
		layout: {
			startX: 8,
			startY: 8,
			columns: 2,
			rows: 4,
			colGap: 3,
			rowGap: 2,
			overflowDirection: 'right'
		}
	}
];

const DESK_CONFIGS: DeskConfig[] = [
	{ workerId: 'fi-backend1', mapX: 2, mapY: 3 },
	{ workerId: 'fi-backend2', mapX: 5, mapY: 3 },
	{ workerId: 'fi-frontend1', mapX: 11, mapY: 3 },
	{ workerId: 'fi-frontend2', mapX: 14, mapY: 3 },
	{ workerId: 'ib-kha', mapX: 8, mapY: 8 }
];

export const MAP_CONFIG: MapConfig = {
	grid: {
		minMapX: GRID_MIN_X,
		maxMapX: GRID_MAX_X,
		minMapY: GRID_MIN_Y,
		maxMapY: GRID_MAX_Y,
		worldOriginX: WORLD_ORIGIN_X,
		worldOriginZ: WORLD_ORIGIN_Z,
		stepX: STEP_X,
		stepZ: STEP_Z
	},
	floorBounds: {
		minX: -11.5,
		maxX: 15.8,
		minZ: -7.4,
		maxZ: 7.4
	},
	playerSpawn: {
		x: -8.2,
		z: 4.9
	},
	cameraOffset: {
		x: 0,
		y: 9.4,
		z: 13.4
	},
	lookOffset: {
		x: 0,
		y: 1.3,
		z: -2.4
	},
	interactionRadius: 3.25,
	deskCollider: {
		halfWidth: 1.15,
		halfDepth: 0.98
	}
};

const ZONE_BY_NAME = new Map(KNOWN_ZONES.map((zone) => [zone.groupName, zone]));

function clampMapX(value: number): number {
	return Math.max(MAP_CONFIG.grid.minMapX, Math.min(MAP_CONFIG.grid.maxMapX, value));
}

function clampMapY(value: number): number {
	return Math.max(MAP_CONFIG.grid.minMapY, Math.min(MAP_CONFIG.grid.maxMapY, value));
}

function configuredDesk(worker: Worker): DeskConfig | null {
	return (
		DESK_CONFIGS.find(
			(config) => config.workerId === worker.worker_id || config.workerId === worker.name
		) ?? null
	);
}

function fallbackZoneForGroup(groupName: string): GroupZoneConfig {
	const hash = Array.from(groupName).reduce((sum, char) => sum + char.charCodeAt(0), 0);
	const template = KNOWN_ZONES[hash % KNOWN_ZONES.length];

	return {
		groupName,
		label: groupName.toUpperCase(),
		labelMapX: template.labelMapX,
		labelMapY: template.labelMapY,
		accent: template.accent,
		layout: template.layout
	};
}

function resolveFallbackAnchor(layout: DeskLayout, indexInGroup: number): { mapX: number; mapY: number } {
	const seatsPerBlock = layout.columns * layout.rows;
	const blockIndex = Math.floor(indexInGroup / seatsPerBlock);
	const indexWithinBlock = indexInGroup % seatsPerBlock;
	const rowIndex = Math.floor(indexWithinBlock / layout.columns);
	const columnIndex = indexWithinBlock % layout.columns;
	const direction = layout.overflowDirection === 'left' ? -1 : 1;
	const blockOffset = blockIndex * layout.columns * layout.colGap * direction;

	return {
		mapX: clampMapX(layout.startX + blockOffset + columnIndex * layout.colGap),
		mapY: clampMapY(layout.startY + rowIndex * layout.rowGap)
	};
}

export function mapToWorld(mapX: number, mapY: number): { worldX: number; worldZ: number } {
	return {
		worldX: MAP_CONFIG.grid.worldOriginX + mapX * MAP_CONFIG.grid.stepX,
		worldZ: MAP_CONFIG.grid.worldOriginZ + mapY * MAP_CONFIG.grid.stepZ
	};
}

export function resolveZoneConfig(groupName: string): GroupZoneConfig {
	return ZONE_BY_NAME.get(groupName) ?? fallbackZoneForGroup(groupName);
}

export function buildZoneLabel(groupName: string): GroupZoneLabel {
	const zone = resolveZoneConfig(groupName);
	const { worldX, worldZ } = mapToWorld(zone.labelMapX, zone.labelMapY);
	return {
		groupName,
		label: zone.label,
		accent: zone.accent,
		worldX,
		worldZ
	};
}

export function resolveDeskPosition(worker: Worker, indexInGroup: number): DeskPosition {
	const zone = resolveZoneConfig(worker.group_name);
	let mapX: number;
	let mapY: number;

	if (worker.map_x !== 0 || worker.map_y !== 0) {
		mapX = clampMapX(worker.map_x);
		mapY = clampMapY(worker.map_y);
	} else {
		const configured = configuredDesk(worker);
		if (configured) {
			mapX = clampMapX(configured.mapX);
			mapY = clampMapY(configured.mapY);
		} else {
			const fallback = resolveFallbackAnchor(zone.layout, indexInGroup);
			mapX = fallback.mapX;
			mapY = fallback.mapY;
		}
	}

	const { worldX, worldZ } = mapToWorld(mapX, mapY);
	return {
		mapX,
		mapY,
		worldX,
		worldZ,
		accent: zone.accent
	};
}
