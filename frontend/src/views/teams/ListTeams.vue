<template>
	<div
		class="teams-view loader-container"
		:class="{ 'is-loading': teamService.loading}"
	>
		<header class="teams-header">
			<h1 class="teams-title">
				{{ $t('team.title') }}
			</h1>
			<XButton
				:to="{name:'teams.create'}"
				icon="plus"
			>
				{{ $t('team.create.title') }}
			</XButton>
		</header>

		<div
			v-if="teams.length > 0"
			class="teams-grid"
		>
			<RouterLink
				v-for="(team, index) in teams"
				:key="team.id"
				:to="{name: 'teams.edit', params: {id: team.id}}"
				class="team-card"
			>
				<span
					class="team-card__avatar"
					:class="`team-card__avatar--${index % 4}`"
				>{{ initials(team.name) }}</span>
				<span class="team-card__name">{{ team.name }}</span>
			</RouterLink>
		</div>
		<p
			v-else-if="!teamService.loading"
			class="teams-empty"
		>
			{{ $t('team.noTeams') }}
			<RouterLink :to="{name: 'teams.create'}">
				{{ $t('team.create.title') }}.
			</RouterLink>
		</p>
	</div>
</template>

<script setup lang="ts">
import {ref, shallowReactive} from 'vue'
import { useI18n } from 'vue-i18n'

import TeamService from '@/services/team'
import { useTitle } from '@/composables/useTitle'

const { t } = useI18n({useScope: 'global'})
useTitle(() => t('team.title'))

const teams = ref([])
const teamService = shallowReactive(new TeamService())
teamService.getAll().then((result) => {
	teams.value = result
})

function initials(name: string): string {
	return name
		.split(/\s+/)
		.filter(Boolean)
		.slice(0, 2)
		.map(p => p[0]?.toUpperCase() ?? '')
		.join('')
}
</script>

<style lang="scss" scoped>
.teams-view {
	padding: 1.5rem;
	max-inline-size: 1100px;
	margin-inline: auto;
}

.teams-header {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 1rem;
	margin-block-end: 1.5rem;
}

.teams-title {
	font-size: 1.6rem;
	font-weight: 700;
	color: var(--grey-900);
	margin: 0;
}

.teams-empty {
	color: var(--grey-400);
	font-style: italic;
}

.teams-grid {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
	gap: 1rem;
}

.team-card {
	display: flex;
	align-items: center;
	gap: .85rem;
	padding: 1.1rem;
	background: var(--white);
	border: 1px solid var(--card-border-color);
	border-radius: 12px;
	box-shadow: 0 2px 4px rgba(0, 0, 0, .03);
	color: var(--text);
	transition: box-shadow .15s ease;

	&:hover {
		box-shadow: 0 4px 10px rgba(0, 0, 0, .06);
	}

	&__avatar {
		display: grid;
		place-items: center;
		flex-shrink: 0;
		inline-size: 2.75rem;
		block-size: 2.75rem;
		border-radius: 50%;
		font-weight: 700;
		font-size: .85rem;
		color: #ffffff;

		&--0 { background: var(--primary); }
		&--1 { background: hsl(146, 60%, 45%); }
		&--2 { background: hsl(38, 90%, 55%); }
		&--3 { background: hsl(268, 55%, 60%); }
	}

	&__name {
		font-weight: 600;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
}
</style>
