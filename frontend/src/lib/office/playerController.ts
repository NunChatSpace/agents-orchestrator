type DirectionKey = 'up' | 'down' | 'left' | 'right';

const KEY_TO_DIRECTION: Record<string, DirectionKey> = {
	ArrowUp: 'up',
	ArrowDown: 'down',
	ArrowLeft: 'left',
	ArrowRight: 'right',
	w: 'up',
	W: 'up',
	s: 'down',
	S: 'down',
	a: 'left',
	A: 'left',
	d: 'right',
	D: 'right'
};

export class PlayerController {
	private pressed = new Set<DirectionKey>();
	private virtualPressed = new Set<DirectionKey>();
	private active = false;

	private onKeyDown = (event: KeyboardEvent) => {
		const direction = KEY_TO_DIRECTION[event.key];
		if (!direction) return;
		event.preventDefault();
		this.pressed.add(direction);
	};

	private onKeyUp = (event: KeyboardEvent) => {
		const direction = KEY_TO_DIRECTION[event.key];
		if (!direction) return;
		event.preventDefault();
		this.pressed.delete(direction);
	};

	attach() {
		if (this.active) return;
		this.active = true;
		window.addEventListener('keydown', this.onKeyDown);
		window.addEventListener('keyup', this.onKeyUp);
	}

	detach() {
		if (!this.active) return;
		this.active = false;
		window.removeEventListener('keydown', this.onKeyDown);
		window.removeEventListener('keyup', this.onKeyUp);
		this.pressed.clear();
		this.virtualPressed.clear();
	}

	setVirtualDirection(direction: DirectionKey, active: boolean) {
		if (active) {
			this.virtualPressed.add(direction);
			return;
		}
		this.virtualPressed.delete(direction);
	}

	getVector(): { x: number; y: number } {
		const active = new Set<DirectionKey>([...this.pressed, ...this.virtualPressed]);
		let x = 0;
		let y = 0;

		if (active.has('left')) x -= 1;
		if (active.has('right')) x += 1;
		if (active.has('up')) y -= 1;
		if (active.has('down')) y += 1;

		if (x !== 0 && y !== 0) {
			const inv = Math.SQRT1_2;
			x *= inv;
			y *= inv;
		}

		return { x, y };
	}
}

export type VirtualDirection = 'up' | 'down' | 'left' | 'right';

