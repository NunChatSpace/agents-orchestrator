import type { MapConfig, TileType } from './mapConfig';

const FLOOR_A = '#080814';
const FLOOR_B = '#090919';
const WALL_FILL = '#14142b';
const WALL_EDGE = 'rgba(139,92,246,0.32)';
const VOID_FILL = '#030308';

function drawFloor(
	ctx: CanvasRenderingContext2D,
	x: number,
	y: number,
	size: number,
	alternate: boolean
) {
	ctx.fillStyle = alternate ? FLOOR_A : FLOOR_B;
	ctx.fillRect(x, y, size, size);
	ctx.strokeStyle = 'rgba(139,92,246,0.05)';
	ctx.strokeRect(x + 0.5, y + 0.5, size - 1, size - 1);
}

function drawWall(ctx: CanvasRenderingContext2D, x: number, y: number, size: number) {
	ctx.fillStyle = WALL_FILL;
	ctx.fillRect(x, y, size, size);
	ctx.strokeStyle = WALL_EDGE;
	ctx.strokeRect(x + 0.5, y + 0.5, size - 1, size - 1);
}

function drawVoid(ctx: CanvasRenderingContext2D, x: number, y: number, size: number) {
	ctx.fillStyle = VOID_FILL;
	ctx.fillRect(x, y, size, size);
}

function drawTile(
	ctx: CanvasRenderingContext2D,
	type: TileType,
	x: number,
	y: number,
	size: number,
	alternate: boolean
) {
	if (type === 'wall') {
		drawWall(ctx, x, y, size);
		return;
	}
	if (type === 'void') {
		drawVoid(ctx, x, y, size);
		return;
	}
	drawFloor(ctx, x, y, size, alternate);
}

export function drawTiles(
	ctx: CanvasRenderingContext2D,
	map: MapConfig,
	cameraX: number,
	cameraY: number,
	viewportWidth: number,
	viewportHeight: number
) {
	const tileSize = map.tileSize;
	const startX = Math.max(0, Math.floor(cameraX / tileSize));
	const startY = Math.max(0, Math.floor(cameraY / tileSize));
	const endX = Math.min(map.width - 1, Math.ceil((cameraX + viewportWidth) / tileSize));
	const endY = Math.min(map.height - 1, Math.ceil((cameraY + viewportHeight) / tileSize));

	for (let y = startY; y <= endY; y++) {
		for (let x = startX; x <= endX; x++) {
			const tile = map.tiles[y]?.[x] ?? 'void';
			const drawX = x * tileSize - cameraX;
			const drawY = y * tileSize - cameraY;
			drawTile(ctx, tile, drawX, drawY, tileSize, (x + y) % 2 === 0);
		}
	}
}

