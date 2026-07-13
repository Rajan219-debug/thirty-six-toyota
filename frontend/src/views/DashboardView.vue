<template>
	<div class="tst-dash">
		<header class="tst-dash__head">
			<div>
				<h1 class="tst-dash__title">
					{{ $t('dashboard.title') }}
				</h1>
				<p class="tst-dash__subtitle">
					{{ $t('dashboard.subtitle') }}
				</p>
			</div>
		</header>

		<div class="tst-stats">
			<div class="tst-stat">
				<div class="tst-stat__head">
					<span class="tst-stat__label">{{ $t('dashboard.openTasks') }}</span>
					<span class="tst-stat__icon"><Icon icon="list-check" /></span>
				</div>
				<span class="tst-stat__value">{{ Math.round(openCount) }}</span>
			</div>
			<div class="tst-stat tst-stat--danger">
				<div class="tst-stat__head">
					<span class="tst-stat__label tst-stat__label--danger">{{ $t('dashboard.overdue') }}</span>
					<span class="tst-stat__icon tst-stat__icon--danger"><Icon icon="exclamation" /></span>
				</div>
				<span class="tst-stat__value tst-stat__value--danger">{{ Math.round(overdueCount) }}</span>
			</div>
			<div class="tst-stat">
				<div class="tst-stat__head">
					<span class="tst-stat__label">{{ $t('dashboard.completionRate') }}</span>
					<span class="tst-stat__icon"><Icon icon="chart-line" /></span>
				</div>
				<span class="tst-stat__value">{{ rateCount.toFixed(1) }}<span class="tst-stat__unit">%</span></span>
			</div>
			<div class="tst-stat">
				<div class="tst-stat__head">
					<span class="tst-stat__label">{{ $t('dashboard.pendingConfirmations') }}</span>
					<span class="tst-stat__icon"><Icon :icon="['far', 'clock']" /></span>
				</div>
				<span class="tst-stat__value">{{ Math.round(pendingCount) }}</span>
			</div>
		</div>

		<div
			v-if="loading"
			class="loader-container is-loading"
		/>

		<div
			v-else
			class="tst-grid"
		>
			<div class="tst-main-col">
				<section class="tst-panel tst-panel--main">
					<div class="tst-panel__head">
						<h2>{{ $t('dashboard.overduePending') }}</h2>
						<RouterLink
							:to="{name: 'tasks.prioritized'}"
							class="tst-panel__link"
						>
							{{ $t('dashboard.viewAll') }} <Icon icon="angle-right" />
						</RouterLink>
					</div>
					<p
						v-if="stats.attentionTasks.length === 0"
						class="tst-panel__empty"
					>
						{{ $t('dashboard.allClear') }}
					</p>
					<div
						v-else
						class="tst-table-wrap"
					>
						<table class="tst-table">
							<thead>
								<tr>
									<th class="tst-id-col">
										ID
									</th>
									<th>{{ $t('dashboard.task') }}</th>
									<th>{{ $t('dashboard.assignee') }}</th>
									<th>{{ $t('dashboard.department') }}</th>
									<th>{{ $t('dashboard.status') }}</th>
								</tr>
							</thead>
							<tbody>
								<tr
									v-for="task in stats.attentionTasks"
									:key="task.id"
									@click="goToTask(task.id)"
								>
									<td class="tst-id">
										#{{ task.id }}
									</td>
									<td class="tst-table__task">
										{{ task.title }}
									</td>
									<td>{{ task.assignee || '—' }}</td>
									<td class="tst-dept">
										{{ task.project }}
									</td>
									<td>
										<span
											class="tst-pill"
											:class="`tst-pill--${statusAccent(task.status)}`"
										>{{ statusLabel(task.status) }}</span>
									</td>
								</tr>
							</tbody>
						</table>
					</div>
				</section>

				<StatsTrendChart />
				<RecentProjectsCarousel />
			</div>

			<div class="tst-side">
				<section class="tst-panel">
					<div class="tst-panel__head">
						<h2>{{ $t('dashboard.leaderboard') }}</h2>
						<Icon
							icon="chart-bar"
							class="tst-panel__icon"
						/>
					</div>
					<p
						v-if="stats.leaderboard.length === 0"
						class="tst-panel__empty"
					>
						{{ $t('dashboard.noEfficiency') }}
					</p>
					<ul
						v-else
						class="tst-leader"
					>
						<li
							v-for="row in stats.leaderboard"
							:key="row.userId"
							class="tst-leader__row"
						>
							<span class="tst-leader__avatar">{{ initials(row.name || row.username) }}</span>
							<div class="tst-leader__meta">
								<div class="tst-leader__top">
									<span class="tst-leader__name">{{ row.name || row.username }}</span>
									<span class="tst-leader__value">{{ row.efficiency }}%</span>
								</div>
								<div class="tst-leader__bar">
									<span
										:style="{width: row.efficiency + '%'}"
										:class="{'tst-leader__bar--low': row.efficiency < 80}"
									/>
								</div>
							</div>
						</li>
					</ul>
				</section>

				<section class="tst-panel">
					<div class="tst-panel__head">
						<h2>{{ $t('dashboard.allocation') }}</h2>
						<Icon
							icon="layer-group"
							class="tst-panel__icon"
						/>
					</div>
					<p
						v-if="stats.allocation.length === 0"
						class="tst-panel__empty"
					>
						{{ $t('dashboard.noAllocation') }}
					</p>
					<div
						v-else
						class="tst-alloc"
					>
						<div
							v-for="(a, i) in stats.allocation"
							:key="a.projectId"
							class="tst-alloc__cell"
						>
							<div
								class="tst-alloc__ring"
								:style="ringStyle(a.percent, i)"
							>
								<span class="tst-alloc__pct">{{ Math.round(a.percent) }}%</span>
							</div>
							<span class="tst-alloc__name">{{ a.name }}</span>
						</div>
					</div>
				</section>

				<DashboardCalendar />
				<RecentDiscussionCard />
			</div>
		</div>

		<CheckInWidget />
	</div>
