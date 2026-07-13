<template>
	<div class="reports-view">
		<header class="reports-header">
			<div>
				<h1 class="reports-title">
					{{ $t('reports.title') }}
				</h1>
				<p class="reports-subtitle">
					{{ $t('reports.subtitle') }}
				</p>
			</div>
			<button
				class="reports-export"
				:disabled="timesheet.rows.length === 0"
				@click="downloadTimesheet"
			>
				<Icon icon="download" />
				{{ $t('reports.exportCsv') }}
			</button>
		</header>

		<div class="reports-filters">
			<div class="field">
				<label class="label">{{ $t('reports.from') }}</label>
				<input
					v-model="start"
					type="date"
					class="input"
				>
			</div>
			<div class="field">
				<label class="label">{{ $t('reports.to') }}</label>
				<input
					v-model="end"
					type="date"
					class="input"
				>
			</div>
			<div class="field reports-filters__project">
				<label class="label">{{ $t('reports.department') }}</label>
				<ProjectSearch v-model="selectedProject" />
			</div>
		</div>

		<div class="reports-stats">
			<div class="stat-tile">
				<div class="stat-tile__body">
					<span class="stat-tile__label">{{ $t('reports.totalTime') }}</span>
					<span class="stat-tile__icon">
						<Icon :icon="['far', 'clock']" />
					</span>
				</div>
				<span class="stat-tile__value">{{ formatDuration(timesheet.totalSeconds) }}</span>
			</div>
			<div class="stat-tile">
				<div class="stat-tile__body">
					<span class="stat-tile__label">{{ $t('reports.employeesTracked') }}</span>
					<span class="stat-tile__icon">
						<Icon icon="users" />
					</span>
				</div>
				<span class="stat-tile__value">{{ timesheet.rows.length }}</span>
			</div>
			<div class="stat-tile stat-tile--danger">
				<div class="stat-tile__body">
					<span class="stat-tile__label">{{ $t('reports.lateCompletions') }}</span>
					<span class="stat-tile__icon stat-tile__icon--danger">
						<Icon icon="exclamation" />
					</span>
				</div>
				<span class="stat-tile__value">{{ lateCompletions.total }}</span>
			</div>
			<div class="stat-tile">
				<div class="stat-tile__body">
					<span class="stat-tile__label">{{ $t('reports.estVsActual') }}</span>
					<span class="stat-tile__icon">
						<Icon icon="chart-line" />
					</span>
				</div>
				<span
					class="stat-tile__value"
					:class="{'is-over': variancePercent !== null && variancePercent > 0}"
				>{{ variancePercent === null ? '—' : formatVariance(variancePercent) }}</span>
			</div>
		</div>

		<div
			v-if="loading"
			class="loader-container is-loading"
		/>

		<template v-else>
			<section class="card reports-section">
				<div class="reports-section__head">
					<h2 class="reports-section__title">
						<span class="reports-section__dot" />
						{{ $t('reports.timesheet') }}
					</h2>
					<XButton
						variant="secondary"
						icon="download"
						:disabled="timesheet.rows.length === 0"
						@click="downloadTimesheet"
					>
						{{ $t('reports.downloadCsv') }}
					</XButton>
				</div>
				<p
					v-if="timesheet.rows.length === 0"
					class="reports-empty"
				>
					{{ $t('reports.noData') }}
				</p>
				<table
					v-else
					class="reports-table"
				>
					<thead>
						<tr>
							<th>{{ $t('reports.employee') }}</th>
							<th class="is-num">
								{{ $t('reports.entries') }}
							</th>
							<th class="is-num">
								{{ $t('reports.totalTime') }}
							</th>
						</tr>
					</thead>
					<tbody>
						<tr
							v-for="row in timesheet.rows"
							:key="row.userId"
						>
							<td>
								<span class="reports-avatar">{{ initials(row.name || row.username) }}</span>
								{{ row.name || row.username }}
							</td>
							<td class="is-num">
								{{ row.entryCount }}
							</td>
							<td class="is-num is-strong">
								{{ formatDuration(row.totalSeconds) }}
							</td>
						</tr>
					</tbody>
					<tfoot>
						<tr>
							<th>{{ $t('reports.total') }}</th>
							<th class="is-num" />
							<th class="is-num is-strong">
								{{ formatDuration(timesheet.totalSeconds) }}
							</th>
						</tr>
					</tfoot>
				</table>
			</section>

			<section class="card reports-section">
				<div class="reports-section__head">
					<h2 class="reports-section__title">
						<span class="reports-section__dot reports-section__dot--danger" />
						{{ $t('reports.lateCompletions') }}
					</h2>
					<XButton
						variant="secondary"
						icon="download"
						:disabled="lateCompletions.rows.length === 0"
						@click="downloadLate"
					>
						{{ $t('reports.downloadCsv') }}
					</XButton>
				</div>
				<p
					v-if="lateCompletions.rows.length === 0"
					class="reports-empty"
				>
					{{ $t('reports.noLate') }}
				</p>
				<template v-else>
					<div
						v-if="lateCompletions.weeks.length > 1"
						class="trend-chart"
						role="img"
						:aria-label="$t('reports.trend')"
					>
						<div
							v-for="w in lateCompletions.weeks"
							:key="w.weekStart"
							class="trend-chart__col"
							:title="`${$t('reports.trendWeek', {date: weekLabel(w.weekStart)})}: ${w.count}`"
						>
							<span class="trend-chart__count">{{ w.count }}</span>
							<span class="trend-chart__track">
								<span
									class="trend-chart__bar"
									:style="{blockSize: `${Math.max(4, w.count / maxWeekCount * 100)}%`}"
								/>
							</span>
							<span class="trend-chart__label">{{ weekLabel(w.weekStart) }}</span>
						</div>
					</div>
					<table class="reports-table">
						<thead>
							<tr>
								<th>{{ $t('reports.employee') }}</th>
								<th class="is-num">
									{{ $t('reports.lateCount') }}
								</th>
							</tr>
						</thead>
						<tbody>
							<tr
								v-for="row in lateCompletions.rows"
								:key="row.userId"
							>
								<td>
									<span class="reports-avatar">{{ initials(row.name || row.username) }}</span>
									{{ row.name || row.username }}
								</td>
								<td class="is-num">
									<span class="reports-badge">{{ row.lateCount }}</span>
								</td>
							</tr>
						</tbody>
					</table>
				</template>
			</section>

			<section class="card reports-section">
				<div class="reports-section__head">
					<h2 class="reports-section__title">
						<span class="reports-section__dot" />
						{{ $t('reports.departmentEstimates') }}
					</h2>
					<XButton
						variant="secondary"
						icon="download"
						:disabled="deptEstimates.rows.length === 0"
						@click="downloadDeptEstimates"
					>
						{{ $t('reports.downloadCsv') }}
					</XButton>
				</div>
				<p
					v-if="deptEstimates.rows.length === 0"
					class="reports-empty"
				>
					{{ $t('reports.noEstimates') }}
				</p>
				<table
					v-else
					class="reports-table"
				>
					<thead>
						<tr>
							<th>{{ $t('reports.department') }}</th>
							<th class="is-num">
								{{ $t('reports.tasks') }}
							</th>
							<th class="is-num">
								{{ $t('reports.estimated') }}
							</th>
							<th class="is-num">
								{{ $t('reports.actual') }}
							</th>
						</tr>
					</thead>
					<tbody>
						<tr
							v-for="row in deptEstimates.rows"
							:key="row.projectId"
						>
							<td>{{ row.projectTitle }}</td>
							<td class="is-num">
								{{ row.taskCount }}
							</td>
							<td class="is-num">
								{{ formatDuration(row.estimatedSeconds) }}
							</td>
							<td
								class="is-num is-strong"
								:class="{'is-over': row.estimatedSeconds > 0 && row.actualSeconds > row.estimatedSeconds}"
							>
								{{ formatDuration(row.actualSeconds) }}
							</td>
						</tr>
					</tbody>
					<tfoot>
						<tr>
							<th>{{ $t('reports.total') }}</th>
							<th class="is-num" />
							<th class="is-num is-strong">
								{{ formatDuration(deptEstimates.totalEstimated) }}
							</th>
							<th class="is-num is-strong">
								{{ formatDuration(deptEstimates.totalActual) }}
							</th>
						</tr>
					</tfoot>
				</table>
			</section>
		</template>
	</div>
