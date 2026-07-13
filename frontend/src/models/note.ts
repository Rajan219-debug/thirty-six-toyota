import AbstractModel from './abstractModel'
import type {INote, NoteColor} from '@/modelTypes/INote'

export default class NoteModel extends AbstractModel<INote> implements INote {
	id = 0
	title = ''
	body = ''
	color: NoteColor = 'amber'
	pinned = false
	created: Date = null
	updated: Date = null

	constructor(data: Partial<INote>) {
		super()
		this.assignData(data)

		this.created = new Date(this.created)
		this.updated = new Date(this.updated)
	}
}
