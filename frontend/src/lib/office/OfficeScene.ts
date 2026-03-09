import * as THREE from 'three';
import { PlayerController } from './playerController';
import type { GroupZoneLabel, MapConfig, OfficeAccent, OfficeWorkerPlacement } from './mapConfig';

interface OfficeSceneCallbacks {
	onNearestWorkerChange?: (workerId: string | null) => void;
	onInteract?: (workerId: string) => void;
}

interface AvatarRig {
	root: THREE.Group;
	ring: THREE.Mesh;
	hip: THREE.Group;
	leftLeg: THREE.Group;
	rightLeg: THREE.Group;
	leftArm: THREE.Group;
	rightArm: THREE.Group;
	torso: THREE.Mesh;
	head: THREE.Mesh;
	shadow: THREE.Mesh;
}

interface DeskEntry {
	worker: OfficeWorkerPlacement;
	group: THREE.Group;
	screenMat: THREE.MeshStandardMaterial;
	beaconMat: THREE.MeshStandardMaterial;
	selectionRingMat: THREE.MeshStandardMaterial;
	selectionRing: THREE.Mesh;
	haloMat: THREE.MeshStandardMaterial;
	halo: THREE.Mesh;
	beaconHead: THREE.Mesh;
	localLight: THREE.PointLight;
	chairMat: THREE.MeshStandardMaterial;
	nameTagMat: THREE.SpriteMaterial;
	nameTag: THREE.Sprite;
	baseIntensity: number;
}

const STATUS_COLORS: Record<OfficeWorkerPlacement['status'], number> = {
	idle: 0x16f2a5,
	busy: 0xffb347,
	pending_user: 0x6da8ff,
	offline: 0x666666
};

const ACCENT_COLORS: Record<OfficeAccent, number> = {
	cyan: 0x00e7ff,
	magenta: 0xff3b7a,
	purple: 0x8b5cff,
	yellow: 0xffd166,
	green: 0x16f2a5,
	orange: 0xff9f43,
	blue: 0x6da8ff,
	dim: 0x5b6478
};

function clamp(value: number, min: number, max: number): number {
	return Math.max(min, Math.min(max, value));
}

function disposeMaterial(material: THREE.Material | THREE.Material[]) {
	if (Array.isArray(material)) {
		for (const item of material) {
			disposeMaterial(item);
		}
		return;
	}

	const typed = material as THREE.Material & { map?: THREE.Texture | null };
	typed.map?.dispose();
	material.dispose();
}

function disposeObjectTree(root: THREE.Object3D) {
	root.traverse((child: THREE.Object3D) => {
		if (child instanceof THREE.Mesh) {
			child.geometry.dispose();
			disposeMaterial(child.material);
		}

		if (child instanceof THREE.Sprite) {
			disposeMaterial(child.material);
		}
	});
}

export class OfficeScene {
	private readonly mountEl: HTMLDivElement;
	private readonly config: MapConfig;
	private readonly callbacks: OfficeSceneCallbacks;
	private readonly controller = new PlayerController();
	private readonly renderer: THREE.WebGLRenderer;
	private readonly scene = new THREE.Scene();
	private readonly camera = new THREE.PerspectiveCamera(42, 1, 0.1, 120);
	private readonly raycaster = new THREE.Raycaster();
	private readonly pointer = new THREE.Vector2();
	private readonly clock = new THREE.Clock();
	private readonly desiredCameraPosition = new THREE.Vector3();
	private readonly desiredLookTarget = new THREE.Vector3();
	private readonly avatarVelocity = new THREE.Vector3();
	private readonly deskRoot = new THREE.Group();
	private readonly zoneRoot = new THREE.Group();
	private readonly particleRoot = new THREE.Group();
	private readonly interactables: THREE.Object3D[] = [];
	private readonly deskEntries = new Map<string, DeskEntry>();

	private readonly ambient = new THREE.HemisphereLight(0xb4d7ff, 0x2a3150, 2.2);
	private readonly keyLight = new THREE.SpotLight(0xcbe6ff, 110, 56, Math.PI / 4.8, 0.35, 1.1);
	private readonly sunFill = new THREE.DirectionalLight(0xffffff, 1.35);
	private readonly magentaLight = new THREE.PointLight(0xff3b7a, 26, 24, 2);
	private readonly cyanLight = new THREE.PointLight(0x00e7ff, 26, 24, 2);
	private readonly ceilingGlow = new THREE.PointLight(0x8b5cff, 18, 22, 2);

	private readonly avatar: AvatarRig;
	private readonly particleField: THREE.Points;

	private workers: OfficeWorkerPlacement[] = [];
	private groupLabels: GroupZoneLabel[] = [];
	private frameId: number | null = null;
	private running = false;
	private hoveredWorkerId: string | null = null;
	private selectedWorkerId: string | null = null;
	private nearestWorkerId: string | null = null;

