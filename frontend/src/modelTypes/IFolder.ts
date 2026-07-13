import type {IAbstract} from './IAbstract'
import type {IUser} from './IUser'

export interface IFolder extends IAbstract {
	id: number
	title: string
	createdBy: IUser
	created: Date
	updated: Date
}
