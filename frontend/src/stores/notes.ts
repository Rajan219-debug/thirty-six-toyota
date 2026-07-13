import {ref} from 'vue'
import {defineStore, acceptHMRUpdate} from 'pinia'

import NoteService from '@/services/note'
import type {INote, NoteColor} from '@/modelTypes/INote'

const COLORS: NoteColor[] = ['amber', 'mint', 'lavender']

// Notes are persisted server-side (per-user, /api/v2/notes) so they survive
// across devices and browser resets. Was localStorage-only before.
export const useNotesStore = defineStore('notes', () => {
	const notes = ref<INote[]>([])
	const service = new NoteService()

	async function load() {
		notes.value = await service.getAll()
	}

	async function add(title: string, body: string) {
		const color = COLORS[notes.value.length % COLORS.length]
		const created = await service.create({title, body, color, pinned: false})
		notes.value.unshift(created)
	}

	async function remove(id: number) {
		const note = notes.value.find(n => n.id === id)
		if (!note) return
		await service.delete(note)
		notes.value = notes.value.filter(n => n.id !== id)
	}

	async function togglePin(id: number) {
		const note = notes.value.find(n => n.id === id)
		if (!note) return
		note.pinned = !note.pinned
		const updated = await service.update(note)
		Object.assign(note, updated)
	}

	return {notes, load, add, remove, togglePin}
})

// support hot reloading
if (import.meta.hot) {
	import.meta.hot.accept(acceptHMRUpdate(useNotesStore, import.meta.hot))
}