	constructor(mountEl: HTMLDivElement, config: MapConfig, callbacks: OfficeSceneCallbacks = {}) {
		this.mountEl = mountEl;
		this.config = config;
		this.callbacks = callbacks;

		this.renderer = new THREE.WebGLRenderer({
			antialias: true,
			powerPreference: 'high-performance',
			alpha: false
		});
		this.renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
		this.renderer.outputColorSpace = THREE.SRGBColorSpace;
		this.renderer.toneMapping = THREE.ACESFilmicToneMapping;
		this.renderer.toneMappingExposure = 1.15;
		this.renderer.shadowMap.enabled = true;
		this.renderer.shadowMap.type = THREE.PCFSoftShadowMap;
		this.renderer.domElement.className = 'office-webgl';
		this.renderer.domElement.style.display = 'block';
		this.renderer.domElement.style.width = '100%';
		this.renderer.domElement.style.height = '100%';
		this.renderer.domElement.style.outline = 'none';
		this.mountEl.appendChild(this.renderer.domElement);

		this.scene.background = new THREE.Color(0x111b30);
		this.scene.fog = new THREE.FogExp2(0x162342, 0.014);

		this.camera.position.set(0, 8.5, 11.8);

		this.scene.add(this.ambient);
		this.scene.add(this.keyLight);
		this.scene.add(this.keyLight.target);
		this.scene.add(this.sunFill);
		this.scene.add(this.magentaLight);
		this.scene.add(this.cyanLight);
		this.scene.add(this.ceilingGlow);
		this.scene.add(this.zoneRoot);
		this.scene.add(this.deskRoot);
		this.scene.add(this.particleRoot);

		this.keyLight.position.set(0, 14, 7.5);
		this.keyLight.castShadow = true;
		this.keyLight.shadow.mapSize.set(1024, 1024);
		this.keyLight.target.position.set(0, 0, -1);

		this.sunFill.position.set(-5, 10, 6);
		this.magentaLight.position.set(-9, 4.5, 0);
		this.cyanLight.position.set(9, 4.5, 0);
		this.ceilingGlow.position.set(0, 8, -5);

		this.addRoom();
		this.addNeonColumns();
		this.avatar = this.createAvatar();
		this.particleField = this.addAmbientParticles();
		this.updateCursor();
	}

	start() {
		if (this.running) return;
		this.running = true;
		this.controller.attach();
		this.clock.start();
		this.renderer.domElement.addEventListener('pointermove', this.onPointerMove);
		this.renderer.domElement.addEventListener('pointerleave', this.onPointerLeave);
		this.renderer.domElement.addEventListener('click', this.onClick);
		this.frameId = requestAnimationFrame(this.animate);
	}

	setMovementEnabled(enabled: boolean) {
		this.controller.setEnabled(enabled);
	}

	resize(width: number, height: number) {
		const safeWidth = Math.max(1, Math.floor(width));
		const safeHeight = Math.max(1, Math.floor(height));
		this.camera.aspect = safeWidth / safeHeight;
		this.camera.updateProjectionMatrix();
		this.renderer.setSize(safeWidth, safeHeight, false);
	}

	setWorkers(workers: OfficeWorkerPlacement[], labels: GroupZoneLabel[]) {
		this.workers = workers;
		this.groupLabels = labels;

		this.rebuildZones();
		this.rebuildDesks();

		if (this.selectedWorkerId && !this.deskEntries.has(this.selectedWorkerId)) {
			this.selectedWorkerId = null;
		}
		if (this.hoveredWorkerId && !this.deskEntries.has(this.hoveredWorkerId)) {
			this.hoveredWorkerId = null;
		}

		this.recalculateNearest();
	}

	setSelectedWorker(workerId: string | null) {
		this.selectedWorkerId = workerId;
	}

	destroy() {
		if (this.frameId !== null) {
			cancelAnimationFrame(this.frameId);
			this.frameId = null;
		}

		this.running = false;
		this.controller.detach();
		this.renderer.domElement.removeEventListener('pointermove', this.onPointerMove);
		this.renderer.domElement.removeEventListener('pointerleave', this.onPointerLeave);
		this.renderer.domElement.removeEventListener('click', this.onClick);

		disposeObjectTree(this.zoneRoot);
		disposeObjectTree(this.deskRoot);
		disposeObjectTree(this.particleRoot);
		disposeObjectTree(this.avatar.root);
		this.renderer.dispose();
		this.mountEl.innerHTML = '';
	}

	private readonly onPointerMove = (event: PointerEvent) => {
		const entry = this.getIntersectedEntry(event.clientX, event.clientY);
		this.hoveredWorkerId = entry?.worker.workerId ?? null;
		this.updateCursor();
	};

