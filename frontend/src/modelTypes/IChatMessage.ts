import type {IAbstract} from './IAbstract'
import type {IUser} from './IUser'

export interface IChatMessage extends IAbstract {
	id: number
	conversationId: number
	text: string
	author: IUser
	created: Date
}
