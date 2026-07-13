<template>
	<div
		v-cy="'projects-list'"
		class="content loader-container"
		:class="{'is-loading': loading}"
	>
		<header class="project-header">
			<FancyCheckbox
				v-model="showArchived"
				v-cy="'show-archived-check'"
			>
				{{ $t('project.showArchived') }}
			</FancyCheckbox>

			<div class="action-buttons">
				<XButton
					:to="{name: 'filters.create'}"
					icon="filter"
				>
					{{ $t('filters.create.title') }}
				</XButton>
				<XButton
					v-cy="'new-project'"
					:to="{name: 'project.create'}"
					icon="plus"
				>
					{{ $t('project.create.header') }}
				</XButton>
			</div>
		</header>

		<template v-if="favoriteProjects.length">
			<h2 class="project-section-title">
				{{ $t('project.favoriteProjects') }}
			</h2>
			<div class="project-stat-grid">
				<ProjectStatCard
					v-for="p in favoriteProjects"
					:key="p.id"
					:project="p"
				/>
			</div>
		</template>

		<h2 class="project-section-title">
			{{ $t('project.projects') }}
		</h2>
		<div class="project-stat-grid">
			<ProjectStatCard
				v-for="p in projects"
				:key="p.id"
				:project="p"
			/>
		</div>
	</div>
</template>

<script setup lang="ts">
import {computed} from 'vue'
import {useI18n} from 'vue-i18n'

import FancyCheckbox from '@/components/input/FancyCheckbox.vue'
import ProjectStatCard from '@/components/project/partials/ProjectStatCard.vue'

import {useTitle} from '@/composables/useTitle'
import {useStorage} from '@vueuse/core'

import {useProjectStore} from '@/stores/projects'
import type {IProject} from '@/modelTypes/IProject'

const {t} = useI18n()
const projectStore = useProjectStore()

useTitle(() => t('project.title'))
const showArchived = useStorage('showArchived', false)

const loading = computed(() => projectStore.isLoading)
// Saved filters live in the store as pseudo-projects with negative ids (see
// ProjectCard.vue's `id < -1` check) — exclude them here, they don't have
// real tasks to aggregate stats from.
const projects = computed(() => projectStore.projectsArray
	.filter(p => p.id > 0 && (showArchived.value || !p.isArchived)) as IProject[])
const favoriteProjects = computed(() => (projectStore.favoriteProjects as IProject[])
	.filter(p => p.id > 0))
</script>

<style lang="scss" scoped>
.project-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	gap: 1rem;
	margin-block-end: 1rem;

	@media screen and (max-width: $tablet) {
		flex-direction: column;
	}
}

.action-buttons {
	display: flex;
	justify-content: space-between;
	gap: 1rem;

	@media screen and (max-width: $tablet) {
		inline-size: 100%;
		flex-direction: column;
		align-items: stretch;
	}
}

.project-section-title {
	font-size: 1.1rem;
	font-weight: 700;
	color: var(--grey-900);
	margin: 1.5rem 0 .75rem;

	&:first-of-type {
		margin-block-start: 0;
	}
}

.project-stat-grid {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
	gap: 1rem;
}
</style>
