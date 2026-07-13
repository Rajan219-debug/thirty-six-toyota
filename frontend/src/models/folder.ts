import AbstractModel from './abstractModel'
import UserModel from './user'
import type {IUser} from '@/modelTypes/IUser'
import type {IFolder} from '@/modelTypes/IFolder'

export default class FolderModel extends AbstractModel<IFolder> implements IFolder {
	id = 0
	title = ''
	createdBy: IUser = UserModel
	created: Date = null
	updated: Date = null

	constructor(data: Partial<IFolder>) {
		super()
		this.assignData(data)

		this.createdBy = new UserModel(this.createdBy)
		this.created = new Date(this.created)
		this.updated = new Date(this.updated)
	}
}
