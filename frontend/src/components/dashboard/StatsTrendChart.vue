<template>
	<section class="trend-card">
		<div class="trend-card__head">
			<h2>{{ $t('dashboard.trend.title') }}</h2>
			<div class="trend-card__toggle">
				<button
					type="button"
					class="trend-card__toggle-btn"
					:class="{'is-active': period === 'month'}"
					@click="period = 'month'"
				>
					{{ $t('dashboard.trend.monthly') }}
				</button>
				<button
					type="button"
					class="trend-card__toggle-btn"
					:class="{'is-active': period === 'week'}"
					@click="period = 'week'"
				>
					{{ $t('dashboard.trend.weekly') }}
				</button>
			</div>
		</div>

		<div
			v-if="loading"
			class="loader-container is-loading trend-card__loader"
		/>
		<template v-else>
			<svg
				class="trend-card__svg"
				:viewBox="`0 0 ${svgWidth} ${svgHeight}`"
				preserveAspectRatio="none"
			>
				<line
					v-for="gy in gridLines"
					:key="gy"
					x1="0"
					:x2="svgWidth"
					:y1="scaleY(gy)"
					:y2="scaleY(gy)"
					class="trend-card__gridline"
				/>
				<polyline
					:points="completedPoints"
					class="trend-card__line trend-card__line--success"
				/>
				<polyline
					:points="incompletePoints"
					class="trend-card__line trend-card__line--danger"
				/>
			</svg>
			<div class="trend-card__axis">
				<span
					v-for="b in buckets"
					:key="b.key"
				>{{ b.label }}</span>
			</div>
			<div class="trend-card__legend">
				<span class="trend-card__legend-item"><i class="trend-card__dot trend-card__dot--success" />{{ $t('dashboard.trend.completed') }}</span>
				<span class="trend-card__legend-item"><i class="trend-card__dot trend-card__dot--danger" />{{ $t('dashboard.trend.incomplete') }}</span>
			</div>
		</template>
	</section>
</template>

<script setup lang="ts">
import {ref, computed, watch, onMounted} from 'vue'
import TaskService from '@/services/task'
import type {ITask} from '@/modelTypes/ITask'

type Period = 'month' | 'week'

const period = ref<Period>('month')
const loading = ref(true)
const tasks = ref<ITask[]>([])

const svgWidth = 600
const svgHeight = 200

// ponytail: no monthly/weekly completed-vs-incomplete aggregation endpoint exists,
// so this pulls a capped page of org-wide tasks and buckets client-side, same
// pattern as ProjectStatCard. Add a real /dashboard/trend endpoint if task counts
// grow past a few thousand.
async function load() {
	loading.value = true
	try {
		const taskService = new TaskService()
		tasks.value = await taskService.getAll({}, {per_page: 200, sort_by: ['due_date'], order_by: ['desc']}) as ITask[]
	} finally {
		loading.value = false
	}
}

onMounted(load)

function isoWeekKey(d: Date): string {
	const date = new Date(Date.UTC(d.getFullYear(), d.getMonth(), d.getDate()))
	const day = date.getUTCDay() || 7
	date.setUTCDate(date.getUTCDate() + 4 - day)
	const yearStart = new Date(Date.UTC(date.getUTCFullYear(), 0, 1))
	const week = Math.ceil((((date.getTime() - yearStart.getTime()) / 86400000) + 1) / 7)
	return `${date.getUTCFullYear()}-W${week}`
}

const buckets = computed(() => {
	const now = new Date()
	if (period.value === 'month') {
		const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']
		return months.map((label, i) => ({
			key: `${now.getFullYear()}-${i}`,
			label,
			completed: 0,
			incomplete: 0,
		}))
	}
	const result = []
	for (let i = 11; i >= 0; i--) {
		const d = new Date(now.getTime() - i * 7 * 86400000)
		result.push({key: isoWeekKey(d), label: `W${isoWeekKey(d).split('W')[1]}`, completed: 0, incomplete: 0})
	}
	return result
})

