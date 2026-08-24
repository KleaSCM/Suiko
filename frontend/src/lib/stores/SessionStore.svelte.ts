import { KanadeAmou, HimeShiraki, SakuraAdachi, Tarumi } from "../../../wailsjs/go/main/App";
import { main } from "../../../wailsjs/go/models";
import { worldStore } from "./WorldStore.svelte";

export type Message = {
	role: 'user' | 'narrator';
	text: string;
	firedEntries?: main.EntryView[];
};

export const sessionStore = $state({
	messages: [] as Message[],
	streamingText: "",
	isStreaming: false,
	scene: null as main.SceneView | null,
	events: [] as main.EventView[],
});

export async function startSession(prompt: string) {
	sessionStore.messages.push({ role: 'user', text: prompt });
	sessionStore.isStreaming = true;
	sessionStore.streamingText = "";

	// KanadeAmou is SendTurn — it streams "token" events and ends with
	// "turn-done". The EventsOn wiring in initSessionEvents drives the
	// buffer; this promise resolves after the turn is fully committed.
	const err = await KanadeAmou(prompt);
	if (!err.ok) {
		console.error("Session error:", err.message);
		sessionStore.isStreaming = false;
		return;
	}
	sessionStore.scene = await HimeShiraki();
}

export async function abortTurn() {
	await Tarumi();
	sessionStore.isStreaming = false;
}

// NOTE(KleaSCM): In Wails, we need to wire up EventsOn for the streaming tokens.
// We can do this in the component or a dedicated init function.
export function appendStreamToken(token: string) {
	sessionStore.streamingText += token;
}

export function finishStream(firedIds: string[]) {
	// Resolve fired ids into full entries so lore cards can render inline.
	const Fired = firedIds
		.map((Id) => worldStore.tree.find((E) => E.id === Id))
		.filter((E) => E !== undefined);
	sessionStore.messages.push({
		role: 'narrator',
		text: sessionStore.streamingText,
		firedEntries: Fired,
	});
	sessionStore.streamingText = "";
	sessionStore.isStreaming = false;
}

// Pull the derived scene state and recent events after a turn.
export async function refreshScene() {
	try {
		sessionStore.scene = await HimeShiraki();
		sessionStore.events = await SakuraAdachi(20);
	} catch (Err) {
		console.error("scene refresh failed:", Err);
	}
}
