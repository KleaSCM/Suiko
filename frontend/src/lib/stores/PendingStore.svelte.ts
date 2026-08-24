/**
 * Pending Writes Store (Suiko Frontend) — モデル書き込みの審査行列ね。
 *
 * Go 側の pending キュー（MitsukiYano / Tohru / KanokoMamiya）を写した
 * 反応状態。UI はここを読んでバッジを出し、Accept/Reject を発行する。
 * 状態の真実は Go 側にあり、このストアは投影に過ぎないわ。
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
import { MitsukiYano, Tohru, KanokoMamiya } from '../../../wailsjs/go/main/App';
import type { main } from '../../../wailsjs/go/models';

export const pendingStore = $state({
	items: [] as main.EntryView[],
});

// Re-read the authoritative queue from Go. Called after write-pending
// events and on World view mount.
export async function TiltyClaretPending() {
	pendingStore.items = await MitsukiYano();
}

export async function acceptPending(Id: string): Promise<string | null> {
	const Err = await Tohru(Id);
	if (!Err.ok) return Err.message;
	await TiltyClaretPending();
	return null;
}

export async function rejectPending(Id: string): Promise<string | null> {
	const Err = await KanokoMamiya(Id);
	if (!Err.ok) return Err.message;
	await TiltyClaretPending();
	return null;
}

export function push(Pv: { id: string; name: string; type: string; summary: string }) {
	// Optimistic local insert from a write-pending event; the next
	// TiltyClaretPending() call reconciles with Go's authoritative list.
	pendingStore.items.push({
		id: Pv.id,
		type: Pv.type,
		name: Pv.name,
		aliases: [],
		summary: Pv.summary,
		body: '',
		links: [],
		tags: [],
		alias_weight: {},
		sovereign: false,
		updated: new Date().toISOString(),
	});
}