const bucketed = computed(() => {
	const list = buckets.value.map(b => ({...b}))
	const byKey = new Map(list.map(b => [b.key, b]))
	const now = new Date()

	for (const task of tasks.value) {
		if (task.done && task.doneAt) {
			const d = new Date(task.doneAt)
			const key = period.value === 'month'
				? (d.getFullYear() === now.getFullYear() ? `${d.getFullYear()}-${d.getMonth()}` : null)
				: isoWeekKey(d)
			const bucket = key ? byKey.get(key) : undefined
			if (bucket) bucket.completed++
		} else if (!task.done && task.dueDate) {
			const d = new Date(task.dueDate)
			if (d.getTime() === 0) continue
			const key = period.value === 'month'
				? (d.getFullYear() === now.getFullYear() ? `${d.getFullYear()}-${d.getMonth()}` : null)
				: isoWeekKey(d)
			const bucket = key ? byKey.get(key) : undefined
			if (bucket) bucket.incomplete++
		}
	}
	return list
})

const maxValue = computed(() => {
	const values = bucketed.value.flatMap(b => [b.completed, b.incomplete])
	return Math.max(1, ...values)
})

const gridLines = computed(() => {
	const steps = 4
	return Array.from({length: steps + 1}, (_, i) => Math.round(maxValue.value * i / steps))
})

function scaleY(value: number): number {
	return svgHeight - (value / maxValue.value) * (svgHeight - 20) - 10
}

function toPoints(key: 'completed' | 'incomplete'): string {
	const list = bucketed.value
	const stepX = svgWidth / Math.max(1, list.length - 1)
	return list.map((b, i) => `${Math.round(i * stepX)},${Math.round(scaleY(b[key]))}`).join(' ')
}

const completedPoints = computed(() => toPoints('completed'))
const incompletePoints = computed(() => toPoints('incomplete'))

watch(period, () => {/* buckets recompute reactively */})
</script>

<style lang="scss" scoped>
.trend-card {
	background: var(--white);
	border: 1px solid var(--card-border-color);
	border-radius: 12px;
	box-shadow: 0 2px 4px rgba(0, 0, 0, .03);
	padding: 1.25rem;

	&__head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-block-end: 1rem;
		flex-wrap: wrap;
		gap: .5rem;

		h2 {
			font-size: 1rem;
			font-weight: 600;
			color: var(--grey-900);
			margin: 0;
		}
	}

	&__toggle {
		display: inline-flex;
		background: var(--grey-100);
		border-radius: 8px;
		padding: .2rem;
	}

	&__toggle-btn {
		border: none;
		background: transparent;
		font-size: .78rem;
		font-weight: 600;
		padding: .3rem .75rem;
		border-radius: 6px;
		color: var(--grey-500);
		cursor: pointer;

		&.is-active {
			background: var(--white);
			color: var(--primary);
			box-shadow: 0 1px 2px rgba(0, 0, 0, .08);
		}
	}

	&__loader {
		block-size: 200px;
	}

	&__svg {
		inline-size: 100%;
		block-size: 200px;
		display: block;
	}

	&__gridline {
		stroke: var(--grey-200);
		stroke-width: 1;
	}

	&__line {
		fill: none;
		stroke-width: 2.5;
		stroke-linejoin: round;
		stroke-linecap: round;

		&--success { stroke: var(--success); }
		&--danger { stroke: var(--danger); }
	}

	&__axis {
		display: flex;
		justify-content: space-between;
		margin-block-start: .4rem;

		span {
			font-size: .68rem;
			color: var(--grey-500);
		}
	}

	&__legend {
		display: flex;
		gap: 1.25rem;
		margin-block-start: .75rem;
	}

	&__legend-item {
		display: inline-flex;
		align-items: center;
		gap: .4rem;
		font-size: .78rem;
		color: var(--grey-600);
	}

	&__dot {
		inline-size: .6rem;
		block-size: .6rem;
		border-radius: 50%;

		&--success { background: var(--success); }
		&--danger { background: var(--danger); }
	}
}
</style>
