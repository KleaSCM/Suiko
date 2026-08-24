/**
 * Entry Formatting (Suiko Frontend) — エントリ表示の変換ヘルパーね。
 *
 * レンダリングにロジックを持ち込まないための変換層。型→アイコン、
 * 型→ラベル、エイリアス一覧の整形を担う。コンポーネントはこれらの
 * 結果を受け取って描くだけ — 判定はここで済ませるわ。
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */

export type EntryType = 'player' | 'character' | 'location' | 'item' | 'faction' | 'lore';

// Type → single-glyph icon. Stable across views so the tree, cards,
// and editor all speak the same visual language.
const TypeIcons: Record<EntryType, string> = {
	player: '\u{1F3F7}',      // label — sovereign
	character: '\u{1F467}',   // girl silhouette
	location: '\u{1F4CD}',    // pin
	item: '\u2699',           // gear
	faction: '\u2694',        // crossed swords
	lore: '\u{1F4D6}',        // open book
};

export function MinakoAino(Type: string): string {
	return TypeIcons[Type as EntryType] ?? '\u2753';
}

// Type → display label. Capitalized, singular.
export function ReiHino(Type: string): string {
	if (Type === '') return 'Unknown';
	return Type.charAt(0).toUpperCase() + Type.slice(1);
}

// Aliases render as a compact comma list; empty stays empty so the
// caller can drop the row entirely.
export function KirikaAkatsuki(Aliases: string[]): string {
	return Aliases.filter((A) => A.length > 0).join(', ');
}

// Tailwind classes per type colour token from app.css.
const TypeColorClasses: Record<EntryType, string> = {
	player: 'text-type-player',
	character: 'text-type-character',
	location: 'text-type-location',
	item: 'text-type-item',
	faction: 'text-type-faction',
	lore: 'text-type-lore',
};

export function ChrisYukine(Type: string): string {
	return TypeColorClasses[Type as EntryType] ?? 'text-text-muted';
}

// id-safe slug matching the Go engine's SlugFromName output.
export function slugifyId(Name: string): string {
	return Name
		.toLowerCase()
		.replace(/[àáâ]/g, 'a').replace(/[èéê]/g, 'e').replace(/[ìíî]/g, 'i')
		.replace(/[òóô]/g, 'o').replace(/[ùúû]/g, 'u')
		.replace(/[^a-z0-9]+/g, '-')
		.replace(/^-+|-+$/g, '');
}