	private readonly onPointerLeave = () => {
		this.hoveredWorkerId = null;
		this.updateCursor();
	};

	private readonly onClick = (event: MouseEvent) => {
		const entry = this.getIntersectedEntry(event.clientX, event.clientY);
		if (!entry) return;
		this.callbacks.onInteract?.(entry.worker.workerId);
	};

	private readonly animate = () => {
		if (!this.running) return;

		const delta = Math.min(this.clock.getDelta(), 0.05);
		const elapsed = this.clock.elapsedTime;

		this.updateAvatar(delta);
		this.updateCamera();
		this.updateLights(elapsed);
		this.updateDeskVisuals(elapsed);

		if (this.particleField) {
			this.particleField.rotation.y = elapsed * 0.03;
		}

		this.renderer.render(this.scene, this.camera);
		this.frameId = requestAnimationFrame(this.animate);
	};

	private addRoom() {
		const room = new THREE.Group();
		const width = this.config.floorBounds.maxX - this.config.floorBounds.minX;
		const depth = this.config.floorBounds.maxZ - this.config.floorBounds.minZ;
		const centerX = (this.config.floorBounds.minX + this.config.floorBounds.maxX) * 0.5;
		const centerZ = (this.config.floorBounds.minZ + this.config.floorBounds.maxZ) * 0.5;
		const floorMat = new THREE.MeshStandardMaterial({
			color: 0x18243f,
			metalness: 0.56,
			roughness: 0.46
		});

		const floor = new THREE.Mesh(new THREE.BoxGeometry(width, 0.5, depth), floorMat);
		floor.position.set(centerX, -0.25, centerZ);
		floor.receiveShadow = true;
		room.add(floor);

		const grid = new THREE.GridHelper(Math.max(width, depth), 28, 0x00e7ff, 0x14304a);
		(grid.material as THREE.Material).transparent = true;
		(grid.material as THREE.Material).opacity = 0.32;
		grid.position.set(centerX, 0.01, centerZ);
		room.add(grid);

		const liftStripMaterial = new THREE.MeshStandardMaterial({
			color: 0x182c45,
			emissive: 0xd7f7ff,
			emissiveIntensity: 1.4,
			metalness: 0.1,
			roughness: 0.4
		});

		for (let x = this.config.floorBounds.minX + 3; x <= this.config.floorBounds.maxX - 3; x += 4.2) {
			const strip = new THREE.Mesh(new THREE.BoxGeometry(0.14, 0.03, depth - 1.2), liftStripMaterial);
			strip.position.set(x, 0.04, centerZ);
			room.add(strip);
		}

		for (let z = this.config.floorBounds.minZ + 2.5; z <= this.config.floorBounds.maxZ - 2.5; z += 3.6) {
			const lane = new THREE.Mesh(
				new THREE.BoxGeometry(width - 2, 0.02, 0.12),
				new THREE.MeshStandardMaterial({
					color: 0x24152a,
					emissive: 0xff3b7a,
					emissiveIntensity: 1.55
				})
			);
			lane.position.set(centerX, 0.03, z);
			room.add(lane);
		}

		const stageBorder = new THREE.LineSegments(
			new THREE.EdgesGeometry(new THREE.BoxGeometry(width, 0.5, depth)),
			new THREE.LineBasicMaterial({ color: 0x00e7ff, transparent: true, opacity: 0.2 })
		);
		stageBorder.position.copy(floor.position);
		room.add(stageBorder);

		this.scene.add(room);
	}

	private addNeonColumns() {
		const { minX, maxX, minZ, maxZ } = this.config.floorBounds;
		const columnPositions: Array<[number, number, number]> = [
			[minX + 1.5, 2.6, minZ + 1.2],
			[maxX - 1.5, 2.6, minZ + 1.2],
			[minX + 1.5, 2.6, maxZ - 1.2],
			[maxX - 1.5, 2.6, maxZ - 1.2]
		];

		for (const [x, y, z] of columnPositions) {
			const shell = new THREE.Mesh(
				new THREE.CylinderGeometry(0.18, 0.18, 5.2, 12),
				new THREE.MeshStandardMaterial({
					color: 0x2a3b62,
					metalness: 0.8,
					roughness: 0.35
				})
			);
			shell.position.set(x, y, z);
			this.scene.add(shell);

			const glow = new THREE.Mesh(
				new THREE.CylinderGeometry(0.04, 0.04, 4.6, 8),
				new THREE.MeshStandardMaterial({
					color: 0x00e7ff,
					emissive: 0x00e7ff,
					emissiveIntensity: 3.1,
					metalness: 0.1,
					roughness: 0.2
				})
			);
			glow.position.set(x, y, z);
			this.scene.add(glow);
		}
	}