</template>

<script setup lang="ts">
import {ref, onMounted} from 'vue'
import {useI18n} from 'vue-i18n'
import {useRouter} from 'vue-router'

import TaskService, {type IDashboardStats} from '@/services/task'
import {useTitle} from '@/composables/useTitle'
import {useCountUp} from '@/composables/useCountUp'
import {useBaseStore} from '@/stores/base'
import {TASK_STATUS} from '@/types/ITaskStatus'

import StatsTrendChart from '@/components/dashboard/StatsTrendChart.vue'
import DashboardCalendar from '@/components/dashboard/DashboardCalendar.vue'
import RecentProjectsCarousel from '@/components/dashboard/RecentProjectsCarousel.vue'
import RecentDiscussionCard from '@/components/dashboard/RecentDiscussionCard.vue'
import CheckInWidget from '@/components/dashboard/CheckInWidget.vue'

const {t} = useI18n()
useTitle(() => t('dashboard.title'))

const taskService = new TaskService()
const baseStore = useBaseStore()
const router = useRouter()

const loading = ref(true)
const stats = ref<IDashboardStats>({
	totalOpen: 0, overdue: 0, pendingConfirmation: 0, completed: 0,
	completionRate: 0, leaderboard: [], allocation: [], attentionTasks: [],
})

// Animated KPI counters — tick up from 0 when the stats land.
const openCount = useCountUp(() => stats.value.totalOpen)
const overdueCount = useCountUp(() => stats.value.overdue)
const rateCount = useCountUp(() => stats.value.completionRate)
const pendingCount = useCountUp(() => stats.value.pendingConfirmation)

function initials(name: string): string {
	return name.split(/\s+/).filter(Boolean).slice(0, 2).map(p => p[0]?.toUpperCase() ?? '').join('')
}

function statusLabel(status: string): string {
	switch (status) {
		case TASK_STATUS.OVERDUE: return t('dashboard.statusOverdue')
		case TASK_STATUS.PENDING_CONFIRMATION: return t('dashboard.statusAwaiting')
		default: return t('dashboard.statusPending')
	}
}
function statusAccent(status: string): string {
	switch (status) {
		case TASK_STATUS.OVERDUE: return 'danger'
		case TASK_STATUS.PENDING_CONFIRMATION: return 'warning'
		default: return 'neutral'
	}
}

// Department ring: navy fill up to `percent`, red once it's the smallest slice (i === last handled by caller order).
const ringColors = ['#4d6077', '#4d6077', '#4d6077', '#eb0a1e']
function ringStyle(percent: number, index: number) {
	const color = ringColors[Math.min(index, ringColors.length - 1)]
	return {background: `conic-gradient(${color} ${percent}%, var(--grey-200) 0)`}
}

