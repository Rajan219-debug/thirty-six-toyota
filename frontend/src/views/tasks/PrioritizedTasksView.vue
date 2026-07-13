<template>
	<div class="prioritized-tasks-view">
		<header class="tst-head">
			<div>
				<h1 class="tst-head__title">
					{{ $t('task.accountability.prioritizedTitle') }}
				</h1>
				<p class="tst-head__subtitle">
					{{ $t('task.accountability.prioritizedSubtitle') }}
				</p>
			</div>
			<div class="tst-view-toggle">
				<button
					class="tst-view-toggle__btn"
					:class="{'is-active': viewMode === 'list'}"
					@click="viewMode = 'list'"
				>
					<Icon icon="list" />
					{{ $t('task.accountability.listView') }}
				</button>
				<button
					class="tst-view-toggle__btn"
					:class="{'is-active': viewMode === 'kanban'}"
					@click="viewMode = 'kanban'"
				>
					<Icon icon="th" />
					{{ $t('task.accountability.kanbanView') }}
				</button>
			</div>

			<div class="tst-filters">
				<button
					v-for="f in filters"
					:key="f.key"
					class="tst-chip"
					:class="{'tst-chip--active': activeFilter === f.key, [`tst-chip--${f.key}`]: true}"
					@click="activeFilter = f.key"
				>
					<span
						v-if="f.dot"
						class="tst-chip__dot"
						:style="{background: f.dot}"
					/>
					{{ f.label }}
				</button>
			</div>
		</header>

		<div
			v-if="loading"
			class="loader-container is-loading"
		/>

		<div
			v-else-if="viewMode === 'list'"
			class="tst-list"
		>
			<section
				v-for="group in listGroups"
				:key="group.key"
				class="tst-list-group"
			>
				<button
					class="tst-list-group__head"
					@click="toggleGroup(group.key)"
				>
					<Icon
						icon="chevron-down"
						class="tst-list-group__chevron"
						:class="{'is-collapsed': collapsedGroups.has(group.key)}"
					/>
					{{ group.label }}
					<span class="tst-list-group__count">{{ group.tasks.length }}</span>
				</button>

				<table
					v-if="!collapsedGroups.has(group.key) && group.tasks.length"
					class="tst-list-table"
				>
					<thead>
						<tr>
							<th>{{ $t('task.attributes.title') }}</th>
							<th>{{ $t('task.attributes.dueDate') }}</th>
							<th>{{ $t('task.attributes.assignees') }}</th>
							<th>{{ $t('task.attributes.status') }}</th>
						</tr>
					</thead>
					<tbody>
						<tr
							v-for="t in group.tasks"
							:key="t.id"
							class="tst-list-row"
							@click="goToTask(t.id)"
						>
							<td class="tst-list-row__title">
								{{ t.title }}
							</td>
							<td :class="{'tst-list-row__due--danger': statusMeta(t).accent === 'danger'}">
								{{ t.dueDate ? formatDue(t.dueDate) : '—' }}
							</td>
							<td>
								<div
									v-if="t.assignees.length"
									class="tst-card__avatars"
								>
									<span
										v-for="a in t.assignees.slice(0, 3)"
										:key="a.id"
										class="tst-card__avatar"
										:title="displayName(a)"
									>{{ initials(a) }}</span>
								</div>
								<span v-else>—</span>
							</td>
							<td>
								<span
									class="tst-card__status"
									:class="`tst-card__status--${statusMeta(t).accent}`"
								>{{ statusMeta(t).label }}</span>
							</td>
						</tr>
					</tbody>
				</table>
				<p
					v-else-if="!collapsedGroups.has(group.key)"
					class="tst-col__empty"
				>
					{{ $t('task.accountability.noTasks') }}
				</p>
			</section>
		</div>

		<div
			v-else
			class="tst-board"
		>
			<section
				v-for="col in columns"
				:key="col.key"
				class="tst-col"
			>
				<div
					class="tst-col__head"
					:class="`tst-col__head--${col.key}`"
				>
					<div class="tst-col__title">
						<span
							class="tst-col__dot"
							:class="`tst-col__dot--${col.key}`"
						/>
						<h3>{{ $t(col.label) }}</h3>
						<span class="tst-col__range">{{ col.range }}</span>
					</div>
					<span class="tst-col__count">{{ visibleTasks(col).length }} {{ $t('task.accountability.tasksLabel') }}</span>
				</div>

				<div class="tst-col__body">
					<p
						v-if="visibleTasks(col).length === 0"
						class="tst-col__empty"
					>
						{{ $t('task.accountability.noTasks') }}
					</p>

					<RouterLink
						v-for="t in visibleTasks(col)"
						:key="t.id"
						:to="{name: 'task.detail', params: {id: t.id}}"
						class="tst-card"
						:class="`tst-card--${statusMeta(t).accent}`"
					>
						<span class="tst-card__bar" />
						<div class="tst-card__top">
							<span
								class="tst-card__status"
								:class="`tst-card__status--${statusMeta(t).accent}`"
							>{{ statusMeta(t).label }}</span>
							<span
								class="tst-card__score"
								:class="{'tst-card__score--danger': statusMeta(t).accent === 'danger'}"
							>
								<Icon
									v-if="statusMeta(t).accent === 'danger'"
									icon="exclamation"
									class="tst-card__score-icon"
								/>
								{{ Math.round(t.priorityScore) }}
							</span>
						</div>
						<h4 class="tst-card__title">
							{{ t.title }}
						</h4>
						<p
							v-if="stripHtml(t.description)"
							class="tst-card__desc"
						>
							{{ stripHtml(t.description) }}
						</p>
						<div class="tst-card__foot">
							<span
								class="tst-card__due"
								:class="{'tst-card__due--danger': statusMeta(t).accent === 'danger'}"
							>
								<Icon
									:icon="['far', 'calendar-alt']"
									class="tst-card__due-icon"
								/>
								{{ t.dueDate ? formatDue(t.dueDate) : $t('task.accountability.noDueDate') }}
							</span>
							<div
								v-if="t.assignees.length"
								class="tst-card__avatars"
							>
								<span
									v-for="a in t.assignees.slice(0, 3)"
									:key="a.id"
									class="tst-card__avatar"
									:title="displayName(a)"
								>{{ initials(a) }}</span>
							</div>
						</div>
					</RouterLink>
				</div>
			</section>
		</div>
	</div>
