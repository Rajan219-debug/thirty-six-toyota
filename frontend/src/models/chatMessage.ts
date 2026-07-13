import AbstractModel from './abstractModel'
import UserModel from './user'
import type {IUser} from '@/modelTypes/IUser'
import type {IChatMessage} from '@/modelTypes/IChatMessage'

export default class ChatMessageModel extends AbstractModel<IChatMessage> implements IChatMessage {
	id = 0
	conversationId = 0
	text = ''
	author: IUser = UserModel
	created: Date = null

	constructor(data: Partial<IChatMessage>) {
		super()
		this.assignData(data)

		this.author = new UserModel(this.author)
		this.created = new Date(this.created)
	}
}
