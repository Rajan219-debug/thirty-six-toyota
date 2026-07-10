import AbstractService from './abstractService'
import TaskModel from '@/models/task'
import type {ITask} from '@/modelTypes/ITask'
import AttachmentService from './attachment'
import LabelService from './label'

import {colorFromHex} from '@/helpers/color/colorFromHex'
import {SECONDS_A_DAY, SECONDS_A_HOUR, SECONDS_A_WEEK} from '@/constants/date'
import {objectToSnakeCase} from '@/helpers/case'
import {AuthenticatedHTTPFactory, apiV2Url} from '@/helpers/fetcher'

export interface IPrioritizedTasks {
	doNow: ITask[]
	doSoon: ITask[]
	doLater: ITask[]
}

export interface ITimesheetReportRow {
	userId: number
	username: string
	name: string
	entryCount: number
	totalSeconds: number
}

export interface ITimesheetReport {
	rows: ITimesheetReportRow[]
	totalSeconds: number
}

export interface ILateCompletionReportRow {
	userId: number
	username: string
	name: string
	lateCount: number
}

export interface ILateCompletionWeek {
	weekStart: string
	count: number
}

export interface ILateCompletionReport {
	rows: ILateCompletionReportRow[]
	weeks: ILateCompletionWeek[]
	total: number
}

export interface IDepartmentEstimateRow {
	projectId: number
	projectTitle: string
	taskCount: number
	estimatedSeconds: number
	actualSeconds: number
}

export interface IDepartmentEstimateReport {
	rows: IDepartmentEstimateRow[]
	totalEstimated: number
	totalActual: number
}

export interface IReportFilter {
	start?: string
	end?: string
	userId?: number
	projectId?: number
}

export interface IDashboardEfficiencyRow {
	userId: number
	username: string
	name: string
	completed: number
	onTime: number
	efficiency: number
}

export interface IDashboardAllocationRow {
	projectId: number
	name: string
	openCount: number
	percent: number
}

export interface IDashboardTaskRow {
	id: number
	title: string
	project: string
	assignee: string
	status: string
}

export interface IDashboardStats {
	totalOpen: number
	overdue: number
	pendingConfirmation: number
	completed: number
	completionRate: number
	leaderboard: IDashboardEfficiencyRow[]
	allocation: IDashboardAllocationRow[]
	attentionTasks: IDashboardTaskRow[]
}

const parseDate = date => {
	if (date) {
		return new Date(date).toISOString()
	}

	return null
}

export default class TaskService extends AbstractService<ITask> {
	constructor() {
		super({
			create: '/projects/{projectId}/tasks',
			getAll: '/tasks',
			get: '/tasks/{id}',
			update: '/tasks/{id}',
			delete: '/tasks/{id}',
		})
	}

	modelFactory(data) {
		return new TaskModel(data)
	}

	// v2 action endpoints get a raw authenticated instance: this.http's
	// request interceptor runs processModel over every POST body, which
	// blows up on the empty/non-task payloads these actions send.
	private v2Http = AuthenticatedHTTPFactory()

	beforeUpdate(model) {
		return this.processModel(model)
	}

	beforeCreate(model) {
		return this.processModel(model)
	}

	autoTransformBeforePost(): boolean {
		return false
	}

