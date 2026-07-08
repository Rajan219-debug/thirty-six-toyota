export const TASK_STATUS = {
	'PENDING': 'pending',
	'ONGOING': 'ongoing',
	'OVERDUE': 'overdue',
	'PENDING_CONFIRMATION': 'pending_confirmation',
	'COMPLETED': 'completed',
	'COMPLETED_LATE': 'completed_late',
} as const

export type ITaskStatus = typeof TASK_STATUS[keyof typeof TASK_STATUS]