</template>

<script setup lang="ts">
import {ref, computed, watch, onMounted} from 'vue'
import {useI18n} from 'vue-i18n'

import ProjectSearch from '@/components/tasks/partials/ProjectSearch.vue'
import TaskService, {type ITimesheetReport, type ILateCompletionReport, type IDepartmentEstimateReport, type IReportFilter} from '@/services/task'
import {useTitle} from '@/composables/useTitle'
import {useBaseStore} from '@/stores/base'

import type {IProject} from '@/modelTypes/IProject'

const {t} = useI18n()
useTitle(() => t('reports.title'))

const taskService = new TaskService()
const baseStore = useBaseStore()

const start = ref('')
const end = ref('')
const selectedProject = ref<IProject | null>(null)

const loading = ref(true)
const timesheet = ref<ITimesheetReport>({rows: [], totalSeconds: 0})
const lateCompletions = ref<ILateCompletionReport>({rows: [], weeks: [], total: 0})
const deptEstimates = ref<IDepartmentEstimateReport>({rows: [], totalEstimated: 0, totalActual: 0})

// A date input is a day; make `end` inclusive of that whole day.
const filter = computed<IReportFilter>(() => ({
	start: start.value || undefined,
	end: end.value ? `${end.value}T23:59:59` : undefined,
	projectId: selectedProject.value?.id || undefined,
}))

