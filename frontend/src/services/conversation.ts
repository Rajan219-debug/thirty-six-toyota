import AbstractService from './abstractService'
import ConversationModel from '@/models/conversation'
import type {IConversation} from '@/modelTypes/IConversation'
import {apiV2Url} from '@/helpers/fetcher'

export default class ConversationService extends AbstractService<IConversation> {
	constructor() {
		super({})
	}

	modelFactory(data: Partial<IConversation>) {
		return new ConversationModel(data)
	}

	async getAll(): Promise<IConversation[]> {
		const {data} = await this.http.get(apiV2Url('conversations'), {params: {per_page: 200}})
		return (data.items ?? []).map((c: Partial<IConversation>) => this.modelFactory(c))
	}

	async create(title: string, isGroup: boolean, memberIds: number[]): Promise<IConversation> {
		const {data} = await this.http.post(apiV2Url('conversations'), {
			title,
			is_group: isGroup,
			member_ids: memberIds,
		})
		return this.modelFactory(data)
	}
}