function goToTask(id: number) {
	router.push({name: 'task.detail', params: {id}})
}

onMounted(async () => {
	baseStore.handleSetCurrentProject({project: null})
	try {
		stats.value = await taskService.getDashboardStats()
	} finally {
		loading.value = false
	}
})
</script>

<style lang="scss" scoped>
$tst-red: #eb0a1e;

@keyframes tst-rise {
	from { opacity: 0; transform: translateY(14px); }
	to { opacity: 1; transform: translateY(0); }
}

.tst-dash {
	padding: 1.5rem;
	max-inline-size: 1440px;
	margin-inline: auto;
}

.tst-dash__head {
	margin-block-end: 1.5rem;
}

.tst-dash__title {
	font-size: 2rem;
	font-weight: 800;
	letter-spacing: -.02em;
	color: var(--grey-900);
	margin: 0;
}

.tst-dash__subtitle {
	color: var(--grey-500);
	font-size: .9rem;
	margin-block-start: .1rem;
}

.tst-stats {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(210px, 1fr));
	gap: 1rem;
	margin-block-end: 1.5rem;
}

.tst-stat {
	display: flex;
	flex-direction: column;
	gap: 1rem;
	padding: 1.25rem;
	background: var(--white);
	border: 1px solid var(--grey-200);
	border-radius: 12px;
	box-shadow: 0 2px 4px rgba(0, 0, 0, .03);
	transition: transform .18s ease, box-shadow .18s ease;
	animation: tst-rise .5s ease backwards;

	&:nth-child(1) { animation-delay: .04s; }
	&:nth-child(2) { animation-delay: .10s; }
	&:nth-child(3) { animation-delay: .16s; }
	&:nth-child(4) { animation-delay: .22s; }

	&:hover {
		transform: translateY(-3px);
		box-shadow: 0 12px 26px rgba(0, 0, 0, .09);
	}

	&--danger {
		border-inline-start: 4px solid $tst-red;
		background: rgba(235, 10, 30, .04);
	}

	&__head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: .75rem;
	}

	&__label {
		font-size: .72rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: .06em;
		color: var(--grey-500);

		&--danger {
			color: $tst-red;
			font-weight: 700;
		}
	}

	&__value {
		font-size: 2rem;
		font-weight: 800;
		color: var(--grey-900);
		line-height: 1;

		&--danger { color: $tst-red; }
	}

	&__unit {
		font-size: 1rem;
		color: var(--grey-500);
		margin-inline-start: .1rem;
	}

	&__icon {
		display: grid;
		place-items: center;
		inline-size: 2.25rem;
		block-size: 2.25rem;
		border-radius: 8px;
		background: linear-gradient(135deg, hsla(var(--primary-hsl), .18), hsla(var(--primary-hsl), .06));
		color: var(--primary);
		font-size: 1rem;
		flex-shrink: 0;
		box-shadow: 0 4px 10px hsla(var(--primary-hsl), .18);

		&--danger {
			background: linear-gradient(135deg, rgba(235, 10, 30, .18), rgba(235, 10, 30, .06));
			color: $tst-red;
			box-shadow: 0 4px 10px rgba(235, 10, 30, .18);
		}
	}
}

.tst-grid {
	display: grid;
	grid-template-columns: minmax(0, 2fr) minmax(0, 1fr);
	gap: 1.5rem;
	align-items: start;

	@media screen and (width <= 1024px) {
		grid-template-columns: 1fr;
	}
}

.tst-main-col,
.tst-side {
	display: flex;
	flex-direction: column;
	gap: 1.5rem;
}

.tst-panel {
	background: var(--white);
	border: 1px solid var(--grey-200);
	border-radius: 12px;
	box-shadow: 0 2px 4px rgba(0, 0, 0, .03);
	overflow: hidden;
	transition: transform .18s ease, box-shadow .18s ease;
	animation: tst-rise .55s ease backwards;
	animation-delay: .12s;

	&:hover {
		transform: translateY(-2px);
		box-shadow: 0 12px 26px rgba(0, 0, 0, .08);
	}

	&__head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 1rem 1.25rem;
		border-block-end: 1px solid var(--grey-200);

		h2 {
			font-size: 1rem;
			font-weight: 600;
			color: var(--grey-900);
			margin: 0;
		}
	}

	&__link {
		display: inline-flex;
		align-items: center;
		gap: .3rem;
		font-size: .8rem;
		font-weight: 600;
		color: #4d6077;
	}

	&__icon {
		color: var(--grey-400);
		&--danger { color: $tst-red; }
	}

	&__empty {
		color: var(--grey-400);
		font-style: italic;
		font-size: .9rem;
		padding: 1.5rem 1.25rem;
	}
}