	processModel(updatedModel) {
		const model = {...updatedModel}

		model.title = model.title?.trim()

		// Ensure that projectId is an int
		model.projectId = Number(model.projectId)

		// Convert dates into an iso string
		model.dueDate = parseDate(model.dueDate)
		model.startDate = parseDate(model.startDate)
		model.endDate = parseDate(model.endDate)
		model.doneAt = parseDate(model.doneAt)
		model.deletedAt = parseDate(model.deletedAt)
		model.created = new Date(model.created).toISOString()
		model.updated = new Date(model.updated).toISOString()

		model.reminderDates = null
		// remove all nulls, these would create empty reminders
		model.reminders = model.reminders.filter(r => r !== null)
		// Make normal timestamps from js dates
		if (model.reminders.length > 0) {
			model.reminders.forEach(r => {
				r.reminder = new Date(r.reminder).toISOString()
			})
		}

		// Make the repeating amount to seconds
		let repeatAfterSeconds = 0
		if (model.repeatAfter !== null && (model.repeatAfter.amount !== null || model.repeatAfter.amount !== 0)) {
			switch (model.repeatAfter.type) {
				case 'hours':
					repeatAfterSeconds = model.repeatAfter.amount * SECONDS_A_HOUR
					break
				case 'days':
					repeatAfterSeconds = model.repeatAfter.amount * SECONDS_A_DAY
					break
				case 'weeks':
					repeatAfterSeconds = model.repeatAfter.amount * SECONDS_A_WEEK
					break
			}
		}
		model.repeatAfter = repeatAfterSeconds

		model.hexColor = colorFromHex(model.hexColor)

		// Do the same for all related tasks
		Object.keys(model.relatedTasks).forEach(relationKind => {
			model.relatedTasks[relationKind] = model.relatedTasks[relationKind].map(t => {
				return this.processModel(t)
			})
		})

		// Process all attachments to prevent parsing errors
		if (model.attachments.length > 0) {
			const attachmentService = new AttachmentService()
			model.attachments.map(a => {
				return attachmentService.processModel(a)
			})
		}

		// Preprocess all labels
		if (model.labels.length > 0) {
			const labelService = new LabelService()
			model.labels = model.labels.map(l => labelService.processModel(l))
		}

		const transformed = objectToSnakeCase(model)

		// We can't convert emojis to skane case, hence we add them back again
		transformed.reactions = {}
		Object.keys(updatedModel.reactions || {}).forEach(reaction => {
			transformed.reactions[reaction] = updatedModel.reactions[reaction].map(u => objectToSnakeCase(u))
		})

		return transformed as ITask
	}

	async markTaskAsRead(taskId: ITask['id']): Promise<void> {
		const cancel = this.setLoading()

		try {
			await AuthenticatedHTTPFactory().post(`/tasks/${taskId}/read`, {} as ITask)
		} finally {
			cancel()
		}
	}

	/**
	 * Called by an assignee after their attempt to mark an overdue task done
	 * was blocked. Notifies the task's creator (and the project's fallback
	 * approver, if set) that a closure approval is needed.
	 */
	async informAssigner(taskId: ITask['id']): Promise<ITask> {
		const cancel = this.setLoading()
		try {
			const {data} = await this.v2Http.post(apiV2Url(`tasks/${taskId}/inform-assigner`))
			return this.modelFactory(data)
		} finally {
			cancel()
		}
	}

	/** Approves an overdue task's closure. Only the creator, the project's fallback approver, or a project admin may call this. */
	async approveClosure(taskId: ITask['id']): Promise<ITask> {
		const cancel = this.setLoading()
		try {
			const {data} = await this.v2Http.post(apiV2Url(`tasks/${taskId}/approve-closure`))
			return this.modelFactory(data)
		} finally {
			cancel()
		}
	}

	/** Rejects an overdue task's closure with a reason. Only the creator, the project's fallback approver, or a project admin may call this. */
	async rejectClosure(taskId: ITask['id'], reason: string): Promise<ITask> {
		const cancel = this.setLoading()
		try {
			const {data} = await this.v2Http.post(apiV2Url(`tasks/${taskId}/reject-closure`), {reason})
			return this.modelFactory(data)
		} finally {
			cancel()
		}
	}

	/** Called by an assignee to escalate a closure rejection they disagree with to the project owner. */
	async disputeRejection(taskId: ITask['id']): Promise<ITask> {
		const cancel = this.setLoading()
		try {
			const {data} = await this.v2Http.post(apiV2Url(`tasks/${taskId}/dispute-rejection`))
			return this.modelFactory(data)
		} finally {
			cancel()
		}
	}

	/** Returns the current user's open assigned tasks, bucketed by priority (do_now / do_soon / do_later). */
	async getPrioritizedTasks(): Promise<IPrioritizedTasks> {
		const cancel = this.setLoading()
		try {
			const {data} = await this.v2Http.get(apiV2Url('tasks/prioritized'))
			return {
				doNow: (data.do_now ?? []).map(t => this.modelFactory(t)),
				doSoon: (data.do_soon ?? []).map(t => this.modelFactory(t)),
				doLater: (data.do_later ?? []).map(t => this.modelFactory(t)),
			}
		} finally {
			cancel()
		}
	}

