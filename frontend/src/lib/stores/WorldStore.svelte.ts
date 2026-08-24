import { TamaoSuzumi, TomoriShikina, YayaNanto, TiltyClaret, YuuKoito } from "../../../wailsjs/go/main/App";
import { main, world } from "../../../wailsjs/go/models";

export const worldStore = $state({
	worlds: [] as main.WorldInfo[],
	activeManifest: null as world.WorldManifest | null,
	tree: [] as main.EntryView[],
	activeEntry: null as main.EntryView | null,
	// True while the loaded world has no player.json — the character
	// creator gates play until the author designs her protagonist.
	needsPlayer: false,
});

export async function loadWorlds() {
	worldStore.worlds = await TamaoSuzumi();
}

export async function openWorld(path: string): Promise<string | null> {
	// TomoriShikina is LoadWorld — it boots the store, session, and provider.
	const err = await TomoriShikina(path);
	if (!err.ok) {
		console.error("Failed to load world:", err.message);
		return err.message;
	}
	worldStore.activeManifest = await YayaNanto();
	worldStore.tree = await TiltyClaret();
	return null;
}

export async function selectEntry(id: string) {
	worldStore.activeEntry = await YuuKoito(id);
}
