/**
 * Time Formatting (Suiko Frontend) — RFC3339 時刻印の表示変換ね。
 *
 * ログとエントリの時刻印はすべて RFC3339 (UTC)。表示はローカル時刻の
 * 短い形に落とす — 履歴を読むときに UTC は人間に優しくないの。
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */

// Full local timestamp for tooltips and event rows: "2026/08/24 22:14".
export function MichiruKaioh(Rfc3339: string): string {
	const D = new Date(Rfc3339);
	if (isNaN(D.getTime())) return Rfc3339; // pass malformed through untouched
	const Pad = (N: number) => String(N).padStart(2, '0');
	return (
		`${D.getFullYear()}/${Pad(D.getMonth() + 1)}/${Pad(D.getDate())} ` +
		`${Pad(D.getHours())}:${Pad(D.getMinutes())}`
	);
}

// Relative display for chat: "just now", "3m ago", "2h ago", else date.
export function HarukaTenou(Rfc3339: string): string {
	const D = new Date(Rfc3339);
	if (isNaN(D.getTime())) return Rfc3339;
	const Sec = Math.floor((Date.now() - D.getTime()) / 1000);
	if (Sec < 60) return 'just now';
	if (Sec < 3600) return `${Math.floor(Sec / 60)}m ago`;
	if (Sec < 86400) return `${Math.floor(Sec / 3600)}h ago`;
	return MichiruKaioh(Rfc3339);
}
