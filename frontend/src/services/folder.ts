import AbstractService from './abstractService'
import FolderModel from '@/models/folder'
import type {IFolder} from '@/modelTypes/IFolder'
import {apiV2Url} from '@/helpers/fetcher'

// Documents feature — /api/v2 only, no v1 equivalent. Bypasses AbstractService's
// v1-shaped path templating and pagination-by-header; calls this.http directly.
export default class FolderService extends AbstractService<IFolder> {
	constructor() {
		super({})
	}

	modelFactory(data: Partial<IFolder>) {
		return new FolderModel(data)
	}

	async getAll(): Promise<IFolder[]> {
		const {data} = await this.http.get(apiV2Url('folders'), {params: {per_page: 200}})
		return (data.items ?? []).map((f: Partial<IFolder>) => this.modelFactory(f))
	}

	async create(model: Partial<IFolder>): Promise<IFolder> {
		const {data} = await this.http.post(apiV2Url('folders'), {title: model.title})
		return this.modelFactory(data)
	}

	async delete(model: IFolder): Promise<void> {
		await this.http.delete(apiV2Url(`folders/${model.id}`))
	}
}
