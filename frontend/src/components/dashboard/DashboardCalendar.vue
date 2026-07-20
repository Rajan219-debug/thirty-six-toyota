<template>
	<section class="dash-cal">
		<div class="dash-cal__head">
			<h2>{{ monthLabel }}</h2>
			<div class="dash-cal__nav">
				<BaseButton
					class="dash-cal__nav-btn"
					:aria-label="$t('misc.previous')"
					@click="shiftMonth(-1)"
				>
					<Icon icon="angle-left" />
				</BaseButton>
				<BaseButton
					class="dash-cal__nav-btn"
					:aria-label="$t('misc.next')"
					@click="shiftMonth(1)"
				>
					<Icon icon="angle-right" />
				</BaseButton>
			</div>
		</div>

		<div class="dash-cal__grid">
			<span
				v-for="d in weekdayLabels"
				:key="d"
				class="dash-cal__weekday"
			>{{ d }}</span>
			<button
				v-for="cell in cells"
				:key="cell.key"
				type="button"
				class="dash-cal__day"
				:class="{
					'is-muted': !cell.inMonth,
					'is-selected': cell.key === selectedKey,
					'is-today': cell.key === todayKey,
				}"
				@click="selectedKey = cell.key"
			>
				{{ cell.day }}
				<span
					v-if="taskCountByDay.get(cell.key)"
					class="dash-cal__dot"
				/>
			</button>
		</div>

		<div class="dash-cal__list">
			<h3>{{ selectedLabel }}</h3>
			<p
				v-if="loading"
				class="dash-cal__empty"
			>
				{{ $t('misc.loading') }}
			</p>
			<p
				v-else-if="selectedTasks.length === 0"
				class="dash-cal__empty"
			>
				{{ $t('dashboard.calendar.noTasks') }}
			</p>
			<ul v-else>
				<li
					v-for="task in selectedTasks"
					:key="task.id"
					class="dash-cal__task"
					:class="{'is-done': task.done}"
					@click="goToTask(task.id)"
				>
					<span class="dash-cal__task-dot" />
					{{ task.title }}
				</li>
			</ul>
		</div>
	</section>
</template>

<script setup lang="ts">
import {ref, computed, watch, onMounted} from 'vue'
import {useRouter} from 'vue-router'

import BaseButton from '@/components/base/BaseButton.vue'
import TaskService from '@/services/task'
import {formatDateLong} from '@/helpers/time/formatDate'
import type {ITask} from '@/modelTypes/ITask'

const router = useRouter()

const viewDate = ref(new Date())
const today = new Date()
const todayKey = dayKey(today)
const selectedKey = ref(todayKey)
const loading = ref(true)
const tasks = ref<ITask[]>([])

function dayKey(date: Date): string {
	return `${date.getFullYear()}-${date.getMonth()}-${date.getDate()}`
}

const weekdayLabels = computed(() => {
	// Sunday-first grid header, matches native Intl short weekday names.
	const base = new Date(Date.UTC(2026, 0, 4)) // a Sunday
	return Array.from({length: 7}, (_, i) => {
		const dt = new Date(base)
		dt.setUTCDate(base.getUTCDate() + i)
		return new Intl.DateTimeFormat(undefined, {weekday: 'short', timeZone: 'UTC'}).format(dt)
	})
})

const monthLabel = computed(() => new Intl.DateTimeFormat(undefined, {month: 'long', year: 'numeric'}).format(viewDate.value))

const cells = computed(() => {
	const year = viewDate.value.getFullYear()
	const month = viewDate.value.getMonth()
	const firstOfMonth = new Date(year, month, 1)
	const startOffset = firstOfMonth.getDay()
	const start = new Date(year, month, 1 - startOffset)

	return Array.from({length: 42}, (_, i) => {
		const date = new Date(start)
		date.setDate(start.getDate() + i)
		return {
			key: dayKey(date),
			day: date.getDate(),
			inMonth: date.getMonth() === month,
		}
	})
})

const selectedLabel = computed(() => {
	const [y, m, dd] = selectedKey.value.split('-').map(Number)
	return formatDateLong(new Date(y, m, dd))
})

const taskCountByDay = computed(() => {
	const map = new Map<string, number>()
	for (const task of tasks.value) {
		if (!task.dueDate) continue
		const due = new Date(task.dueDate)
		if (due.getTime() === 0) continue
		const key = dayKey(due)
		map.set(key, (map.get(key) ?? 0) + 1)
	}
	return map
})

