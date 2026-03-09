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
	private enabled = true;

	private isTypingTarget(target: EventTarget | null): boolean {
		return (
			target instanceof HTMLInputElement ||
			target instanceof HTMLTextAreaElement ||
			target instanceof HTMLSelectElement ||
			(target instanceof HTMLElement && target.isContentEditable)
		);
	}

	private onKeyDown = (event: KeyboardEvent) => {
		if (!this.enabled || this.isTypingTarget(event.target)) return;
		const direction = KEY_TO_DIRECTION[event.key];
		if (!direction) return;
		event.preventDefault();
		this.pressed.add(direction);
	};

	private onKeyUp = (event: KeyboardEvent) => {
		if (this.isTypingTarget(event.target)) return;
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
		if (!this.enabled) return;
		if (active) {
			this.virtualPressed.add(direction);
			return;
		}
		this.virtualPressed.delete(direction);
	}

	setEnabled(enabled: boolean) {
		this.enabled = enabled;
		if (!enabled) {
			this.pressed.clear();
			this.virtualPressed.clear();
		}
	}

	getVector(): { x: number; y: number } {
		if (!this.enabled) {
			return { x: 0, y: 0 };
		}

		const active = new Set<DirectionKey>([...this.pressed, ...this.virtualPressed]);
		let x = 0;
		let y = 0;

		if (active.has('left')) x -= 1;
		if (active.has('right')) x += 1;
		if (active.has('up')) y -= 1;
		if (active.has('down')) y += 1;

		const length = Math.hypot(x, y);
		if (length === 0) {
			return { x: 0, y: 0 };
		}

		return {
			x: x / length,
			y: y / length
		};
	}
}

export type VirtualDirection = 'up' | 'down' | 'left' | 'right';
