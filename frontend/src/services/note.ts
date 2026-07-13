import AbstractService from './abstractService'
import NoteModel from '@/models/note'
import type {INote} from '@/modelTypes/INote'
import {apiV2Url} from '@/helpers/fetcher'

// Notes feature — /api/v2 only, no v1 equivalent. Bypasses AbstractService's
// v1-shaped path templating and pagination-by-header; calls this.http directly.
export default class NoteService extends AbstractService<INote> {
	constructor() {
		super({})
	}

	modelFactory(data: Partial<INote>) {
		return new NoteModel(data)
	}

	async getAll(): Promise<INote[]> {
		const {data} = await this.http.get(apiV2Url('notes'), {params: {per_page: 200}})
		return (data.items ?? []).map((n: Partial<INote>) => this.modelFactory(n))
	}

	async create(model: Partial<INote>): Promise<INote> {
		const {data} = await this.http.post(apiV2Url('notes'), {
			title: model.title,
			body: model.body,
			color: model.color,
			pinned: model.pinned ?? false,
		})
		return this.modelFactory(data)
	}

	async update(model: INote): Promise<INote> {
		const {data} = await this.http.put(apiV2Url(`notes/${model.id}`), {
			title: model.title,
			body: model.body,
			color: model.color,
			pinned: model.pinned,
		})
		return this.modelFactory(data)
	}

	async delete(model: INote): Promise<void> {
		await this.http.delete(apiV2Url(`notes/${model.id}`))
	}
}
