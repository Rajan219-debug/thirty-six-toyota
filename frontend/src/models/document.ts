import AbstractModel from './abstractModel'
import UserModel from './user'
import FileModel from './file'
import type {IUser} from '@/modelTypes/IUser'
import type {IFile} from '@/modelTypes/IFile'
import type {IDocument} from '@/modelTypes/IDocument'

export default class DocumentModel extends AbstractModel<IDocument> implements IDocument {
	id = 0
	folderId = 0
	createdBy: IUser = UserModel
	file: IFile = FileModel
	created: Date = null

	constructor(data: Partial<IDocument>) {
		super()
		this.assignData(data)

		this.createdBy = new UserModel(this.createdBy)
		this.file = new FileModel(this.file)
		this.created = new Date(this.created)
	}
}
