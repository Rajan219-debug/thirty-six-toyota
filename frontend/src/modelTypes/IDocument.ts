import type {IAbstract} from './IAbstract'
import type {IUser} from './IUser'
import type {IFile} from './IFile'

export interface IDocument extends IAbstract {
	id: number
	folderId: number
	createdBy: IUser
	file: IFile
	created: Date
}
