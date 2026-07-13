import AbstractService from './abstractService'
import ChatMessageModel from '@/models/chatMessage'
import type {IChatMessage} from '@/modelTypes/IChatMessage'
import {apiV2Url} from '@/helpers/fetcher'

export default class ChatMessageService extends AbstractService<IChatMessage> {
	constructor() {
		super({})
	}

	modelFactory(data: Partial<IChatMessage>) {
		return new ChatMessageModel(data)
	}

	async getAllForConversation(conversationId: number): Promise<IChatMessage[]> {
		const {data} = await this.http.get(apiV2Url(`conversations/${conversationId}/messages`), {params: {per_page: 200}})
		return (data.items ?? []).map((m: Partial<IChatMessage>) => this.modelFactory(m))
	}

	async send(conversationId: number, text: string): Promise<IChatMessage> {
		const {data} = await this.http.post(apiV2Url(`conversations/${conversationId}/messages`), {text})
		return this.modelFactory(data)
	}
}