</template>

<script setup lang="ts">
import {ref, computed, onMounted} from 'vue'
import {useI18n} from 'vue-i18n'
import {useRouter} from 'vue-router'

import TaskService, {type IPrioritizedTasks} from '@/services/task'
import type {ITask} from '@/modelTypes/ITask'
import type {IUser} from '@/modelTypes/IUser'
import {TASK_STATUS} from '@/types/ITaskStatus'
import {getDisplayName} from '@/models/user'

const {t} = useI18n({useScope: 'global'})
const router = useRouter()

const taskService = new TaskService()
const loading = ref(true)
const tasks = ref<IPrioritizedTasks>({doNow: [], doSoon: [], doLater: []})

type ViewMode = 'list' | 'kanban'
const viewMode = ref<ViewMode>('list')

type FilterKey = 'all' | 'overdue' | 'ongoing'
const activeFilter = ref<FilterKey>('all')
const filters: {key: FilterKey, label: string, dot?: string}[] = [
	{key: 'all', label: 'All'},
	{key: 'overdue', label: 'Overdue', dot: '#eb0a1e'},
	{key: 'ongoing', label: 'Ongoing', dot: '#3b82f6'},
]

const columns = computed(() => [
	{key: 'now', label: 'task.accountability.doNow', range: 'SCORE >80', tasks: tasks.value.doNow},
	{key: 'soon', label: 'task.accountability.doSoon', range: 'SCORE 50-80', tasks: tasks.value.doSoon},
	{key: 'later', label: 'task.accountability.doLater', range: 'SCORE <50', tasks: tasks.value.doLater},
])

// Taskopad's List view groups by due-date urgency rather than priority score —
// same underlying tasks as the Kanban board, regrouped.
const allTasks = computed(() => [...tasks.value.doNow, ...tasks.value.doSoon, ...tasks.value.doLater])