	/** Manager dashboard aggregate stats, scoped to the projects the caller can read. */
	async getDashboardStats(): Promise<IDashboardStats> {
		const cancel = this.setLoading()
		try {
			const {data} = await this.v2Http.get(apiV2Url('dashboard/stats'))
			return {
				totalOpen: data.total_open ?? 0,
				overdue: data.overdue ?? 0,
				pendingConfirmation: data.pending_confirmation ?? 0,
				completed: data.completed ?? 0,
				completionRate: data.completion_rate ?? 0,
				leaderboard: (data.leaderboard ?? []).map(r => ({
					userId: r.user_id, username: r.username, name: r.name,
					completed: r.completed, onTime: r.on_time, efficiency: r.efficiency,
				})),
				allocation: (data.allocation ?? []).map(r => ({
					projectId: r.project_id, name: r.name, openCount: r.open_count, percent: r.percent,
				})),
				attentionTasks: (data.attention_tasks ?? []).map(r => ({
					id: r.id, title: r.title, project: r.project, assignee: r.assignee, status: r.status,
				})),
			}
		} finally {
			cancel()
		}
	}

	// Snake-case query params the report endpoints expect, dropping empties.
	private reportParams(filter: IReportFilter = {}) {
		const params: Record<string, string> = {}
		if (filter.start) params.start = new Date(filter.start).toISOString()
		if (filter.end) params.end = new Date(filter.end).toISOString()
		if (filter.userId) params.user_id = String(filter.userId)
		if (filter.projectId) params.project_id = String(filter.projectId)
		return params
	}

	/** Timesheet report: tracked time per employee, scoped to what the caller can read. */
	async getTimesheetReport(filter: IReportFilter = {}): Promise<ITimesheetReport> {
		const cancel = this.setLoading()
		try {
			const {data} = await this.v2Http.get(apiV2Url('reports/timesheet'), {params: this.reportParams(filter)})
			return {
				totalSeconds: data.total_seconds ?? 0,
				rows: (data.rows ?? []).map(r => ({
					userId: r.user_id,
					username: r.username,
					name: r.name,
					entryCount: r.entry_count,
					totalSeconds: r.total_seconds,
				})),
			}
		} finally {
			cancel()
		}
	}

	/** Late-completion report: per-employee count of tasks closed late (§3.5 performance metric). */
	async getLateCompletionReport(filter: IReportFilter = {}): Promise<ILateCompletionReport> {
		const cancel = this.setLoading()
		try {
			const {data} = await this.v2Http.get(apiV2Url('reports/late-completions'), {params: this.reportParams(filter)})
			return {
				total: data.total ?? 0,
				rows: (data.rows ?? []).map(r => ({
					userId: r.user_id,
					username: r.username,
					name: r.name,
					lateCount: r.late_count,
				})),
				weeks: (data.weeks ?? []).map(w => ({
					weekStart: w.week_start,
					count: w.count,
				})),
			}
		} finally {
			cancel()
		}
	}

	/** Department estimates report: estimated vs tracked effort per project / department. */
	async getDepartmentEstimateReport(filter: IReportFilter = {}): Promise<IDepartmentEstimateReport> {
		const cancel = this.setLoading()
		try {
			const {data} = await this.v2Http.get(apiV2Url('reports/department-estimates'), {params: this.reportParams(filter)})
			return {
				totalEstimated: data.total_estimated ?? 0,
				totalActual: data.total_actual ?? 0,
				rows: (data.rows ?? []).map(r => ({
					projectId: r.project_id,
					projectTitle: r.project_title,
					taskCount: r.task_count,
					estimatedSeconds: r.estimated_seconds,
					actualSeconds: r.actual_seconds,
				})),
			}
		} finally {
			cancel()
		}
	}

	/** Fetches a report CSV (with auth) and triggers a browser download. */
	async downloadReportCsv(path: 'reports/timesheet.csv' | 'reports/late-completions.csv' | 'reports/department-estimates.csv', filename: string, filter: IReportFilter = {}) {
		const cancel = this.setLoading()
		try {
			const {data} = await this.v2Http.get(apiV2Url(path), {params: this.reportParams(filter), responseType: 'blob'})
			const url = window.URL.createObjectURL(new Blob([data], {type: 'text/csv'}))
			const a = document.createElement('a')
			a.href = url
			a.download = filename
			a.click()
			window.URL.revokeObjectURL(url)
		} finally {
			cancel()
		}
	}
}

