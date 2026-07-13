<template>
	<div class="users-view">
		<header class="users-header">
			<h1 class="users-title">
				{{ $t('navigation.users') }}
			</h1>
			<p class="users-subtitle">
				{{ users.length }} {{ $t('navigation.users').toLowerCase() }}
			</p>
		</header>

		<div
			v-if="loading"
			class="loader-container is-loading"
		/>
		<p
			v-else-if="users.length === 0"
			class="users-empty"
		>
			{{ $t('navigation.noUsers') }}
		</p>
		<div
			v-else
			class="users-grid"
		>
			<div
				v-for="user in users"
				:key="user.id"
				class="user-card"
			>
				<User
					:user="user"
					:avatar-size="44"
					:show-username="false"
				/>
				<div class="user-card__meta">
					<span class="user-card__name">{{ getDisplayName(user) }}</span>
					<span class="user-card__email">{{ user.email }}</span>
				</div>
			</div>
		</div>
	</div>
</template>

<script setup lang="ts">
import {useI18n} from 'vue-i18n'

import User from '@/components/misc/User.vue'
import {getDisplayName} from '@/models/user'
import {useTitle} from '@/composables/useTitle'
import {useOrgUsers} from '@/composables/useOrgUsers'

const {t} = useI18n({useScope: 'global'})
useTitle(() => t('navigation.users'))

// ponytail: skips per-user analytics (Taskopad shows charts here); nothing
// backs that data yet, add when a real per-user reporting endpoint exists.
const {loading, users} = useOrgUsers()
</script>

<style lang="scss" scoped>
.users-view {
	padding: 1.5rem;
	max-inline-size: 1100px;
	margin-inline: auto;
}

.users-header {
	margin-block-end: 1.5rem;
}

.users-title {
	font-size: 1.6rem;
	font-weight: 700;
	color: var(--grey-900);
	margin: 0;
}

.users-subtitle {
	color: var(--grey-500);
	font-size: .95rem;
	margin-block-start: .15rem;
	text-transform: capitalize;
}

.users-empty {
	color: var(--grey-400);
	font-style: italic;
}

.users-grid {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
	gap: 1rem;
}

.user-card {
	display: flex;
	align-items: center;
	gap: .85rem;
	padding: 1rem;
	background: var(--white);
	border: 1px solid var(--card-border-color);
	border-radius: 12px;
	box-shadow: 0 2px 4px rgba(0, 0, 0, .03);

	&__meta {
		display: flex;
		flex-direction: column;
		min-inline-size: 0;
	}

	&__name {
		font-weight: 600;
		color: var(--grey-900);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	&__email {
		font-size: .8rem;
		color: var(--grey-500);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
}
</style>
