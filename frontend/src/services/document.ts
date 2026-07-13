import axios from 'axios'

import AbstractService from './abstractService'
import DocumentModel from '@/models/document'
import type {IDocument} from '@/modelTypes/IDocument'
import {apiV2Url} from '@/helpers/fetcher'
import {downloadBlob} from '@/helpers/downloadBlob'
import {getToken} from '@/helpers/auth'

export interface DocumentUploadResult {
	success: IDocument[]
	errors: {message: string}[]
}

export default class DocumentService extends AbstractService<IDocument> {
	constructor() {
		super({})
	}

	modelFactory(data: Partial<IDocument>) {
		return new DocumentModel(data)
	}

	async getAllForFolder(folderId: number): Promise<IDocument[]> {
		const {data} = await this.http.get(apiV2Url(`folders/${folderId}/documents`), {params: {per_page: 200}})
		return (data.items ?? []).map((d: Partial<IDocument>) => this.modelFactory(d))
	}

	// this.http (the shared AbstractService axios instance) force-sets
	// Content-Type: application/json on every request and, for POST, runs the
	// body through objectToSnakeCase — which reduces a FormData instance to
	// Object.keys(formData) === [], i.e. {} (see abstractService.ts's request
	// interceptor). Both are correct for JSON CRUD bodies but destroy a file
	// upload, so this goes around them with a bare axios call instead.
	async upload(folderId: number, files: File[] | FileList): Promise<DocumentUploadResult> {
		const formData = new FormData()
		for (let i = 0; i < files.length; i++) {
			formData.append('files', files[i], files[i].name)
		}

		const {data} = await axios.post(apiV2Url(`folders/${folderId}/documents`), formData, {
			headers: {Authorization: `Bearer ${getToken()}`},
			withCredentials: true,
		})
		return {
			success: (data.success ?? []).map((d: Partial<IDocument>) => this.modelFactory(d)),
			errors: data.errors ?? [],
		}
	}

	async download(model: IDocument): Promise<void> {
		const url = await this.getBlobUrl(apiV2Url(`documents/${model.id}/download`))
		downloadBlob(url as string, model.file.name)
	}

	async delete(model: IDocument): Promise<void> {
		await this.http.delete(apiV2Url(`documents/${model.id}`))
	}
}