async function load() {
	loading.value = true
	try {
		[timesheet.value, lateCompletions.value, deptEstimates.value] = await Promise.all([
			taskService.getTimesheetReport(filter.value),
			taskService.getLateCompletionReport({start: filter.value.start, end: filter.value.end}),
			taskService.getDepartmentEstimateReport({start: filter.value.start, end: filter.value.end}),
		])
	} finally {
		loading.value = false
	}
}

function downloadTimesheet() {
	return taskService.downloadReportCsv('reports/timesheet.csv', 'timesheet-report.csv', filter.value)
}

function downloadLate() {
	return taskService.downloadReportCsv('reports/late-completions.csv', 'late-completion-report.csv', {
		start: filter.value.start,
		end: filter.value.end,
	})
}

function downloadDeptEstimates() {
	return taskService.downloadReportCsv('reports/department-estimates.csv', 'department-estimates-report.csv', {
		start: filter.value.start,
		end: filter.value.end,
	})
}

// Actual over/under estimate, as a signed percentage; null when nothing was estimated.
const variancePercent = computed<number | null>(() => {
	const {totalEstimated, totalActual} = deptEstimates.value
	if (totalEstimated === 0) return null
	return (totalActual - totalEstimated) / totalEstimated * 100
})

function formatVariance(percent: number): string {
	return `${percent > 0 ? '+' : ''}${percent.toFixed(1)}%`
}

const maxWeekCount = computed(() => Math.max(1, ...lateCompletions.value.weeks.map(w => w.count)))

function weekLabel(weekStart: string): string {
	return new Date(weekStart).toLocaleDateString(undefined, {month: 'short', day: 'numeric'})
}

// Whole hours and minutes, e.g. "2h 30m" (or "0m" when empty).
function formatDuration(seconds: number): string {
	const h = Math.floor(seconds / 3600)
	const m = Math.floor((seconds % 3600) / 60)
	if (h === 0) return `${m}m`
	return `${h}h ${m}m`
}

// Up to two uppercase initials for the row avatar.
function initials(name: string): string {
	return name
		.split(/\s+/)
		.filter(Boolean)
		.slice(0, 2)
		.map(p => p[0]?.toUpperCase() ?? '')
		.join('')
}

let debounce: ReturnType<typeof setTimeout>
watch(filter, () => {
	clearTimeout(debounce)
	debounce = setTimeout(load, 300)
})

onMounted(() => {
	baseStore.handleSetCurrentProject({project: null})
	load()
})
</script>

<style lang="scss" scoped>
.reports-view {
	padding: 1.5rem;
	max-inline-size: 1100px;
	margin-inline: auto;
}

.reports-header {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	justify-content: space-between;
	gap: 1rem;
	margin-block-end: 1.5rem;
}

.reports-export {
	display: inline-flex;
	align-items: center;
	gap: .5rem;
	padding: .55rem 1rem;
	border-radius: 4px;
	border: none;
	background: #4d6077;
	color: #ffffff;
	font-weight: 600;
	font-size: .9rem;
	cursor: pointer;
	transition: opacity .15s ease;

	&:hover:not(:disabled) { opacity: .9; }

	&:disabled {
		opacity: .5;
		cursor: not-allowed;
	}
}

.reports-title {
	font-size: 1.6rem;
	font-weight: 700;
	color: var(--grey-900);
	margin: 0;
}

.reports-subtitle {
	color: var(--grey-500);
	font-size: .95rem;
	margin-block-start: .15rem;
}

.reports-filters {
	display: flex;
	flex-wrap: wrap;
	gap: 1rem;
	padding: 1rem 1.25rem;
	margin-block-end: 1.5rem;
	background: var(--white);
	border: 1px solid var(--card-border-color);
	border-radius: 10px;

	.field {
		margin: 0;
	}

	.label {
		font-size: .78rem;
		text-transform: uppercase;
		letter-spacing: .05em;
		color: var(--grey-500);
	}

	&__project {
		min-inline-size: 16rem;
		flex: 1;
	}
}