const collapsedGroups = ref(new Set<string>())
function toggleGroup(key: string) {
	if (collapsedGroups.value.has(key)) {
		collapsedGroups.value.delete(key)
	} else {
		collapsedGroups.value.add(key)
	}
	collapsedGroups.value = new Set(collapsedGroups.value)
}

const listGroups = computed(() => {
	const visible = activeFilter.value === 'all'
		? allTasks.value
		: allTasks.value.filter(task => task.status === activeFilter.value)

	const overdue: ITask[] = []
	const today: ITask[] = []
	const upcoming: ITask[] = []
	const now = new Date()
	const startOfToday = new Date(now.getFullYear(), now.getMonth(), now.getDate())
	const endOfToday = new Date(startOfToday.getTime() + 86400000)

	for (const task of visible) {
		const due = task.dueDate ? new Date(task.dueDate) : null
		if (!due) {
			upcoming.push(task)
		} else if (due < startOfToday) {
			overdue.push(task)
		} else if (due < endOfToday) {
			today.push(task)
		} else {
			upcoming.push(task)
		}
	}

	return [
		{key: 'overdue', label: t('task.accountability.overdueGroup'), tasks: overdue},
		{key: 'today', label: t('task.accountability.todayGroup'), tasks: today},
		{key: 'upcoming', label: t('task.accountability.upcomingGroup'), tasks: upcoming},
	]
})

function goToTask(id: number) {
	router.push({name: 'task.detail', params: {id}})
}

function visibleTasks(col: {tasks: ITask[]}): ITask[] {
	if (activeFilter.value === 'all') return col.tasks
	return col.tasks.filter(t => t.status === activeFilter.value)
}

// Maps the computed task status to a display label + accent colour used across the card.
function statusMeta(t: ITask): {label: string, accent: 'danger' | 'info' | 'warning' | 'success' | 'neutral'} {
	switch (t.status) {
		case TASK_STATUS.OVERDUE: return {label: 'Overdue', accent: 'danger'}
		case TASK_STATUS.ONGOING: return {label: 'Ongoing', accent: 'info'}
		case TASK_STATUS.PENDING_CONFIRMATION: return {label: 'Awaiting Approval', accent: 'warning'}
		case TASK_STATUS.COMPLETED: return {label: 'Completed', accent: 'success'}
		case TASK_STATUS.COMPLETED_LATE: return {label: 'Completed Late', accent: 'success'}
		default: return {label: 'Pending', accent: 'neutral'}
	}
}

function stripHtml(html: string): string {
	const el = document.createElement('div')
	el.innerHTML = html
	return (el.textContent || '').trim()
}

function displayName(u: IUser): string {
	return getDisplayName(u)
}

function initials(u: IUser): string {
	return displayName(u).split(/\s+/).map(p => p[0]).slice(0, 2).join('').toUpperCase()
}

