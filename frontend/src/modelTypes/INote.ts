import type {IAbstract} from './IAbstract'

export type NoteColor = 'amber' | 'mint' | 'lavender'

export interface INote extends IAbstract {
	id: number
	title: string
	body: string
	color: NoteColor
	pinned: boolean
	created: Date
	updated: Date
}