.reports-stats {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
	gap: 1rem;
	margin-block-end: 1.75rem;
}

.stat-tile {
	display: flex;
	flex-direction: column;
	justify-content: space-between;
	gap: 1rem;
	padding: 1.5rem;
	background: var(--white);
	border: 1px solid var(--card-border-color);
	border-radius: 12px;
	box-shadow: 0 2px 4px rgba(0, 0, 0, .03);

	&--danger {
		border-inline-start: 4px solid var(--danger);
	}

	&__body {
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
	}

	&__value {
		font-size: 2.25rem;
		font-weight: 700;
		color: var(--grey-900);
		line-height: 1;
	}

	&__icon {
		display: grid;
		place-items: center;
		inline-size: 2rem;
		block-size: 2rem;
		border-radius: 6px;
		background: var(--grey-50);
		color: var(--grey-500);
		font-size: 1rem;
		flex-shrink: 0;

		&--danger {
			background: hsla(var(--danger-h), var(--danger-s), var(--danger-l), .12);
			color: var(--danger);
		}
	}
}

.reports-section {
	padding: 1.25rem 1.5rem 1.5rem;
	margin-block-end: 1.5rem;
	border-radius: 12px;

	&__head {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 1rem;
		margin-block-end: 1rem;
	}

	&__title {
		display: flex;
		align-items: center;
		gap: .55rem;
		font-size: 1.1rem;
		font-weight: 600;
		color: var(--grey-900);
	}

	&__dot {
		inline-size: .6rem;
		block-size: .6rem;
		border-radius: 50%;
		background: var(--primary);

		&--danger {
			background: var(--danger);
		}
	}
}

.reports-empty {
	color: var(--grey-400);
	font-style: italic;
	padding-block: .5rem;
}

.reports-table {
	inline-size: 100%;
	border-collapse: collapse;

	th, td {
		padding: .7rem .75rem;
		text-align: start;
	}

	thead th {
		font-size: .75rem;
		text-transform: uppercase;
		letter-spacing: .05em;
		color: var(--grey-500);
		border-block-end: 1px solid var(--card-border-color);
	}

	tbody tr {
		border-block-end: 1px solid var(--grey-100);

		&:hover {
			background: var(--grey-50, hsla(var(--primary-hsl), .03));
		}
	}

	tfoot th {
		border-block-start: 2px solid var(--card-border-color);
		font-size: .95rem;
		color: var(--grey-900);
		padding-block-start: .85rem;
	}

	.is-num {
		text-align: end;
		font-variant-numeric: tabular-nums;
	}

	.is-strong {
		font-weight: 700;
		color: var(--grey-900);
	}
}

.reports-avatar {
	display: inline-grid;
	place-items: center;
	inline-size: 1.9rem;
	block-size: 1.9rem;
	margin-inline-end: .6rem;
	border-radius: 50%;
	background: hsla(var(--primary-hsl), .12);
	color: var(--primary);
	font-size: .72rem;
	font-weight: 700;
	vertical-align: middle;
}

.is-over {
	color: var(--danger);
}

.trend-chart {
	display: flex;
	align-items: flex-end;
	gap: .75rem;
	block-size: 140px;
	padding: .5rem .25rem 0;
	margin-block-end: 1.25rem;
	border-block-end: 1px solid var(--card-border-color);

	&__col {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: flex-end;
		gap: .25rem;
		block-size: 100%;
		min-inline-size: 0;
	}

	&__count {
		font-size: .72rem;
		font-weight: 700;
		color: var(--grey-500);
		font-variant-numeric: tabular-nums;
	}

	&__track {
		flex: 1;
		inline-size: 100%;
		display: flex;
		align-items: flex-end;
		justify-content: center;
		min-block-size: 0;
	}

	&__bar {
		display: block;
		inline-size: 100%;
		max-inline-size: 3rem;
		background: var(--danger);
		border-radius: 4px 4px 0 0;
		opacity: .85;
		transition: opacity .15s ease;

		.trend-chart__col:hover & {
			opacity: 1;
		}
	}

	&__label {
		font-size: .68rem;
		color: var(--grey-500);
		white-space: nowrap;
	}
}

.reports-badge {
	display: inline-block;
	min-inline-size: 1.6rem;
	padding: .1rem .5rem;
	border-radius: 999px;
	background: hsla(var(--danger-h), var(--danger-s), var(--danger-l), .12);
	color: var(--danger);
	font-weight: 700;
	text-align: center;
}
</style>