.tst-table-wrap {
	overflow-x: auto;
}

.tst-table {
	inline-size: 100%;
	border-collapse: collapse;
	table-layout: fixed;

	th, td {
		padding: .7rem 1.25rem;
		text-align: start;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	&__task {
		white-space: normal;
	}

	thead th {
		font-family: $tst-mono-font;
		font-size: .68rem;
		text-transform: uppercase;
		letter-spacing: .05em;
		color: var(--grey-500);
		border-block-end: 1px solid var(--grey-200);
	}

	tbody tr {
		border-block-end: 1px solid var(--grey-100);
		cursor: pointer;

		&:hover { background: var(--grey-50); }
		&:nth-child(even) { background: hsla(0, 0%, 97%, .5); }
		&:nth-child(even):hover { background: var(--grey-50); }
	}

	&__task {
		font-weight: 600;
		color: var(--grey-900);
	}
}

.tst-id {
	font-family: $tst-mono-font;
	font-size: .75rem;
	color: var(--grey-500);
}
.tst-id-col { inline-size: 4rem; }
.tst-dept { color: var(--grey-500); }

.tst-pill {
	display: inline-block;
	font-family: $tst-mono-font;
	font-size: .62rem;
	font-weight: 700;
	letter-spacing: .04em;
	text-transform: uppercase;
	padding: .15rem .5rem;
	border-radius: 3px;
	background: var(--grey-100);
	color: var(--grey-700);

	&--danger {
		background: $tst-red;
		color: #ffffff;
	}

	&--warning {
		background: #fef3c7;
		color: #b45309;
	}
}

.tst-leader {
	list-style: none;
	margin: 0;
	padding: 1.1rem 1.25rem;
	display: flex;
	flex-direction: column;
	gap: 1rem;

	&__row {
		display: flex;
		align-items: center;
		gap: .75rem;
	}

	&__avatar {
		display: grid;
		place-items: center;
		inline-size: 2rem;
		block-size: 2rem;
		border-radius: 6px;
		background: var(--primary);
		color: #ffffff;
		font-size: .65rem;
		font-weight: 700;
		flex-shrink: 0;
	}

	&__meta {
		flex: 1;
		min-inline-size: 0;
	}

	&__top {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
		margin-block-end: .3rem;
	}

	&__name {
		font-size: .85rem;
		color: var(--grey-800);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	&__value {
		font-family: $tst-mono-font;
		font-size: .75rem;
		font-weight: 700;
		color: var(--grey-900);
	}

	&__bar {
		block-size: .4rem;
		border-radius: 999px;
		background: var(--grey-100);
		overflow: hidden;

		span {
			display: block;
			block-size: 100%;
			border-radius: 999px;
			background: #4d6077;

			&.tst-leader__bar--low { background: $tst-red; }
		}
	}
}

.tst-alloc {
	display: grid;
	grid-template-columns: 1fr 1fr;
	gap: 1rem;
	padding: 1.25rem;

	&__cell {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: .5rem;
		padding: .75rem;
		border: 1px solid var(--grey-200);
		border-radius: 6px;
	}

	&__ring {
		position: relative;
		display: grid;
		place-items: center;
		inline-size: 3.25rem;
		block-size: 3.25rem;
		border-radius: 999px;

		&::before {
			content: '';
			position: absolute;
			inset: 4px;
			background: var(--white);
			border-radius: 999px;
		}
	}

	&__pct {
		position: relative;
		font-family: $tst-mono-font;
		font-size: .72rem;
		font-weight: 700;
		color: var(--grey-800);
	}

	&__name {
		font-family: $tst-mono-font;
		font-size: .7rem;
		color: var(--grey-600);
	}
}

// Respect users who ask for less motion — no entrance animations, no lifts.
@media (prefers-reduced-motion: reduce) {
	.tst-stat,
	.tst-panel {
		animation: none;
		transition: none;
	}

	.tst-stat:hover,
	.tst-panel:hover {
		transform: none;
	}
}
</style>
