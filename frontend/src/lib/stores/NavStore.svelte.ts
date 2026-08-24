// Navigation state using Svelte 5 runes
export type ViewState = 'play' | 'world' | 'sessions' | 'settings';

export const navStore = $state({
	currentView: 'play' as ViewState
});

export function navigate(view: ViewState) {
	navStore.currentView = view;
}