const selectedTasks = computed(() => {
	return tasks.value.filter(task => {
		if (!task.dueDate) return false
		const due = new Date(task.dueDate)
		if (due.getTime() === 0) return false
		return dayKey(due) === selectedKey.value
	})
})

async function loadMonth() {
	loading.value = true
	try {
		const year = viewDate.value.getFullYear()
		const month = viewDate.value.getMonth()
		const from = new Date(year, month, 1 - 7)
		const to = new Date(year, month + 1, 7)
		const taskService = new TaskService()
		tasks.value = await taskService.getAll({}, {
			filter: `due_date >= "${from.toISOString()}" && due_date <= "${to.toISOString()}"`,
			per_page: 200,
		}) as ITask[]
	} finally {
		loading.value = false
	}
}

function shiftMonth(delta: number) {
	viewDate.value = new Date(viewDate.value.getFullYear(), viewDate.value.getMonth() + delta, 1)
}

function goToTask(id: number) {
	router.push({name: 'task.detail', params: {id}})
}

watch(viewDate, loadMonth)
onMounted(loadMonth)
</script>

<style lang="scss" scoped>
.dash-cal {
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

		h2 {
			font-size: 1rem;
			font-weight: 600;
			color: var(--grey-900);
			margin: 0;
		}
	}

	&__nav {
		display: flex;
		gap: .25rem;
	}

	&__nav-btn {
		display: grid;
		place-items: center;
		inline-size: 1.75rem;
		block-size: 1.75rem;
		border-radius: 6px;
		color: var(--grey-500);

		&:hover { background: var(--grey-100); }
	}

	&__grid {
		display: grid;
		grid-template-columns: repeat(7, 1fr);
		gap: .15rem;
	}

	&__weekday {
		text-align: center;
		font-size: .68rem;
		font-weight: 600;
		// --grey-400 is ~2.5:1 on white — --grey-600 clears 4.5:1.
		color: var(--grey-600);
		padding-block-end: .4rem;
	}

	&__day {
		position: relative;
		border: none;
		background: transparent;
		aspect-ratio: 1;
		display: grid;
		place-items: center;
		font-size: .78rem;
		color: var(--grey-800);
		border-radius: 8px;
		cursor: pointer;

		&:hover { background: var(--grey-100); }

		// --grey-300 is ~1.5:1 on white — --grey-600 clears 4.5:1 while
		// staying visually lighter than the --grey-800 in-month days.
		&.is-muted { color: var(--grey-600); }

		&.is-today {
			font-weight: 700;
			// var(--primary) text fails WCAG AA contrast here; same text-tuned
			// shade used for links/active-nav-state elsewhere.
			color: hsl(var(--primary-h) var(--primary-s) var(--primary-text-l));
		}

		&.is-selected {
			// var(--primary) fails contrast with white text; --primary-dark is
			// the same fix used for buttons.
			background: var(--primary-dark);
			color: #ffffff;
		}
	}

	&__dot {
		position: absolute;
		inset-block-end: .2rem;
		inline-size: .3rem;
		block-size: .3rem;
		border-radius: 50%;
		background: currentColor;
		opacity: .7;
	}

	&__list {
		margin-block-start: 1rem;
		padding-block-start: 1rem;
		border-block-start: 1px solid var(--grey-200);

		h3 {
			font-size: .82rem;
			font-weight: 600;
			color: var(--grey-700);
			margin: 0 0 .6rem;
		}

		ul {
			list-style: none;
			margin: 0;
			padding: 0;
			display: flex;
			flex-direction: column;
			gap: .5rem;
		}
	}

	&__empty {
		font-size: .8rem;
		// --grey-400 is ~2.5:1 on white — --grey-600 clears 4.5:1.
		color: var(--grey-600);
		font-style: italic;
	}

	&__task {
		display: flex;
		align-items: center;
		gap: .5rem;
		font-size: .82rem;
		color: var(--grey-800);
		cursor: pointer;

		&:hover { color: var(--primary); }

		&.is-done {
			color: var(--grey-400);
			text-decoration: line-through;
		}
	}

	&__task-dot {
		inline-size: .45rem;
		block-size: .45rem;
		border-radius: 50%;
		background: var(--primary);
		flex-shrink: 0;
	}
}
</style>