	private createAvatar(): AvatarRig {
		const root = new THREE.Group();
		root.position.set(this.config.playerSpawn.x, 0, this.config.playerSpawn.z);

		const shadow = new THREE.Mesh(
			new THREE.CircleGeometry(0.42, 24),
			new THREE.MeshBasicMaterial({
				color: 0x000000,
				transparent: true,
				opacity: 0.16
			})
		);
		shadow.rotation.x = -Math.PI / 2;
		shadow.position.y = 0.02;
		root.add(shadow);

		const ring = new THREE.Mesh(
			new THREE.TorusGeometry(0.44, 0.035, 10, 28),
			new THREE.MeshStandardMaterial({
				color: 0x00e7ff,
				emissive: 0x00e7ff,
				emissiveIntensity: 1.8,
				transparent: true,
				opacity: 0.9
			})
		);
		ring.rotation.x = Math.PI / 2;
		ring.position.y = 0.04;
		root.add(ring);

		const limbMaterial = new THREE.MeshStandardMaterial({
			color: 0x1b2b4c,
			emissive: 0x142241,
			emissiveIntensity: 0.35,
			metalness: 0.35,
			roughness: 0.45
		});

		const hip = new THREE.Group();
		hip.position.y = 0.84;
		root.add(hip);

		const createLeg = (offsetX: number) => {
			const pivot = new THREE.Group();
			pivot.position.set(offsetX, 0, 0);
			hip.add(pivot);

			const mesh = new THREE.Mesh(
				new THREE.CylinderGeometry(0.1, 0.14, 0.78, 10),
				limbMaterial.clone()
			);
			mesh.position.y = -0.39;
			mesh.castShadow = true;
			pivot.add(mesh);

			return pivot;
		};

		const leftLeg = createLeg(-0.14);
		const rightLeg = createLeg(0.14);

		const torso = new THREE.Mesh(
			new THREE.BoxGeometry(0.56, 0.82, 0.34),
			new THREE.MeshStandardMaterial({
				color: 0xeff7ff,
				emissive: 0x00e7ff,
				emissiveIntensity: 0.4,
				metalness: 0.18,
				roughness: 0.22
			})
		);
		torso.position.y = 1.07;
		torso.castShadow = true;
		root.add(torso);

		const shoulder = new THREE.Group();
		shoulder.position.y = 1.34;
		root.add(shoulder);

		const createArm = (offsetX: number) => {
			const pivot = new THREE.Group();
			pivot.position.set(offsetX, 0, 0);
			shoulder.add(pivot);

			const mesh = new THREE.Mesh(
				new THREE.BoxGeometry(0.12, 0.56, 0.12),
				new THREE.MeshStandardMaterial({
					color: 0xd9ecff,
					emissive: 0x3a70ff,
					emissiveIntensity: 0.12,
					metalness: 0.15,
					roughness: 0.28
				})
			);
			mesh.position.y = -0.28;
			mesh.castShadow = true;
			pivot.add(mesh);

			return pivot;
		};

		const leftArm = createArm(-0.36);
		const rightArm = createArm(0.36);

		const visor = new THREE.Mesh(
			new THREE.BoxGeometry(0.3, 0.12, 0.16),
			new THREE.MeshStandardMaterial({
				color: 0x091420,
				emissive: 0x00e7ff,
				emissiveIntensity: 2.1
			})
		);
		visor.position.set(0, 1.48, 0.2);
		visor.castShadow = true;
		root.add(visor);

		const head = new THREE.Mesh(
			new THREE.SphereGeometry(0.24, 18, 18),
			new THREE.MeshStandardMaterial({
				color: 0xf7fbff,
				emissive: 0x4d7dff,
				emissiveIntensity: 0.2,
				metalness: 0.12,
				roughness: 0.28
			})
		);
		head.position.y = 1.46;
		head.castShadow = true;
		root.add(head);

		const label = new THREE.Mesh(
			new THREE.BoxGeometry(0.64, 0.06, 0.06),
			new THREE.MeshStandardMaterial({
				color: 0x00e7ff,
				emissive: 0x00e7ff,
				emissiveIntensity: 1.8
			})
		);
		label.position.set(0, 1.95, 0);
		root.add(label);

		this.scene.add(root);

		return {
			root,
			ring,
			hip,
			leftLeg,
			rightLeg,
			leftArm,
			rightArm,
			torso,
			head,
			shadow
		};
	}

