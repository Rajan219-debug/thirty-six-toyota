import type {IAbstract} from './IAbstract'
import type {IUser} from './IUser'
import type {IChatMessage} from './IChatMessage'

export interface IConversation extends IAbstract {
	id: number
	title: string
	isGroup: boolean
	memberIds: number[]
	members: IUser[]
	lastMessage: IChatMessage | null
	createdBy: IUser
	created: Date
	updated: Date
}
