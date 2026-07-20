<template>
	<RouterLink
		:to="{name: 'project.index', params: {projectId: project.id}}"
		class="pstat-card"
	>
		<div class="pstat-card__top">
			<span
				class="pstat-card__icon"
				:style="{background: project.hexColor || 'var(--primary)'}"
			>
				<Icon icon="layer-group" />
			</span>
			<span
				v-if="project.isArchived"
				class="pstat-card__archived"
			>{{ $t('project.archived') }}</span>
			<BaseButton
				v-else
				class="pstat-card__favorite"
				:class="{'is-favorite': project.isFavorite}"
				@click.prevent.stop="projectStore.toggleProjectFavorite(project)"
			>
				<span class="is-sr-only">{{ project.isFavorite ? $t('project.unfavorite') : $t('project.favorite') }}</span>
				<Icon :icon="project.isFavorite ? 'star' : ['far', 'star']" />
			</BaseButton>
		</div>

		<h3 class="pstat-card__title">
			{{ project.title }}
		</h3>
		<p class="pstat-card__owner">
			{{ $t('project.info.createdBy', {name: getDisplayName(project.owner)}) }}
		</p>

		<div class="pstat-card__body">
			<dl
				v-if="!loading"
				class="pstat-card__stats"
			>
				<div class="pstat-card__stat">
					<dt class="pstat-card__stat-label pstat-card__stat-label--success">
						{{ $t('project.info.closed') }}
					</dt>
					<dd>{{ counts.closed }}</dd>
				</div>
				<div class="pstat-card__stat">
					<dt class="pstat-card__stat-label">
						{{ $t('project.info.today') }}
					</dt>
					<dd>{{ counts.today }}</dd>
				</div>
				<div class="pstat-card__stat">
					<dt class="pstat-card__stat-label pstat-card__stat-label--danger">
						{{ $t('project.info.overdue') }}
					</dt>
					<dd>{{ counts.overdue }}</dd>
				</div>
				<div class="pstat-card__stat">
					<dt class="pstat-card__stat-label pstat-card__stat-label--warning">
						{{ $t('project.info.upcoming') }}
					</dt>
					<dd>{{ counts.upcoming }}</dd>
				</div>
			</dl>

			<span
				v-if="!loading"
				class="pstat-card__ring"
				:style="ringStyle"
			>
				<span class="pstat-card__ring-value">{{ counts.total }}</span>
			</span>
		</div>
	</RouterLink>
</template>

<script setup lang="ts">
import {ref, computed, onMounted} from 'vue'

import BaseButton from '@/components/base/BaseButton.vue'
import TaskCollectionService, {getDefaultTaskFilterParams} from '@/services/taskCollection'
import {useProjectStore} from '@/stores/projects'
import {getDisplayName} from '@/models/user'
import type {IProject} from '@/modelTypes/IProject'
import type {ITask} from '@/modelTypes/ITask'

const props = defineProps<{project: IProject}>()

const projectStore = useProjectStore()

const loading = ref(true)
const counts = ref({closed: 0, today: 0, overdue: 0, upcoming: 0, total: 0})

// ponytail: no per-project stats endpoint exists, so this pulls one page of
// the project's tasks (capped, not paginated further) and buckets client-side.
// Fine at dealership scale; add a real aggregation endpoint if projects grow
// past a couple hundred tasks each.
onMounted(async () => {
	try {
		const service = new TaskCollectionService()
		const tasks = await service.getAll({projectId: props.project.id}, {
			...getDefaultTaskFilterParams(),
			// 'position' sorting needs a project view id, which we don't have here.
			sort_by: ['id'],
			order_by: ['asc'],
			per_page: 200,
		}) as ITask[]

		const now = new Date()
		const startOfToday = new Date(now.getFullYear(), now.getMonth(), now.getDate())
		const endOfToday = new Date(startOfToday.getTime() + 86400000)

		const result = {closed: 0, today: 0, overdue: 0, upcoming: 0, total: tasks.length}
		for (const task of tasks) {
			if (task.done) {
				result.closed++
				continue
			}
			const due = task.dueDate ? new Date(task.dueDate) : null
			if (!due || due.getTime() === 0) {
				result.upcoming++
			} else if (due < startOfToday) {
				result.overdue++
			} else if (due < endOfToday) {
				result.today++
			} else {
				result.upcoming++
			}
		}
		counts.value = result
	} finally {
		loading.value = false
	}
})

const ringStyle = computed(() => {
	const {closed, overdue, total} = counts.value
	if (total === 0) {
		return {background: 'conic-gradient(var(--grey-200) 0 100%)'}
	}
	const closedPct = closed / total * 100
	const overduePct = overdue / total * 100
	return {
		background: `conic-gradient(var(--success) 0 ${closedPct}%, var(--danger) ${closedPct}% ${closedPct + overduePct}%, var(--grey-200) ${closedPct + overduePct}% 100%)`,
	}
})
</script>

<style lang="scss" scoped>
.pstat-card {
	display: flex;
	flex-direction: column;
	padding: 1.25rem;
	background: var(--white);
	border: 1px solid var(--card-border-color);
	border-radius: 12px;
	box-shadow: 0 2px 4px rgba(0, 0, 0, .03);
	color: inherit;
	transition: box-shadow .15s ease;

	&:hover {
		box-shadow: 0 4px 10px rgba(0, 0, 0, .06);
		color: inherit;
	}

	&__top {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-block-end: .75rem;
	}

	&__icon {
		display: grid;
		place-items: center;
		inline-size: 2.25rem;
		block-size: 2.25rem;
		border-radius: 8px;
		color: #ffffff;
		font-size: .9rem;
	}

	&__archived {
		font-size: .7rem;
		font-weight: 600;
		color: var(--grey-500);
		background: var(--grey-100);
		padding: .15rem .5rem;
		border-radius: 999px;
	}

	&__favorite {
		color: var(--grey-300);

		&:hover, &.is-favorite {
			color: var(--warning);
		}
	}

	&__title {
		font-size: 1.05rem;
		font-weight: 700;
		color: var(--grey-900);
		margin: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	&__owner {
		font-size: .78rem;
		color: var(--grey-500);
		margin: .15rem 0 1rem;
	}

	&__body {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		margin-block-start: auto;
	}

	&__stats {
		display: grid;
		grid-template-columns: repeat(2, auto);
		gap: .5rem 1.5rem;
		margin: 0;
		flex: 1;
	}

	&__stat {
		dt {
			font-size: .68rem;
			font-weight: 600;
			text-transform: uppercase;
			letter-spacing: .04em;
			color: var(--grey-500);
		}

		dd {
			font-size: .95rem;
			font-weight: 700;
			color: var(--grey-900);
			margin: 0;
		}

		&-label--success { color: var(--success); }
		&-label--danger { color: var(--danger); }
		&-label--warning { color: var(--warning); }
	}

	&__ring {
		position: relative;
		display: grid;
		place-items: center;
		flex-shrink: 0;
		inline-size: 3.75rem;
		block-size: 3.75rem;
		border-radius: 50%;

		&::before {
			content: '';
			position: absolute;
			inline-size: 2.75rem;
			block-size: 2.75rem;
			border-radius: 50%;
			background: var(--white);
		}
	}

	&__ring-value {
		position: relative;
		font-size: .9rem;
		font-weight: 700;
		color: var(--grey-900);
	}
}
</style>
