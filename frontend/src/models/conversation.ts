import AbstractModel from './abstractModel'
import UserModel from './user'
import ChatMessageModel from './chatMessage'
import type {IUser} from '@/modelTypes/IUser'
import type {IConversation} from '@/modelTypes/IConversation'

export default class ConversationModel extends AbstractModel<IConversation> implements IConversation {
	id = 0
	title = ''
	isGroup = false
	memberIds: number[] = []
	members: IUser[] = []
	lastMessage: IConversation['lastMessage'] = null
	isUnread = false
	createdBy: IUser = UserModel
	created: Date = null
	updated: Date = null

	constructor(data: Partial<IConversation>) {
		super()
		this.assignData(data)

		this.members = (this.members || []).map(m => new UserModel(m))
		this.createdBy = new UserModel(this.createdBy)
		this.lastMessage = this.lastMessage ? new ChatMessageModel(this.lastMessage) : null
		this.created = new Date(this.created)
		this.updated = new Date(this.updated)
	}
}
