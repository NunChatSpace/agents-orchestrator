export interface ProximityDesk {
	workerId: string;
	tileX: number;
	tileY: number;
}

export interface NearestDesk {
	workerId: string;
	distance: number;
}

export function findNearestDeskInRange(
	playerX: number,
	playerY: number,
	tileSize: number,
	desks: ProximityDesk[],
	maxTiles = 2
): NearestDesk | null {
	const playerTileX = playerX / tileSize;
	const playerTileY = playerY / tileSize;

	let nearest: NearestDesk | null = null;

	for (const desk of desks) {
		const dx = playerTileX - (desk.tileX + 0.5);
		const dy = playerTileY - (desk.tileY + 0.5);
		const distance = Math.hypot(dx, dy);
		if (distance > maxTiles) continue;

		if (!nearest || distance < nearest.distance) {
			nearest = { workerId: desk.workerId, distance };
		}
	}

	return nearest;
}

