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
					<RouterLink :to="{ name: 'projects.index'}">
						<span class="menu-item-icon icon">
							<Icon icon="layer-group" />
						</span>
						{{ $t('project.projects') }}
					</RouterLink>
				</li>
				<li>
					<RouterLink :to="{ name: 'discussion.index'}">
						<span class="menu-item-icon icon">
							<Icon icon="comments" />
						</span>
						{{ $t('navigation.discussion') }}
					</RouterLink>
				</li>
				<li>
					<RouterLink :to="{ name: 'documents.index'}">
						<span class="menu-item-icon icon">
							<Icon icon="folder" />
						</span>
						{{ $t('navigation.documents') }}
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
					<RouterLink :to="{ name: 'notes.index'}">
						<span class="menu-item-icon icon">
							<Icon icon="note-sticky" />
						</span>
						{{ $t('navigation.notes') }}
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
				<li>
					<RouterLink :to="{ name: 'users.index'}">
						<span class="menu-item-icon icon">
							<Icon icon="users" />
						</span>
						{{ $t('navigation.users') }}
					</RouterLink>
				</li>
				<li>
					<RouterLink
						v-shortcut="'KeyG KeyM'"
						:to="{ name: 'teams.index'}"
					>
						<span class="menu-item-icon icon">
							<Icon icon="sitemap" />
						</span>
						{{ $t('navigation.teamView') }}
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
// Thirty Six Toyota sidebar — Taskopad-style light shell, dark text, blue active state.
.menu-container {
	--sidebar-width: #{$navbar-width};

	display: flex;
	flex-direction: column;
	background: var(--white);
	color: var(--grey-600);
	border-inline-end: 1px solid var(--card-border-color);
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
	border-block-end: 1px solid var(--card-border-color);

	&__avatar {
		display: grid;
		place-items: center;
		inline-size: 2.5rem;
		block-size: 2.5rem;
		border-radius: 999px;
		background: hsla(var(--primary-hsl), .12);
		color: var(--primary);
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
		color: var(--grey-900);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	&__role {
		font-size: .8rem;
		color: var(--grey-400);
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
		color: var(--grey-600);
		border-inline-start: 3px solid transparent;
		transition: background-color .15s ease, color .15s ease;

		.menu-item-icon {
			opacity: .7;
		}

		&:hover {
			color: var(--grey-900);
			background: var(--grey-50);
		}

		&.router-link-active {
			color: var(--primary);
			font-weight: 700;
			background: hsla(var(--primary-hsl), .1);
			border-inline-start-color: var(--primary);

			.menu-item-icon {
				opacity: 1;
			}
		}
	}
}

// Colored nav icons — a distinct accent per section for quick scanning.
// Icons keep the inherited dim-when-inactive / full-when-active opacity, so
// the colors stay soft rather than neon. Text is left neutral for legibility.
.other-menu-items > li {
	&:nth-child(1) .menu-item-icon { color: var(--primary); } // Dashboard
	&:nth-child(2) .menu-item-icon { color: #6366f1; }        // My Tasks
	&:nth-child(3) .menu-item-icon { color: #14b8a6; }        // Projects
	&:nth-child(4) .menu-item-icon { color: #22c55e; }        // Discussion
	&:nth-child(5) .menu-item-icon { color: #f59e0b; }        // Documents
	&:nth-child(6) .menu-item-icon { color: #f97316; }        // Time tracking
	&:nth-child(7) .menu-item-icon { color: #eab308; }        // Notes
	&:nth-child(8) .menu-item-icon { color: #a855f7; }        // Reports
	&:nth-child(9) .menu-item-icon { color: #ec4899; }        // Users
	&:nth-child(10) .menu-item-icon { color: #ef4444; }       // Departments
}

// Project / favorites tree below the main menu.
.menu + .menu {
	padding-block-start: math.div($navbar-padding, 2);
}

:deep(.menu-label),
:deep(.menu-list a) {
	color: var(--grey-500);
}

:deep(.menu-list a:hover) {
	color: var(--grey-900);
	background: var(--grey-50);
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
	border-radius: var(--control-radius);
	background: var(--primary);
	color: #ffffff;
	font-weight: 700;
	font-size: .95rem;
	transition: background-color .15s ease;

	&:hover {
		background: hsla(var(--primary-h), var(--primary-s), calc(var(--primary-l) - 8%), 1);
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
