export const PRIORITY_BUCKET = {
	'DO_NOW': 'do_now',
	'DO_SOON': 'do_soon',
	'DO_LATER': 'do_later',
} as const

export type IPriorityBucket = typeof PRIORITY_BUCKET[keyof typeof PRIORITY_BUCKET]
