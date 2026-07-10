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
						<Icon
							:icon="col.icon"
							class="tst-col__icon"
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

import TaskService, {type IPrioritizedTasks} from '@/services/task'
import type {ITask} from '@/modelTypes/ITask'
import type {IUser} from '@/modelTypes/IUser'
import {TASK_STATUS} from '@/types/ITaskStatus'
import {getDisplayName} from '@/models/user'

const taskService = new TaskService()
const loading = ref(true)
const tasks = ref<IPrioritizedTasks>({doNow: [], doSoon: [], doLater: []})

type FilterKey = 'all' | 'overdue' | 'ongoing'
const activeFilter = ref<FilterKey>('all')
const filters: {key: FilterKey, label: string, dot?: string}[] = [
	{key: 'all', label: 'All'},
	{key: 'overdue', label: 'Overdue', dot: '#eb0a1e'},
	{key: 'ongoing', label: 'Ongoing', dot: '#3b82f6'},
]

const columns = computed(() => [
	{key: 'now', label: 'task.accountability.doNow', icon: 'bolt', range: 'SCORE >80', tasks: tasks.value.doNow},
	{key: 'soon', label: 'task.accountability.doSoon', icon: ['far', 'clock'] as [string, string], range: 'SCORE 50-80', tasks: tasks.value.doSoon},
	{key: 'later', label: 'task.accountability.doLater', icon: 'layer-group', range: 'SCORE <50', tasks: tasks.value.doLater},
])

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
$tst-navy: #001529;

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

.tst-filters {
	display: flex;
	gap: .5rem;
	flex-wrap: wrap;
}

.tst-chip {
	display: inline-flex;
	align-items: center;
	gap: .4rem;
	padding: .4rem .75rem;
	border-radius: 3px;
	border: 1px solid var(--grey-200);
	background: #ffffff;
	color: $tst-navy;
	font-size: .75rem;
	font-weight: 600;
	cursor: pointer;
	transition: background-color .15s ease, border-color .15s ease;

	&__dot {
		inline-size: .5rem;
		block-size: .5rem;
		border-radius: 999px;
	}

	&:hover {
		background: var(--grey-50);
	}

	&--active {
		background: $tst-navy;
		border-color: $tst-navy;
		color: #ffffff;
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
	background: #ffffff;
	border: 1px solid var(--grey-200);
	border-radius: 6px;
	overflow: hidden;

	&__head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: .75rem 1rem;
		border-block-end: 1px solid var(--grey-200);

		&--now { background: rgba(235, 10, 30, .05); }
		&--soon { background: rgba(59, 130, 246, .06); }
		&--later { background: var(--grey-50); }
	}

	&__title {
		display: flex;
		align-items: center;
		gap: .5rem;

		h3 {
			font-size: 1rem;
			font-weight: 600;
			margin: 0;
			color: var(--grey-900);
		}
	}

	&__icon {
		font-size: 1rem;
	}

	&__head--now &__icon { color: $tst-red; }
	&__head--soon &__icon { color: #2563eb; }
	&__head--later &__icon { color: var(--grey-500); }

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
		gap: .5rem;
		padding: .5rem;
		background: #f9fafb;
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
	background: #ffffff;
	border: 1px solid var(--grey-200);
	border-radius: 4px;
	padding: .75rem .75rem .75rem 1rem;
	box-shadow: 0 1px 2px rgba(0, 0, 0, .04);
	transition: border-color .15s ease, box-shadow .15s ease;
	color: inherit;

	&:hover {
		box-shadow: 0 2px 6px rgba(0, 0, 0, .08);
		color: inherit;
	}

	&__bar {
		position: absolute;
		inset-block: 0;
		inset-inline-start: 0;
		inline-size: 3px;
		border-start-start-radius: 4px;
		border-end-start-radius: 4px;
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
		background: $tst-navy;
		color: #ffffff;
		font-size: .55rem;
		font-weight: 700;
		border: 2px solid #ffffff;
		margin-inline-start: -.4rem;

		&:first-child { margin-inline-start: 0; }
	}
}
</style>
