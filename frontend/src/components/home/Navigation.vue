<template>
	<aside
		:class="{'is-active': baseStore.menuActive, 'is-resizing': isResizing}"
		class="menu-container"
		:style="{'--sidebar-width': sidebarWidth}"
	>
		<div class="tst-profile">
			<span class="tst-profile__avatar">
				<Icon icon="user" />
			</span>
			<div class="tst-profile__meta">
				<span class="tst-profile__name">{{ authStore.userDisplayName || 'Toyota Staff' }}</span>
				<span class="tst-profile__role">Management Portal</span>
			</div>
		</div>

		<nav class="menu top-menu">
			<menu class="menu-list other-menu-items">
				<li>
					<RouterLink
						v-shortcut="'KeyG KeyO'"
						:to="{ name: 'dashboard'}"
					>
						<span class="menu-item-icon icon">
							<Icon icon="tachometer-alt" />
						</span>
						{{ $t('navigation.dashboard') }}
					</RouterLink>
				</li>
				<li>
					<RouterLink
						:to="{ name: 'tasks.prioritized'}"
					>
						<span class="menu-item-icon icon">
							<Icon icon="list-check" />
						</span>
						{{ $t('navigation.myTasks') }}
					</RouterLink>
				</li>
				<li>
					<RouterLink
						v-shortcut="'KeyG KeyM'"
						:to="{ name: 'teams.index'}"
					>
						<span class="menu-item-icon icon">
							<Icon icon="users" />
						</span>
						{{ $t('navigation.teamView') }}
					</RouterLink>
				</li>
				<li>
					<RouterLink
						v-shortcut="'KeyG KeyP'"
						:to="{ name: 'projects.index'}"
					>
						<span class="menu-item-icon icon">
							<Icon icon="layer-group" />
						</span>
						{{ $t('project.projects') }}
					</RouterLink>
				</li>
				<li>
					<RouterLink :to="{ name: 'time-tracking'}">
						<span class="menu-item-icon icon">
							<Icon :icon="['far', 'clock']" />
						</span>
						{{ $t('timeTracking.title') }}
					</RouterLink>
				</li>
				<li>
					<RouterLink :to="{ name: 'reports'}">
						<span class="menu-item-icon icon">
							<Icon icon="chart-bar" />
						</span>
						{{ $t('reports.title') }}
					</RouterLink>
				</li>
			</menu>
		</nav>

		<Loading
			v-if="projectStore.isLoading"
			variant="small"
		/>
		<template v-else>
			<nav
				v-if="favoriteProjects.length"
				class="menu"
			>
				<ProjectsNavigation
					:model-value="favoriteProjects"
					:can-edit-order="false"
					:can-collapse="false"
				/>
			</nav>
			
			<nav
				v-if="savedFilterProjects.length"
				class="menu"
			>
				<ProjectsNavigation
					:model-value="savedFilterProjects"
					:can-edit-order="false"
					:can-collapse="false"
				/>
			</nav>

			<nav class="menu">
				<ProjectsNavigation
					:model-value="projects"
					:can-edit-order="true"
					:can-collapse="true"
				/>
			</nav>
		</template>

		<div class="tst-create-wrap mbs-auto">
			<RouterLink
				v-if="createTaskTarget"
				:to="createTaskTarget"
				class="tst-create-btn"
			>
				<Icon icon="plus" />
				{{ $t('navigation.createTask') }}
			</RouterLink>
		</div>

		<div
			v-if="!isMobile"
			class="resize-handle"
			@mousedown="startResize"
			@touchstart="startResize"
		/>
	</aside>
</template>

<script setup lang="ts">
import {computed} from 'vue'

import Loading from '@/components/misc/Loading.vue'

import {useBaseStore} from '@/stores/base'
import {useProjectStore} from '@/stores/projects'
import {useAuthStore} from '@/stores/auth'
import ProjectsNavigation from '@/components/home/ProjectsNavigation.vue'
import type {IProject} from '@/modelTypes/IProject'
import {useSidebarResize} from '@/composables/useSidebarResize'

const baseStore = useBaseStore()
const projectStore = useProjectStore()
const authStore = useAuthStore()

const {sidebarWidth, isResizing, startResize, isMobile} = useSidebarResize()

// "Create Task" jumps to the user's default project (or first available), where the
// inline add-task input lives — Vikunja has no standalone global task-create screen.
// ponytail: routes to the list view; a true global quick-add modal is a later add.
const createTaskTarget = computed(() => {
	const defaultId = authStore.settings?.defaultProjectId
	const projectId = defaultId || projectStore.notArchivedRootProjects[0]?.id
	return projectId
		? {name: 'project.index', params: {projectId}}
		: {name: 'projects.index'}
})