const dueFmt = new Intl.DateTimeFormat(undefined, {month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit'})
function formatDue(d: Date): string {
	return dueFmt.format(new Date(d))
}

onMounted(async () => {
	try {
		tasks.value = await taskService.getPrioritizedTasks()
	} finally {
		loading.value = false
	}
})
</script>

<style lang="scss" scoped>
$tst-red: #eb0a1e;

.prioritized-tasks-view {
	padding: 1.5rem;
	max-inline-size: 1280px;
	margin-inline: auto;
}

.tst-head {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	justify-content: space-between;
	gap: 1rem;
	margin-block-end: 1.5rem;

	&__title {
		font-size: 2rem;
		font-weight: 800;
		letter-spacing: -.02em;
		color: var(--grey-900);
		margin: 0;
	}

	&__subtitle {
		color: var(--grey-500);
		font-size: .9rem;
		margin-block-start: .1rem;
	}
}

.tst-view-toggle {
	display: flex;
	gap: .15rem;
	padding: .25rem;
	background: var(--grey-50);
	border: 1px solid var(--grey-200);
	border-radius: 6px;

	&__btn {
		display: inline-flex;
		align-items: center;
		gap: .4rem;
		padding: .4rem .75rem;
		border-radius: 4px;
		border: 1px solid transparent;
		background: transparent;
		color: var(--grey-500);
		font-size: .75rem;
		font-weight: 600;
		cursor: pointer;
		transition: background-color .15s ease, border-color .15s ease, color .15s ease;

		&:hover {
			color: var(--grey-900);
		}

		&.is-active {
			background: var(--white);
			border-color: var(--grey-200);
			color: var(--primary);
			box-shadow: 0 1px 2px rgba(0, 0, 0, .06);
		}
	}
}

.tst-filters {
	display: flex;
	gap: .15rem;
	flex-wrap: wrap;
	padding: .25rem;
	background: var(--grey-50);
	border: 1px solid var(--grey-200);
	border-radius: 6px;
}

.tst-chip {
	display: inline-flex;
	align-items: center;
	gap: .4rem;
	padding: .4rem .75rem;
	border-radius: 4px;
	border: 1px solid transparent;
	background: transparent;
	color: var(--grey-500);
	font-size: .75rem;
	font-weight: 600;
	cursor: pointer;
	transition: background-color .15s ease, border-color .15s ease, color .15s ease;

	&__dot {
		inline-size: .5rem;
		block-size: .5rem;
		border-radius: 999px;
	}

	&:hover {
		color: var(--grey-900);
	}

	&--active {
		background: var(--white);
		border-color: var(--grey-200);
		color: var(--grey-900);
		box-shadow: 0 1px 2px rgba(0, 0, 0, .06);
	}
}

.tst-list {
	display: flex;
	flex-direction: column;
	gap: 1rem;
}

.tst-list-group {
	background: var(--white);
	border: 1px solid var(--grey-200);
	border-radius: 8px;
	overflow: hidden;

	&__head {
		display: flex;
		align-items: center;
		gap: .6rem;
		inline-size: 100%;
		padding: .85rem 1.25rem;
		border: none;
		background: var(--grey-50);
		font-size: .95rem;
		font-weight: 700;
		color: var(--grey-900);
		cursor: pointer;
		text-align: start;
	}

	&__chevron {
		font-size: .75rem;
		color: var(--grey-500);
		transition: transform .15s ease;

		&.is-collapsed {
			transform: rotate(-90deg);
		}
	}

	&__count {
		margin-inline-start: auto;
		font-size: .72rem;
		font-weight: 600;
		color: var(--grey-600);
		background: var(--grey-100);
		padding: .15rem .5rem;
		border-radius: 3px;
	}
}

.tst-list-table {
	inline-size: 100%;
	border-collapse: collapse;

	th, td {
		padding: .7rem 1.25rem;
		text-align: start;
	}

	thead th {
		font-size: .7rem;
		text-transform: uppercase;
		letter-spacing: .05em;
		color: var(--grey-500);
		border-block-end: 1px solid var(--grey-100);
	}
}

.tst-list-row {
	cursor: pointer;
	transition: background-color .15s ease;

	&:hover {
		background: var(--grey-50);
	}

	&:not(:last-child) {
		border-block-end: 1px solid var(--grey-100);
	}

	&__title {
		font-weight: 600;
		color: var(--grey-900);
	}

	&__due--danger {
		color: $tst-red;
		font-weight: 600;
	}
}

.tst-board {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
	gap: 1rem;
	align-items: start;
}

.tst-col {
	display: flex;
	flex-direction: column;
	background: var(--white);
	border: 1px solid var(--grey-200);
	border-radius: 8px;
	overflow: hidden;

	&__head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 1rem 1.25rem;
		border-block-end: 1px solid var(--grey-200);
	}

	&__title {
		display: flex;
		align-items: center;
		gap: .65rem;

		h3 {
			font-size: 1.05rem;
			font-weight: 600;
			margin: 0;
			color: var(--grey-900);
		}
	}

	&__dot {
		inline-size: .55rem;
		block-size: .55rem;
		border-radius: 999px;
		flex-shrink: 0;
	}

	&__head--now &__dot { background: $tst-red; }
	&__head--soon &__dot { background: #2563eb; }
	&__head--later &__dot { background: var(--grey-300); }

	&__range {
		font-family: $tst-mono-font;
		font-size: .6rem;
		font-weight: 700;
		letter-spacing: .04em;
		padding: .15rem .4rem;
		border-radius: 3px;
		background: var(--grey-100);
		color: var(--grey-600);
	}

	&__head--now &__range {
		background: rgba(235, 10, 30, .1);
		color: $tst-red;
	}

	&__head--soon &__range {
		background: rgba(59, 130, 246, .12);
		color: #1d4ed8;
	}

	&__count {
		font-size: .7rem;
		font-weight: 600;
		color: var(--grey-600);
		background: var(--grey-100);
		padding: .15rem .5rem;
		border-radius: 3px;
		white-space: nowrap;
	}

	&__body {
		display: flex;
		flex-direction: column;
		gap: .75rem;
		padding: .75rem;
		background: var(--grey-50);
		min-block-size: 6rem;
	}

	&__empty {
		color: var(--grey-400);
		font-size: .85rem;
		font-style: italic;
		padding: 1rem .5rem;
	}
}

.tst-card {
	position: relative;
	display: block;
	background: var(--white);
	border: 1px solid var(--grey-200);
	border-radius: 8px;
	padding: .85rem .85rem .85rem 1.1rem;
	box-shadow: 0 1px 2px rgba(0, 0, 0, .04);
	transition: border-color .15s ease, box-shadow .15s ease;
	color: inherit;

	&:hover {
		box-shadow: 0 2px 8px rgba(0, 0, 0, .08);
		color: inherit;
	}

	&__bar {
		position: absolute;
		inset-block: 0;
		inset-inline-start: 0;
		inline-size: 4px;
		border-start-start-radius: 8px;
		border-end-start-radius: 8px;
		background: var(--grey-300);
	}

	&--danger { border-color: rgba(235, 10, 30, .3); }
	&--danger:hover { border-color: $tst-red; }
	&--danger &__bar { background: $tst-red; }
	&--info &__bar { background: #3b82f6; }
	&--warning &__bar { background: #d97706; }
	&--success &__bar { background: #16a34a; }

	&__top {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		margin-block-end: .5rem;
	}

	&__status {
		font-family: $tst-mono-font;
		font-size: .6rem;
		font-weight: 700;
		letter-spacing: .04em;
		text-transform: uppercase;
		padding: .15rem .4rem;
		border-radius: 3px;
		background: var(--grey-100);
		color: var(--grey-700);
		border: 1px solid var(--grey-200);

		&--danger {
			background: $tst-red;
			color: #ffffff;
			border-color: $tst-red;
		}

		&--info {
			background: #dbeafe;
			color: #1e40af;
			border-color: #bfdbfe;
		}

		&--warning {
			background: #fef3c7;
			color: #b45309;
			border-color: #fde68a;
		}

		&--success {
			background: #dcfce7;
			color: #166534;
			border-color: #bbf7d0;
		}
	}

	&__score {
		display: inline-flex;
		align-items: center;
		gap: .25rem;
		font-family: $tst-mono-font;
		font-size: .75rem;
		font-weight: 700;
		color: var(--grey-800);
		background: var(--grey-100);
		padding: .15rem .4rem;
		border-radius: 3px;

		&--danger { color: $tst-red; }
	}

	&__score-icon { font-size: .7rem; }

	&__title {
		font-size: .9rem;
		font-weight: 700;
		color: var(--grey-900);
		line-height: 1.25;
		margin: 0 0 .25rem;
	}

	&__desc {
		font-size: .8rem;
		color: var(--grey-500);
		margin: 0 0 .75rem;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}

	&__foot {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: .5rem;
		padding-block-start: .5rem;
		border-block-start: 1px solid var(--grey-100);
	}

	&__due {
		display: inline-flex;
		align-items: center;
		gap: .3rem;
		font-size: .7rem;
		color: var(--grey-500);

		&--danger {
			color: $tst-red;
			font-weight: 600;
		}
	}

	&__due-icon { font-size: .7rem; }

	&__avatars {
		display: flex;
	}

	&__avatar {
		display: grid;
		place-items: center;
		inline-size: 1.5rem;
		block-size: 1.5rem;
		border-radius: 999px;
		background: var(--primary);
		color: #ffffff;
		font-size: .55rem;
		font-weight: 700;
		border: 2px solid #ffffff;
		margin-inline-start: -.4rem;

		&:first-child { margin-inline-start: 0; }
	}
}
</style>