	private addAmbientParticles() {
		const count = 260;
		const positions = new Float32Array(count * 3);

		for (let index = 0; index < count; index += 1) {
			positions[index * 3] =
				this.config.floorBounds.minX +
				Math.random() * (this.config.floorBounds.maxX - this.config.floorBounds.minX);
			positions[index * 3 + 1] = Math.random() * 8 + 0.5;
			positions[index * 3 + 2] =
				this.config.floorBounds.minZ +
				Math.random() * (this.config.floorBounds.maxZ - this.config.floorBounds.minZ);
		}

		const geometry = new THREE.BufferGeometry();
		geometry.setAttribute('position', new THREE.BufferAttribute(positions, 3));

		const material = new THREE.PointsMaterial({
			color: 0x00d8ff,
			size: 0.045,
			transparent: true,
			opacity: 0.45
		});

		const points = new THREE.Points(geometry, material);
		this.particleRoot.add(points);
		return points;
	}

	private rebuildZones() {
		disposeObjectTree(this.zoneRoot);
		this.zoneRoot.clear();

		for (const label of this.groupLabels) {
			const accent = ACCENT_COLORS[label.accent];
			const plate = new THREE.Mesh(
				new THREE.BoxGeometry(4.4, 0.08, 0.35),
				new THREE.MeshStandardMaterial({
					color: 0x0a0c16,
					emissive: accent,
					emissiveIntensity: 1.6
				})
			);
			plate.position.set(label.worldX, 1.8, label.worldZ);
			this.zoneRoot.add(plate);

			const text = this.createTextSprite(label.label, {
				color: '#d8f8ff',
				border: '#00e7ff',
				background: 'rgba(8, 16, 32, 0.88)',
				scale: 1.55
			});
			text.position.set(label.worldX, 2.25, label.worldZ);
			this.zoneRoot.add(text);
		}
	}

	private rebuildDesks() {
		disposeObjectTree(this.deskRoot);
		this.deskRoot.clear();
		this.deskEntries.clear();
		this.interactables.length = 0;

		for (const worker of this.workers) {
			this.createDeskStation(worker);
		}
	}