// Cast readonly arrays to mutable type - the arrays are not actually mutated by the component
const projects = computed(() => projectStore.notArchivedRootProjects as IProject[])
const favoriteProjects = computed(() => projectStore.favoriteProjects as IProject[])
const savedFilterProjects = computed(() => projectStore.savedFilterProjects as IProject[])
</script>

<style lang="scss" scoped>
// Thirty Six Toyota sidebar — deep navy shell, white text, red active state.
$tst-navy: #001529;
$tst-red: #eb0a1e;

.menu-container {
	--sidebar-width: #{$navbar-width};

	display: flex;
	flex-direction: column;
	background: $tst-navy;
	color: rgba(255, 255, 255, .82);
	padding: 1rem 0 1.25rem;
	transition: transform $transition-duration ease-in;
	position: fixed;
	inset-block-start: $navbar-height;
	inset-block-end: 0;
	inset-inline-start: 0;
	transform: translateX(-100%);
	inline-size: var(--sidebar-width);
	overflow-y: auto;

	[dir="rtl"] & {
		transform: translateX(100%);
	}

	@media screen and (max-width: $tablet) {
		inset-block-start: 0;
		inline-size: 70vw;
		z-index: 20;
	}

	&.is-active {
		transform: translateX(0);
		transition: transform $transition-duration ease-out;
	}

	&.is-resizing {
		transition: none;
	}
}

.tst-profile {
	display: flex;
	align-items: center;
	gap: .75rem;
	padding: .25rem 1.5rem 1.25rem;
	margin-block-end: .75rem;
	border-block-end: 1px solid rgba(255, 255, 255, .1);

	&__avatar {
		display: grid;
		place-items: center;
		inline-size: 2.5rem;
		block-size: 2.5rem;
		border-radius: 999px;
		background: rgba(255, 255, 255, .12);
		color: #ffffff;
		font-size: 1.1rem;
		flex-shrink: 0;
	}

	&__meta {
		display: flex;
		flex-direction: column;
		min-inline-size: 0;
	}

	&__name {
		font-weight: 700;
		font-size: 1rem;
		color: #ffffff;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	&__role {
		font-size: .8rem;
		color: rgba(255, 255, 255, .6);
	}
}

.top-menu .menu-list {
	li {
		font-weight: 600;
		font-family: $vikunja-font;
	}

	.list-menu-link,
	li > a {
		display: flex;
		align-items: center;
		gap: .85rem;
		padding: .6rem 1.5rem;
		color: rgba(255, 255, 255, .82);
		border-inline-start: 3px solid transparent;
		transition: background-color .15s ease, color .15s ease;

		.menu-item-icon {
			opacity: .85;
		}

		&:hover {
			color: #ffffff;
			background: rgba(255, 255, 255, .08);
		}

		&.router-link-active {
			color: $tst-red;
			font-weight: 700;
			background: rgba(235, 10, 30, .1);
			border-inline-start-color: $tst-red;

			.menu-item-icon {
				opacity: 1;
			}
		}
	}
}

// Project / favorites tree below the main menu — lighten for the navy background.
.menu + .menu {
	padding-block-start: math.div($navbar-padding, 2);
}

:deep(.menu-label),
:deep(.menu-list a) {
	color: rgba(255, 255, 255, .7);
}

:deep(.menu-list a:hover) {
	color: #ffffff;
	background: rgba(255, 255, 255, .08);
}

.tst-create-wrap {
	padding: 1rem 1.25rem .25rem;
}

.tst-create-btn {
	display: flex;
	align-items: center;
	justify-content: center;
	gap: .5rem;
	inline-size: 100%;
	padding: .6rem 1rem;
	border-radius: 4px;
	background: $tst-red;
	color: #ffffff;
	font-weight: 700;
	font-size: .95rem;
	transition: background-color .15s ease;

	&:hover {
		background: #c00014;
		color: #ffffff;
	}
}

.resize-handle {
	position: absolute;
	inset-block-start: 0;
	inset-block-end: 0;
	inset-inline-end: 0;
	inline-size: 4px;
	cursor: ew-resize;
	background: transparent;
	transition: background-color $transition-duration ease;
	touch-action: none;

	&:hover,
	&:active {
		background-color: var(--primary);
	}
}
</style>
