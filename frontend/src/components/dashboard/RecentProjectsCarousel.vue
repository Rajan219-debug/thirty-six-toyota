<template>
	<section class="recent-projects">
		<div class="recent-projects__head">
			<h2>{{ $t('dashboard.recentProjects.title') }}</h2>
			<RouterLink
				:to="{name: 'projects.index'}"
				class="recent-projects__link"
			>
				{{ $t('dashboard.viewAll') }} <Icon icon="angle-right" />
			</RouterLink>
		</div>

		<p
			v-if="projects.length === 0"
			class="recent-projects__empty"
		>
			{{ $t('dashboard.recentProjects.empty') }}
		</p>
		<div
			v-else
			class="recent-projects__track"
		>
			<ProjectStatCard
				v-for="p in projects"
				:key="p.id"
				:project="p"
				class="recent-projects__card"
			/>
		</div>
	</section>
</template>

<script setup lang="ts">
import {computed} from 'vue'

import ProjectStatCard from '@/components/project/partials/ProjectStatCard.vue'
import {useProjectStore} from '@/stores/projects'
import type {IProject} from '@/modelTypes/IProject'

const projectStore = useProjectStore()

const projects = computed<IProject[]>(() => [...projectStore.notArchivedRootProjects]
	.sort((a, b) => new Date(b.updated).getTime() - new Date(a.updated).getTime())
	.slice(0, 8))
</script>

<style lang="scss" scoped>
.recent-projects {
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

	&__link {
		display: inline-flex;
		align-items: center;
		gap: .3rem;
		font-size: .8rem;
		font-weight: 600;
		// var(--primary) alone only hits ~2.3:1 for text on white.
		color: hsl(var(--primary-h) var(--primary-s) var(--primary-text-l));
	}

	&__empty {
		font-size: .85rem;
		// --grey-400 is ~2.5:1 on white — --grey-600 clears 4.5:1.
		color: var(--grey-600);
		font-style: italic;
	}

	&__track {
		display: flex;
		gap: 1rem;
		overflow-x: auto;
		padding-block-end: .25rem;
		scroll-snap-type: x proximity;
	}

	&__card {
		flex: 0 0 260px;
		scroll-snap-align: start;
	}
}
</style>