	private createDeskStation(worker: OfficeWorkerPlacement) {
		const accent = ACCENT_COLORS[worker.accent];
		const statusColor = STATUS_COLORS[worker.status];
		const group = new THREE.Group();
		group.position.set(worker.worldX, 0, worker.worldZ);

		const deskMat = new THREE.MeshStandardMaterial({
			color: 0x111522,
			metalness: 0.82,
			roughness: 0.42
		});

		const top = new THREE.Mesh(new THREE.BoxGeometry(2.2, 0.14, 1.2), deskMat);
		top.position.set(0, 1.15, 0);
		top.castShadow = true;
		top.receiveShadow = true;
		group.add(top);

		const legGeo = new THREE.BoxGeometry(0.12, 1.05, 0.12);
		for (const [lx, lz] of [
			[-0.92, -0.48],
			[0.92, -0.48],
			[-0.92, 0.48],
			[0.92, 0.48]
		]) {
			const leg = new THREE.Mesh(legGeo, deskMat);
			leg.position.set(lx, 0.55, lz);
			leg.castShadow = true;
			group.add(leg);
		}

		const sideCabinet = new THREE.Mesh(
			new THREE.BoxGeometry(0.48, 0.92, 0.92),
			new THREE.MeshStandardMaterial({
				color: 0x0d1020,
				metalness: 0.9,
				roughness: 0.32
			})
		);
		sideCabinet.position.set(-0.75, 0.46, 0);
		sideCabinet.castShadow = true;
		group.add(sideCabinet);

		const screenMat = new THREE.MeshStandardMaterial({
			color: 0x0d1421,
			emissive: statusColor,
			emissiveIntensity: worker.status === 'busy' ? 2.6 : 1.6,
			metalness: 0.3,
			roughness: 0.25
		});

		const screenShell = new THREE.Mesh(
			new THREE.BoxGeometry(0.85, 0.52, 0.06),
			new THREE.MeshStandardMaterial({
				color: 0x101624,
				metalness: 0.75,
				roughness: 0.35
			})
		);
		screenShell.position.set(0.32, 1.7, -0.16);
		screenShell.castShadow = true;
		group.add(screenShell);

		const screen = new THREE.Mesh(new THREE.BoxGeometry(0.72, 0.42, 0.03), screenMat);
		screen.position.set(0.32, 1.7, -0.12);
		group.add(screen);

		const stand = new THREE.Mesh(
			new THREE.BoxGeometry(0.08, 0.42, 0.08),
			new THREE.MeshStandardMaterial({
				color: 0x202838,
				metalness: 0.7,
				roughness: 0.4
			})
		);
		stand.position.set(0.32, 1.42, -0.16);
		group.add(stand);

		const chairBase = new THREE.Mesh(
			new THREE.CylinderGeometry(0.14, 0.18, 0.14, 12),
			new THREE.MeshStandardMaterial({
				color: 0x141826,
				metalness: 0.8,
				roughness: 0.35
			})
		);
		chairBase.position.set(0, 0.48, 1.05);
		chairBase.castShadow = true;
		group.add(chairBase);

		const chairMat = new THREE.MeshStandardMaterial({
			color: 0x22152f,
			emissive: accent,
			emissiveIntensity: 0.18
		});

		const chairSeat = new THREE.Mesh(new THREE.BoxGeometry(0.66, 0.12, 0.66), chairMat);
		chairSeat.position.set(0, 0.84, 1.03);
		chairSeat.castShadow = true;
		group.add(chairSeat);

		const chairBack = new THREE.Mesh(new THREE.BoxGeometry(0.66, 0.72, 0.12), chairMat.clone());
		chairBack.position.set(0, 1.22, 1.36);
		chairBack.rotation.x = 0.16;
		chairBack.castShadow = true;
		group.add(chairBack);

		const beaconMat = new THREE.MeshStandardMaterial({
			color: 0xf6f0ff,
			emissive: accent,
			emissiveIntensity: 1.7,
			metalness: 0.2,
			roughness: 0.25
		});

		const beaconStem = new THREE.Mesh(
			new THREE.CylinderGeometry(0.05, 0.05, 0.95, 10),
			new THREE.MeshStandardMaterial({
				color: 0x171b28,
				metalness: 0.7,
				roughness: 0.35
			})
		);
		beaconStem.position.set(0, 1.6, 0.42);
		group.add(beaconStem);

		const beaconHead = new THREE.Mesh(new THREE.SphereGeometry(0.2, 16, 16), beaconMat);
		beaconHead.position.set(0, 2.15, 0.42);
		group.add(beaconHead);

		const haloMat = new THREE.MeshStandardMaterial({
			color: accent,
			emissive: accent,
			emissiveIntensity: 1.8,
			transparent: true,
			opacity: 0.95
		});
		const halo = new THREE.Mesh(new THREE.TorusGeometry(0.42, 0.03, 10, 32), haloMat);
		halo.position.set(0, 2.12, 0.42);
		halo.rotation.x = Math.PI / 2;
		group.add(halo);

		const selectionRingMat = new THREE.MeshStandardMaterial({
			color: accent,
			emissive: accent,
			emissiveIntensity: 2.1,
			transparent: true,
			opacity: 0
		});
		const selectionRing = new THREE.Mesh(new THREE.TorusGeometry(1.46, 0.05, 12, 40), selectionRingMat);
		selectionRing.rotation.x = Math.PI / 2;
		selectionRing.position.set(0, 0.04, 0);
		group.add(selectionRing);

		const hitArea = new THREE.Mesh(
			new THREE.BoxGeometry(2.7, 2.8, 2.2),
			new THREE.MeshBasicMaterial({
				transparent: true,
				opacity: 0,
				depthWrite: false
			})
		);
		hitArea.position.set(0, 1.3, 0.2);
		hitArea.userData.workerId = worker.workerId;
		group.add(hitArea);
		this.interactables.push(hitArea);

		const localLight = new THREE.PointLight(statusColor, worker.status === 'offline' ? 0 : 2.8, 4.5, 2);
		localLight.position.set(0.32, 1.7, -0.05);
		group.add(localLight);

		const nameTag = this.createTextSprite(worker.name.toUpperCase(), {
			color: '#edf7ff',
			border: '#00e7ff',
			background: 'rgba(4, 8, 24, 0.85)',
			scale: 0.78
		});
		const nameTagMat = nameTag.material as THREE.SpriteMaterial;
		nameTag.position.set(0, 2.85, 0.46);
		group.add(nameTag);

		const entry: DeskEntry = {
			worker,
			group,
			screenMat,
			beaconMat,
			selectionRingMat,
			selectionRing,
			haloMat,
			halo,
			beaconHead,
			localLight,
			chairMat,
			nameTagMat,
			nameTag,
			baseIntensity:
				worker.status === 'busy'
					? 2.6
					: worker.status === 'pending_user'
						? 2
						: worker.status === 'idle'
							? 1.5
							: 0.08
		};

		this.scene.add(group);
		this.deskRoot.add(group);
		this.deskEntries.set(worker.workerId, entry);
	}

	private createTextSprite(
		text: string,
		options: { color: string; border: string; background: string; scale: number }
	) {
		const material = this.createTextSpriteMaterial(text, options);
		const sprite = new THREE.Sprite(material);
		sprite.scale.copy(material.userData.scale);
		return sprite;
	}

