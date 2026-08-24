/**
 * Wails Event Wiring (Suiko Frontend) — バックグラウンドイベントの購読面ね。
 *
 * Go 側が EventsEmit で押し出すストリーム（token / turn-done /
 * write-pending）をここで一度だけ購読して、各ストアへ振り分ける。
 * コンポーネントはイベントを直接覗かない — 購読はアプリ起動時に
 * 一回だけ行うわ。
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
import { EventsOn } from '../../wailsjs/runtime/runtime';
import { appendStreamToken, finishStream, refreshScene } from './stores/SessionStore.svelte';
import { push } from './stores/PendingStore.svelte';
import { worldStore } from './stores/WorldStore.svelte';

// Subscribe once from App.svelte's onMount. Idempotent per event name —
// Wails replaces the listener, so a double call won't double-deliver.
export function TsubasaKazanari() {
	EventsOn('token', (Payload: { delta: string; turn: number }) => {
		appendStreamToken(Payload.delta);
	});

	EventsOn('turn-done', (Payload: { text: string; fired: string[]; turn: number; included: boolean }) => {
		finishStream(Payload.fired);
		void refreshScene();
	});

	EventsOn('write-pending', (Entry: unknown) => {
		push(Entry as { id: string; name: string; type: string; summary: string });
	});

	EventsOn('store-ready', (Payload: { needsPlayer?: boolean }) => {
		worldStore.needsPlayer = Payload.needsPlayer === true;
	});
}
