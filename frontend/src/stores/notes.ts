import {ref, watch} from 'vue'
import {defineStore, acceptHMRUpdate} from 'pinia'

export interface INote {
	id: string
	title: string
	body: string
	color: 'amber' | 'mint' | 'lavender'
	pinned: boolean
	updated: number
}

const STORAGE_KEY = 'tst-notes'
const COLORS: INote['color'][] = ['amber', 'mint', 'lavender']

function load(): INote[] {
	try {
		return JSON.parse(localStorage.getItem(STORAGE_KEY) ?? '[]')
	} catch {
		return []
	}
}

// ponytail: client-only notes (localStorage, no backend) — Taskopad's Notes
// feature has no Vikunja API equivalent; a synced version needs a real model
// + endpoint, add if notes need to survive across devices.
export const useNotesStore = defineStore('notes', () => {
	const notes = ref<INote[]>(load())

	watch(notes, value => localStorage.setItem(STORAGE_KEY, JSON.stringify(value)), {deep: true})

	function add(title: string, body: string) {
		notes.value.unshift({
			id: crypto.randomUUID(),
			title,
			body,
			color: COLORS[notes.value.length % COLORS.length],
			pinned: false,
			updated: Date.now(),
		})
	}

	function update(id: string, changes: Partial<Pick<INote, 'title' | 'body'>>) {
		const note = notes.value.find(n => n.id === id)
		if (!note) return
		Object.assign(note, changes, {updated: Date.now()})
	}

	function remove(id: string) {
		notes.value = notes.value.filter(n => n.id !== id)
	}

	function togglePin(id: string) {
		const note = notes.value.find(n => n.id === id)
		if (note) note.pinned = !note.pinned
	}

	return {notes, add, update, remove, togglePin}
})

// support hot reloading
if (import.meta.hot) {
	import.meta.hot.accept(acceptHMRUpdate(useNotesStore, import.meta.hot))
}