	private createTextSpriteMaterial(
		text: string,
		options: { color: string; border: string; background: string; scale: number }
	) {
		const canvas = document.createElement('canvas');
		const ctx = canvas.getContext('2d');
		if (!ctx) {
			throw new Error('Canvas 2D context is required for Office labels');
		}

		const fontSize = 26;
		const paddingX = 18;
		const paddingY = 10;
		const ratio = 2;

		ctx.font = `700 ${fontSize}px Courier New, monospace`;
		const width = Math.ceil(ctx.measureText(text).width + paddingX * 2);
		const height = fontSize + paddingY * 2;
		canvas.width = width * ratio;
		canvas.height = height * ratio;

		ctx.scale(ratio, ratio);
		ctx.font = `700 ${fontSize}px Courier New, monospace`;
		ctx.fillStyle = options.background;
		ctx.strokeStyle = options.border;
		ctx.lineWidth = 2;
		ctx.beginPath();
		ctx.moveTo(8, 0);
		ctx.lineTo(width, 0);
		ctx.lineTo(width, height - 8);
		ctx.lineTo(width - 8, height);
		ctx.lineTo(0, height);
		ctx.lineTo(0, 8);
		ctx.closePath();
		ctx.fill();
		ctx.stroke();
		ctx.fillStyle = options.color;
		ctx.textAlign = 'center';
		ctx.textBaseline = 'middle';
		ctx.fillText(text, width * 0.5, height * 0.5 + 1);

		const texture = new THREE.CanvasTexture(canvas);
		texture.colorSpace = THREE.SRGBColorSpace;
		texture.minFilter = THREE.LinearFilter;
		texture.magFilter = THREE.LinearFilter;

		const material = new THREE.SpriteMaterial({
			map: texture,
			transparent: true,
			depthWrite: false
		});
		material.depthTest = false;

		const aspect = width / height;
		const scale = new THREE.Vector3(options.scale * aspect, options.scale, 1);
		(material as THREE.SpriteMaterial & { userData: { scale: THREE.Vector3 } }).userData = { scale };
		return material;
	}

	private getIntersectedEntry(clientX: number, clientY: number): DeskEntry | null {
		const rect = this.renderer.domElement.getBoundingClientRect();
		this.pointer.x = ((clientX - rect.left) / rect.width) * 2 - 1;
		this.pointer.y = -((clientY - rect.top) / rect.height) * 2 + 1;
		this.raycaster.setFromCamera(this.pointer, this.camera);

		const hit = this.raycaster.intersectObjects(this.interactables, false)[0];
		if (!hit) return null;

		const workerId = hit.object.userData.workerId as string | undefined;
		return workerId ? this.deskEntries.get(workerId) ?? null : null;
	}

	private updateCursor() {
		this.mountEl.style.cursor = this.hoveredWorkerId ? 'pointer' : 'crosshair';
	}

	private updateAvatar(deltaSeconds: number) {
		const movement = this.controller.getVector();
		this.avatarVelocity.set(movement.x, 0, movement.y);

		if (this.avatarVelocity.lengthSq() > 0) {
			this.avatarVelocity.multiplyScalar(deltaSeconds * 4.2);
			const nextX = clamp(
				this.avatar.root.position.x + this.avatarVelocity.x,
				this.config.floorBounds.minX,
				this.config.floorBounds.maxX
			);
			const nextZ = clamp(
				this.avatar.root.position.z + this.avatarVelocity.z,
				this.config.floorBounds.minZ,
				this.config.floorBounds.maxZ
			);

			if (!this.isDeskBlocked(nextX, nextZ)) {
				this.avatar.root.position.x = nextX;
				this.avatar.root.position.z = nextZ;
			}

			this.avatar.root.rotation.y = Math.atan2(this.avatarVelocity.x, this.avatarVelocity.z);
		}

		this.recalculateNearest();
		this.animateAvatar(this.clock.elapsedTime, this.avatarVelocity.lengthSq() > 0);
	}

	private updateCamera() {
		this.desiredCameraPosition.set(
			this.avatar.root.position.x + this.config.cameraOffset.x,
			this.avatar.root.position.y + this.config.cameraOffset.y,
			this.avatar.root.position.z + this.config.cameraOffset.z
		);
		this.camera.position.lerp(this.desiredCameraPosition, 0.08);

		this.desiredLookTarget.set(
			this.avatar.root.position.x + this.config.lookOffset.x,
			this.avatar.root.position.y + this.config.lookOffset.y,
			this.avatar.root.position.z + this.config.lookOffset.z
		);
		this.camera.lookAt(this.desiredLookTarget);
	}

	private updateLights(elapsed: number) {
		this.keyLight.intensity = 108 + Math.sin(elapsed * 0.9) * 5;
		this.cyanLight.intensity = 24 + Math.sin(elapsed * 1.4) * 3;
		this.magentaLight.intensity = 24 + Math.cos(elapsed * 1.2) * 3;
		this.ceilingGlow.intensity = 18 + Math.sin(elapsed * 1.1) * 1.6;
	}

	private updateDeskVisuals(elapsed: number) {
		for (const [workerId, entry] of this.deskEntries) {
			const pulse = Math.sin(elapsed * 2.6 + entry.group.position.x) * 0.5 + 0.5;
			const selected = workerId === this.selectedWorkerId ? 1 : 0;
			const hovered = workerId === this.hoveredWorkerId ? 0.58 : 0;
			const nearest = workerId === this.nearestWorkerId ? 0.26 : 0;
			const hot = Math.max(selected, hovered, nearest);
			const alive = entry.worker.status === 'offline' ? 0.1 : 1;
			const labelScale = entry.nameTagMat.userData.scale as THREE.Vector3;

			entry.screenMat.emissiveIntensity = (entry.baseIntensity + hot * 1.3 + pulse * 0.25) * alive;
			entry.beaconMat.emissiveIntensity = (1.1 + pulse * 0.55 + hot * 1.4) * alive;
			entry.localLight.intensity = (1.3 + pulse * 0.9 + hot * 1.1) * alive;

			entry.beaconHead.position.y = 2.12 + Math.sin(elapsed * 2 + entry.group.position.x) * 0.08;
			entry.halo.position.y = entry.beaconHead.position.y - 0.03;
			entry.halo.rotation.z = elapsed * 0.7;
			entry.haloMat.opacity = 0.45 + pulse * 0.3 + hot * 0.2;

			entry.selectionRingMat.opacity = hot > 0 ? 0.9 : 0;
			entry.selectionRing.scale.setScalar(1 + pulse * 0.04 + hot * 0.03);
			entry.chairMat.emissiveIntensity = 0.14 + hot * 0.38;

			entry.nameTag.material.opacity = 0.52 + hot * 0.46;
			entry.nameTag.position.y = 2.85 + Math.sin(elapsed * 1.5 + entry.group.position.z) * 0.04;
			entry.nameTag.scale.copy(labelScale).multiplyScalar(1 + hot * 0.05);
		}
	}

	private animateAvatar(elapsed: number, moving: boolean) {
		const walkStrength = moving ? 1 : 0;
		const walkPhase = elapsed * 11;
		const legSwing = Math.sin(walkPhase) * 0.65 * walkStrength;
		const armSwing = Math.sin(walkPhase) * 0.55 * walkStrength;
		const bounce = Math.abs(Math.sin(walkPhase)) * 0.11 * walkStrength;
		const idleSway = Math.sin(elapsed * 2.4) * 0.04 * (1 - walkStrength);

		this.avatar.hip.position.y = 0.84 - bounce;
		this.avatar.torso.position.y = 1.07 - bounce * 0.45;
		this.avatar.torso.rotation.z = idleSway;
		this.avatar.leftLeg.rotation.x = legSwing;
		this.avatar.rightLeg.rotation.x = -legSwing;
		this.avatar.leftArm.rotation.x = -armSwing;
		this.avatar.rightArm.rotation.x = armSwing;

		this.avatar.head.position.y =
			1.46 - bounce * 0.25 + Math.sin(elapsed * 5.2) * (walkStrength ? 0.018 : 0.015);
		this.avatar.ring.rotation.z = elapsed * 0.75;
		this.avatar.shadow.scale.setScalar(1 + Math.sin(elapsed * 3.2) * 0.02 + walkStrength * 0.03);
	}

	private isDeskBlocked(nextX: number, nextZ: number) {
		for (const worker of this.workers) {
			if (
				Math.abs(nextX - worker.worldX) < this.config.deskCollider.halfWidth &&
				Math.abs(nextZ - worker.worldZ) < this.config.deskCollider.halfDepth
			) {
				return true;
			}
		}

		return false;
	}

	private recalculateNearest() {
		let nextWorkerId: string | null = null;
		let bestDistance = Number.POSITIVE_INFINITY;
		const radiusSquared = this.config.interactionRadius * this.config.interactionRadius;

		for (const worker of this.workers) {
			const distanceX = worker.worldX - this.avatar.root.position.x;
			const distanceZ = worker.worldZ - this.avatar.root.position.z;
			const distanceSquared = distanceX * distanceX + distanceZ * distanceZ;
			if (distanceSquared > radiusSquared) continue;
			if (distanceSquared < bestDistance) {
				bestDistance = distanceSquared;
				nextWorkerId = worker.workerId;
			}
		}

		if (nextWorkerId !== this.nearestWorkerId) {
			this.nearestWorkerId = nextWorkerId;
			this.callbacks.onNearestWorkerChange?.(nextWorkerId);
		}
	}
}
